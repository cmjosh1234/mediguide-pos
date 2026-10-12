import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:user_app/app/providers/app_providers.dart';
import 'package:user_app/core/network/api_client.dart';
import 'package:user_app/core/services/firebase_service.dart';
import 'package:user_app/features/authentication/data/datasources/auth_remote_datasource.dart';
import 'package:user_app/features/notifications/data/models/notification_preferences.dart';
import 'package:user_app/features/notifications/presentation/controllers/notification_preferences_controller.dart';
import 'package:user_app/features/notifications/presentation/screens/notification_preferences_page.dart';

class _Preferences extends NotificationPreferencesController {
  static Map<String, dynamic>? saved;
  @override
  Future<NotificationPreferencesState> build() async =>
      NotificationPreferencesState(
        preferences: NotificationPreferences.fromJson({
          'routine_digest': true,
          'quiet_hours_timezone': 'Africa/Kampala',
        }),
        devices: const [],
      );
  @override
  Future<void> updatePreferences(Map<String, dynamic> changes) async {
    saved = changes;
  }
}

class _CatalogApi extends BackendApiService {
  @override
  Future<Map<String, dynamic>> requestJson(
    String path, {
    required String method,
    Map<String, dynamic>? body,
    Map<String, String>? query,
    bool includeAuth = true,
  }) async => {
    'data': {
      'total_pages': 1,
      'items': path.endsWith('regions')
          ? [
              {
                'id': 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',
                'name': 'West Nile',
              },
            ]
          : [
              {
                'id': 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb',
                'title': 'Malaria prevention',
                'program_area': 'Malaria',
              },
            ],
    },
  };
}

void main() {
  for (final scale in [1.0, 2.0]) {
    testWidgets(
      'notification preferences and interest picker fit a narrow phone at $scale',
      (tester) async {
        tester.view.physicalSize = const Size(320, 740);
        tester.view.devicePixelRatio = 1;
        addTearDown(tester.view.resetPhysicalSize);
        addTearDown(tester.view.resetDevicePixelRatio);
        SharedPreferences.setMockInitialValues({});
        final preferences = await SharedPreferences.getInstance();
        final api = _CatalogApi();
        final firebase = MediGuideFirebaseService(
          api,
          AuthService(api),
          preferences,
        );
        addTearDown(firebase.dispose);
        await tester.pumpWidget(
          ProviderScope(
            overrides: [
              firebaseServiceProvider.overrideWithValue(firebase),
              backendApiServiceProvider.overrideWithValue(api),
              notificationPreferencesControllerProvider.overrideWith(
                _Preferences.new,
              ),
            ],
            child: MaterialApp(
              builder: (context, child) => MediaQuery(
                data: MediaQuery.of(
                  context,
                ).copyWith(textScaler: TextScaler.linear(scale)),
                child: child!,
              ),
              home: const NotificationPreferencesPage(),
            ),
          ),
        );
        await tester.pumpAndSettle();
        expect(tester.takeException(), isNull);
        final target = find.text('Personalize routine alerts');
        for (
          var i = 0;
          i < 80 && target.hitTestable().evaluate().isEmpty;
          i++
        ) {
          final position = tester
              .state<ScrollableState>(find.byType(Scrollable).first)
              .position;
          final delta = target.evaluate().isEmpty
              ? 150.0
              : tester.getCenter(target).dy - 370;
          position.jumpTo(
            (position.pixels + delta).clamp(0.0, position.maxScrollExtent),
          );
          await tester.pumpAndSettle();
        }
        await tester.tap(find.text('Personalize routine alerts'));
        await tester.pumpAndSettle();
        expect(tester.takeException(), isNull);
        await tester.tap(find.text('Malaria'));
        await tester.tap(find.text('Save'));
        await tester.pumpAndSettle();
        expect((_Preferences.saved?['interests'] as Map)['program_areas'], [
          'Malaria',
        ]);
        expect(tester.takeException(), isNull);
      },
    );
  }
}
