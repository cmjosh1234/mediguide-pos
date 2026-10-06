import 'package:responsive_framework/responsive_framework.dart';
import 'package:user_app/core/storage/local_cache_service.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:user_app/app/providers/app_providers.dart';
import 'package:user_app/app/router/app_router.dart';
import 'package:user_app/features/authentication/data/models/user.dart';
import 'package:user_app/features/authentication/presentation/controllers/auth_controller.dart';
import 'package:user_app/features/authentication/presentation/controllers/auth_state.dart';
import 'package:user_app/features/navigation/presentation/controllers/main_navigation_controller.dart';
import 'package:user_app/features/drugs/data/models/drug.dart';
import 'package:user_app/features/drugs/presentation/widgets/drug_details_bottom_sheet.dart';
import 'package:user_app/features/abbreviations/data/models/abbreviation.dart';
import 'package:user_app/features/abbreviations/presentation/widgets/abbreviation_detail_modal.dart';
import 'package:user_app/features/ai_assistant/data/services/ai_context_service.dart';
import 'package:user_app/features/guidelines/data/repositories/progress_usage_repository.dart';
import 'package:user_app/shared/providers/connectivity_provider.dart';
import 'package:user_app/shared/providers/usage_tracking_provider.dart';
import 'helpers/test_local_store.dart';
import 'usage_coverage_test.dart' show QueuedUsageApi;

class MemoryUsageCache extends LocalCacheService {
  MemoryUsageCache(super.database);
  final rows = <String, Map<String, dynamic>>{};
  @override
  Future<void> put({
    required String type,
    required String id,
    required Map<String, dynamic> data,
    String scope = 'public',
    String searchableText = '',
    Map<String, dynamic>? metadata,
    String? version,
    DateTime? remoteUpdatedAt,
    Duration? ttl,
  }) async {
    rows['$scope/$type/$id'] = data;
  }

  @override
  Future<List<Map<String, dynamic>>> list({
    required String type,
    String scope = 'public',
    String? search,
    int limit = 30,
    int offset = 0,
  }) async => rows.entries
      .where((e) => e.key.startsWith('$scope/$type/'))
      .map((e) => e.value)
      .toList();
  @override
  Future<Map<String, dynamic>?> get({
    required String type,
    required String id,
    String scope = 'public',
  }) async => rows['$scope/$type/$id'];
  @override
  Future<void> tombstone({
    required String type,
    required String id,
    String scope = 'public',
  }) async {
    rows.remove('$scope/$type/$id');
  }
}

class UsageAuth extends AuthController {
  @override
  Future<AuthState> build() async =>
      const AuthState.authenticated(User(id: 'owner'));
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  test(
    'route visits and main tabs record once without route IDs or query text',
    () async {
      final store = TestLocalStore();
      addTearDown(store.close);
      final api = QueuedUsageApi()..offline = false;
      final usage = UsageRepository(api, store.cache, () => 'owner');
      final router = GoRouter(
        initialLocation: '/main',
        routes: [
          GoRoute(
            path: '/main',
            builder: (_, _) => const Scaffold(body: Text('Main')),
          ),
          GoRoute(
            path: '/outbreak-hub/:id',
            builder: (_, _) => const Scaffold(body: Text('Outbreak')),
          ),
        ],
      );
      addTearDown(router.dispose);
      final container = ProviderContainer(
        overrides: [
          authControllerProvider.overrideWith(UsageAuth.new),
          appRouterProvider.overrideWithValue(router),
          usageRepositoryProvider.overrideWithValue(usage),
          connectivityProvider.overrideWith((_) => Stream.value(true)),
        ],
      );
      addTearDown(container.dispose);
      await container.read(authControllerProvider.future);
      container.read(usageTrackingProvider);
      await usage.sync();
      container.read(mainNavigationIndexProvider.notifier).state = 1;
      await usage.sync();
      container.read(mainNavigationIndexProvider.notifier).state = 1;
      await usage.sync();
      router.go('/outbreak-hub/private-id?query=private-text');
      await usage.sync();
      expect(api.calls.map((c) => c['body']['feature']).toList(), [
        'home',
        'search',
        'outbreaks',
      ]);
      for (final call in api.calls) {
        expect(call.toString(), isNot(contains('private-id')));
        expect(call.toString(), isNot(contains('private-text')));
      }
    },
  );
  testWidgets('drug and abbreviation details record one event per opening', (
    tester,
  ) async {
    final store = TestLocalStore();
    addTearDown(store.close);
    final api = QueuedUsageApi()..offline = false;
    final usage = UsageRepository(
      api,
      MemoryUsageCache(store.database),
      () => 'owner',
    );
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          usageRepositoryProvider.overrideWithValue(usage),
          aiContextServiceProvider.overrideWithValue(AiContextService()),
        ],
        child: MaterialApp(
          builder: (context, child) => ResponsiveBreakpoints.builder(
            child: child!,
            breakpoints: const [
              Breakpoint(start: 0, end: 599, name: MOBILE),
              Breakpoint(start: 600, end: double.infinity, name: TABLET),
            ],
          ),
          home: Builder(
            builder: (context) => Scaffold(
              body: Column(
                children: [
                  TextButton(
                    onPressed: () => DrugDetailsBottomSheet.show(
                      context: context,
                      drug: const Drug(id: 'drug', name: 'Test drug'),
                    ),
                    child: const Text('Open drug'),
                  ),
                  TextButton(
                    onPressed: () => AbbreviationDetailModal.show(
                      context,
                      const Abbreviation(
                        id: 'abbreviation',
                        abbreviation: 'ABC',
                        meaning: 'Meaning',
                      ),
                    ),
                    child: const Text('Open abbreviation'),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
    await tester.tap(find.text('Open drug'));
    await tester.pumpAndSettle();
    await usage.sync();
    expect(api.calls, hasLength(1));
    await tester.pump();
    await usage.sync();
    expect(api.calls, hasLength(1));
    Navigator.of(tester.element(find.byType(DrugDetailsBottomSheet))).pop();
    await tester.pumpAndSettle();
    await tester.tap(find.text('Open abbreviation'));
    await tester.pumpAndSettle();
    await usage.sync();
    expect(api.calls, hasLength(2));
    expect(api.calls.map((c) => c['path']).toList(), [
      '/api/v2/drugs/drug/usage',
      '/api/v2/usage/abbreviations',
    ]);
    await tester.pumpWidget(const SizedBox());
  });
}
