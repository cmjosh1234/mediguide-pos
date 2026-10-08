import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:user_app/app/providers/app_providers.dart';
import 'package:user_app/core/network/api_client.dart';
import 'package:user_app/features/discovery/data/repositories/discovery_repository.dart';
import 'package:user_app/features/discovery/presentation/screens/discovery_pages.dart';
import 'package:user_app/features/discovery/presentation/widgets/discovery_widgets.dart';

import '../../helpers/test_local_store.dart';

// Keeps every request pending so the pages stay in their loading state.
final class _PendingApi extends BackendApiService {
  @override
  Future<Map<String, dynamic>> requestJson(
    String path, {
    required String method,
    Map<String, dynamic>? body,
    Map<String, String>? query,
    bool includeAuth = true,
  }) => Completer<Map<String, dynamic>>().future;
}

void main() {
  // The directory pages show their loading skeleton inside a scrolling list.
  // A nested ListView there failed layout in debug builds ("Vertical viewport
  // was given unbounded height"), which left the content hubs list blank.
  for (final (name, page) in [
    ('content hub directory', const ContentHubDirectoryPage()),
    ('disease directory', const DiseaseDirectoryPage()),
  ]) {
    testWidgets('$name lays out its loading state', (tester) async {
      final store = TestLocalStore();
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            discoveryRepositoryProvider.overrideWithValue(
              DiscoveryRepository(_PendingApi(), store.cache),
            ),
            diseaseHubsEnabledProvider.overrideWithValue(true),
            genericHubsEnabledProvider.overrideWithValue(true),
          ],
          child: MaterialApp(home: page),
        ),
      );
      await tester.pump(const Duration(milliseconds: 300));

      expect(tester.takeException(), isNull);
      expect(find.byType(DiscoverySkeleton), findsOneWidget);

      await tester.pumpWidget(const SizedBox());
      await tester.runAsync(store.close);
    });
  }
}
