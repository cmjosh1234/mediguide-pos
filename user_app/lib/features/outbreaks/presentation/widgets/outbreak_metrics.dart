import 'package:flutter/material.dart';
import 'package:intl/intl.dart';

import 'package:user_app/core/constants/app_spacing.dart';
import 'package:user_app/core/utils/date_utils.dart';
import 'package:user_app/features/outbreaks/data/models/outbreak_models.dart';

class OutbreakMetricGrid extends StatelessWidget {
  const OutbreakMetricGrid({
    super.key,
    required this.metrics,
    this.compact = false,
  });

  final List<OutbreakMetric> metrics;
  final bool compact;

  @override
  Widget build(BuildContext context) => LayoutBuilder(
    builder: (context, constraints) {
      if (compact) {
        final colors = Theme.of(context).colorScheme;
        final largeText = MediaQuery.textScalerOf(context).scale(1) > 1.3;
        return Container(
          decoration: BoxDecoration(
            color: colors.surfaceContainerLowest,
            borderRadius: BorderRadius.circular(16),
            border: Border.all(color: colors.outlineVariant),
          ),
          child: Column(
            children: [
              for (var index = 0; index < metrics.length; index++) ...[
                if (index > 0) Divider(height: 1, color: colors.outlineVariant),
                Padding(
                  padding: const EdgeInsets.all(16),
                  child: Semantics(
                    label:
                        '${metrics[index].label}: ${metrics[index].value} ${metrics[index].unit}',
                    child: largeText
                        ? Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              _MetricValue(metric: metrics[index]),
                              const SizedBox(height: 6),
                              Text(metrics[index].label),
                            ],
                          )
                        : Row(
                            children: [
                              SizedBox(
                                width: 100,
                                child: _MetricValue(metric: metrics[index]),
                              ),
                              const SizedBox(width: 16),
                              Expanded(
                                child: Text(
                                  metrics[index].label,
                                  style: Theme.of(context).textTheme.bodyMedium,
                                ),
                              ),
                            ],
                          ),
                  ),
                ),
              ],
            ],
          ),
        );
      }
      final columns = constraints.maxWidth >= 900
          ? 4
          : constraints.maxWidth >= 600
          ? 3
          : constraints.maxWidth < 340
          ? 1
          : 2;
      final itemWidth =
          (constraints.maxWidth - AppSpacing.sm * (columns - 1)) / columns;
      return Wrap(
        spacing: AppSpacing.sm,
        runSpacing: AppSpacing.sm,
        children: [
          for (final metric in metrics)
            SizedBox(
              width: itemWidth,
              child: OutbreakMetricCard(metric: metric),
            ),
        ],
      );
    },
  );
}

class _MetricValue extends StatelessWidget {
  const _MetricValue({required this.metric});
  final OutbreakMetric metric;
  @override
  Widget build(BuildContext context) => Column(
    crossAxisAlignment: CrossAxisAlignment.start,
    children: [
      Text(
        metric.value,
        style: Theme.of(context).textTheme.titleLarge?.copyWith(
          fontWeight: FontWeight.w800,
          color: Theme.of(context).colorScheme.primary,
        ),
      ),
      if (metric.unit.isNotEmpty &&
          !metric.label.toLowerCase().contains(metric.unit.toLowerCase()))
        Text(metric.unit, style: Theme.of(context).textTheme.bodySmall),
    ],
  );
}

class OutbreakMetricCard extends StatelessWidget {
  const OutbreakMetricCard({super.key, required this.metric});

  final OutbreakMetric metric;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    final value = metricDisplayValue(metric);
    final unit = metric.unit.trim();
    final provenance = metricProvenance(metric);
    return Semantics(
      label: [
        '${metric.label}: $value $unit'.trim(),
        if (provenance.isNotEmpty) provenance,
      ].join('. '),
      child: Container(
        padding: const EdgeInsets.all(AppSpacing.md),
        decoration: BoxDecoration(
          color: colors.surfaceContainerLowest,
          borderRadius: BorderRadius.circular(16),
          border: Border.all(color: colors.outlineVariant),
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              unit.isEmpty ? value : '$value $unit',
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
              style: Theme.of(
                context,
              ).textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.w900),
            ),
            const SizedBox(height: 5),
            Text(
              metric.label,
              style: Theme.of(
                context,
              ).textTheme.bodySmall?.copyWith(color: colors.onSurfaceVariant),
            ),
            if (provenance.isNotEmpty) ...[
              const SizedBox(height: AppSpacing.xs),
              Text(
                provenance,
                style: Theme.of(context).textTheme.labelSmall?.copyWith(
                  color: colors.onSurfaceVariant,
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

// Categorical metrics carry only a text value, so the numeric value is
// preferred when present and the text value is the fallback (as on the web).
String metricDisplayValue(OutbreakMetric metric) => metric.numericValue == null
    ? metric.value
    : NumberFormat.decimalPattern().format(metric.numericValue);

String metricProvenance(OutbreakMetric metric) => [
  metric.sourceReference.trim(),
  if (metric.asOf != null)
    'as of ${AppDateUtils.formatDate(metric.asOf!.toLocal())}',
].where((part) => part.isNotEmpty).join(' · ');
