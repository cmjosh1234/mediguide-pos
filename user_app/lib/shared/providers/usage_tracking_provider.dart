import 'dart:async';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:user_app/app/providers/app_providers.dart';
import 'package:user_app/app/router/app_router.dart';
import 'package:user_app/features/authentication/presentation/controllers/auth_controller.dart';
import 'connectivity_provider.dart';
import 'package:user_app/features/navigation/presentation/controllers/main_navigation_controller.dart';

// Only feature names leave the device: never route IDs, search terms or URLs.
String? usageFeatureForLocation(String location) {
  final path = Uri.tryParse(location)?.path ?? '';
  final first = path.split('/').where((part) => part.isNotEmpty).firstOrNull;
  return switch (first) {
    'main' || 'home' => 'home',
    'guidelines' || 'guidelines-indexer' => 'guidelines',
    'public' when path.startsWith('/public/guidelines') => 'guidelines',
    'drug-index' => 'drugs',
    'health-infrastructure' || 'health-facilities' => 'facilities',
    'ministry-directory' => 'ministry_directory',
    'abbreviations' => 'abbreviations',
    'ai-assistant' => 'ai',
    'tools' || 'calculators' => 'tools',
    'document-reader' || 'generic-viewer' || 'viewer' => 'documents',
    'offline-content' => 'downloads',
    'library' => 'library',
    'outbreak-hub' || 'situation-reports' => 'outbreaks',
    'search' => 'search',
    'notifications' || 'notification-preferences' => 'notifications',
    'chats' => 'conversations',
    'profile' => 'profile',
    'faq' || 'help-center' || 'about-us' || 'terms-and-conditions' => 'support',
    'diseases' || 'hubs' => 'discovery',
    'more' || 'all-actions' => 'more',
    _ => null,
  };
}

final usageTrackingProvider = Provider<void>((ref) {
  final repository = ref.watch(usageRepositoryProvider);
  final router = ref.watch(appRouterProvider);
  String? lastPath;
  String? lastUser;
  void recordVisit() {
    final userId = ref.read(authControllerProvider).valueOrNull?.user?.id;
    final routePath = router.routeInformationProvider.value.uri.path;
    final mainIndex = ref.read(mainNavigationIndexProvider).clamp(0, 4);
    final path = routePath == '/main' ? '$routePath:$mainIndex' : routePath;
    if (userId == null) {
      lastUser = null;
      lastPath = null;
      return;
    }
    if (userId == lastUser && path == lastPath) return;
    lastUser = userId;
    lastPath = path;
    final feature = routePath == '/main'
        ? const ['home', 'search', 'library', 'tools', 'profile'][mainIndex]
        : usageFeatureForLocation(path);
    if (feature != null) unawaited(repository.feature(feature));
  }

  router.routeInformationProvider.addListener(recordVisit);
  ref.onDispose(
    () => router.routeInformationProvider.removeListener(recordVisit),
  );
  ref.listen(mainNavigationIndexProvider, (_, _) => recordVisit());
  ref.listen(authControllerProvider, (_, next) {
    recordVisit();
    if (next.valueOrNull?.user != null) unawaited(repository.sync());
  });
  ref.listen(connectivityProvider, (_, next) {
    if (next.valueOrNull == true) unawaited(repository.sync());
  });
  final timer = Timer.periodic(
    const Duration(minutes: 1),
    (_) => unawaited(repository.sync()),
  );
  ref.onDispose(timer.cancel);
  recordVisit();
  unawaited(repository.sync());
});
