import 'package:flutter/material.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';

/// Each public content type gets its own label, icon, tint and call to action
/// so a guideline, a live outbreak and a situation report read differently at
/// a glance. Keep this table in step with the web portal's resource-kinds.ts.
final class ResourceKind {
  const ResourceKind({
    required this.label,
    required this.icon,
    required this.action,
    this.tint,
    this.container,
  });

  final String label;
  final IconData icon;
  final String action;

  /// Null for neutral kinds, which follow the theme's muted colours.
  final Color? tint;
  final Color? container;

  /// The tint for text and icons, lightened on dark surfaces.
  Color foreground(ColorScheme colors) {
    final value = tint;
    if (value == null) return colors.onSurfaceVariant;
    return colors.brightness == Brightness.dark
        ? Color.lerp(value, Colors.white, 0.45)!
        : value;
  }

  /// The icon tile background, a translucent tint on dark surfaces.
  Color background(ColorScheme colors) {
    final value = tint;
    if (value == null) return colors.surfaceContainerHighest;
    return colors.brightness == Brightness.dark
        ? value.withValues(alpha: 0.22)
        : container!;
  }
}

const _guideline = Color(0xFF1A735A);
const _outbreak = Color(0xFFB42328);
const _report = Color(0xFF0B6FA4);
const _tool = Color(0xFF4F46E5);
const _drug = Color(0xFF9A5B00);

const _kinds = <String, ResourceKind>{
  'guideline': ResourceKind(
    label: 'Guideline',
    icon: LucideIcons.bookOpen,
    action: 'Read guideline',
    tint: _guideline,
    container: Color(0xFFE7F4EF),
  ),
  'outbreak': ResourceKind(
    label: 'Outbreak',
    icon: LucideIcons.siren,
    action: 'Open outbreak',
    tint: _outbreak,
    container: Color(0xFFFDECEC),
  ),
  'outbreak_document': ResourceKind(
    label: 'Response document',
    icon: LucideIcons.fileText,
    action: 'Read document',
    tint: _outbreak,
    container: Color(0xFFFDECEC),
  ),
  'situation_report': ResourceKind(
    label: 'Situation report',
    icon: LucideIcons.chartLine,
    action: 'Read report',
    tint: _report,
    container: Color(0xFFE6F4FB),
  ),
  'clinical_tool': ResourceKind(
    label: 'Clinical tool',
    icon: LucideIcons.calculator,
    action: 'Open tool',
    tint: _tool,
    container: Color(0xFFECEEFE),
  ),
  'algorithm': ResourceKind(
    label: 'Algorithm',
    icon: LucideIcons.workflow,
    action: 'Open algorithm',
    tint: _tool,
    container: Color(0xFFECEEFE),
  ),
  'drug_reference': ResourceKind(
    label: 'Drug reference',
    icon: LucideIcons.pill,
    action: 'Open drug reference',
    tint: _drug,
    container: Color(0xFFFFF4E0),
  ),
  'approved_external_url': ResourceKind(
    label: 'External resource',
    icon: LucideIcons.externalLink,
    action: 'Visit website',
  ),
};

ResourceKind resourceKindOf(String contentType) {
  final known = _kinds[contentType];
  if (known != null) return known;
  final words = contentType.replaceAll('_', ' ').trim();
  return ResourceKind(
    label: words.isEmpty
        ? 'Resource'
        : '${words[0].toUpperCase()}${words.substring(1)}',
    icon: LucideIcons.fileText,
    action: 'Open',
  );
}
