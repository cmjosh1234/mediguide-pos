import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:user_app/app/providers/app_providers.dart';
import 'package:user_app/core/network/api_client.dart';
import 'package:user_app/features/authentication/data/models/user.dart';
import 'package:user_app/features/authentication/data/repositories/user_repository.dart';
import 'package:user_app/features/authentication/presentation/controllers/auth_controller.dart';
import 'package:user_app/features/authentication/presentation/controllers/auth_state.dart';
import 'package:user_app/features/authentication/presentation/screens/account_action_page.dart';

class _Api extends BackendApiService {
  final paths = <String>[];
  bool fail = false;
  @override
  Future<Map<String, dynamic>> requestJson(
    String path, {
    required String method,
    Map<String, dynamic>? body,
    Map<String, String>? query,
    bool includeAuth = true,
  }) async {
    expect(includeAuth, false);
    paths.add(path);
    if (fail) throw StateError('expired');
    return {
      'data': {'accepted': true, 'verified': true, 'logged_out': true},
    };
  }
}

class _Auth extends AuthController {
  bool signedIn = true;
  bool initiallyVerified = false;
  @override
  Future<AuthState> build() async => AuthState.authenticated(
    User(
      id: 'one',
      email: 'clinician@example.test',
      emailVerified: initiallyVerified,
    ),
  );
  @override
  Future<void> refreshProfile() async {
    state = const AsyncData(
      AuthState.authenticated(
        User(id: 'one', email: 'clinician@example.test', emailVerified: true),
      ),
    );
  }

  @override
  Future<void> logout() async {
    signedIn = false;
    state = const AsyncData(AuthState.unauthenticated());
  }
}

Future<void> _host(
  WidgetTester tester,
  _Api api,
  _Auth auth, {
  bool verification = true,
  String token = '',
  double scale = 1,
}) async {
  tester.view.physicalSize = const Size(320, 640);
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);
  final router = GoRouter(
    routes: [
      GoRoute(
        path: '/',
        builder: (_, _) =>
            AccountActionPage(verification: verification, token: token),
      ),
      GoRoute(
        path: '/login',
        builder: (_, _) => const Scaffold(body: Text('Login destination')),
      ),
      GoRoute(
        path: '/profile',
        builder: (_, _) => const Scaffold(body: Text('Profile destination')),
      ),
      GoRoute(
        path: '/forgot-password',
        builder: (_, _) => const Scaffold(body: Text('Recovery destination')),
      ),
    ],
  );
  addTearDown(router.dispose);
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        authControllerProvider.overrideWith(() => auth),
        userRepositoryProvider.overrideWithValue(UserRepository(api)),
      ],
      child: MaterialApp.router(
        routerConfig: router,
        builder: (context, child) => MediaQuery(
          data: MediaQuery.of(
            context,
          ).copyWith(textScaler: TextScaler.linear(scale)),
          child: child!,
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

Future<void> _tap(WidgetTester tester, String text) async {
  final finder = find.text(text);
  await tester.ensureVisible(finder);
  await tester.pumpAndSettle();
  await tester.tap(finder);
  await tester.pumpAndSettle();
}

void main() {
  testWidgets(
    'verification confirms only on user action and refreshes ownership status',
    (tester) async {
      final api = _Api();
      await _host(tester, api, _Auth(), token: 'single-use');
      expect(api.paths, isEmpty);
      await _tap(tester, 'Verify email');
      expect(api.paths, ['/api/v2/auth/email-verification/confirm']);
      expect(find.text('Your email is verified'), findsOneWidget);
    },
  );
  testWidgets(
    'a token still requires confirmation when the signed-in email is already verified',
    (tester) async {
      final api = _Api();
      await _host(
        tester,
        api,
        _Auth()..initiallyVerified = true,
        token: 'another-account-token',
      );
      expect(find.text('Verify email'), findsOneWidget);
      await _tap(tester, 'Verify email');
      expect(api.paths, ['/api/v2/auth/email-verification/confirm']);
    },
  );
  testWidgets(
    'expired verification permits resend and prevents repeated sends',
    (tester) async {
      final api = _Api()..fail = true;
      await _host(tester, api, _Auth(), token: 'expired');
      await _tap(tester, 'Verify email');
      expect(find.textContaining('expired or already used'), findsOneWidget);
      api.fail = false;
      await _tap(tester, 'Send verification link');
      expect(api.paths.last, '/api/v2/auth/email-verification/request');
      expect(find.textContaining('new link will be emailed'), findsOneWidget);
      final button = tester.widget<OutlinedButton>(find.byType(OutlinedButton));
      expect(button.onPressed, isNull);
      await tester.pumpWidget(const SizedBox());
    },
  );
  for (final scale in [1.0, 2.0]) {
    testWidgets('reset validates and clears the session at text scale $scale', (
      tester,
    ) async {
      final api = _Api();
      final auth = _Auth();
      await _host(
        tester,
        api,
        auth,
        verification: false,
        token: 'reset',
        scale: scale,
      );
      await tester.enterText(find.byType(TextFormField).at(0), 'weak');
      await tester.enterText(find.byType(TextFormField).at(1), 'different');
      await _tap(tester, 'Update password');
      expect(api.paths, isEmpty);
      await tester.enterText(
        find.byType(TextFormField).at(0),
        'StrongPassword9',
      );
      await tester.enterText(
        find.byType(TextFormField).at(1),
        'StrongPassword9',
      );
      await _tap(tester, 'Update password');
      expect(api.paths, ['/api/v2/auth/password-reset/confirm']);
      expect(auth.signedIn, isFalse);
      expect(find.text('Password updated'), findsOneWidget);
      expect(tester.takeException(), isNull);
      await _tap(tester, 'Sign in');
      expect(find.text('Login destination'), findsOneWidget);
    });
  }
  testWidgets('missing reset token directs user to request a fresh link', (
    tester,
  ) async {
    await _host(tester, _Api(), _Auth(), verification: false);
    await _tap(tester, 'Request reset link');
    expect(find.text('Recovery destination'), findsOneWidget);
  });
  test(
    'account metrics report both successful and failed requests without PII',
    () async {
      final api = _Api();
      final events = <String>[];
      final repo = UserRepository(
        api,
        recordMetric: (event) async {
          events.add(event);
        },
      );
      await repo.requestEmailVerification('clinician@example.test');
      api.fail = true;
      await expectLater(
        repo.confirmEmailVerification('secret-token'),
        throwsStateError,
      );
      expect(events, [
        'email_verification_request_attempt',
        'email_verification_request_success',
        'email_verification_confirm_attempt',
        'email_verification_confirm_failed',
      ]);
      expect(events.join(), isNot(contains('clinician')));
      expect(events.join(), isNot(contains('secret-token')));
    },
  );
}
