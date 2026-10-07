import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';
import 'package:user_app/app/router/route_names.dart';
import 'package:user_app/features/discovery/data/models/discovery_models.dart';

class DiscoveryDetailHeader extends StatelessWidget {
  const DiscoveryDetailHeader({
    super.key,
    required this.title,
    required this.description,
    this.label = '',
  });
  final String title, description, label;

  @override
  Widget build(BuildContext context) => Column(
    crossAxisAlignment: CrossAxisAlignment.start,
    children: [
      if (label.isNotEmpty) ...[
        Text(
          label,
          style: Theme.of(context).textTheme.labelMedium?.copyWith(
            color: Theme.of(context).colorScheme.primary,
          ),
        ),
        const SizedBox(height: 6),
      ],
      Text(
        title,
        style: Theme.of(
          context,
        ).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w800),
      ),
      if (description.trim().isNotEmpty) ...[
        const SizedBox(height: 8),
        Text(
          description,
          style: Theme.of(context).textTheme.bodyMedium?.copyWith(
            height: 1.5,
            color: Theme.of(context).colorScheme.onSurfaceVariant,
          ),
        ),
      ],
    ],
  );
}

class HubSectionList extends StatelessWidget {
  const HubSectionList({
    super.key,
    required this.hubSlug,
    required this.pillars,
  });
  final String hubSlug;
  final List<DiscoveryPillar> pillars;

  @override
  Widget build(BuildContext context) {
    final available = pillars
        .where((pillar) => pillar.resourceCount > 0)
        .toList();
    final pending = pillars
        .where((pillar) => pillar.resourceCount == 0)
        .toList();
    final colors = Theme.of(context).colorScheme;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        for (final pillar in available)
          Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: Material(
              color: colors.surfaceContainerLowest,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(14),
                side: BorderSide(color: colors.outlineVariant),
              ),
              clipBehavior: Clip.antiAlias,
              child: ListTile(
                contentPadding: const EdgeInsets.symmetric(
                  horizontal: 16,
                  vertical: 8,
                ),
                leading: Container(
                  width: 40,
                  height: 40,
                  decoration: BoxDecoration(
                    color: colors.primaryContainer,
                    borderRadius: BorderRadius.circular(10),
                  ),
                  child: Icon(
                    LucideIcons.folderOpen,
                    size: 20,
                    color: colors.primary,
                  ),
                ),
                title: Text(
                  pillar.name,
                  style: Theme.of(
                    context,
                  ).textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w700),
                ),
                subtitle: Text(
                  '${pillar.resourceCount} ${pillar.resourceCount == 1 ? 'resource' : 'resources'}',
                ),
                trailing: const Icon(LucideIcons.chevronRight, size: 18),
                onTap: () =>
                    context.push(AppRoutes.hubPillar(hubSlug, pillar.slug)),
              ),
            ),
          ),
        if (pending.isNotEmpty)
          ExpansionTile(
            tilePadding: const EdgeInsets.symmetric(horizontal: 8),
            title: Text(
              'Sections awaiting resources',
              style: Theme.of(context).textTheme.bodyMedium,
            ),
            subtitle: Text(
              '${pending.length} ${pending.length == 1 ? 'section' : 'sections'}',
            ),
            children: [
              for (final pillar in pending)
                ListTile(
                  dense: true,
                  title: Text(pillar.name),
                  subtitle: const Text('No published resources yet'),
                  enabled: false,
                ),
            ],
          ),
      ],
    );
  }
}

class DiscoveryAiBar extends StatelessWidget {
  const DiscoveryAiBar({super.key, required this.onPressed});
  final VoidCallback onPressed;
  @override
  Widget build(BuildContext context) => Material(
    color: Theme.of(context).colorScheme.surface,
    child: SafeArea(
      top: false,
      child: Padding(
        padding: const EdgeInsets.fromLTRB(16, 8, 16, 8),
        child: OutlinedButton.icon(
          onPressed: onPressed,
          icon: const Icon(LucideIcons.sparkles, size: 20),
          label: const Text('Ask AI'),
        ),
      ),
    ),
  );
}

String resourceTypeLabel(String type) => switch (type) {
  'guideline' => 'Guideline',
  'situation_report' => 'Situation report',
  'clinical_tool' => 'Clinical tool',
  'outbreak' => 'Outbreak response',
  'outbreak_document' => 'Response document',
  'approved_external_url' => 'External resource',
  'internal_route' => 'App resource',
  _ => type.replaceAll('_', ' '),
};
