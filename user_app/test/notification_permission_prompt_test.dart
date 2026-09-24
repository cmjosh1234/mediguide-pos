import 'package:flutter_test/flutter_test.dart';
import 'package:user_app/core/services/firebase_service.dart';
import 'package:user_app/features/notifications/presentation/controllers/notification_permission_prompt_controller.dart';

void main() {
  final now = DateTime(2026, 9, 24, 9);

  bool offer(
    AppNotificationPermissionState state, {
    bool pushAvailable = true,
    DateTime? dismissedAt,
  }) => shouldOfferNotificationPermission(
    pushAvailable: pushAvailable,
    state: state,
    dismissedAt: dismissedAt,
    now: now,
  );

  test('offers notifications while the system can still ask', () {
    expect(offer(AppNotificationPermissionState.notDetermined), isTrue);
    // Android 13+ reports a never-requested permission as denied.
    expect(offer(AppNotificationPermissionState.denied), isTrue);
    // iOS quiet delivery can be upgraded to full alerts.
    expect(offer(AppNotificationPermissionState.provisional), isTrue);
  });

  test('stops once the user has answered', () {
    expect(offer(AppNotificationPermissionState.authorized), isFalse);
    expect(offer(AppNotificationPermissionState.permanentlyDenied), isFalse);
  });

  test('never offers when push is switched off', () {
    for (final state in AppNotificationPermissionState.values) {
      expect(offer(state, pushAvailable: false), isFalse, reason: '$state');
    }
  });

  test('"Not now" hides the prompt for the snooze period', () {
    final justInside = now.subtract(
      notificationPromptSnooze - const Duration(minutes: 1),
    );
    final elapsed = now.subtract(notificationPromptSnooze);
    expect(
      offer(AppNotificationPermissionState.notDetermined, dismissedAt: now),
      isFalse,
    );
    expect(
      offer(
        AppNotificationPermissionState.notDetermined,
        dismissedAt: justInside,
      ),
      isFalse,
    );
    expect(
      offer(AppNotificationPermissionState.notDetermined, dismissedAt: elapsed),
      isTrue,
    );
  });
}
