import 'dart:async';
import 'package:user_app/core/utils/app_message.dart';
import 'package:user_app/features/notifications/presentation/controllers/notification_sync.dart';
import 'package:user_app/shared/providers/usage_tracking_provider.dart';
import 'package:user_app/features/calculators/data/repositories/calculator_usage_tracker.dart';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:responsive_framework/responsive_framework.dart';
import 'package:user_app/app/router/app_router.dart';
import 'package:user_app/app/providers/app_providers.dart';
import 'package:user_app/app/theme/app_theme.dart';
import 'package:user_app/core/config/app_keys.dart';
import 'package:user_app/core/utils/common.dart';
import 'package:user_app/features/authentication/presentation/controllers/auth_controller.dart';
import 'package:user_app/features/settings/presentation/controllers/app_settings_controller.dart';
import 'package:user_app/shared/providers/connectivity_provider.dart';
import 'package:user_app/features/settings/presentation/controllers/language_controller.dart';
import 'package:user_app/l10n/app_translations.dart';
import 'package:user_app/core/debug/debug_tools_overlay.dart';
import 'package:user_app/features/notifications/domain/notification_action_resolver.dart';
import 'package:user_app/features/home/presentation/screens/guest_home_page.dart';
import 'package:user_app/features/home/presentation/screens/home_page.dart';
import 'package:user_app/features/outbreaks/presentation/providers/outbreak_providers.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:user_app/features/settings/presentation/widgets/app_update_coordinator.dart';

class MediGuideApp extends ConsumerWidget {
  const MediGuideApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    ref.watch(authControllerProvider);
    ref.watch(backendReconnectProvider);
    ref.watch(notificationSyncProvider);
    ref.watch(calculatorUsageSyncProvider);
    ref.watch(usageTrackingProvider);
    final router = ref.watch(appRouterProvider);
    ref.listen(outbreakBannerEnabledProvider, (_, next) {
      if (!next.hasValue) return;
      ref.invalidate(guestHomeOutbreaksProvider);
      ref.invalidate(homeOutbreakBannerProvider);
      ref.invalidate(publicOutbreaksProvider);
      ref.invalidate(publicSituationReportsProvider);
    });
    ref.listen(firebaseOpenedMessageProvider, (_, message) {
      final data = message.valueOrNull?.data;
      if (data == null) return;
      _invalidateOutbreakContent(ref, data);
      final target = NotificationActionResolver.fromPushData(data);
      final deliveryId = data['delivery_id']?.toString().trim() ?? '';
      final notificationId = data['notification_id']?.toString().trim() ?? '';
      final repository = ref.read(notificationRepositoryProvider);
      final sync = ref.read(notificationSyncProvider);
      final suffix = data['message_id']?.toString() ?? deliveryId;
      unawaited(
        (() async {
          try {
            if (repository.userId.trim().isNotEmpty) {
              if (notificationId.isNotEmpty) {
                await repository.markRead(notificationId);
              }
              if (deliveryId.isNotEmpty) {
                await repository.recordOpen(
                  deliveryId,
                  eventId: 'push-open-$suffix',
                );
              }
            }
          } catch (_) {
            // Opening content must work even when read-state sync fails.
          } finally {
            unawaited(sync.sync(force: true));
          }
        })(),
      );
      unawaited(
        (() async {
          try {
            var opened = false;
            if (target?.location case final location?) {
              router.push(location);
              opened = true;
            } else if (target?.externalUri case final uri?) {
              opened = await launchUrl(
                uri,
                mode: LaunchMode.externalApplication,
              );
            } else {
              router.push('/notifications');
            }
            if (opened &&
                deliveryId.isNotEmpty &&
                repository.userId.trim().isNotEmpty) {
              await repository.recordClick(
                deliveryId,
                eventId: 'push-click-$suffix',
              );
            }
          } catch (_) {
            final context = AppKeys.navigatorKey.currentContext;
            if (context != null && context.mounted) {
              AppMessage.error(
                context,
                'Unable to open this update. It remains in your inbox.',
              );
            }
          }
        })(),
      );
    });
    ref.listen(firebaseForegroundMessageProvider, (_, message) {
      final foreground = message.valueOrNull;
      if (foreground == null) return;
      _invalidateOutbreakContent(ref, foreground.data);
      unawaited(ref.read(notificationSyncProvider).sync(force: true));
      if (foreground.notification == null ||
          !ref.read(firebaseServiceProvider).pushFeatureEnabled) {
        return;
      }
      // Urgent alerts retain their OS channel; routine updates stay discreet.
      if (foreground.notification?.android?.channelId ==
              'mediguide_emergency' ||
          foreground.data['priority'] == 'urgent' ||
          foreground.data['priority'] == 'high') {
        return;
      }
      final context = AppKeys.navigatorKey.currentContext;
      if (context == null || !context.mounted) return;
      final target = NotificationActionResolver.fromPushData(foreground.data);
      if (target?.location ==
          router.routeInformationProvider.value.uri.toString()) {
        return;
      }
      AppMessage.show(
        context,
        message: foreground.notification?.title ?? 'New update in your inbox',
        duration: const Duration(seconds: 6),
        actionLabel: 'View',
        onAction: () => router.push('/notifications'),
        replaceCurrent: false,
      );
    });
    final languageCode =
        ref.watch(languageControllerProvider).valueOrNull?.currentCode ?? 'en';
    AppTranslation.setLocale(Locale(languageCode));
    final themeMode = ref.watch(
      appSettingsControllerProvider.select(
        (settings) => settings.materialThemeMode,
      ),
    );

    return GestureDetector(
      onTap: Common.dismissKeyboard,
      child: MaterialApp.router(
        routerConfig: router,
        scaffoldMessengerKey: AppKeys.scaffoldMessengerKey,
        theme: AppTheme.light,
        darkTheme: AppTheme.dark,
        themeMode: themeMode,
        locale: AppTranslation.locale,
        debugShowCheckedModeBanner: false,
        builder: (context, child) => AppUpdateCoordinator(
          child: DebugToolsOverlay(
            child: ResponsiveBreakpoints.builder(
              child: child!,
              breakpoints: const [
                Breakpoint(start: 0, end: 450, name: MOBILE),
                Breakpoint(start: 451, end: 800, name: TABLET),
                Breakpoint(start: 801, end: 1920, name: DESKTOP),
                Breakpoint(start: 1921, end: double.infinity, name: '4K'),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

void _invalidateOutbreakContent(WidgetRef ref, Map<String, dynamic> data) {
  final type = [data['action_type'], data['resource_type'], data['type']]
      .whereType<Object>()
      .map((value) => value.toString().toLowerCase())
      .join(' ');
  if (!type.contains('outbreak') && !type.contains('situation_report')) return;
  ref.invalidate(guestHomeOutbreaksProvider);
  ref.invalidate(homeOutbreakBannerProvider);
  ref.invalidate(publicOutbreaksProvider);
  ref.invalidate(publicSituationReportsProvider);
  final resourceId = data['resource_id']?.toString().trim() ?? '';
  if (resourceId.isEmpty) return;
  if (type.contains('situation_report')) {
    ref.invalidate(publicSituationReportProvider(resourceId));
  } else {
    ref.invalidate(publicOutbreakProvider(resourceId));
  }
}
