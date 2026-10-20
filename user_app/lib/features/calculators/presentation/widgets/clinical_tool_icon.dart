import 'package:flutter/widgets.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';

/// Lucide icon names a tool's `icon` field may store, matching the dashboard's
/// curated picker (dashboard/components/ui/clinical-icons.ts). Keep in sync.
const Map<String, IconData> clinicalToolIcons = {
  'calculator': LucideIcons.calculator,
  'list-checks': LucideIcons.listChecks,
  'clipboard-list': LucideIcons.clipboardList,
  'clipboard-check': LucideIcons.clipboardCheck,
  'gauge': LucideIcons.gauge,
  'timer': LucideIcons.timer,
  'weight': LucideIcons.weight,
  'ruler': LucideIcons.ruler,
  'stethoscope': LucideIcons.stethoscope,
  'activity': LucideIcons.activity,
  'heart': LucideIcons.heart,
  'heart-pulse': LucideIcons.heartPulse,
  'droplet': LucideIcons.droplet,
  'droplets': LucideIcons.droplets,
  'brain': LucideIcons.brain,
  'baby': LucideIcons.baby,
  'calendar-heart': LucideIcons.calendarHeart,
  'hand-heart': LucideIcons.handHeart,
  'person-standing': LucideIcons.personStanding,
  'thermometer': LucideIcons.thermometer,
  'siren': LucideIcons.siren,
  'badge-alert': LucideIcons.badgeAlert,
  'bandage': LucideIcons.bandage,
  'bone': LucideIcons.bone,
  'pill': LucideIcons.pill,
  'syringe': LucideIcons.syringe,
  'flask-conical': LucideIcons.flaskConical,
  'microscope': LucideIcons.microscope,
  'dna': LucideIcons.dna,
  'wind': LucideIcons.wind,
  'bean': LucideIcons.bean,
  'eye': LucideIcons.eye,
  'ear': LucideIcons.ear,
  'hand': LucideIcons.hand,
  'salad': LucideIcons.salad,
  'biohazard': LucideIcons.biohazard,
  'bug': LucideIcons.bug,
  'worm': LucideIcons.worm,
  'ribbon': LucideIcons.ribbon,
  'skull': LucideIcons.skull,
  'shield-alert': LucideIcons.shieldAlert,
  'shield-check': LucideIcons.shieldCheck,
  'shield-plus': LucideIcons.shieldPlus,
};

// Emoji and pictographic symbol blocks; Dart's RegExp has no
// Extended_Pictographic property, so match by code point instead.
bool _isEmoji(String value) => value.runes.any(
  (rune) =>
      (rune >= 0x1F000 && rune <= 0x1FAFF) ||
      (rune >= 0x2600 && rune <= 0x27BF) ||
      (rune >= 0x2300 && rune <= 0x23FF) ||
      (rune >= 0x2B00 && rune <= 0x2BFF),
);

/// Shows a tool's stored icon: a curated Lucide name renders as that icon, an
/// emoji renders as text, and anything else renders [fallback].
class ClinicalToolIcon extends StatelessWidget {
  const ClinicalToolIcon({
    super.key,
    required this.icon,
    required this.fallback,
    required this.size,
    required this.color,
  });

  final String icon;
  final IconData fallback;
  final double size;
  final Color color;

  @override
  Widget build(BuildContext context) {
    final value = icon.trim();
    final named = clinicalToolIcons[value.toLowerCase()];
    if (named != null) return Icon(named, size: size, color: color);
    if (_isEmoji(value)) {
      return Text(value, style: TextStyle(fontSize: size, height: 1));
    }
    return Icon(fallback, size: size, color: color);
  }
}
