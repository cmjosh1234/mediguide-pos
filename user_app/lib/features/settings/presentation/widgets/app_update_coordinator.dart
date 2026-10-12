import 'dart:async';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:user_app/core/config/app_config.dart';
import 'package:user_app/core/config/app_keys.dart';
import 'package:user_app/core/utils/app_message.dart';
import 'package:user_app/features/settings/data/app_update_client.dart';
import 'package:user_app/features/settings/presentation/controllers/app_update_controller.dart';

final automaticAppUpdatesEnabledProvider = Provider<bool>(
  (ref) =>
      kReleaseMode && (AppConfig.current.isProduction || firebaseTesterUpdates),
);

class AppUpdateCoordinator extends ConsumerStatefulWidget {
  const AppUpdateCoordinator({required this.child, super.key});
  final Widget child;
  @override
  ConsumerState<AppUpdateCoordinator> createState() =>
      _AppUpdateCoordinatorState();
}

class _AppUpdateCoordinatorState extends ConsumerState<AppUpdateCoordinator> {
  AppLifecycleListener? _lifecycle;
  bool _foreground = true;
  AppUpdateResult? _shownNotice;
  @override
  void initState() {
    super.initState();
    _lifecycle = AppLifecycleListener(
      onResume: () {
        _foreground = true;
        _check();
        _showNotice(ref.read(appUpdateControllerProvider).valueOrNull);
      },
      onInactive: () => _foreground = false,
    );
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) _check();
    });
  }

  void _check() {
    if (!ref.read(automaticAppUpdatesEnabledProvider)) return;
    unawaited(
      ref.read(appUpdateControllerProvider.notifier).check(automatic: true),
    );
  }

  void _showNotice(AppUpdateResult? result) {
    if (result == null) {
      _shownNotice = null;
      return;
    }
    if (!_foreground || !mounted) return;
    if (_shownNotice == result) return;
    final message = switch (result) {
      AppUpdateResult.available =>
        'A MediGuide update is available. You can keep using the app while it downloads.',
      AppUpdateResult.storeUpdateAvailable =>
        'A MediGuide update is available on Google Play.',
      AppUpdateResult.criticalAvailable =>
        'An important MediGuide update is available. Update when you are ready to restart.',
      AppUpdateResult.readyToInstall =>
        'Your update is ready. Restart when you have finished your current task.',
      AppUpdateResult.failed =>
        'The update could not finish. You can retry from Profile.',
      _ => null,
    };
    if (message == null) {
      _shownNotice = null;
      return;
    }
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted || !_foreground) return;
      if (AppKeys.scaffoldMessengerKey.currentState == null) return;
      _shownNotice = result;
      AppMessage.show(
        context,
        message: message,
        duration: result == AppUpdateResult.failed
            ? const Duration(seconds: 6)
            : const Duration(days: 1),
        showCloseIcon: true,
        actionLabel: result == AppUpdateResult.failed
            ? null
            : result == AppUpdateResult.readyToInstall
            ? 'Restart'
            : 'Update',
        onAction: result == AppUpdateResult.failed
            ? null
            : () {
                final controller = ref.read(
                  appUpdateControllerProvider.notifier,
                );
                if (result == AppUpdateResult.readyToInstall) {
                  unawaited(controller.installUpdate());
                } else if (result == AppUpdateResult.storeUpdateAvailable) {
                  unawaited(_openPlayStore());
                } else {
                  unawaited(controller.startUpdate());
                }
              },
      );
    });
    // Stream/lifecycle callbacks can arrive while the reader is idle and may
    // not otherwise request a frame for the queued notification.
    WidgetsBinding.instance.ensureVisualUpdate();
  }

  Future<void> _openPlayStore() async {
    try {
      await launchUrl(
        Uri.https('play.google.com', '/store/apps/details', {
          'id': 'com.mediguide.ug',
        }),
        mode: LaunchMode.externalApplication,
      );
    } catch (_) {
      /* The reader remains usable if the store cannot open. */
    }
  }

  @override
  Widget build(BuildContext context) {
    ref.listen(
      appUpdateControllerProvider,
      (_, next) => _showNotice(next.valueOrNull),
    );
    return widget.child;
  }

  @override
  void dispose() {
    _lifecycle?.dispose();
    super.dispose();
  }
}
