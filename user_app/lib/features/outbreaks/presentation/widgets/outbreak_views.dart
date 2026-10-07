import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:intl/intl.dart';

import 'package:user_app/app/router/route_names.dart';
import 'package:user_app/app/providers/app_providers.dart';
import 'package:user_app/core/constants/app_spacing.dart';
import 'package:user_app/core/config/app_config.dart';
import 'package:user_app/core/utils/app_message.dart';
import 'package:user_app/core/widgets/app_error_view.dart';
import 'package:user_app/core/widgets/app_loading_view.dart';
import 'package:user_app/features/documents/presentation/screens/document_reader_page.dart';
import 'package:user_app/features/notifications/domain/notification_action_resolver.dart';
import 'package:user_app/features/notifications/presentation/widgets/notification_permission_prompt_card.dart';
import 'package:user_app/features/outbreaks/data/models/outbreak_models.dart';
import 'package:user_app/features/outbreaks/presentation/providers/outbreak_providers.dart';
import 'package:user_app/features/outbreaks/presentation/widgets/outbreak_metrics.dart';
import 'package:user_app/shared/widgets/clinical_icon_tile.dart';
import 'package:user_app/shared/widgets/section_header.dart';

export 'package:user_app/features/outbreaks/presentation/providers/outbreak_providers.dart';

// ===========================================================================
// OUTBREAK HUB
// ===========================================================================

class OutbreakHubPage extends ConsumerWidget {
  const OutbreakHubPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final outbreaks = ref.watch(publicOutbreaksProvider);

    return Scaffold(
      appBar: AppBar(
        titleSpacing: AppSpacing.md,
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              'Outbreak response hub',
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: Theme.of(
                context,
              ).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w800),
            ),
            Text(
              'Public emergency updates and response guidance',
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: Theme.of(context).textTheme.bodySmall?.copyWith(
                color: Theme.of(context).colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ),
        actions: [
          IconButton(
            tooltip: 'Situation reports',
            onPressed: () {
              context.push(AppRoutes.situationReports);
            },
            icon: const Icon(LucideIcons.fileChartColumn),
          ),
          AppSpacing.hGapXs,
        ],
      ),
      body: outbreaks.when(
        loading: () =>
            const AppLoadingView(message: 'Loading public outbreak updates...'),
        error: (error, _) => AppErrorView(
          error: error,
          title: 'Public updates unavailable',
          message: 'Outbreak information cannot be loaded right now.',
          onRetry: () {
            ref.invalidate(publicOutbreaksProvider);
          },
        ),
        data: (page) {
          final items = page.items;
          if (items.isEmpty) {
            return const _PublicEmptyState(
              icon: LucideIcons.shieldCheck,
              title: 'No published outbreak updates',
              message:
                  'There are currently no public outbreak records available.',
            );
          }

          return RefreshIndicator(
            onRefresh: () =>
                ref.read(publicOutbreaksProvider.notifier).refresh(),
            child: ListView(
              physics: const AlwaysScrollableScrollPhysics(),
              padding: const EdgeInsets.fromLTRB(
                AppSpacing.md,
                AppSpacing.md,
                AppSpacing.md,
                AppSpacing.xxxl,
              ),
              children: [
                _FreshnessBanner(metadata: page.cache),
                if (page.cache.isOffline || page.cache.isStale)
                  AppSpacing.gapSm,
                const _OutbreakHubIntro(),

                const NotificationPermissionPromptCard(
                  padding: EdgeInsets.only(top: AppSpacing.md),
                ),

                AppSpacing.gapLg,

                const SectionHeader(
                  title: 'Published outbreaks',
                  subtitle: 'Current and historical public response updates',
                  icon: LucideIcons.siren,
                ),

                AppSpacing.gapSm,

                for (final outbreak in items)
                  Padding(
                    padding: const EdgeInsets.only(bottom: AppSpacing.sm),
                    child: _OutbreakCard(
                      outbreak: outbreak,
                      onTap: () {
                        context.push(AppRoutes.outbreak(outbreak.id));
                      },
                    ),
                  ),
                if (page.hasMore)
                  OutlinedButton.icon(
                    onPressed: () =>
                        ref.read(publicOutbreaksProvider.notifier).loadMore(),
                    icon: const Icon(LucideIcons.chevronsDown),
                    label: const Text('Load more updates'),
                  ),
              ],
            ),
          );
        },
      ),
    );
  }
}

// ===========================================================================
// OUTBREAK DETAIL
// ===========================================================================

class OutbreakDetailPage extends ConsumerWidget {
  const OutbreakDetailPage({super.key, required this.outbreakId});

  final String outbreakId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final outbreak = ref.watch(publicOutbreakProvider(outbreakId));

    return Scaffold(
      appBar: AppBar(
        title: const Text('Response hub'),
        actions: [
          IconButton(
            tooltip: 'Copy outbreak link',
            onPressed: () {
              _copyLink(context, AppRoutes.outbreak(outbreakId));
            },
            icon: const Icon(LucideIcons.share2),
          ),
          AppSpacing.hGapXs,
        ],
      ),
      body: outbreak.when(
        loading: () => const AppLoadingView(message: 'Loading response hub...'),
        error: (error, _) => error is PublicContentUnavailableException
            ? _PublicUnavailableState(
                message: error.message,
                withdrawn: error.isWithdrawn,
              )
            : AppErrorView(
                error: error,
                title: 'Response hub unavailable',
                onRetry: () {
                  ref.invalidate(publicOutbreakProvider(outbreakId));
                },
              ),
        data: (content) {
          return _OutbreakDetail(content: content);
        },
      ),
    );
  }
}

// ===========================================================================
// SITUATION REPORT LIST
// ===========================================================================

class SituationReportsPage extends ConsumerWidget {
  const SituationReportsPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final reports = ref.watch(publicSituationReportsProvider);

    return Scaffold(
      appBar: AppBar(
        titleSpacing: AppSpacing.md,
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              'Situation reports',
              style: Theme.of(
                context,
              ).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w800),
            ),
            Text(
              'Published outbreak and emergency reports',
              style: Theme.of(context).textTheme.bodySmall?.copyWith(
                color: Theme.of(context).colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ),
      ),
      body: reports.when(
        loading: () =>
            const AppLoadingView(message: 'Loading situation reports...'),
        error: (error, _) => AppErrorView(
          error: error,
          title: 'Situation reports unavailable',
          onRetry: () {
            ref.invalidate(publicSituationReportsProvider);
          },
        ),
        data: (page) {
          final items = page.items;
          if (items.isEmpty) {
            return const _PublicEmptyState(
              icon: LucideIcons.fileText,
              title: 'No published situation reports',
              message: 'Reports will appear here after publication.',
            );
          }

          return RefreshIndicator(
            onRefresh: () =>
                ref.read(publicSituationReportsProvider.notifier).refresh(),
            child: ListView(
              physics: const AlwaysScrollableScrollPhysics(),
              padding: const EdgeInsets.fromLTRB(
                AppSpacing.md,
                AppSpacing.md,
                AppSpacing.md,
                AppSpacing.xxxl,
              ),
              children: [
                _FreshnessBanner(metadata: page.cache),
                if (page.cache.isOffline || page.cache.isStale)
                  AppSpacing.gapSm,
                const _SituationReportsIntro(),

                AppSpacing.gapLg,

                for (final report in items)
                  Padding(
                    padding: const EdgeInsets.only(bottom: AppSpacing.sm),
                    child: _SituationReportCard(
                      report: report,
                      onTap: () {
                        context.push(AppRoutes.situationReport(report.id));
                      },
                    ),
                  ),
                if (page.hasMore)
                  OutlinedButton.icon(
                    onPressed: () => ref
                        .read(publicSituationReportsProvider.notifier)
                        .loadMore(),
                    icon: const Icon(LucideIcons.chevronsDown),
                    label: const Text('Load more reports'),
                  ),
              ],
            ),
          );
        },
      ),
    );
  }
}

// ===========================================================================
// SITUATION REPORT DETAIL
// ===========================================================================

class SituationReportDetailPage extends ConsumerWidget {
  const SituationReportDetailPage({super.key, required this.reportId});

  final String reportId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final report = ref.watch(publicSituationReportProvider(reportId));

    return Scaffold(
      appBar: AppBar(
        title: const Text('Situation report'),
        actions: [
          IconButton(
            tooltip: 'Copy report link',
            onPressed: () {
              _copyLink(context, AppRoutes.situationReport(reportId));
            },
            icon: const Icon(LucideIcons.share2),
          ),
          AppSpacing.hGapXs,
        ],
      ),
      body: report.when(
        loading: () => const AppLoadingView(message: 'Loading report...'),
        error: (error, _) => error is PublicContentUnavailableException
            ? _PublicUnavailableState(
                message: error.message,
                withdrawn: error.isWithdrawn,
              )
            : AppErrorView(
                error: error,
                title: 'Situation report unavailable',
                onRetry: () {
                  ref.invalidate(publicSituationReportProvider(reportId));
                },
              ),
        data: (content) {
          return _SituationReportView(content: content);
        },
      ),
    );
  }
}

// ===========================================================================
// HUB INTRO
// ===========================================================================

class _OutbreakHubIntro extends StatelessWidget {
  const _OutbreakHubIntro();

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;

    return Container(
      padding: const EdgeInsets.all(AppSpacing.md),
      decoration: BoxDecoration(
        color: colors.surfaceContainerLow,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: colors.outlineVariant),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: 42,
            height: 42,
            decoration: BoxDecoration(
              color: colors.errorContainer,
              borderRadius: BorderRadius.circular(12),
            ),
            child: Icon(
              LucideIcons.siren,
              color: colors.onErrorContainer,
              size: 21,
            ),
          ),

          AppSpacing.hGapMd,

          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  'Emergency information',
                  style: Theme.of(
                    context,
                  ).textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w800),
                ),
                const SizedBox(height: 3),
                Text(
                  'Review official outbreak updates, response resources and situation reports.',
                  style: Theme.of(context).textTheme.bodySmall?.copyWith(
                    color: colors.onSurfaceVariant,
                    height: 1.4,
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

// ===========================================================================
// OUTBREAK CARD
// ===========================================================================

class _OutbreakCard extends StatelessWidget {
  const _OutbreakCard({required this.outbreak, required this.onTap});

  final PublicOutbreak outbreak;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    final largeText = MediaQuery.textScalerOf(context).scale(1) >= 1.5;

    final tone = _statusTone(
      context,
      status: outbreak.status,
      visualTone: outbreak.visualTone,
    );

    return Semantics(
      button: true,
      label: '${outbreak.title}. Status ${_capitalize(outbreak.status)}',
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          onTap: onTap,
          borderRadius: BorderRadius.circular(16),
          child: Ink(
            padding: const EdgeInsets.all(AppSpacing.md),
            decoration: BoxDecoration(
              color: colors.surfaceContainerLowest,
              borderRadius: BorderRadius.circular(16),
              border: Border.all(color: tone.withValues(alpha: 0.22)),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                if (largeText) ...[
                  Align(
                    alignment: Alignment.centerLeft,
                    child: _StatusChip(status: outbreak.status, color: tone),
                  ),
                  AppSpacing.gapSm,
                ],

                Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Container(
                      width: 42,
                      height: 42,
                      decoration: BoxDecoration(
                        color: tone.withValues(alpha: 0.10),
                        borderRadius: BorderRadius.circular(12),
                      ),
                      child: Icon(LucideIcons.siren, color: tone, size: 21),
                    ),

                    AppSpacing.hGapMd,

                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            outbreak.title,
                            maxLines: 2,
                            overflow: TextOverflow.ellipsis,
                            style: Theme.of(context).textTheme.titleMedium
                                ?.copyWith(fontWeight: FontWeight.w800),
                          ),

                          if (outbreak.geographicArea.trim().isNotEmpty) ...[
                            const SizedBox(height: 3),
                            Row(
                              children: [
                                Icon(
                                  LucideIcons.mapPin,
                                  size: 14,
                                  color: colors.onSurfaceVariant,
                                ),
                                const SizedBox(width: 4),
                                Expanded(
                                  child: Text(
                                    outbreak.geographicArea,
                                    maxLines: 1,
                                    overflow: TextOverflow.ellipsis,
                                    style: Theme.of(context).textTheme.bodySmall
                                        ?.copyWith(
                                          color: colors.onSurfaceVariant,
                                        ),
                                  ),
                                ),
                              ],
                            ),
                          ],
                        ],
                      ),
                    ),

                    if (!largeText) ...[
                      AppSpacing.hGapSm,
                      _StatusChip(status: outbreak.status, color: tone),
                    ],
                  ],
                ),

                if (outbreak.summary.trim().isNotEmpty) ...[
                  AppSpacing.gapMd,

                  Text(
                    outbreak.summary,
                    maxLines: 3,
                    overflow: TextOverflow.ellipsis,
                    style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                      color: colors.onSurfaceVariant,
                      height: 1.45,
                    ),
                  ),
                ],

                AppSpacing.gapMd,

                Row(
                  children: [
                    Icon(
                      LucideIcons.clock3,
                      size: 14,
                      color: colors.onSurfaceVariant,
                    ),
                    const SizedBox(width: 5),
                    Expanded(
                      child: Text(
                        outbreak.lastUpdate == null
                            ? 'Update time unavailable'
                            : 'Updated ${_date(context, outbreak.lastUpdate)}',
                        style: Theme.of(context).textTheme.labelSmall?.copyWith(
                          color: colors.onSurfaceVariant,
                        ),
                      ),
                    ),
                    Icon(
                      LucideIcons.chevronRight,
                      size: 18,
                      color: colors.onSurfaceVariant,
                    ),
                  ],
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

// ===========================================================================
// OUTBREAK DETAIL
// ===========================================================================

class _OutbreakDetail extends ConsumerWidget {
  const _OutbreakDetail({required this.content});

  final PublicContent<PublicOutbreakDetail> content;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final detail = content.value;
    final outbreak = detail.outbreak;
    final apiDrivenHubsEnabled = ref.watch(
      backendManagedOutbreakHubsEnabledProvider,
    );
    final configuredHub = apiDrivenHubsEnabled
        ? ref.watch(publicOutbreakHubProvider(outbreak.id)).valueOrNull
        : null;
    final hasQuickAccess =
        detail.resources.isNotEmpty ||
        detail.reports.isNotEmpty ||
        (configuredHub?.pillars.isNotEmpty ?? false);

    final tone = _statusTone(
      context,
      status: outbreak.status,
      visualTone: outbreak.visualTone,
    );

    return ListView(
      padding: const EdgeInsets.fromLTRB(
        AppSpacing.md,
        AppSpacing.md,
        AppSpacing.md,
        AppSpacing.xxxl,
      ),
      children: [
        _FreshnessBanner(metadata: content.cache),
        if (content.cache.isOffline || content.cache.isStale) AppSpacing.gapSm,
        _OutbreakHero(outbreak: outbreak, color: tone),

        if (content.partialFailures.isNotEmpty) ...[
          AppSpacing.gapSm,
          _PartialContentBanner(sections: content.partialFailures),
        ],

        if (outbreak.metrics.isNotEmpty) ...[
          AppSpacing.gapLg,
          const SectionHeader(
            title: 'Current situation',
            subtitle: 'Published response indicators',
            icon: LucideIcons.chartNoAxesColumn,
          ),
          AppSpacing.gapSm,
          OutbreakMetricGrid(metrics: outbreak.metrics),
        ],

        if (hasQuickAccess) ...[
          AppSpacing.gapLg,
          const SectionHeader(
            title: 'Quick access',
            subtitle: 'Open essential response guidance and reports',
            icon: LucideIcons.layoutGrid,
          ),
          AppSpacing.gapSm,
          _OutbreakQuickAccessGrid(
            detail: detail,
            configuredHub: configuredHub,
          ),
        ],

        if (detail.updates.isNotEmpty) ...[
          AppSpacing.gapLg,

          const SectionHeader(
            title: 'Latest updates',
            subtitle: 'Published outbreak developments',
            icon: LucideIcons.clock3,
          ),

          AppSpacing.gapSm,

          for (final update in detail.updates)
            _UpdateCard(
              title: update.title,
              summary: update.summary,
              date: update.publishedAt,
            ),
        ],

        if (detail.resources.isNotEmpty) ...[
          AppSpacing.gapLg,

          const SectionHeader(
            title: 'Quick resources',
            subtitle: 'Guidance and response material',
            icon: LucideIcons.layoutGrid,
          ),

          AppSpacing.gapSm,

          for (final resource in detail.resources)
            _ResourceTile(
              title: resource.title,
              type: resource.resourceType,
              onTap: () {
                _openOutbreakResource(context, resource);
              },
            ),
        ],

        if (detail.reports.isNotEmpty) ...[
          AppSpacing.gapLg,

          const SectionHeader(
            title: 'Situation reports',
            subtitle: 'Published official reports',
            icon: LucideIcons.fileChartColumn,
          ),

          AppSpacing.gapSm,

          for (final report in detail.reports)
            _SituationReportCard(
              report: report,
              onTap: () {
                context.push(AppRoutes.situationReport(report.id));
              },
            ),
        ],
      ],
    );
  }
}

class _OutbreakQuickAccessGrid extends StatelessWidget {
  const _OutbreakQuickAccessGrid({required this.detail, this.configuredHub});

  final PublicOutbreakDetail detail;
  final PublicOutbreakHub? configuredHub;

  @override
  Widget build(BuildContext context) {
    if (configuredHub != null && configuredHub!.pillars.isNotEmpty) {
      final actions = configuredHub!.pillars
          .map(
            (pillar) => _OutbreakQuickAction(
              label: pillar.name,
              icon: _configuredPillarIcon(pillar.icon, pillar.slug),
              onTap: () => context.push(
                AppRoutes.outbreakSectionFor(detail.outbreak.id, pillar.slug),
              ),
            ),
          )
          .toList(growable: false);
      return _OutbreakQuickActionWrap(actions: actions);
    }
    final actions = <_OutbreakQuickAction>[
      if (detail.reports.isNotEmpty)
        _OutbreakQuickAction(
          label: 'Situation reports',
          icon: LucideIcons.fileChartColumn,
          onTap: () =>
              context.push(AppRoutes.situationReport(detail.reports.first.id)),
        ),
    ];

    return _OutbreakQuickActionWrap(actions: actions);
  }
}

class _OutbreakQuickActionWrap extends StatelessWidget {
  const _OutbreakQuickActionWrap({required this.actions});

  final List<_OutbreakQuickAction> actions;

  @override
  Widget build(BuildContext context) => LayoutBuilder(
    builder: (context, constraints) {
      final columns = constraints.maxWidth >= 700
          ? 4
          : constraints.maxWidth >= 340
          ? 3
          : 2;
      final width =
          (constraints.maxWidth - AppSpacing.sm * (columns - 1)) / columns;
      return Wrap(
        spacing: AppSpacing.sm,
        runSpacing: AppSpacing.sm,
        children: [
          for (final action in actions)
            SizedBox(
              width: width,
              child: _OutbreakQuickAccessTile(action: action),
            ),
        ],
      );
    },
  );
}

IconData _configuredPillarIcon(String configured, String slug) {
  final value = configured.trim().isEmpty ? slug : configured.trim();
  return switch (value) {
    'case-definition' || 'case_definitions' => LucideIcons.badgeHelp,
    'screening-triage' || 'screening' => LucideIcons.listChecks,
    'surveillance-guidance' || 'surveillance' => LucideIcons.radioTower,
    'ipc-ppe' || 'shield-check' => LucideIcons.shieldCheck,
    'isolation' => LucideIcons.squareActivity,
    'clinical-management' || 'clinical-care' => LucideIcons.stethoscope,
    'laboratory' || 'flask' => LucideIcons.flaskConical,
    'medicines' || 'pill' => LucideIcons.pill,
    'forms' || 'file-text' => LucideIcons.fileText,
    'training' || 'graduation-cap' => LucideIcons.graduationCap,
    'situation-reports' || 'file-chart' => LucideIcons.fileChartColumn,
    'contacts' || 'users' => LucideIcons.users,
    'faqs' || 'help-circle' => LucideIcons.messageCircleQuestion,
    _ => LucideIcons.folderOpen,
  };
}

class _OutbreakQuickAccessTile extends StatelessWidget {
  const _OutbreakQuickAccessTile({required this.action});

  final _OutbreakQuickAction action;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    return Card(
      margin: EdgeInsets.zero,
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: action.onTap,
        child: Padding(
          padding: const EdgeInsets.symmetric(
            horizontal: AppSpacing.sm,
            vertical: AppSpacing.md,
          ),
          child: Column(
            children: [
              Icon(action.icon, color: colors.primary, size: 22),
              const SizedBox(height: 7),
              Text(
                action.label,
                textAlign: TextAlign.center,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
                style: Theme.of(
                  context,
                ).textTheme.labelSmall?.copyWith(fontWeight: FontWeight.w700),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _OutbreakQuickAction {
  const _OutbreakQuickAction({
    required this.label,
    required this.icon,
    required this.onTap,
  });

  final String label;
  final IconData icon;
  final VoidCallback onTap;
}

// ===========================================================================
// OUTBREAK HERO
// ===========================================================================

class _OutbreakHero extends StatelessWidget {
  const _OutbreakHero({required this.outbreak, required this.color});

  final PublicOutbreak outbreak;
  final Color color;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;

    return Container(
      padding: const EdgeInsets.all(AppSpacing.lg),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.08),
        borderRadius: BorderRadius.circular(20),
        border: Border.all(color: color.withValues(alpha: 0.25)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              _StatusChip(status: outbreak.status, color: color),

              const Spacer(),

              if (outbreak.lastUpdate != null)
                Text(
                  _date(context, outbreak.lastUpdate),
                  style: Theme.of(context).textTheme.labelSmall?.copyWith(
                    color: colors.onSurfaceVariant,
                  ),
                ),
            ],
          ),

          AppSpacing.gapMd,

          Text(
            outbreak.title,
            style: Theme.of(
              context,
            ).textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.w900),
          ),

          if (outbreak.geographicArea.trim().isNotEmpty) ...[
            AppSpacing.gapSm,

            Row(
              children: [
                Icon(
                  LucideIcons.mapPin,
                  size: 16,
                  color: colors.onSurfaceVariant,
                ),
                const SizedBox(width: 5),
                Expanded(
                  child: Text(
                    outbreak.geographicArea,
                    style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                      color: colors.onSurfaceVariant,
                    ),
                  ),
                ),
              ],
            ),
          ],

          if (outbreak.summary.trim().isNotEmpty) ...[
            AppSpacing.gapMd,

            Text(
              outbreak.summary,
              style: Theme.of(
                context,
              ).textTheme.bodyMedium?.copyWith(height: 1.5),
            ),
          ],

          if (outbreak.sourceOrganization.trim().isNotEmpty) ...[
            AppSpacing.gapMd,

            Row(
              children: [
                Icon(
                  LucideIcons.landmark,
                  size: 15,
                  color: colors.onSurfaceVariant,
                ),
                const SizedBox(width: 5),
                Expanded(
                  child: Text(
                    'Source: ${outbreak.sourceOrganization}',
                    style: Theme.of(context).textTheme.bodySmall?.copyWith(
                      color: colors.onSurfaceVariant,
                    ),
                  ),
                ),
              ],
            ),
          ],
        ],
      ),
    );
  }
}

// ===========================================================================
// UPDATE CARD
// ===========================================================================

class _UpdateCard extends StatelessWidget {
  const _UpdateCard({
    required this.title,
    required this.summary,
    required this.date,
  });

  final String title;
  final String summary;
  final DateTime? date;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;

    return Container(
      margin: const EdgeInsets.only(bottom: AppSpacing.sm),
      padding: const EdgeInsets.all(AppSpacing.md),
      decoration: BoxDecoration(
        color: colors.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: colors.outlineVariant),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: 34,
            height: 34,
            decoration: BoxDecoration(
              color: colors.primaryContainer,
              borderRadius: BorderRadius.circular(10),
            ),
            child: Icon(LucideIcons.circleDot, color: colors.primary, size: 17),
          ),

          AppSpacing.hGapMd,

          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  title,
                  style: Theme.of(
                    context,
                  ).textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w700),
                ),

                if (date != null) ...[
                  const SizedBox(height: 3),
                  Text(
                    _date(context, date),
                    style: Theme.of(context).textTheme.labelSmall?.copyWith(
                      color: colors.onSurfaceVariant,
                    ),
                  ),
                ],

                if (summary.trim().isNotEmpty) ...[
                  const SizedBox(height: 5),
                  Text(
                    summary,
                    style: Theme.of(context).textTheme.bodySmall?.copyWith(
                      color: colors.onSurfaceVariant,
                      height: 1.4,
                    ),
                  ),
                ],
              ],
            ),
          ),
        ],
      ),
    );
  }
}

// ===========================================================================
// RESOURCE TILE
// ===========================================================================

class _ResourceTile extends StatelessWidget {
  const _ResourceTile({
    required this.title,
    required this.type,
    required this.onTap,
  });

  final String title;
  final String type;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.only(bottom: AppSpacing.sm),
      child: ListTile(
        leading: const ClinicalIconTile(icon: LucideIcons.externalLink),
        title: Text(title),
        subtitle: type.trim().isEmpty ? null : Text(type),
        trailing: const Icon(LucideIcons.chevronRight),
        onTap: onTap,
      ),
    );
  }
}

// ===========================================================================
// SITUATION REPORT INTRO
// ===========================================================================

class _SituationReportsIntro extends StatelessWidget {
  const _SituationReportsIntro();

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;

    return Container(
      padding: const EdgeInsets.all(AppSpacing.md),
      decoration: BoxDecoration(
        color: colors.surfaceContainerLow,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: colors.outlineVariant),
      ),
      child: Row(
        children: [
          Container(
            width: 42,
            height: 42,
            decoration: BoxDecoration(
              color: colors.primaryContainer,
              borderRadius: BorderRadius.circular(12),
            ),
            child: Icon(
              LucideIcons.fileChartColumn,
              color: colors.primary,
              size: 21,
            ),
          ),

          AppSpacing.hGapMd,

          Expanded(
            child: Text(
              'Situation reports provide official summaries of outbreak status, response actions and key indicators.',
              style: Theme.of(context).textTheme.bodySmall?.copyWith(
                color: colors.onSurfaceVariant,
                height: 1.4,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

// ===========================================================================
// SITUATION REPORT CARD
// ===========================================================================

class _SituationReportCard extends StatelessWidget {
  const _SituationReportCard({required this.report, required this.onTap});

  final dynamic report;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;

    return Container(
      margin: const EdgeInsets.only(bottom: AppSpacing.sm),
      decoration: BoxDecoration(
        color: colors.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: colors.outlineVariant),
      ),
      child: ListTile(
        minVerticalPadding: AppSpacing.md,
        leading: const ClinicalIconTile(icon: LucideIcons.fileChartColumn),
        title: Text(report.title, maxLines: 2, overflow: TextOverflow.ellipsis),
        subtitle: Text(
          [
            report.geographicArea,
            _date(context, report.publicationDate),
            report.sourceOrganization,
          ].where((value) => value.toString().trim().isNotEmpty).join(' • '),
          maxLines: 2,
          overflow: TextOverflow.ellipsis,
        ),
        trailing: const Icon(LucideIcons.chevronRight, size: 18),
        onTap: onTap,
      ),
    );
  }
}

// ===========================================================================
// SITUATION REPORT DETAIL
// ===========================================================================

class _SituationReportView extends StatelessWidget {
  const _SituationReportView({required this.content});

  final PublicContent<PublicSituationReport> content;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    final report = content.value;

    return ListView(
      padding: const EdgeInsets.fromLTRB(
        AppSpacing.md,
        AppSpacing.md,
        AppSpacing.md,
        AppSpacing.xxxl,
      ),
      children: [
        _FreshnessBanner(metadata: content.cache),
        if (content.cache.isOffline || content.cache.isStale) AppSpacing.gapSm,
        Text(
          report.title,
          style: Theme.of(context).textTheme.titleLarge?.copyWith(
            fontWeight: FontWeight.w800,
            height: 1.25,
          ),
        ),
        AppSpacing.gapMd,
        if (report.publicationDate != null)
          Text(
            'Published ${DateFormat('d MMM y').format(report.publicationDate!)}',
            style: Theme.of(
              context,
            ).textTheme.bodySmall?.copyWith(color: colors.onSurfaceVariant),
          ),
        if (report.sourceOrganization.trim().isNotEmpty) ...[
          const SizedBox(height: 6),
          Text(
            report.sourceOrganization,
            style: Theme.of(
              context,
            ).textTheme.bodySmall?.copyWith(color: colors.onSurfaceVariant),
          ),
        ],
        if (report.geographicArea.trim().isNotEmpty) ...[
          const SizedBox(height: 8),
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Icon(
                LucideIcons.mapPin,
                size: 16,
                color: colors.onSurfaceVariant,
              ),
              const SizedBox(width: 6),
              Expanded(
                child: Text(
                  report.geographicArea,
                  style: Theme.of(context).textTheme.bodySmall?.copyWith(
                    color: colors.onSurfaceVariant,
                  ),
                ),
              ),
            ],
          ),
        ],
        if (report.attachments.isNotEmpty) ...[
          AppSpacing.gapLg,

          const SectionHeader(
            title: 'Report documents',
            subtitle: 'Published with this report',
            icon: LucideIcons.paperclip,
          ),

          AppSpacing.gapSm,

          for (final attachment in report.attachments)
            _ResourceTile(
              title: attachment.title,
              type: attachment.description,
              onTap: () {
                _openOutbreakResource(context, attachment);
              },
            ),
        ] else if (report.reportAssetUrl.trim().isNotEmpty) ...[
          // Reports from before attachments carry a single uploaded PDF.
          AppSpacing.gapMd,
          OutlinedButton.icon(
            onPressed: () => context.push(
              AppRoutes.documentReader,
              extra: DocumentReaderArgs(
                title: report.title,
                source: report.reportAssetUrl,
              ),
            ),
            icon: const Icon(LucideIcons.fileText, size: 18),
            label: const Text('Read full report'),
          ),
        ],
        if (report.metrics.isNotEmpty) ...[
          AppSpacing.gapLg,

          Text(
            'Key indicators',
            style: Theme.of(
              context,
            ).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700),
          ),
          AppSpacing.gapSm,
          OutbreakMetricGrid(metrics: report.metrics, compact: true),
        ],
        if (report.summary.trim().isNotEmpty) ...[
          AppSpacing.gapLg,
          _ReportSummary(summary: report.summary),
        ],

        if (report.keyHighlights.isNotEmpty) ...[
          AppSpacing.gapLg,

          Text(
            'Key highlights',
            style: Theme.of(
              context,
            ).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700),
          ),

          AppSpacing.gapSm,

          for (final highlight in report.keyHighlights)
            _HighlightTile(text: highlight),
        ],
      ],
    );
  }
}

class _ReportSummary extends StatefulWidget {
  const _ReportSummary({required this.summary});
  final String summary;
  @override
  State<_ReportSummary> createState() => _ReportSummaryState();
}

class _ReportSummaryState extends State<_ReportSummary> {
  bool expanded = false;
  @override
  Widget build(BuildContext context) => Column(
    crossAxisAlignment: CrossAxisAlignment.start,
    children: [
      Text(
        'Summary',
        style: Theme.of(
          context,
        ).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700),
      ),
      const SizedBox(height: 8),
      Text(
        widget.summary,
        maxLines: expanded ? null : 3,
        overflow: expanded ? TextOverflow.visible : TextOverflow.ellipsis,
        style: Theme.of(context).textTheme.bodyMedium?.copyWith(height: 1.5),
      ),
      TextButton(
        onPressed: () => setState(() => expanded = !expanded),
        child: Text(expanded ? 'Show less' : 'Read summary'),
      ),
    ],
  );
}

// ===========================================================================
// HIGHLIGHT
// ===========================================================================

class _HighlightTile extends StatelessWidget {
  const _HighlightTile({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;

    return Container(
      margin: const EdgeInsets.only(bottom: AppSpacing.sm),
      padding: const EdgeInsets.all(AppSpacing.md),
      decoration: BoxDecoration(
        color: colors.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: colors.outlineVariant),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(LucideIcons.circleCheck, size: 18, color: colors.primary),

          AppSpacing.hGapSm,

          Expanded(
            child: Text(
              text,
              style: Theme.of(
                context,
              ).textTheme.bodyMedium?.copyWith(height: 1.45),
            ),
          ),
        ],
      ),
    );
  }
}

// ===========================================================================
// STATUS CHIP
// ===========================================================================

class _StatusChip extends StatelessWidget {
  const _StatusChip({required this.status, required this.color});

  final String status;
  final Color color;

  @override
  Widget build(BuildContext context) {
    final label = status.trim().isEmpty ? 'Published' : _capitalize(status);

    return Semantics(
      label: 'Status $label',
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 9, vertical: 5),
        decoration: BoxDecoration(
          color: color.withValues(alpha: 0.10),
          borderRadius: BorderRadius.circular(999),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(LucideIcons.activity, size: 13, color: color),
            const SizedBox(width: 5),
            Text(
              label,
              style: Theme.of(context).textTheme.labelSmall?.copyWith(
                color: color,
                fontWeight: FontWeight.w800,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// ===========================================================================
// META CHIP
// ===========================================================================

class _PublicEmptyState extends StatelessWidget {
  const _PublicEmptyState({
    required this.icon,
    required this.title,
    required this.message,
  });

  final IconData icon;
  final String title;
  final String message;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;

    return LayoutBuilder(
      builder: (context, constraints) {
        return SingleChildScrollView(
          physics: const AlwaysScrollableScrollPhysics(),
          child: ConstrainedBox(
            constraints: BoxConstraints(minHeight: constraints.maxHeight),
            child: Center(
              child: Padding(
                padding: AppSpacing.pagePadding,
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 420),
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      ClinicalIconTile(icon: icon, size: 72, iconSize: 34),

                      AppSpacing.gapLg,

                      Text(
                        title,
                        style: Theme.of(context).textTheme.titleLarge?.copyWith(
                          fontWeight: FontWeight.w800,
                        ),
                        textAlign: TextAlign.center,
                      ),

                      AppSpacing.gapSm,

                      Text(
                        message,
                        textAlign: TextAlign.center,
                        style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                          color: colors.onSurfaceVariant,
                          height: 1.45,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ),
        );
      },
    );
  }
}

class _FreshnessBanner extends StatelessWidget {
  const _FreshnessBanner({required this.metadata});

  final PublicCacheMetadata metadata;

  @override
  Widget build(BuildContext context) {
    if (!metadata.isOffline && !metadata.isStale && !metadata.isWithdrawn) {
      return const SizedBox.shrink();
    }
    final colors = Theme.of(context).colorScheme;
    final message = metadata.isWithdrawn
        ? 'This public item has been withdrawn.'
        : metadata.isStale
        ? 'Offline copy — verify critical details when connectivity returns.'
        : 'Showing a verified offline copy.';
    return Semantics(
      liveRegion: true,
      label: message,
      child: Container(
        width: double.infinity,
        padding: const EdgeInsets.all(AppSpacing.sm),
        decoration: BoxDecoration(
          color: metadata.isStale
              ? colors.errorContainer
              : colors.secondaryContainer,
          borderRadius: BorderRadius.circular(12),
        ),
        child: Row(
          children: [
            Icon(
              metadata.isStale
                  ? LucideIcons.triangleAlert
                  : LucideIcons.cloudOff,
              size: 18,
            ),
            AppSpacing.hGapSm,
            Expanded(child: Text(message)),
          ],
        ),
      ),
    );
  }
}

class _PublicUnavailableState extends StatelessWidget {
  const _PublicUnavailableState({
    required this.message,
    required this.withdrawn,
  });

  final String message;
  final bool withdrawn;

  @override
  Widget build(BuildContext context) => _PublicEmptyState(
    icon: withdrawn ? LucideIcons.fileX2 : LucideIcons.cloudOff,
    title: withdrawn ? 'Publication withdrawn' : 'Content unavailable',
    message: message,
  );
}

class _PartialContentBanner extends StatelessWidget {
  const _PartialContentBanner({required this.sections});

  final List<String> sections;

  @override
  Widget build(BuildContext context) => Container(
    width: double.infinity,
    padding: const EdgeInsets.all(AppSpacing.sm),
    decoration: BoxDecoration(
      color: Theme.of(context).colorScheme.tertiaryContainer,
      borderRadius: BorderRadius.circular(12),
    ),
    child: Text(
      'Some sections could not be refreshed: ${sections.join(', ')}. Pull to retry.',
    ),
  );
}

// ===========================================================================
// HELPERS
// ===========================================================================

Color _tone(BuildContext context, String value) {
  final colors = Theme.of(context).colorScheme;

  return switch (value.trim().toLowerCase()) {
    'critical' || 'emergency' => colors.error,
    'success' || 'resolved' => colors.tertiary,
    'info' => colors.primary,
    'neutral' => colors.onSurfaceVariant,
    _ => colors.secondary,
  };
}

Color _statusTone(
  BuildContext context, {
  required String status,
  required String visualTone,
}) {
  final colors = Theme.of(context).colorScheme;
  return switch (status.trim().toLowerCase()) {
    'active' => colors.error,
    'monitoring' => colors.tertiary,
    'contained' => colors.primary,
    'closed' => colors.onSurfaceVariant,
    _ => _tone(context, visualTone),
  };
}

String _date(BuildContext context, DateTime? value) {
  if (value == null) {
    return '';
  }

  final local = value.toLocal();

  return MaterialLocalizations.of(context).formatMediumDate(local);
}

String _capitalize(String value) {
  final normalized = value.trim();

  if (normalized.isEmpty) {
    return '';
  }

  return '${normalized[0].toUpperCase()}'
      '${normalized.substring(1).toLowerCase()}';
}

Future<void> _copyLink(BuildContext context, String value) async {
  final api = Uri.parse(AppConfig.current.apiBaseUrl);
  final absolute = api.replace(path: value, query: null, fragment: null);
  try {
    await Clipboard.setData(ClipboardData(text: absolute.toString()));
    if (context.mounted) {
      AppMessage.success(context, 'Link copied.');
    }
  } catch (_) {
    if (context.mounted) {
      AppMessage.error(context, 'The outbreak link could not be copied.');
    }
  }
}

Future<void> _openOutbreakResource(
  BuildContext context,
  PublicOutbreakResource resource,
) async {
  final target = NotificationActionResolver.fromOutbreakResource(
    type: resource.resourceType,
    url: resource.url,
    assetUrl: resource.assetUrl,
  );
  if (target?.location case final String location) {
    context.push(location);
    return;
  }
  if (target?.externalUri case final Uri uri) {
    try {
      final launched = await launchUrl(
        uri,
        mode: LaunchMode.externalApplication,
      );
      if (!launched && context.mounted) {
        AppMessage.error(context, 'Unable to open this trusted resource.');
      }
    } catch (_) {
      if (context.mounted) {
        AppMessage.error(context, 'Unable to open this trusted resource.');
      }
    }
    return;
  }
  if (context.mounted) {
    AppMessage.warning(context, 'This resource link is unavailable.');
  }
}
