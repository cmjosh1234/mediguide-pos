import 'package:firebase_core/firebase_core.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:user_app/core/network/api_client.dart';
import 'package:user_app/core/services/firebase_service.dart';

void main() {
  BackendApiException apiError(int statusCode) =>
      BackendApiException('failed', statusCode: statusCode);

  test('failures that clear up on their own are not reported', () {
    // 0 is how the API client reports an unreachable server.
    for (final status in [0, 401, 408, 429, 500, 502, 503]) {
      expect(
        isReportableRegistrationFailure(apiError(status)),
        isFalse,
        reason: 'HTTP $status',
      );
    }
  });

  test('rejected registrations are reported', () {
    for (final status in [400, 403, 404, 409, 422]) {
      expect(
        isReportableRegistrationFailure(apiError(status)),
        isTrue,
        reason: 'HTTP $status',
      );
    }
  });

  test('platform errors are reported', () {
    expect(
      isReportableRegistrationFailure(
        FirebaseException(plugin: 'firebase_messaging', code: 'unknown'),
      ),
      isTrue,
    );
    expect(isReportableRegistrationFailure(StateError('unexpected')), isTrue);
  });
}
