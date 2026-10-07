import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';
import 'package:user_app/app/providers/app_providers.dart';
import 'package:user_app/app/router/app_router.dart';
import 'package:user_app/core/utils/app_message.dart';
import 'package:user_app/features/ai_assistant/data/models/ai_context.dart';
import 'package:user_app/features/discovery/data/models/discovery_models.dart';
import 'package:user_app/features/discovery/presentation/widgets/discovery_widgets.dart';

class ContentHubDirectoryPage extends ConsumerStatefulWidget {
  const ContentHubDirectoryPage({super.key});

  @override
  ConsumerState<ContentHubDirectoryPage> createState() =>
      _ContentHubDirectoryPageState();
}

class _ContentHubDirectoryPageState
    extends ConsumerState<ContentHubDirectoryPage> {
  final search = TextEditingController();
  late Future<DiscoveryValue<List<DiscoveryHub>>> request;

  @override
  void initState() {
    super.initState();
    reload();
  }

  @override
  void dispose() {
    search.dispose();
    super.dispose();
  }

  void reload() =>
      request = ref.read(discoveryRepositoryProvider).hubs(search: search.text);

  @override
  Widget build(BuildContext context) {
    final diseaseHubsEnabled = ref.watch(diseaseHubsEnabledProvider);
    final genericHubsEnabled = ref.watch(genericHubsEnabledProvider);
    return Scaffold(
      appBar: AppBar(title: const Text('Content hubs')),
      body: !diseaseHubsEnabled && !genericHubsEnabled
          ? const EmptyState('Content hubs are not enabled yet.')
          : RefreshIndicator(
              onRefresh: () async {
                setState(reload);
                await request;
              },
              child: ListView(
                physics: const AlwaysScrollableScrollPhysics(),
                padding: const EdgeInsets.all(16),
                children: [
                  TextField(
                    controller: search,
                    textInputAction: TextInputAction.search,
                    decoration: const InputDecoration(
                      prefixIcon: Icon(LucideIcons.search),
                      hintText: 'Search disease and clinical hubs',
                    ),
                    onSubmitted: (_) => setState(reload),
                  ),
                  const SizedBox(height: 16),
                  FutureBuilder<DiscoveryValue<List<DiscoveryHub>>>(
                    future: request,
                    builder: (context, snapshot) {
                      if (snapshot.connectionState != ConnectionState.done) {
                        return const DiscoverySkeleton();
                      }
                      if (snapshot.hasError) {
                        return ErrorState(onRetry: () => setState(reload));
                      }
                      final result = snapshot.data!;
                      final hubs = result.value.where((hub) {
                        final diseaseHub = hub.diseases.isNotEmpty;
                        return diseaseHub
                            ? diseaseHubsEnabled
                            : genericHubsEnabled;
                      }).toList();
                      if (result.offline) {
                        WidgetsBinding.instance.addPostFrameCallback((_) {
                          if (mounted) {
                            AppMessage.warning(
                              context,
                              'Offline: showing saved content hubs.',
                            );
                          }
                        });
                      }
                      if (hubs.isEmpty) {
                        return const EmptyState(
                          'No published content hubs are available.',
                        );
                      }
                      return Column(children: hubs.map(HubTile.new).toList());
                    },
                  ),
                ],
              ),
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
  late Future<DiscoveryValue<List<DiscoveryDisease>>> request;
  @override
  void initState() {
    super.initState();
    reload();
  }

  @override
  void dispose() {
    search.dispose();
    super.dispose();
  }

  void reload() => request = ref
      .read(discoveryRepositoryProvider)
      .diseases(search: search.text);
  @override
  Widget build(BuildContext context) {
    if (!ref.watch(diseaseTaxonomyEnabledProvider)) {
      return Scaffold(
        appBar: AppBar(title: const Text('Diseases & conditions')),
        body: const EmptyState('Disease discovery is not enabled yet.'),
      );
    }
    return Scaffold(
      appBar: AppBar(title: const Text('Diseases & conditions')),
      body: RefreshIndicator(
        onRefresh: () async {
          setState(reload);
          await request;
        },
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            TextField(
              controller: search,
              textInputAction: TextInputAction.search,
              decoration: const InputDecoration(
                prefixIcon: Icon(LucideIcons.search),
                hintText: 'Search official names or aliases',
              ),
              onSubmitted: (_) => setState(reload),
            ),
            const SizedBox(height: 16),
            FutureBuilder<DiscoveryValue<List<DiscoveryDisease>>>(
              future: request,
              builder: (context, snapshot) {
                if (snapshot.connectionState != ConnectionState.done) {
                  return const DiscoverySkeleton();
                }
                if (snapshot.hasError) {
                  return ErrorState(onRetry: () => setState(reload));
                }
                final result = snapshot.data!;
                if (result.offline) {
                  WidgetsBinding.instance.addPostFrameCallback((_) {
                    if (mounted) {
                      AppMessage.warning(
                        context,
                        'Offline: showing saved disease content.',
                      );
                    }
                  });
                }
                if (result.value.isEmpty) {
                  return const EmptyState(
                    'No active diseases with public content found.',
                  );
                }
                return Column(children: diseaseTiles(context, result.value));
              },
            ),
          ],
        ),
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

  void reload() =>
      request = ref.read(discoveryRepositoryProvider).disease(widget.slug);

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

  void reload() =>
      request = ref.read(discoveryRepositoryProvider).hub(widget.slug);

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(title: const Text('Content hub')),
    floatingActionButton: ref.watch(pillarRagMetadataEnabledProvider)
        ? FloatingActionButton.extended(
            onPressed: () => context.push(
              AppRoutes.aiAssistant,
              extra: AiContext.genericPage(
                title: 'Content hub: ${widget.slug}',
                content: 'Search approved content in this hub.',
                metadata: {'hub_slug': widget.slug},
              ),
            ),
            icon: const Icon(LucideIcons.sparkles),
            label: const Text('Ask AI'),
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
        return ListView(
          padding: const EdgeInsets.all(16),
          children: [
            Text(
              hub.name,
              style: Theme.of(
                context,
              ).textTheme.headlineMedium?.copyWith(fontWeight: FontWeight.w800),
            ),
            if (hub.description.isNotEmpty) Text(hub.description),
            if (hub.diseases.isNotEmpty)
              Wrap(
                spacing: 8,
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
            if (snapshot.data!.offline)
              const Card(
                child: ListTile(
                  leading: Icon(LucideIcons.cloudOff),
                  title: Text('Showing saved hub content'),
                ),
              ),
            if (hub.outbreak != null) OutbreakBanner(hub.outbreak!),
            const SectionHeading('Quick access'),
            if (hub.pillars.isEmpty)
              const EmptyState('This hub has no published sections yet.')
            else
              GridView.count(
                shrinkWrap: true,
                physics: const NeverScrollableScrollPhysics(),
                crossAxisCount: MediaQuery.sizeOf(context).width > 650 ? 4 : 2,
                children: hub.pillars
                    .map(
                      (pillar) => InkWell(
                        onTap: () => context.push(
                          AppRoutes.hubPillar(hub.slug, pillar.slug),
                        ),
                        child: Card(
                          child: Padding(
                            padding: const EdgeInsets.all(10),
                            child: Column(
                              mainAxisAlignment: MainAxisAlignment.center,
                              children: [
                                const Icon(LucideIcons.folderOpen),
                                Text(
                                  pillar.name,
                                  textAlign: TextAlign.center,
                                  style: const TextStyle(
                                    fontWeight: FontWeight.w700,
                                  ),
                                ),
                                Text('${pillar.resourceCount} resources'),
                              ],
                            ),
                          ),
                        ),
                      ),
                    )
                    .toList(),
              ),
            ...hubResources(hub, featuredOnly: true).isEmpty
                ? const <Widget>[]
                : <Widget>[
                    const SectionHeading('Featured resources'),
                    ...hubResources(
                      hub,
                      featuredOnly: true,
                    ).map(ResourceTile.new),
                  ],
            ...hubResources(hub, contentType: 'situation_report').isEmpty
                ? const <Widget>[]
                : <Widget>[
                    const SectionHeading('Situation reports'),
                    ...hubResources(
                      hub,
                      contentType: 'situation_report',
                    ).map(ResourceTile.new),
                  ],
            ...hubResources(hub).isEmpty
                ? const <Widget>[
                    EmptyState(
                      'Published resources will appear here when they are available.',
                      title: 'No resources yet',
                    ),
                  ]
                : <Widget>[
                    const SectionHeading('Latest updates'),
                    ...hubResources(hub).take(5).map(ResourceTile.new),
                  ],
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

  void reload() =>
      request = ref.read(discoveryRepositoryProvider).hub(widget.hubSlug);

  @override
  void dispose() {
    searchController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(title: const Text('Hub section')),
    floatingActionButton: ref.watch(pillarRagMetadataEnabledProvider)
        ? FloatingActionButton.extended(
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
            icon: const Icon(LucideIcons.sparkles),
            label: const Text('Ask AI'),
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
            Text(
              pillar.name,
              style: Theme.of(
                context,
              ).textTheme.headlineMedium?.copyWith(fontWeight: FontWeight.w800),
            ),
            if (pillar.description.isNotEmpty) Text(pillar.description),
            if (pillar.children.isNotEmpty) ...[
              const SectionHeading('Sections'),
              ...pillar.children.map(
                (child) => Card(
                  child: ListTile(
                    leading: const Icon(LucideIcons.folderOpen),
                    title: Text(child.name),
                    subtitle: Text('${child.resourceCount} resources'),
                    trailing: const Icon(LucideIcons.chevronRight),
                    onTap: () => context.push(
                      AppRoutes.hubPillar(widget.hubSlug, child.slug),
                    ),
                  ),
                ),
              ),
            ],
            const SizedBox(height: 16),
            TextField(
              controller: searchController,
              decoration: const InputDecoration(
                prefixIcon: Icon(LucideIcons.search),
                hintText: 'Search this section',
              ),
              onChanged: (value) => setState(() => query = value),
            ),
            const SizedBox(height: 8),
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
                    child: Text(value.replaceAll('_', ' ')),
                  ),
                ),
              ],
              onChanged: (value) => setState(() => kind = value ?? ''),
            ),
            Text('${shown.length} public resources'),
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
