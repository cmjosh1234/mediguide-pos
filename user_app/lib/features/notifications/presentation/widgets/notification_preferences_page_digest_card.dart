part of '../screens/notification_preferences_page.dart';

class _DigestCard extends ConsumerWidget {
  const _DigestCard({required this.preferences});
  final NotificationPreferences preferences;
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final controller = ref.read(
      notificationPreferencesControllerProvider.notifier,
    );
    return _Section(
      title: 'Routine update digest',
      description:
          'Keep routine updates in the inbox immediately and receive one daily push summary. High-priority and urgent alerts keep their normal delivery. Uses your quiet-hours timezone.',
      children: [
        _PreferenceSwitch(
          icon: LucideIcons.calendar,
          title: 'Daily summary',
          subtitle: 'Reduce interruptions from routine updates',
          value: preferences.routineDigest,
          onChanged: (value) =>
              controller.updatePreferences({'routine_digest': value}),
        ),
        if (preferences.routineDigest)
          ListTile(
            title: const Text('Summary time'),
            subtitle: Text(
              '${preferences.digestTime} · ${preferences.quietHoursTimezone}',
            ),
            trailing: const Icon(Icons.edit_outlined),
            onTap: () async {
              final parts = preferences.digestTime.split(':');
              final selected = await showTimePicker(
                context: context,
                initialTime: TimeOfDay(
                  hour: int.tryParse(parts.first) ?? 18,
                  minute: int.tryParse(parts.last) ?? 0,
                ),
              );
              if (selected != null) {
                await controller.updatePreferences({
                  'digest_time':
                      '${selected.hour.toString().padLeft(2, '0')}:${selected.minute.toString().padLeft(2, '0')}',
                });
              }
            },
          ),
      ],
    );
  }
}
