import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:user_app/app/providers/app_providers.dart';
import 'package:user_app/features/notifications/data/models/notification_preferences.dart';

final notificationPreferencesControllerProvider =
    AsyncNotifierProvider.autoDispose<
      NotificationPreferencesController,
      NotificationPreferencesState
    >(NotificationPreferencesController.new);

final class NotificationPreferencesState {
  const NotificationPreferencesState({
    required this.preferences,
    required this.devices,
    this.devicesUnavailable = false,
  });

  final NotificationPreferences preferences;
  final List<NotificationDevice> devices;
  final bool devicesUnavailable;
}

class NotificationPreferencesController
    extends AutoDisposeAsyncNotifier<NotificationPreferencesState> {
  @override
  Future<NotificationPreferencesState> build() => _load();

  Future<NotificationPreferencesState> _load() async {
    final repository = ref.read(notificationRepositoryProvider);
    final preferences = await repository.getPreferences();
    final (devices, unavailable) = await _loadDevices();
    await ref
        .read(firebaseServiceProvider)
        .setOutbreakTopicPreference(
          outbreakAlerts: preferences.outbreakAlerts,
          pushEnabled: preferences.pushEnabled,
        );
    return NotificationPreferencesState(
      preferences: preferences,
      devices: devices,
      devicesUnavailable: unavailable,
    );
  }

  Future<(List<NotificationDevice>, bool)> _loadDevices({
    List<NotificationDevice> fallback = const [],
  }) async {
    try {
      return (
        await ref.read(notificationRepositoryProvider).listDevices(),
        false,
      );
    } catch (_) {
      return (fallback, true);
    }
  }

  Future<void> refresh() async {
    state = const AsyncLoading<NotificationPreferencesState>().copyWithPrevious(
      state,
    );
    state = await AsyncValue.guard(_load);
  }

  Future<void> updatePreferences(Map<String, dynamic> changes) async {
    final current = state.valueOrNull;
    if (current == null) return;
    state = const AsyncLoading<NotificationPreferencesState>().copyWithPrevious(
      state,
    );
    state = await AsyncValue.guard(() async {
      final preferences = await ref
          .read(notificationRepositoryProvider)
          .updatePreferences(changes);
      final (devices, unavailable) = await _loadDevices(
        fallback: current.devices,
      );
      await ref
          .read(firebaseServiceProvider)
          .setOutbreakTopicPreference(
            outbreakAlerts: preferences.outbreakAlerts,
            pushEnabled: preferences.pushEnabled,
          );
      return NotificationPreferencesState(
        preferences: preferences,
        devices: devices,
        devicesUnavailable: unavailable,
      );
    });
  }

  Future<void> setDevicePushEnabled(String deviceId, bool enabled) async {
    final current = state.valueOrNull;
    if (current == null) return;
    state = const AsyncLoading<NotificationPreferencesState>().copyWithPrevious(
      state,
    );
    state = await AsyncValue.guard(() async {
      final updated = await ref
          .read(notificationRepositoryProvider)
          .setDevicePushEnabled(deviceId, enabled);
      return NotificationPreferencesState(
        preferences: current.preferences,
        devices: [
          for (final device in current.devices)
            if (device.id == updated.id) updated else device,
        ],
      );
    });
  }
}
