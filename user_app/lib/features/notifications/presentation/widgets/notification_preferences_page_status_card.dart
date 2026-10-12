part of '../screens/notification_preferences_page.dart';

class _NotificationStatusCard extends ConsumerWidget {
  const _NotificationStatusCard({required this.state});
  final NotificationPreferencesState state;
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final service = ref.watch(firebaseServiceProvider);
    final currentDevice = state.devices
        .where((device) => device.id == service.currentDeviceId)
        .firstOrNull;
    return ValueListenableBuilder<String>(
      valueListenable: service.registrationStatus,
      builder: (context, registration, _) => _Section(
        title: 'Notification status',
        description:
            'System permission, account preferences and device registration must all allow push delivery.',
        children: [
          if (state.devicesUnavailable)
            const ListTile(
              title: Text('Device status unavailable'),
              subtitle: Text(
                'Your preferences are loaded. Pull down to retry device status.',
              ),
            ),
          ListTile(
            title: const Text('Push service'),
            subtitle: Text(
              !service.enabled
                  ? 'Unavailable in this installation. Your inbox still works.'
                  : service.pushFeatureEnabled
                  ? 'Enabled'
                  : 'Temporarily disabled by the service',
            ),
          ),
          ListTile(
            title: const Text('Account push preference'),
            subtitle: Text(state.preferences.pushEnabled ? 'On' : 'Off'),
          ),
          ListTile(
            title: const Text('This installation'),
            subtitle: Text(
              currentDevice != null && !currentDevice.notificationsEnabled
                  ? 'Push disabled for this device. Enable it in Registered devices below.'
                  : registration,
            ),
          ),
          if (service.lastRegisteredAt != null)
            ListTile(
              title: const Text('Last registration'),
              subtitle: Text(
                service.lastRegisteredAt!.toLocal().toString().split('.').first,
              ),
            ),
          ValueListenableBuilder<bool>(
            valueListenable: service.deliveryDiagnostics,
            builder: (context, enabled, _) => SwitchListTile(
              title: const Text('Share delivery diagnostics'),
              subtitle: const Text(
                'Help measure whether alerts reach this installation. Exports messaging identifiers and delivery timing to Firebase. Off by default.',
              ),
              value: enabled,
              onChanged: service.enabled
                  ? (value) async {
                      final saved = await service.setDeliveryDiagnostics(value);
                      if (!saved && context.mounted) {
                        AppMessage.error(
                          context,
                          'Unable to change delivery diagnostics. Try again.',
                        );
                      }
                    }
                  : null,
            ),
          ),
          Align(
            alignment: Alignment.centerRight,
            child: TextButton(
              onPressed: !service.enabled || !state.preferences.pushEnabled
                  ? null
                  : () async {
                      await service.registerCurrentDevice(force: true);
                      await ref
                          .read(
                            notificationPreferencesControllerProvider.notifier,
                          )
                          .refresh();
                    },
              child: const Text('Refresh registration'),
            ),
          ),
        ],
      ),
    );
  }
}
