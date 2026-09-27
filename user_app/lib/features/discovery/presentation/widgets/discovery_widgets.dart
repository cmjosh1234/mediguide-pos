import 'dart:async';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:user_app/app/router/app_router.dart';
import 'package:user_app/core/utils/app_message.dart';
import 'package:user_app/core/widgets/app_skeleton.dart';
import 'package:user_app/features/discovery/data/models/discovery_models.dart';

class HubTile extends StatelessWidget {
  const HubTile(this.hub, {super.key});
  final DiscoveryHub hub;

  @override
  Widget build(BuildContext context) => Card(
    child: ListTile(
      leading: const Icon(LucideIcons.layoutGrid),
      title: Text(hub.name),
      subtitle: Text(
        [
          if (hub.diseases.isNotEmpty)
            hub.diseases.map((disease) => disease.name).join(', '),
          if (hub.outbreak != null) 'Outbreak response',
          hub.description,
        ].where((value) => value.isNotEmpty).join(' · '),
        maxLines: 3,
        overflow: TextOverflow.ellipsis,
      ),
      trailing: const Icon(LucideIcons.chevronRight),
      onTap: () => context.push(AppRoutes.hub(hub.slug)),
    ),
  );
}

class ResourceTile extends StatelessWidget {
  const ResourceTile(this.resource, {super.key});
  final DiscoveryResource resource;

  @override
  Widget build(BuildContext context) => Card(
    child: ListTile(
      leading: const Icon(LucideIcons.fileText),
      title: Text(resource.title),
      subtitle: Text(
        [
          resource.contentType.replaceAll('_', ' '),
          resource.source,
          if (resource.version.isNotEmpty) 'Version ${resource.version}',
          if (resource.publicationDate.isNotEmpty)
            'Published ${shortDate(resource.publicationDate)}',
          if (resource.effectiveAt.isNotEmpty)
            'Effective ${shortDate(resource.effectiveAt)}',
          if (resource.reviewAt.isNotEmpty)
            'Review ${shortDate(resource.reviewAt)}',
          if (resource.expiresAt.isNotEmpty)
            'Expires ${shortDate(resource.expiresAt)}',
          if (resource.provenance.isNotEmpty) 'Source: ${resource.provenance}',
        ].where((value) => value.isNotEmpty).join(' · '),
      ),
      onTap: () {
        if (resource.contentType == 'approved_external_url') {
          unawaited(_openApprovedExternalResource(context, resource));
          return;
        }
        final route = mobileRoute(resource);
        if (route == null) {
          AppMessage.info(
            context,
            'A reader for this resource is not available yet.',
          );
          return;
        }
        context.push(route);
      },
    ),
  );
}

Future<void> _openApprovedExternalResource(
  BuildContext context,
  DiscoveryResource resource,
) async {
  final uri = Uri.tryParse(resource.route);
  if (uri == null || uri.scheme != 'https' || uri.host.isEmpty) {
    AppMessage.error(context, 'This external resource address is invalid.');
    return;
  }
  final approved = await showDialog<bool>(
    context: context,
    builder: (dialogContext) => AlertDialog(
      title: const Text('Open external resource?'),
      content: Text(
        'You are leaving MediGuide to open ${uri.host}. Continue only if you trust this approved source.',
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(dialogContext, false),
          child: const Text('Cancel'),
        ),
        FilledButton(
          onPressed: () => Navigator.pop(dialogContext, true),
          child: const Text('Open'),
        ),
      ],
    ),
  );
  if (approved != true || !context.mounted) return;
  if (!await launchUrl(uri, mode: LaunchMode.externalApplication) &&
      context.mounted) {
    AppMessage.error(context, 'The external resource could not be opened.');
  }
}

class OutbreakBanner extends StatelessWidget {
  const OutbreakBanner(this.value, {super.key});
  final Map<String, dynamic> value;

  @override
  Widget build(BuildContext context) => Card(
    color: Theme.of(context).colorScheme.errorContainer,
    child: Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            'ACTIVE OUTBREAK · ${value['status'] ?? ''}',
            style: const TextStyle(fontWeight: FontWeight.w800),
          ),
          Text(
            '${value['title'] ?? ''}',
            style: Theme.of(context).textTheme.titleLarge,
          ),
          Text('${value['geographic_area'] ?? ''}'),
          if (value['metrics'] is List)
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: (value['metrics'] as List)
                  .whereType<Map>()
                  .expand(
                    (metric) => metric.entries.map(
                      (entry) => Chip(
                        label: Text(
                          '${entry.value} ${entry.key.toString().replaceAll('_', ' ')}',
                        ),
                      ),
                    ),
                  )
                  .toList(),
            ),
        ],
      ),
    ),
  );
}

class SectionHeading extends StatelessWidget {
  const SectionHeading(this.text, {super.key});
  final String text;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.only(top: 24, bottom: 8),
    child: Text(
      text,
      style: Theme.of(
        context,
      ).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w800),
    ),
  );
}

class DiscoverySkeleton extends StatelessWidget {
  const DiscoverySkeleton({super.key});

  @override
  Widget build(BuildContext context) => AppShimmer(
    child: ListView(
      padding: const EdgeInsets.all(16),
      children: const [
        AppSkeleton(height: 44),
        SizedBox(height: 16),
        AppSkeleton(height: 120),
        SizedBox(height: 12),
        AppSkeleton(height: 120),
      ],
    ),
  );
}

class ErrorState extends StatelessWidget {
  const ErrorState({required this.onRetry, super.key});
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) => Center(
    child: FilledButton(onPressed: onRetry, child: const Text('Try again')),
  );
}

class EmptyState extends StatelessWidget {
  const EmptyState(this.message, {super.key});
  final String message;

  @override
  Widget build(BuildContext context) => Center(
    child: Padding(
      padding: const EdgeInsets.all(24),
      child: Text(message, textAlign: TextAlign.center),
    ),
  );
}

List<DiscoveryPillar> flatten(List<DiscoveryPillar> values) =>
    values.expand((item) => [item, ...flatten(item.children)]).toList();

List<DiscoveryResource> resources(DiscoveryPillar pillar) => [
  ...pillar.items,
  ...pillar.children.expand(resources),
];

List<Widget> diseaseTiles(
  BuildContext context,
  List<DiscoveryDisease> diseases, {
  String parentId = '',
  int depth = 0,
}) {
  final ids = diseases.map((item) => item.id).toSet();
  final rows = parentId.isEmpty
      ? diseases.where(
          (item) => item.parentId.isEmpty || !ids.contains(item.parentId),
        )
      : diseases.where((item) => item.parentId == parentId);
  return [
    for (final disease in rows) ...[
      Padding(
        padding: EdgeInsets.only(left: depth * 18.0),
        child: Card(
          child: ListTile(
            leading: const Icon(LucideIcons.activity),
            title: Text(disease.name),
            subtitle: disease.description.isEmpty
                ? null
                : Text(
                    disease.description,
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                  ),
            trailing: const Icon(LucideIcons.chevronRight),
            onTap: () => context.push(AppRoutes.disease(disease.slug)),
          ),
        ),
      ),
      ...diseaseTiles(
        context,
        diseases,
        parentId: disease.id,
        depth: depth + 1,
      ),
    ],
  ];
}

String? mobileRoute(DiscoveryResource value) => switch (value.contentType) {
  'guideline' => AppRoutes.publicGuideline(value.id),
  'outbreak' => AppRoutes.outbreak(value.id),
  'situation_report' => AppRoutes.situationReport(value.id),
  'clinical_tool' => AppRoutes.calculator(value.id),
  'drug_reference' => AppRoutes.drugIndex,
  'algorithm' => algorithmRoute(value.route, value.id),
  'internal_route' => value.route.startsWith('/') ? value.route : null,
  _ => null,
};

String? algorithmRoute(String route, String blockId) {
  final parts = Uri.tryParse(route)?.pathSegments ?? const <String>[];
  if (parts.length >= 2 && parts[0] == 'guidelines') {
    return AppRoutes.publicGuidelineAlgorithmView(parts[1], blockId);
  }
  return null;
}

String shortDate(String value) =>
    value.length >= 10 ? value.substring(0, 10) : value;

List<DiscoveryResource> hubResources(
  DiscoveryHub hub, {
  bool featuredOnly = false,
  String? contentType,
}) {
  final seen = <String>{};
  final result = <DiscoveryResource>[];
  for (final pillar in flatten(hub.pillars)) {
    for (final resource in pillar.items) {
      if (contentType != null && resource.contentType != contentType) continue;
      if (featuredOnly && !resource.featured) continue;
      if (seen.add('${resource.contentType}:${resource.id}')) {
        result.add(resource);
      }
    }
  }
  result.sort(
    (left, right) => right.publicationDate.compareTo(left.publicationDate),
  );
  return result;
}
