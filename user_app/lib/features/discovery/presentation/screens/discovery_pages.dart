import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';
import 'package:user_app/app/providers/app_providers.dart';
import 'package:user_app/app/router/app_router.dart';
import 'package:user_app/features/ai_assistant/data/models/ai_context.dart';
import 'package:user_app/features/discovery/data/models/discovery_models.dart';
import 'package:user_app/features/discovery/presentation/widgets/discovery_widgets.dart';
import 'package:user_app/features/discovery/presentation/widgets/discovery_directory_scaffold.dart';
import 'package:user_app/features/discovery/presentation/widgets/discovery_detail_widgets.dart';

class ContentHubDirectoryPage extends ConsumerStatefulWidget {
  const ContentHubDirectoryPage({super.key});

  @override
  ConsumerState<ContentHubDirectoryPage> createState() =>
      _ContentHubDirectoryPageState();
}

class _ContentHubDirectoryPageState
    extends ConsumerState<ContentHubDirectoryPage> {
  final search = TextEditingController();
  Timer? debounce;
  late Future<DiscoveryValue<List<DiscoveryHub>>> request;

  @override
  void initState() {
    super.initState();
    reload();
  }

  @override
  void dispose() {
    debounce?.cancel();
    search.dispose();
    super.dispose();
  }

  void reload() {
    request = ref
        .read(discoveryRepositoryProvider)
        .hubs(search: search.text.trim());
  }

  void searchChanged(String _) {
    debounce?.cancel();
    setState(() {});
    debounce = Timer(const Duration(milliseconds: 300), () => setState(reload));
  }

  void submit() {
    debounce?.cancel();
    setState(reload);
  }

  void clear() {
    search.clear();
    submit();
  }

  Future<void> refresh() async {
    submit();
    try {
      await request;
    } catch (_) {
      // FutureBuilder displays the request error with a retry action.
    }
  }

  @override
  Widget build(BuildContext context) {
    final diseaseEnabled = ref.watch(diseaseHubsEnabledProvider);
    final genericEnabled = ref.watch(genericHubsEnabledProvider);
    final hasSearch = search.text.trim().isNotEmpty;
    return DiscoveryDirectoryScaffold(
      title: 'Content hubs',
      subtitle: 'Guidance and resources by topic',
      searchHint: 'Search hubs',
      search: search,
      onChanged: searchChanged,
      onSubmitted: submit,
      onClear: clear,
      onRefresh: refresh,
      child: !diseaseEnabled && !genericEnabled
          ? const EmptyState(
              'Content hubs will appear here when this feature is available.',
              title: 'Content hubs are not available yet',
            )
          : FutureBuilder<DiscoveryValue<List<DiscoveryHub>>>(
              future: request,
              builder: (context, snapshot) {
                if (snapshot.connectionState != ConnectionState.done) {
                  return const DiscoverySkeleton();
                }
                if (snapshot.hasError) {
                  return ErrorState(onRetry: submit);
                }
                final result = snapshot.data!;
                final hubs = result.value
                    .where(
                      (hub) => hub.diseases.isNotEmpty
                          ? diseaseEnabled
                          : genericEnabled,
                    )
                    .toList();
                return DiscoveryDirectoryResults(
                  label: hasSearch ? 'Matching hubs' : 'Available hubs',
                  count: hubs.length,
                  offline: result.offline,
                  child: hubs.isEmpty
                      ? EmptyState(
                          hasSearch
                              ? 'Try another topic or clear your search to browse all hubs.'
                              : 'Published clinical and disease resource collections will appear here.',
                          title: hasSearch
                              ? 'No matching content hubs'
                              : 'No content hubs yet',
                          isSearch: hasSearch,
                          actionLabel: hasSearch ? 'Clear search' : 'Refresh',
                          onAction: hasSearch ? clear : submit,
                        )
                      : Column(children: hubs.map(HubTile.new).toList()),
                );
              },
            ),
    );
  }
}

class DiseaseDirectoryPage extends ConsumerStatefulWidget {
  const DiseaseDirectoryPage({super.key});
  @override
  ConsumerState<DiseaseDirectoryPage> createState() =>
      _DiseaseDirectoryPageState();
}

class _DiseaseDirectoryPageState extends ConsumerState<DiseaseDirectoryPage> {
  final search = TextEditingController();
  Timer? debounce;
  late Future<DiscoveryValue<List<DiscoveryDisease>>> request;

  @override
  void initState() {
    super.initState();
    reload();
  }

  @override
  void dispose() {
    debounce?.cancel();
    search.dispose();
    super.dispose();
  }

  void reload() {
    request = ref
        .read(discoveryRepositoryProvider)
        .diseases(search: search.text.trim());
  }

  void searchChanged(String _) {
    debounce?.cancel();
    setState(() {});
    debounce = Timer(const Duration(milliseconds: 300), () => setState(reload));
  }

  void submit() {
    debounce?.cancel();
    setState(reload);
  }

  void clear() {
    search.clear();
    submit();
  }

  Future<void> refresh() async {
    submit();
    try {
      await request;
    } catch (_) {
      // FutureBuilder displays the request error with a retry action.
    }
  }

  @override
  Widget build(BuildContext context) {
    final hasSearch = search.text.trim().isNotEmpty;
    return DiscoveryDirectoryScaffold(
      title: 'Diseases & conditions',
      subtitle: 'Clinical guidance by condition',
      searchHint: 'Search conditions',
      search: search,
      onChanged: searchChanged,
      onSubmitted: submit,
      onClear: clear,
      onRefresh: refresh,
      child: !ref.watch(diseaseTaxonomyEnabledProvider)
          ? const EmptyState(
              'Conditions and their clinical resources will appear here when this feature is available.',
              title: 'Disease discovery is not available yet',
            )
          : FutureBuilder<DiscoveryValue<List<DiscoveryDisease>>>(
              future: request,
              builder: (context, snapshot) {
                if (snapshot.connectionState != ConnectionState.done) {
                  return const DiscoverySkeleton();
                }
                if (snapshot.hasError) {
                  return ErrorState(onRetry: submit);
                }
                final result = snapshot.data!;
                return DiscoveryDirectoryResults(
                  label: hasSearch
                      ? 'Matching conditions'
                      : 'Available conditions',
                  count: result.value.length,
                  offline: result.offline,
                  child: result.value.isEmpty
                      ? EmptyState(
                          hasSearch
                              ? 'Try another name or alias, or clear your search to see all conditions.'
                              : 'Conditions with approved public guidance will appear here.',
                          title: hasSearch
                              ? 'No matching conditions'
                              : 'No conditions yet',
                          isSearch: hasSearch,
                          actionLabel: hasSearch ? 'Clear search' : 'Refresh',
                          onAction: hasSearch ? clear : submit,
                        )
                      : Column(children: diseaseTiles(context, result.value)),
                );
              },
            ),
    );
  }
}

class DiseaseDetailPage extends ConsumerStatefulWidget {
  const DiseaseDetailPage({required this.slug, super.key});
  final String slug;

  @override
  ConsumerState<DiseaseDetailPage> createState() => _DiseaseDetailPageState();
}

class _DiseaseDetailPageState extends ConsumerState<DiseaseDetailPage> {
  late Future<DiscoveryValue<DiscoveryDisease>> request;

  @override
  void initState() {
    super.initState();
    reload();
  }

  void reload() {
    request = ref.read(discoveryRepositoryProvider).disease(widget.slug);
  }

  @override
  Widget build(BuildContext context) {
    if (!ref.watch(diseaseTaxonomyEnabledProvider)) {
      return Scaffold(
        appBar: AppBar(title: const Text('Disease')),
        body: const EmptyState('Disease discovery is not enabled yet.'),
      );
    }
    return Scaffold(
      appBar: AppBar(title: const Text('Disease')),
      floatingActionButton: ref.watch(diseaseContentAssignmentEnabledProvider)
          ? FloatingActionButton.extended(
              onPressed: () => context.push(
                AppRoutes.aiAssistant,
                extra: AiContext.genericPage(
                  title: 'Disease: ${widget.slug}',
                  content: 'Search approved content for this disease.',
                  metadata: {'disease_slug': widget.slug},
                ),
              ),
              icon: const Icon(LucideIcons.sparkles),
              label: const Text('Ask AI'),
            )
          : null,
      body: FutureBuilder<DiscoveryValue<DiscoveryDisease>>(
        future: request,
        builder: (context, snapshot) {
          if (snapshot.connectionState != ConnectionState.done) {
            return const DiscoverySkeleton();
          }
          if (snapshot.hasError) {
            return ErrorState(onRetry: () => setState(reload));
          }
          final disease = snapshot.data!.value;
          return ListView(
            padding: const EdgeInsets.all(16),
            children: [
              Text(
                disease.name,
                style: Theme.of(context).textTheme.headlineMedium?.copyWith(
                  fontWeight: FontWeight.w800,
                ),
              ),
              if (disease.description.isNotEmpty) Text(disease.description),
              if (snapshot.data!.offline)
                const Card(
                  child: ListTile(
                    leading: Icon(LucideIcons.cloudOff),
                    title: Text('Showing saved disease content'),
                  ),
                ),
              if (disease.aliases.isNotEmpty)
                Text('Also known as: ${disease.aliases.join(', ')}'),
              if (disease.children.isNotEmpty) ...[
                const SectionHeading('Related conditions'),
                ...disease.children.map(
                  (item) => ListTile(
                    title: Text(item.name),
                    onTap: () => context.push(AppRoutes.disease(item.slug)),
                  ),
                ),
              ],
              if (ref.watch(diseaseHubsEnabledProvider)) ...[
                const SectionHeading('Content hubs'),
                if (disease.hubs.isEmpty)
                  const EmptyState(
                    'Published hubs for this condition will appear here.',
                    title: 'No content hubs yet',
                  )
                else
                  ...disease.hubs.map((hub) => HubTile(hub)),
              ],
              const SectionHeading('Approved resources'),
              if (disease.resources.isEmpty)
                const EmptyState(
                  'Approved resources for this condition will appear here.',
                  title: 'No resources yet',
                )
              else
                ...disease.resources.map((item) => ResourceTile(item)),
            ],
          );
        },
      ),
    );
  }
}

class ContentHubPage extends ConsumerStatefulWidget {
  const ContentHubPage({required this.slug, super.key});
  final String slug;

  @override
  ConsumerState<ContentHubPage> createState() => _ContentHubPageState();
}

class _ContentHubPageState extends ConsumerState<ContentHubPage> {
  late Future<DiscoveryValue<DiscoveryHub>> request;

  @override
  void initState() {
    super.initState();
    reload();
  }

  void reload() {
    request = ref.read(discoveryRepositoryProvider).hub(widget.slug);
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(title: const Text('Content hub')),
    bottomNavigationBar: ref.watch(pillarRagMetadataEnabledProvider)
        ? DiscoveryAiBar(
            onPressed: () => context.push(
              AppRoutes.aiAssistant,
              extra: AiContext.genericPage(
                title: 'Content hub: ${widget.slug}',
                content: 'Search approved content in this hub.',
                metadata: {'hub_slug': widget.slug},
              ),
            ),
          )
        : null,
    body: FutureBuilder<DiscoveryValue<DiscoveryHub>>(
      future: request,
      builder: (context, snapshot) {
        if (snapshot.connectionState != ConnectionState.done) {
          return const DiscoverySkeleton();
        }
        if (snapshot.hasError) {
          return ErrorState(onRetry: () => setState(reload));
        }
        final hub = snapshot.data!.value;
        final enabled = hub.diseases.isEmpty
            ? ref.watch(genericHubsEnabledProvider)
            : ref.watch(diseaseHubsEnabledProvider);
        if (!enabled) {
          return const EmptyState('This content hub is not enabled yet.');
        }
        final all = hubResources(hub);
        final featured = hubResources(hub, featuredOnly: true);
        final featuredKeys = featured
            .map((item) => '${item.contentType}:${item.id}')
            .toSet();
        final reports = all
            .where(
              (item) =>
                  item.contentType == 'situation_report' &&
                  !featuredKeys.contains('${item.contentType}:${item.id}'),
            )
            .toList();
        final latest = all
            .where(
              (item) =>
                  !featuredKeys.contains('${item.contentType}:${item.id}') &&
                  item.contentType != 'situation_report',
            )
            .take(5)
            .toList();
        return ListView(
          padding: const EdgeInsets.fromLTRB(16, 16, 16, 32),
          children: [
            DiscoveryDetailHeader(
              title: hub.name,
              description: hub.description,
              label: 'Clinical resources',
            ),
            if (hub.diseases.isNotEmpty)
              Padding(
                padding: const EdgeInsets.only(top: 12),
                child: Wrap(
                  spacing: 8,
                  runSpacing: 4,
                  children: hub.diseases
                      .map(
                        (disease) => ActionChip(
                          label: Text(disease.name),
                          onPressed: () =>
                              context.push(AppRoutes.disease(disease.slug)),
                        ),
                      )
                      .toList(),
                ),
              ),
            if (snapshot.data!.offline)
              const Card(
                child: ListTile(
                  leading: Icon(LucideIcons.cloudOff),
                  title: Text('Showing saved hub content'),
                ),
              ),
            if (hub.outbreak != null) ...[
              const SizedBox(height: 16),
              OutbreakBanner(hub.outbreak!),
            ],
            const SectionHeading('Browse sections'),
            if (hub.pillars.isEmpty)
              const EmptyState(
                'Published sections will appear here.',
                title: 'No sections yet',
              )
            else
              HubSectionList(hubSlug: hub.slug, pillars: hub.pillars),
            if (featured.isNotEmpty) ...[
              const SectionHeading('Featured resources'),
              ...featured.map(ResourceTile.new),
            ],
            if (reports.isNotEmpty) ...[
              const SectionHeading('Situation reports'),
              ...reports.map(ResourceTile.new),
            ],
            if (latest.isNotEmpty) ...[
              const SectionHeading('Latest resources'),
              ...latest.map(ResourceTile.new),
            ],
            if (all.isEmpty)
              const EmptyState(
                'Published resources will appear here when they are available.',
                title: 'No resources yet',
              ),
          ],
        );
      },
    ),
  );
}

class ContentPillarPage extends ConsumerStatefulWidget {
  const ContentPillarPage({
    required this.hubSlug,
    required this.pillarSlug,
    super.key,
  });
  final String hubSlug, pillarSlug;
  @override
  ConsumerState<ContentPillarPage> createState() => _ContentPillarPageState();
}

class _ContentPillarPageState extends ConsumerState<ContentPillarPage> {
  String query = '', kind = '';
  final searchController = TextEditingController();
  late Future<DiscoveryValue<DiscoveryHub>> request;

  @override
  void initState() {
    super.initState();
    reload();
  }

  void reload() {
    request = ref.read(discoveryRepositoryProvider).hub(widget.hubSlug);
  }

  @override
  void dispose() {
    searchController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(title: const Text('Hub section')),
    bottomNavigationBar: ref.watch(pillarRagMetadataEnabledProvider)
        ? DiscoveryAiBar(
            onPressed: () => context.push(
              AppRoutes.aiAssistant,
              extra: AiContext.genericPage(
                title: 'Hub section: ${widget.pillarSlug}',
                content: 'Search approved content in this hub section.',
                metadata: {
                  'hub_slug': widget.hubSlug,
                  'pillar_slug': widget.pillarSlug,
                },
              ),
            ),
          )
        : null,
    body: FutureBuilder<DiscoveryValue<DiscoveryHub>>(
      future: request,
      builder: (context, snapshot) {
        if (snapshot.connectionState != ConnectionState.done) {
          return const DiscoverySkeleton();
        }
        if (snapshot.hasError) {
          return ErrorState(onRetry: () => setState(reload));
        }
        final hub = snapshot.data?.value;
        if (hub != null) {
          final enabled = hub.diseases.isEmpty
              ? ref.watch(genericHubsEnabledProvider)
              : ref.watch(diseaseHubsEnabledProvider);
          if (!enabled) {
            return const EmptyState('This content hub is not enabled yet.');
          }
        }
        final matches = hub == null
            ? <DiscoveryPillar>[]
            : flatten(
                hub.pillars,
              ).where((item) => item.slug == widget.pillarSlug).toList();
        if (matches.isEmpty) {
          return const EmptyState('This hub section is unavailable.');
        }
        final pillar = matches.first;
        final all = resources(pillar);
        final kinds = all.map((e) => e.contentType).toSet().toList()..sort();
        final shown = all
            .where(
              (item) =>
                  (kind.isEmpty || item.contentType == kind) &&
                  '${item.title} ${item.description}'.toLowerCase().contains(
                    query.trim().toLowerCase(),
                  ),
            )
            .toList();
        return ListView(
          padding: const EdgeInsets.all(16),
          children: [
            if (snapshot.data!.offline)
              const Card(
                child: ListTile(
                  leading: Icon(LucideIcons.cloudOff),
                  title: Text('Showing saved hub content'),
                ),
              ),
            DiscoveryDetailHeader(
              title: pillar.name,
              description: pillar.description,
              label: hub!.name,
            ),
            if (pillar.children.isNotEmpty) ...[
              const SectionHeading('Browse subsections'),
              HubSectionList(hubSlug: widget.hubSlug, pillars: pillar.children),
            ],
            const SizedBox(height: 16),
            TextField(
              controller: searchController,
              textInputAction: TextInputAction.search,
              decoration: InputDecoration(
                prefixIcon: const Icon(LucideIcons.search),
                hintText: 'Search resources',
                suffixIcon: query.isEmpty
                    ? null
                    : IconButton(
                        tooltip: 'Clear search',
                        icon: const Icon(LucideIcons.x),
                        onPressed: () => setState(() {
                          searchController.clear();
                          query = '';
                        }),
                      ),
              ),
              onChanged: (value) => setState(() => query = value),
            ),
            if (kinds.length > 1) ...[
              const SizedBox(height: 12),
              DropdownButtonFormField<String>(
                key: ValueKey(kind),
                initialValue: kind,
                decoration: const InputDecoration(labelText: 'Content type'),
                items: [
                  const DropdownMenuItem(
                    value: '',
                    child: Text('All content types'),
                  ),
                  ...kinds.map(
                    (value) => DropdownMenuItem(
                      value: value,
                      child: Text(resourceTypeLabel(value)),
                    ),
                  ),
                ],
                isExpanded: true,
                onChanged: (value) => setState(() => kind = value ?? ''),
              ),
            ],
            const SizedBox(height: 20),
            Text(
              '${shown.length} ${shown.length == 1 ? 'resource' : 'resources'}',
              style: Theme.of(
                context,
              ).textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w700),
            ),
            const SizedBox(height: 8),
            if (shown.isEmpty)
              EmptyState(
                query.trim().isNotEmpty || kind.isNotEmpty
                    ? 'Try another search or clear your filters to see all resources.'
                    : 'Published resources for this section will appear here.',
                title: query.trim().isNotEmpty || kind.isNotEmpty
                    ? 'No matching resources'
                    : 'No resources yet',
                isSearch: query.trim().isNotEmpty || kind.isNotEmpty,
                actionLabel: query.trim().isNotEmpty || kind.isNotEmpty
                    ? 'Clear filters'
                    : 'Refresh',
                onAction: () {
                  if (query.trim().isNotEmpty || kind.isNotEmpty) {
                    searchController.clear();
                    setState(() {
                      query = '';
                      kind = '';
                    });
                  } else {
                    setState(reload);
                  }
                },
              )
            else
              ...shown.map((item) => ResourceTile(item)),
          ],
        );
      },
    ),
  );
}
