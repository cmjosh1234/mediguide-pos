import 'dart:async';
import 'dart:io';
import 'package:flutter_test/flutter_test.dart';
import 'package:user_app/core/network/api_client.dart';
import 'package:user_app/features/calculators/data/repositories/calculator_local_repository.dart';
import 'package:user_app/features/calculators/data/repositories/calculator_repository.dart';
import 'package:user_app/features/calculators/data/repositories/calculator_usage_tracker.dart';
import 'helpers/test_local_store.dart';

class UsageApi extends BackendApiService {
  bool offline = true;
  bool loseStartResponse = false;
  bool loseEndResponse = false;
  Completer<void>? startGate;
  Completer<void>? startEntered;
  final starts = <Map<String, dynamic>>[];
  final ends = <Map<String, dynamic>>[];
  final records = <String, Map<String, dynamic>>{};

  @override
  Future<Map<String, dynamic>> requestJson(
    String path, {
    required String method,
    Map<String, dynamic>? body,
    Map<String, String>? query,
    bool includeAuth = true,
  }) async {
    expect(includeAuth, isTrue);
    if (offline) throw const SocketException('offline');
    if (path.contains('/bad-tool/')) {
      throw const FormatException('rejected session');
    }
    if (method == 'POST') {
      starts.add(Map.from(body!));
      startEntered?.complete();
      await startGate?.future;
      final key = body['idempotency_key'] as String;
      final record = records.putIfAbsent(
        key,
        () => {
          'id': 'server-$key',
          'user_id': 'owner',
          'calculator_id': 'tool',
          'session_start': body['session_start'],
          'calculator_type': body['calculator_type'],
        },
      );
      if (loseStartResponse) {
        loseStartResponse = false;
        throw const SocketException('response lost');
      }
      return {'data': record};
    }
    ends.add(Map.from(body!));
    final record = records.values.firstWhere(
      (r) => path.endsWith(r['id'] as String),
    );
    record['session_end'] = body['session_end'];
    if (loseEndResponse) {
      loseEndResponse = false;
      throw const SocketException('response lost');
    }
    return {'data': record};
  }
}

void main() {
  late TestLocalStore store;
  late CalculatorLocalRepository local;
  late UsageApi api;
  late CalculatorUsageTracker tracker;
  String? currentUser;
  final start = DateTime.utc(2026, 10, 6, 12);
  Future<String> begin() => tracker.begin(
    userId: 'owner',
    calculatorId: 'tool',
    versionId: 'cached-published-version',
    calculatorType: 'calculator',
    start: start,
  );
  Future<void> settle() async {
    // A sync request made while syncing joins the next pass; wait for persisted state.
    await Future<void>.delayed(const Duration(milliseconds: 30));
    await tracker.sync();
    await Future<void>.delayed(const Duration(milliseconds: 30));
  }

  setUp(() {
    store = TestLocalStore();
    local = CalculatorLocalRepository(store.cache);
    api = UsageApi();
    currentUser = 'owner';
    tracker = CalculatorUsageTracker(
      CalculatorRepository(api, local),
      local,
      () => currentUser,
    );
  });
  tearDown(() async {
    await settle();
    await store.close();
  });

  test(
    'offline short visit persists and replays once after tracker restart',
    () async {
      final id = await begin();
      await tracker.end('owner', id, start.add(const Duration(seconds: 1)));
      await settle();
      expect(await local.pendingUsage('owner'), hasLength(1));
      tracker = CalculatorUsageTracker(
        CalculatorRepository(api, local),
        local,
        () => currentUser,
      );
      api.offline = false;
      await settle();
      expect(api.starts, hasLength(1));
      expect(api.ends, hasLength(1));
      expect(
        api.starts.single['calculator_version_id'],
        'cached-published-version',
      );
      expect(api.starts.single['idempotency_key'], id);
      expect(
        api.ends.single['session_end'],
        start.add(const Duration(seconds: 1)).toIso8601String(),
      );
      expect(await local.pendingUsage('owner'), isEmpty);
    },
  );

  test(
    'lost start and end responses retain the same key and end on retry',
    () async {
      final id = await begin();
      await tracker.end('owner', id, start.add(const Duration(seconds: 2)));
      await settle();
      api.offline = false;
      api.loseStartResponse = true;
      await settle();
      expect(await local.pendingUsage('owner'), hasLength(1));
      api.loseEndResponse = true;
      await settle();
      expect(await local.pendingUsage('owner'), hasLength(1));
      await settle();
      expect(api.records, hasLength(1));
      expect(api.starts, hasLength(2));
      expect(api.starts.map((s) => s['idempotency_key']).toSet(), {id});
      expect(api.ends, hasLength(2));
      expect(api.ends.first, api.ends.last);
      expect(await local.pendingUsage('owner'), isEmpty);
    },
  );

  test(
    'closing while start request is slow saves original end immediately',
    () async {
      api.offline = false;
      api.startGate = Completer();
      api.startEntered = Completer();
      final id = await begin();
      await api.startEntered!.future;
      final end = start.add(const Duration(milliseconds: 300));
      await tracker.end('owner', id, end);
      expect(
        (await local.getUsage('owner', id))!['session_end'],
        end.toIso8601String(),
      );
      api.startGate!.complete();
      api.startEntered = null;
      await settle();
      expect(api.ends.single['session_end'], end.toIso8601String());
      expect(await local.pendingUsage('owner'), isEmpty);
    },
  );

  test('does not replay another user queue after account switch', () async {
    final id = await begin();
    await tracker.end('owner', id, start.add(const Duration(seconds: 1)));
    await settle();
    currentUser = 'another-user';
    api.offline = false;
    await settle();
    expect(api.starts, isEmpty);
    currentUser = null;
    await settle();
    expect(api.starts, isEmpty);
    currentUser = 'owner';
    await settle();
    expect(api.starts, hasLength(1));
  });

  test(
    'active session is not resent on every sync and duplicate end is stable',
    () async {
      api.offline = false;
      final id = await begin();
      await settle();
      await settle();
      expect(api.starts, hasLength(1));
      expect(api.ends, isEmpty);
      await tracker.end('owner', id, start.add(const Duration(seconds: 1)));
      await tracker.end('owner', id, start.add(const Duration(seconds: 9)));
      await settle();
      expect(api.ends, hasLength(1));
      expect(
        api.ends.single['session_end'],
        start.add(const Duration(seconds: 1)).toIso8601String(),
      );
    },
  );
  test('one rejected session does not block other queued visits', () async {
    currentUser = null;
    final bad = await tracker.begin(
      userId: 'owner',
      calculatorId: 'bad-tool',
      versionId: 'invalid-version',
      calculatorType: 'calculator',
      start: start,
    );
    await tracker.end('owner', bad, start.add(const Duration(seconds: 1)));
    final good = await begin();
    await tracker.end('owner', good, start.add(const Duration(seconds: 1)));
    currentUser = 'owner';
    api.offline = false;
    await settle();
    expect(api.records, hasLength(1));
    expect(api.ends, hasLength(1));
    expect((await local.pendingUsage('owner')).single['id'], bad);
  });
}
