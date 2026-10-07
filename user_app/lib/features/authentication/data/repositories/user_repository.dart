import 'package:user_app/core/network/api_client.dart';
import 'package:user_app/features/authentication/data/models/user.dart';

final class UserRepository {
  UserRepository(this._api, {this.recordMetric});
  final Future<void> Function(String)? recordMetric;

  final BackendApiService _api;

  Future<User> updateProfile(String userId, Map<String, dynamic> fields) async {
    final response = await _api.requestJson(
      '/api/v2/users/$userId',
      method: 'PATCH',
      body: _normalizeOutgoing(fields),
    );
    return User.fromJson(_data(response));
  }

  Future<User> refreshProfile() async {
    final response = await _api.requestJson('/api/v2/me', method: 'GET');
    return User.fromJson(_data(response));
  }

  Future<void> changePassword({
    required String currentPassword,
    required String newPassword,
    required String newPasswordConfirm,
  }) async {
    await _api.requestJson(
      '/api/v2/me/password',
      method: 'POST',
      body: {
        'current_password': currentPassword,
        'new_password': newPassword,
        'new_password_confirm': newPasswordConfirm,
      },
    );
  }

  Future<Map<String, dynamic>> _accountAction(
    String path,
    String event,
    Map<String, dynamic> body,
  ) async {
    Future<void> record(String suffix) async {
      try {
        await recordMetric?.call('${event}_$suffix');
      } catch (_) {
        /* Telemetry never blocks account access. */
      }
    }

    await record('attempt');
    try {
      final response = await _api.requestJson(
        path,
        method: 'POST',
        body: body,
        includeAuth: false,
      );
      await record('success');
      return _data(response);
    } catch (_) {
      await record('failed');
      rethrow;
    }
  }

  Future<Map<String, dynamic>> requestPasswordReset(String email) =>
      _accountAction(
        '/api/v2/auth/password-reset/request',
        'password_reset_request',
        {'email': email.trim()},
      );
  Future<void> confirmPasswordReset({
    required String token,
    required String password,
    required String passwordConfirm,
  }) async {
    await _accountAction(
      '/api/v2/auth/password-reset/confirm',
      'password_reset_confirm',
      {
        'token': token,
        'password': password,
        'password_confirm': passwordConfirm,
      },
    );
  }

  Future<Map<String, dynamic>> requestEmailVerification(String email) =>
      _accountAction(
        '/api/v2/auth/email-verification/request',
        'email_verification_request',
        {'email': email.trim()},
      );
  Future<void> confirmEmailVerification(String token) async {
    await _accountAction(
      '/api/v2/auth/email-verification/confirm',
      'email_verification_confirm',
      {'token': token},
    );
  }

  Map<String, dynamic> _data(Map<String, dynamic> response) {
    final data = response['data'];
    return data is Map ? Map<String, dynamic>.from(data) : response;
  }

  Map<String, dynamic> _normalizeOutgoing(Map<String, dynamic> fields) {
    return fields.map((key, value) => MapEntry(_snakeCase(key), value));
  }

  String _snakeCase(String value) => value.replaceAllMapped(
    RegExp(r'([a-z0-9])([A-Z])'),
    (match) => '${match.group(1)}_${match.group(2)!.toLowerCase()}',
  );
}
