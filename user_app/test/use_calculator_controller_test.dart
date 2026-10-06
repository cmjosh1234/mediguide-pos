import 'dart:async';
import 'dart:io';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:user_app/app/providers/app_providers.dart';
import 'package:user_app/core/network/api_client.dart';
import 'package:user_app/features/authentication/data/models/user.dart';
import 'package:user_app/features/authentication/presentation/controllers/auth_controller.dart';
import 'package:user_app/features/authentication/presentation/controllers/auth_state.dart';
import 'package:user_app/features/calculators/data/models/calculator.dart';
import 'package:user_app/features/calculators/data/repositories/calculator_local_repository.dart';
import 'package:user_app/features/calculators/data/repositories/calculator_repository.dart';
import 'package:user_app/features/calculators/data/repositories/calculator_usage_tracker.dart';
import 'package:user_app/features/calculators/presentation/controllers/use_calculator_controller.dart';
import 'helpers/test_local_store.dart';

class TestAuth extends AuthController {
  TestAuth(this.user);
  final User? user;
  @override
  Future<AuthState> build() async => user == null
      ? const AuthState.unauthenticated()
      : AuthState.authenticated(user!);
}

class DefinitionApi extends BackendApiService {
  bool fail = false;
  Completer<void>? gate;
  @override
  Future<Map<String, dynamic>> requestJson(
    String path, {
    required String method,
    Map<String, dynamic>? body,
    Map<String, String>? query,
    bool includeAuth = true,
  }) async {
    await gate?.future;
    if (fail) throw const SocketException('definition unavailable');
    return {
      'data': {
        'calculator_id': 'tool',
        'version_id': 'published-version',
        'runtime_type': 'schema_v1',
        'semantic_version': '1.0.0',
        'definition_checksum': 'checksum',
        'definition': {
          'schema_version': '1.0',
          'tool_type': 'calculator',
          'title': 'BMI',
          'version': '1.0.0',
          'locale': 'en',
          'inputs': [],
          'sections': [],
          'calculation': [],
          'rules': [],
          'outputs': [],
          'interpretations': [],
          'completion': {'mode': 'none', 'reset_confirmation': true},
          'test_cases': [],
        },
      },
    };
  }
}

class CapturingTracker extends CalculatorUsageTracker {
  CapturingTracker(
    CalculatorRepository repository,
    CalculatorLocalRepository local,
  ) : super(repository, local, () => 'owner');
  int starts = 0;
  final versions = <String>[];
  final ends = <DateTime>[];
  Completer<String>? gate;
  @override
  Future<String> begin({
    required String userId,
    required String calculatorId,
    required String versionId,
    required String calculatorType,
    required DateTime start,
  }) async {
    starts++;
    versions.add(versionId);
    return gate == null ? 'session' : gate!.future;
  }

  @override
  Future<void> end(String userId, String id, DateTime end) async {
    ends.add(end);
  }
}

void main() {
  late TestLocalStore store;
  late DefinitionApi api;
  late CapturingTracker tracker;
  late ProviderContainer container;
  const request = UseCalculatorRequest(
    id: 'tool',
    calculator: Calculator(id: 'tool'),
  );
  final provider = useCalculatorControllerProvider(request);
  void setup({bool signedIn = true}) {
    store = TestLocalStore();
    api = DefinitionApi();
    final local = CalculatorLocalRepository(store.cache);
    final repository = CalculatorRepository(api, local);
    tracker = CapturingTracker(repository, local);
    container = ProviderContainer(
      overrides: [
        authControllerProvider.overrideWith(
          () => TestAuth(signedIn ? const User(id: 'owner') : null),
        ),
        calculatorRepositoryProvider.overrideWithValue(repository),
        calculatorUsageTrackerProvider.overrideWithValue(tracker),
      ],
    );
  }

  tearDown(() async {
    container.dispose();
    await Future<void>.delayed(Duration.zero);
    await store.close();
  });

  test(
    'tracks only after definition loads, with rendered version and short end',
    () async {
      setup();
      await container.read(authControllerProvider.future);
      final sub = container.listen(provider, (_, _) {});
      await container.read(provider.future);
      expect(tracker.starts, 1);
      expect(tracker.versions, ['published-version']);
      sub.close();
      await container.pump();
      await Future<void>.delayed(Duration.zero);
      expect(tracker.ends, hasLength(1));
    },
  );
  test('failed definition does not count a visit', () async {
    setup();
    api.fail = true;
    await container.read(authControllerProvider.future);
    final sub = container.listen(provider, (_, _) {});
    await expectLater(
      container.read(provider.future),
      throwsA(isA<SocketException>()),
    );
    expect(tracker.starts, 0);
    sub.close();
  });
  test('anonymous visit is excluded from authenticated tracking', () async {
    setup(signedIn: false);
    await container.read(authControllerProvider.future);
    final sub = container.listen(provider, (_, _) {});
    await container.read(provider.future);
    expect(tracker.starts, 0);
    sub.close();
  });
  test(
    'dispose captures end before slow start completes without reading disposed ref',
    () async {
      setup();
      await container.read(authControllerProvider.future);
      tracker.gate = Completer<String>();
      final sub = container.listen(provider, (_, _) {});
      await container.read(provider.future);
      final before = DateTime.now().toUtc();
      sub.close();
      await container.pump();
      final after = DateTime.now().toUtc();
      await Future<void>.delayed(const Duration(milliseconds: 50));
      tracker.gate!.complete('session');
      await Future<void>.delayed(Duration.zero);
      expect(tracker.ends, hasLength(1));
      expect(tracker.ends.single.isBefore(before), isFalse);
      expect(tracker.ends.single.isAfter(after), isFalse);
    },
  );
}
