part of '../screens/notification_preferences_page.dart';

class _QuietHoursCard extends ConsumerWidget {
  const _QuietHoursCard({required this.preferences});

  final NotificationPreferences preferences;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final controller = ref.read(
      notificationPreferencesControllerProvider.notifier,
    );
    final start = preferences.quietHoursStart ?? '22:00';
    final end = preferences.quietHoursEnd ?? '06:00';
    return _Section(
      title: 'Quiet hours',
      description:
          'Delay non-emergency push alerts until your quiet period ends.',
      children: [
        _PreferenceSwitch(
          icon: LucideIcons.moon,
          title: 'Use quiet hours',
          subtitle: '$start–$end · ${preferences.quietHoursTimezone}',
          value: preferences.quietHoursEnabled,
          onChanged: (enabled) => controller.updatePreferences({
            'quiet_hours_enabled': enabled,
            'quiet_hours_start': start,
            'quiet_hours_end': end,
            'quiet_hours_timezone': preferences.quietHoursTimezone,
          }),
        ),
        ListTile(
          enabled: preferences.quietHoursEnabled,
          leading: const Icon(LucideIcons.clock3),
          title: const Text('Quiet period'),
          subtitle: Text('$start–$end'),
          trailing: const Icon(LucideIcons.chevronRight),
          onTap: preferences.quietHoursEnabled
              ? () => _pickQuietHours(context, ref, start, end)
              : null,
        ),
        ListTile(
          leading: const Icon(LucideIcons.globe2),
          title: const Text('Time zone for quiet hours and digest'),
          subtitle: DropdownButtonFormField<String>(
            key: ValueKey(preferences.quietHoursTimezone),
            initialValue: preferences.quietHoursTimezone,
            isExpanded: true,
            items: [
              for (final zone in {
                ..._quietHourTimezones,
                preferences.quietHoursTimezone,
              })
                DropdownMenuItem(
                  value: zone,
                  child: Text(
                    zone,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
            ],
            onChanged: (value) {
              if (value != null) {
                controller.updatePreferences({'quiet_hours_timezone': value});
              }
            },
          ),
        ),
      ],
    );
  }

  Future<void> _pickQuietHours(
    BuildContext context,
    WidgetRef ref,
    String start,
    String end,
  ) async {
    TimeOfDay parse(String value) {
      final parts = value.split(':');
      return TimeOfDay(
        hour: int.tryParse(parts.first) ?? 22,
        minute: int.tryParse(parts.length > 1 ? parts[1] : '') ?? 0,
      );
    }

    String format(TimeOfDay value) =>
        '${value.hour.toString().padLeft(2, '0')}:${value.minute.toString().padLeft(2, '0')}';
    final selectedStart = await showTimePicker(
      context: context,
      initialTime: parse(start),
      helpText: 'Quiet hours start',
    );
    if (selectedStart == null || !context.mounted) return;
    final selectedEnd = await showTimePicker(
      context: context,
      initialTime: parse(end),
      helpText: 'Quiet hours end',
    );
    if (selectedEnd == null) return;
    await ref
        .read(notificationPreferencesControllerProvider.notifier)
        .updatePreferences({
          'quiet_hours_enabled': true,
          'quiet_hours_start': format(selectedStart),
          'quiet_hours_end': format(selectedEnd),
          'quiet_hours_timezone': preferences.quietHoursTimezone,
        });
  }
}

const _quietHourTimezones = <String>[
  'Africa/Kampala',
  'Africa/Nairobi',
  'Africa/Kigali',
  'Africa/Dar_es_Salaam',
  'Africa/Juba',
  'Africa/Lusaka',
  'Africa/Johannesburg',
  'UTC',
];
