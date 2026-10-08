import 'dart:async';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:user_app/app/router/app_router.dart';
import 'package:user_app/core/constants/app_spacing.dart';
import 'package:user_app/core/utils/app_message.dart';
import 'package:user_app/core/utils/date_utils.dart';
import 'package:user_app/core/widgets/app_skeleton.dart';
import 'package:user_app/core/widgets/app_error_view.dart';
import 'package:user_app/core/widgets/empty_state.dart' as states;
import 'package:user_app/features/discovery/data/models/discovery_models.dart';
import 'package:user_app/features/discovery/presentation/widgets/resource_kind.dart';
import 'package:user_app/features/outbreaks/data/models/outbreak_models.dart';
import 'package:user_app/features/outbreaks/presentation/widgets/outbreak_metrics.dart';

class HubTile extends StatelessWidget {
  const HubTile(this.hub, {super.key});
  final DiscoveryHub hub;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    final text = Theme.of(context).textTheme;
    final outbreak = hub.outbreak != null;
    final topic = hub.diseases.map((disease) => disease.name).join(', ');
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: Card(
        margin: EdgeInsets.zero,
        clipBehavior: Clip.antiAlias,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(16),
          side: BorderSide(color: colors.outlineVariant),
        ),
        child: InkWell(
          onTap: () => context.push(AppRoutes.hub(hub.slug)),
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Container(
                  width: 40,
                  height: 40,
                  decoration: BoxDecoration(
                    color: outbreak
                        ? colors.tertiaryContainer
                        : colors.primaryContainer,
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: Icon(
                    outbreak ? LucideIcons.shieldPlus : LucideIcons.bookOpen,
                    color: outbreak
                        ? colors.onTertiaryContainer
                        : colors.primary,
                    size: 20,
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        hub.name,
                        style: text.titleSmall?.copyWith(
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                      if (topic.isNotEmpty) ...[
                        const SizedBox(height: 4),
                        Text(
                          topic,
                          style: text.bodySmall?.copyWith(
                            color: colors.onSurfaceVariant,
                          ),
                        ),
                      ],
                      const SizedBox(height: 8),
                      Container(
                        padding: const EdgeInsets.symmetric(
                          horizontal: 8,
                          vertical: 4,
                        ),
                        decoration: BoxDecoration(
                          color: outbreak
                              ? colors.tertiaryContainer
                              : colors.surfaceContainerLow,
                          borderRadius: BorderRadius.circular(6),
                        ),
                        child: Text(
                          outbreak ? 'Outbreak response' : 'Clinical resources',
                          style: text.labelSmall?.copyWith(
                            color: outbreak
                                ? colors.onTertiaryContainer
                                : colors.onSurfaceVariant,
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
                const SizedBox(width: 8),
                Icon(
                  LucideIcons.chevronRight,
                  color: colors.onSurfaceVariant,
                  size: 18,
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _DiscoveryTopicTile extends StatelessWidget {
  const _DiscoveryTopicTile({
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.onTap,
  });
  final IconData icon;
  final String title, subtitle;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    return Card(
      margin: const EdgeInsets.only(bottom: 10),
      clipBehavior: Clip.antiAlias,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(color: colors.outlineVariant),
      ),
      child: ListTile(
        contentPadding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
        minVerticalPadding: 10,
        horizontalTitleGap: 12,
        leading: Container(
          width: 36,
          height: 36,
          decoration: BoxDecoration(
            color: colors.primaryContainer,
            borderRadius: BorderRadius.circular(12),
          ),
          child: Icon(icon, color: colors.primary, size: 19),
        ),
        title: Text(
          title,
          style: Theme.of(
            context,
          ).textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w700),
        ),
        subtitle: subtitle.isEmpty
            ? null
            : Padding(
                padding: const EdgeInsets.only(top: 4),
                child: Text(
                  subtitle,
                  maxLines: MediaQuery.textScalerOf(context).scale(14) > 20
                      ? null
                      : 2,
                  overflow: TextOverflow.ellipsis,
                  style: Theme.of(context).textTheme.bodySmall?.copyWith(
                    color: colors.onSurfaceVariant,
                    height: 1.4,
                  ),
                ),
              ),
        trailing: Icon(
          LucideIcons.chevronRight,
          color: colors.onSurfaceVariant,
          size: 20,
        ),
        onTap: onTap,
      ),
    );
  }
}

class ResourceTile extends StatelessWidget {
  const ResourceTile(this.resource, {super.key});
  final DiscoveryResource resource;

  bool get _external => resource.contentType == 'approved_external_url';

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    final kind = resourceKindOf(resource.contentType);
    final tint = kind.foreground(colors);
    final routable = _external
        ? resource.route.isNotEmpty
        : mobileRoute(resource) != null;
    final dates = [
      ('Published', resource.publicationDate),
      ('Effective', resource.effectiveAt),
      ('Next review', resource.reviewAt),
      ('Expires', resource.expiresAt),
    ].where((entry) => entry.$2.isNotEmpty).toList();
    final hasMeta =
        resource.source.isNotEmpty ||
        resource.provenance.isNotEmpty ||
        dates.isNotEmpty;
    final muted = theme.textTheme.bodySmall?.copyWith(
      color: colors.onSurfaceVariant,
    );
    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: () => _open(context),
        child: Padding(
          padding: const EdgeInsets.all(AppSpacing.md),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Container(
                    width: 36,
                    height: 36,
                    alignment: Alignment.center,
                    decoration: BoxDecoration(
                      color: kind.background(colors),
                      borderRadius: BorderRadius.circular(10),
                    ),
                    child: Icon(kind.icon, size: 18, color: tint),
                  ),
                  const SizedBox(width: 10),
                  Expanded(
                    child: Text(
                      kind.label,
                      style: theme.textTheme.labelLarge?.copyWith(
                        color: tint,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                  ),
                  if (resource.version.isNotEmpty)
                    Container(
                      padding: const EdgeInsets.symmetric(
                        horizontal: AppSpacing.sm,
                        vertical: 2,
                      ),
                      decoration: BoxDecoration(
                        border: Border.all(color: colors.outlineVariant),
                        borderRadius: BorderRadius.circular(999),
                      ),
                      child: Text(
                        RegExp(r'^\d').hasMatch(resource.version)
                            ? 'v${resource.version}'
                            : resource.version,
                        style: muted?.copyWith(fontWeight: FontWeight.w600),
                      ),
                    ),
                ],
              ),
              const SizedBox(height: 12),
              Text(
                resource.title,
                style: theme.textTheme.titleMedium?.copyWith(
                  fontWeight: FontWeight.w700,
                ),
              ),
              if (resource.description.isNotEmpty) ...[
                const SizedBox(height: AppSpacing.xs),
                Text(
                  resource.description,
                  maxLines: 3,
                  overflow: TextOverflow.ellipsis,
                  style: theme.textTheme.bodyMedium?.copyWith(
                    color: colors.onSurfaceVariant,
                  ),
                ),
              ],
              if (hasMeta) ...[
                const SizedBox(height: 12),
                Divider(
                  height: 1,
                  color: Color.alphaBlend(
                    tint.withValues(alpha: 0.18),
                    colors.outlineVariant,
                  ),
                ),
                const SizedBox(height: 12),
                if (resource.source.isNotEmpty)
                  Text(
                    resource.source,
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: colors.onSurface,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                if (resource.provenance.isNotEmpty)
                  Text(resource.provenance, style: muted),
                if (dates.isNotEmpty) const SizedBox(height: AppSpacing.xs),
                for (final (term, value) in dates)
                  Text.rich(
                    TextSpan(
                      children: [
                        TextSpan(text: '$term ', style: muted),
                        TextSpan(
                          text: resourceDate(value),
                          style: theme.textTheme.bodySmall,
                        ),
                      ],
                    ),
                  ),
              ],
              if (routable) ...[
                const SizedBox(height: 12),
                Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Text(
                      kind.action,
                      style: theme.textTheme.labelLarge?.copyWith(
                        color: tint,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                    if (_external) ...[
                      const SizedBox(width: AppSpacing.xs),
                      Icon(LucideIcons.externalLink, size: 14, color: tint),
                    ],
                  ],
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }

  void _open(BuildContext context) {
    if (_external) {
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
  }
}

// Publication dates are calendar dates sent as UTC midnight, so they are
// formatted without converting to local time (which could shift the day).
String resourceDate(String value) {
  final parsed = DateTime.tryParse(value);
  return parsed == null ? shortDate(value) : AppDateUtils.formatDate(parsed);
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
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    final metrics = hubOutbreakMetrics(value['metrics']);
    final verifiedAt = DateTime.tryParse('${value['last_verified_at'] ?? ''}');
    final status = '${value['status'] ?? ''}'.replaceAll('_', ' ');
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: colors.errorContainer.withValues(alpha: .35),
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: colors.outlineVariant),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(LucideIcons.siren, size: 18, color: colors.error),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  status == 'active' ? 'Active outbreak' : 'Outbreak update',
                  style: Theme.of(
                    context,
                  ).textTheme.labelLarge?.copyWith(color: colors.error),
                ),
              ),
            ],
          ),
          const SizedBox(height: 8),
          Text(
            '${value['title'] ?? ''}',
            style: Theme.of(
              context,
            ).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700),
          ),
          if ('${value['geographic_area'] ?? ''}'.isNotEmpty) ...[
            const SizedBox(height: 4),
            Text(
              '${value['geographic_area']}',
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ],
          if (verifiedAt != null) ...[
            const SizedBox(height: 4),
            Text(
              'Verified ${AppDateUtils.formatDate(verifiedAt.toLocal())}',
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ],
          if (metrics.isNotEmpty) ...[
            const SizedBox(height: 12),
            OutbreakMetricGrid(metrics: metrics),
          ],
        ],
      ),
    );
  }
}

// Hub payloads carry outbreak metrics as loosely typed maps (and may come from
// the offline cache), so each field is read leniently rather than through the
// strict generated OutbreakMetric.fromJson, which throws on unexpected types.
List<OutbreakMetric> hubOutbreakMetrics(Object? raw) {
  final metrics = (raw is List ? raw : const []).whereType<Map>().map((item) {
    final key = '${item['key'] ?? ''}';
    final label = '${item['label'] ?? ''}'.trim();
    final numeric = item['numeric_value'];
    return OutbreakMetric(
      key: key,
      label: label.isEmpty ? key.replaceAll('_', ' ') : label,
      value: '${item['value'] ?? ''}'.trim(),
      unit: '${item['unit'] ?? ''}',
      numericValue: numeric is num ? numeric.toDouble() : null,
      asOf: DateTime.tryParse('${item['as_of'] ?? ''}'),
      sourceReference: '${item['source_reference'] ?? ''}',
      sortOrder: item['sort_order'] is num
          ? (item['sort_order'] as num).toInt()
          : 0,
    );
  }).toList();
  metrics.sort((a, b) => a.sortOrder.compareTo(b.sortOrder));
  return metrics;
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
      ).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700),
    ),
  );
}

class DiscoverySkeleton extends StatelessWidget {
  const DiscoverySkeleton({super.key});

  @override
  Widget build(BuildContext context) => AppShimmer(
    child: Padding(
      padding: const EdgeInsets.symmetric(vertical: 16),
      child: Column(
        children: const [
          AppSkeleton(height: 44),
          SizedBox(height: 16),
          AppSkeleton(height: 120),
          SizedBox(height: 12),
          AppSkeleton(height: 120),
        ],
      ),
    ),
  );
}

class ErrorState extends StatelessWidget {
  const ErrorState({required this.onRetry, super.key});
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) => AppErrorView(
    error: 'Unable to load this content.',
    message: 'Check your connection and try again.',
    onRetry: onRetry,
  );
}

class EmptyState extends StatelessWidget {
  const EmptyState(
    this.message, {
    super.key,
    this.title = 'Nothing to show yet',
    this.actionLabel,
    this.onAction,
    this.isSearch = false,
  });
  final String message;
  final String title;
  final String? actionLabel;
  final VoidCallback? onAction;
  final bool isSearch;

  @override
  Widget build(BuildContext context) => states.EmptyState(
    icon: isSearch ? LucideIcons.searchX : LucideIcons.folderOpen,
    title: title,
    description: message,
    actionLabel: actionLabel,
    onAction: onAction,
    actionIcon: isSearch ? LucideIcons.filterX : LucideIcons.refreshCw,
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
        child: _DiscoveryTopicTile(
          icon: LucideIcons.activity,
          title: disease.name,
          subtitle: disease.description,
          onTap: () => context.push(AppRoutes.disease(disease.slug)),
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
