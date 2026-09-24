import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:user_app/app/providers/app_providers.dart';
import 'package:user_app/core/services/firebase_service.dart';
import 'package:user_app/features/authentication/presentation/controllers/auth_controller.dart';

const _dismissedAtKey = 'notification_permission_prompt_dismissed_at';

/// How long "Not now" hides the prompt before it is offered again.
const notificationPromptSnooze = Duration(days: 14);

/// Whether a signed-in user should be offered notifications. The system
/// dialog is shown only after the user taps "Turn on", so a prompt the user
/// ignores never uses up the one request iOS allows.
bool shouldOfferNotificationPermission({
  required bool pushAvailable,
  required AppNotificationPermissionState state,
  required DateTime? dismissedAt,
  required DateTime now,
}) {
  if (!pushAvailable) return false;
  final askable = switch (state) {
    // Android 13+ reports a permission it has never asked for as denied.
    AppNotificationPermissionState.notDetermined ||
    AppNotificationPermissionState.denied ||
    AppNotificationPermissionState.provisional => true,
    AppNotificationPermissionState.authorized ||
    AppNotificationPermissionState.permanentlyDenied => false,
  };
  if (!askable) return false;
  return dismissedAt == null ||
      now.difference(dismissedAt) >= notificationPromptSnooze;
}

final notificationPermissionPromptProvider =
    NotifierProvider<NotificationPermissionPromptController, bool>(
      NotificationPermissionPromptController.new,
    );

final class NotificationPermissionPromptController extends Notifier<bool> {
  @override
  bool build() {
    final signedIn = ref.watch(
      authControllerProvider.select(
        (value) => value.valueOrNull?.isAuthenticated ?? false,
      ),
    );
    if (!signedIn) return false;
    final MediGuideFirebaseService service;
    try {
      service = ref.watch(firebaseServiceProvider);
    } on StateError {
      // Isolated previews and widget tests do not initialize Firebase.
      return false;
    }
    final state =
        ref.watch(notificationPermissionProvider).valueOrNull ??
        service.permissionState;
    final dismissedAt = ref
        .read(sharedPreferencesProvider)
        .getInt(_dismissedAtKey);
    return shouldOfferNotificationPermission(
      pushAvailable: service.pushNotificationsEnabled,
      state: state,
      dismissedAt: dismissedAt == null
          ? null
          : DateTime.fromMillisecondsSinceEpoch(dismissedAt),
      now: DateTime.now(),
    );
  }

  /// Shows the system dialog and returns the resulting permission. The prompt
  /// hides itself through the permission stream once the answer is known.
  Future<AppNotificationPermissionState> turnOn() {
    return ref.read(firebaseServiceProvider).requestNotificationPermission();
  }

  Future<void> dismiss() async {
    state = false;
    await ref
        .read(sharedPreferencesProvider)
        .setInt(_dismissedAtKey, DateTime.now().millisecondsSinceEpoch);
  }
}
