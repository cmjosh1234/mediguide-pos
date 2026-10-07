import 'dart:typed_data';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:user_app/core/network/api_client.dart';

class _Adapter implements HttpClientAdapter {
  final paths = <String>[];
  Map<String, dynamic>? registration;
  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? stream,
    Future<void>? cancelFuture,
  ) async {
    paths.add(options.path);
    if (options.path.endsWith('/register')) {
      registration = Map<String, dynamic>.from(options.data as Map);
      return ResponseBody.fromString(
        '{"success":true,"data":{"id":"created-user"}}',
        201,
        headers: {
          Headers.contentTypeHeader: ['application/json'],
        },
      );
    }
    return ResponseBody.fromString(
      '{"success":false,"error":"temporarily unavailable"}',
      503,
      headers: {
        Headers.contentTypeHeader: ['application/json'],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}

void main() {
  test(
    'creation followed by failed login directs to sign in rather than retry registration',
    () async {
      final adapter = _Adapter();
      final dio = Dio(BaseOptions(baseUrl: 'https://example.test'))
        ..httpClientAdapter = adapter;
      final api = BackendApiService(dio: dio);
      await expectLater(
        api.register(
          email: 'new@example.test',
          password: 'Password9',
          passwordConfirm: 'Password9',
          additionalData: {'name': 'New User'},
        ),
        throwsA(isA<AccountCreatedSignInRequired>()),
      );
      expect(adapter.paths, ['/api/v2/auth/register', '/api/v2/auth/login']);
      expect(adapter.registration?['password_confirm'], 'Password9');
    },
  );
  test('password mismatch is rejected before creation', () async {
    final adapter = _Adapter();
    final api = BackendApiService(dio: Dio()..httpClientAdapter = adapter);
    await expectLater(
      api.register(
        email: 'new@example.test',
        password: 'Password9',
        passwordConfirm: 'Different9',
      ),
      throwsArgumentError,
    );
    expect(adapter.paths, isEmpty);
  });
}
