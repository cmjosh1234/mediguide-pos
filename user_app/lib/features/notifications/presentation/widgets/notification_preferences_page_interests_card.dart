part of '../screens/notification_preferences_page.dart';

final _interestCatalogProvider =
    FutureProvider.autoDispose<Map<String, List<Map<String, dynamic>>>>((
      ref,
    ) async {
      final api = ref.watch(backendApiServiceProvider);
      Future<List<Map<String, dynamic>>> catalog(String path) async {
        final items = <Map<String, dynamic>>[];
        for (var page = 1; page <= 50; page++) {
          final response = await api.requestJson(
            path,
            method: 'GET',
            includeAuth: false,
            query: {'page': '$page', 'per_page': '100'},
          );
          final data = response['data'];
          final raw = data is Map ? data['items'] : data;
          final batch = (raw as List? ?? [])
              .whereType<Map>()
              .map((row) => Map<String, dynamic>.from(row))
              .toList();
          items.addAll(batch);
          final total = data is Map
              ? (data['total_pages'] as num?)?.toInt()
              : 1;
          if (batch.isEmpty || page >= (total ?? 1)) break;
        }
        return items;
      }

      final results = await Future.wait([
        catalog('/api/public/guidelines'),
        catalog('/api/public/regions'),
      ]);
      return {'guidelines': results[0], 'regions': results[1]};
    });

class _InterestsCard extends ConsumerWidget {
  const _InterestsCard({required this.preferences});
  final NotificationPreferences preferences;
  @override
  Widget build(BuildContext context, WidgetRef ref) => _Section(
    title: 'Your interests',
    description:
        'Choose program areas, regions and guidelines to follow. Empty selections include all. National and urgent alerts remain available when their categories are enabled.',
    children: [
      ListTile(
        title: const Text('Personalize routine alerts'),
        subtitle: Text(
          '${preferences.programAreas.length} program areas · ${preferences.regionIds.length} regions · ${preferences.followedGuidelineIds.length} followed guidelines',
        ),
        trailing: const Icon(Icons.chevron_right),
        onTap: () => showDialog<void>(
          context: context,
          builder: (_) => _InterestDialog(preferences: preferences),
        ),
      ),
    ],
  );
}

class _InterestDialog extends ConsumerStatefulWidget {
  const _InterestDialog({required this.preferences});
  final NotificationPreferences preferences;
  @override
  ConsumerState<_InterestDialog> createState() => _InterestDialogState();
}

class _InterestDialogState extends ConsumerState<_InterestDialog> {
  late final areas = widget.preferences.programAreas.toSet();
  late final regions = widget.preferences.regionIds.toSet();
  late final guidelines = widget.preferences.followedGuidelineIds.toSet();
  late bool followedOnly = widget.preferences.followedOnly;
  String search = '';
  bool saving = false;
  Widget choices(
    String title,
    Map<String, String> options,
    Set<String> selected,
  ) => Column(
    crossAxisAlignment: CrossAxisAlignment.start,
    children: [
      Padding(
        padding: const EdgeInsets.symmetric(vertical: 8),
        child: Text(title, style: const TextStyle(fontWeight: FontWeight.bold)),
      ),
      for (final entry in options.entries.where(
        (entry) => search.isEmpty || entry.value.toLowerCase().contains(search),
      ))
        CheckboxListTile(
          dense: true,
          contentPadding: EdgeInsets.zero,
          title: Text(entry.value),
          value: selected.contains(entry.key),
          onChanged: (value) => setState(() {
            value == true
                ? selected.add(entry.key)
                : selected.remove(entry.key);
          }),
        ),
    ],
  );
  @override
  Widget build(BuildContext context) {
    final catalog = ref.watch(_interestCatalogProvider);
    return Dialog.fullscreen(
      child: Scaffold(
        appBar: AppBar(
          title: const Text(
            'Choose your interests',
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
          ),
        ),
        body: catalog.when(
          loading: () => const Center(child: CircularProgressIndicator()),
          error: (_, _) => ListView(
            padding: const EdgeInsets.all(16),
            children: [
              const Text('Unable to load choices. Check your connection.'),
              TextButton(
                onPressed: () => ref.invalidate(_interestCatalogProvider),
                child: const Text('Retry'),
              ),
            ],
          ),
          data: (data) {
            final publications = data['guidelines']!;
            final areaOptions = {
              for (final area in {
                ...areas,
                ...publications.map(
                  (row) => row['program_area']?.toString() ?? '',
                ),
              }.where((area) => area.isNotEmpty))
                area: area,
            };
            return ListView(
              padding: const EdgeInsets.all(16),
              children: [
                TextField(
                  decoration: const InputDecoration(
                    labelText: 'Search choices',
                  ),
                  onChanged: (value) =>
                      setState(() => search = value.trim().toLowerCase()),
                ),
                SwitchListTile(
                  contentPadding: EdgeInsets.zero,
                  title: const Text('Only followed guideline updates'),
                  value: followedOnly,
                  onChanged: (value) => setState(() => followedOnly = value),
                ),
                choices('Program areas', areaOptions, areas),
                choices('Regions', {
                  for (final row in data['regions']!)
                    row['id'].toString(): row['name']?.toString() ?? 'Region',
                }, regions),
                choices('Follow guidelines', {
                  for (final row in publications)
                    row['id'].toString():
                        row['title']?.toString() ?? 'Guideline',
                }, guidelines),
              ],
            );
          },
        ),
        bottomNavigationBar: SafeArea(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Wrap(
              alignment: WrapAlignment.end,
              spacing: 12,
              runSpacing: 8,
              children: [
                TextButton(
                  onPressed: saving ? null : () => Navigator.pop(context),
                  child: const Text('Cancel'),
                ),
                FilledButton(
                  onPressed: saving || !catalog.hasValue
                      ? null
                      : () async {
                          setState(() => saving = true);
                          await ref
                              .read(
                                notificationPreferencesControllerProvider
                                    .notifier,
                              )
                              .updatePreferences({
                                'interests': {
                                  'program_areas': areas.toList(),
                                  'region_ids': regions.toList(),
                                  'followed_guideline_ids': guidelines.toList(),
                                  'followed_only': followedOnly,
                                },
                              });
                          if (!mounted || !context.mounted) return;
                          if (ref
                              .read(notificationPreferencesControllerProvider)
                              .hasError) {
                            setState(() => saving = false);
                          } else {
                            Navigator.pop(context);
                          }
                        },
                  child: Text(saving ? 'Saving…' : 'Save'),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
