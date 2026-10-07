import 'dart:io';
import 'package:flutter_test/flutter_test.dart';
import 'package:user_app/core/network/api_client.dart';
import 'package:user_app/features/guidelines/data/repositories/progress_usage_repository.dart';
import 'package:user_app/shared/providers/usage_tracking_provider.dart';
import 'package:user_app/app/router/route_names.dart';
import 'helpers/test_local_store.dart';

class QueuedUsageApi extends BackendApiService {
  bool offline = true;
  bool loseResponse = false;
  final calls = <Map<String, dynamic>>[];
  final keys = <String>{};
  @override
  Future<Map<String, dynamic>> requestJson(
    String path, {
    required String method,
    Map<String, dynamic>? body,
    Map<String, String>? query,
    bool includeAuth = true,
  }) async {
    expect(includeAuth, isTrue);
    expect(body?.containsKey('user_id'), isFalse);
    if (offline) throw const SocketException('offline');
    calls.add({'path': path, 'body': Map<String, dynamic>.from(body!)});
    keys.add((body['idempotency_key'] ?? body['event_id']) as String);
    if (loseResponse) {
      loseResponse = false;
      throw const SocketException('lost response');
    }
    return {'success': true};
  }
}

void main() {
  test('all content events survive offline and repository restart', () async {
    final store = TestLocalStore();
    addTearDown(store.close);
    final api = QueuedUsageApi();
    var usage = UsageRepository(api, store.cache, () => 'owner');
    await usage.guideline('document');
    await usage.abbreviation('abbreviation');
    await usage.drug('drug');
    await usage.facility('facility');
    await usage.ai();
    await usage.feature('outbreaks');
    await usage.sync();
    expect(
      await store.cache.list(type: 'usage_pending', scope: 'user:owner'),
      hasLength(6),
    );
    usage = UsageRepository(api, store.cache, () => 'owner');
    api.offline = false;
    await usage.sync();
    expect(api.keys, hasLength(6));
    expect(
      await store.cache.list(type: 'usage_pending', scope: 'user:owner'),
      isEmpty,
    );
    final guidelines = api.calls
        .where((c) => c['path'] == '/api/v2/usage/guidelines')
        .toList();
    expect(guidelines.map((c) => c['body']['resource_type']).toSet(), {
      'guideline_document',
    });
    expect(
      api.calls.map((c) => c['path']).toSet(),
      containsAll([
        '/api/v2/drugs/drug/usage',
        '/api/v2/facilities/facility/usage',
        '/api/v2/usage/abbreviations',
        '/api/v2/usage/ai',
        '/api/v2/usage/features',
      ]),
    );
  });
  test(
    'upgrade removes retired guideline events while syncing document usage',
    () async {
      final store = TestLocalStore();
      addTearDown(store.close);
      final api = QueuedUsageApi()..offline = false;
      await store.cache.put(
        type: 'usage_pending',
        id: 'old-reader-event',
        scope: 'user:owner',
        data: {
          'id': 'old-reader-event',
          'path': '/api/v2/usage/guidelines',
          'body': {
            'resource_type': 'medical_guideline',
            'resource_id': 'retired-id',
            'idempotency_key': 'old-reader-event',
          },
        },
      );
      final usage = UsageRepository(api, store.cache, () => 'owner');
      await usage.guideline('document-id');
      await usage.sync();
      expect(api.calls, hasLength(1));
      expect(api.calls.single['body']['resource_type'], 'guideline_document');
      expect(
        await store.cache.list(type: 'usage_pending', scope: 'user:owner'),
        isEmpty,
      );
    },
  );
  test('lost response is replayed with unchanged key', () async {
    final store = TestLocalStore();
    addTearDown(store.close);
    final api = QueuedUsageApi();
    final usage = UsageRepository(api, store.cache, () => 'owner');
    await usage.ai();
    await usage.sync();
    api.offline = false;
    api.loseResponse = true;
    await usage.sync();
    expect(
      await store.cache.list(type: 'usage_pending', scope: 'user:owner'),
      hasLength(1),
    );
    await usage.sync();
    expect(api.calls, hasLength(2));
    expect(api.keys, hasLength(1));
    expect(
      await store.cache.list(type: 'usage_pending', scope: 'user:owner'),
      isEmpty,
    );
  });
  test('anonymous usage and another account never send owner events', () async {
    final store = TestLocalStore();
    addTearDown(store.close);
    final api = QueuedUsageApi();
    String? owner;
    final usage = UsageRepository(api, store.cache, () => owner);
    await usage.ai();
    expect(
      await store.cache.list(type: 'usage_pending', scope: 'user:owner'),
      isEmpty,
    );
    owner = 'owner';
    await usage.drug('drug');
    await usage.sync();
    owner = 'other';
    api.offline = false;
    await usage.sync();
    expect(api.calls, isEmpty);
    owner = 'owner';
    await usage.sync();
    expect(api.calls, hasLength(1));
  });
  test(
    'feature mapping covers public app destinations without sensitive route data',
    () {
      for (final route in AppRoutes.publicRoutes) {
        if ([
          '/',
          '/login',
          '/register',
          '/onboarding',
          '/forgot-password',
          '/reset-password',
          '/verify-email',
        ].contains(route)) {
          continue;
        }
        expect(usageFeatureForLocation(route), isNotNull, reason: route);
      }
      expect(
        usageFeatureForLocation('/guidelines/private-id?query=private+text'),
        'guidelines',
      );
      expect(usageFeatureForLocation('/chats/private-id'), 'conversations');
      expect(
        usageFeatureForLocation('/reset-password?token=private-token'),
        isNull,
      );
      expect(
        usageFeatureForLocation('/verify-email?token=private-token'),
        isNull,
      );
      expect(
        usageFeatureForLocation('/outbreak-hub/private-id/sections/another-id'),
        'outbreaks',
      );
      expect(
        usageFeatureForLocation('/clinical-tools/review/private-id'),
        isNull,
      );
    },
  );
  test(
    'notification delivery events persist and retain stable event IDs on retry',
    () async {
      final store = TestLocalStore();
      addTearDown(store.close);
      final api = QueuedUsageApi();
      final usage = UsageRepository(api, store.cache, () => 'owner');
      await usage.notificationDelivery(
        ownerId: 'other',
        deliveryId: 'delivery',
        eventType: 'open',
        eventId: 'wrong-owner',
      );
      await usage.notificationDelivery(
        ownerId: 'owner',
        deliveryId: 'delivery',
        eventType: 'open',
        eventId: 'push-open',
      );
      await usage.sync();
      final pending = await store.cache.list(
        type: 'usage_pending',
        scope: 'user:owner',
      );
      expect(pending, hasLength(1));
      final occurredAt = pending.single['body']['occurred_at'];
      api.offline = false;
      api.loseResponse = true;
      await usage.sync();
      await usage.sync();
      expect(api.keys, {'push-open'});
      expect(api.calls.map((c) => c['body']['occurred_at']).toSet(), {
        occurredAt,
      });
      expect(
        await store.cache.list(type: 'usage_pending', scope: 'user:owner'),
        isEmpty,
      );
    },
  );
}
