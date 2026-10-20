part of '../screens/tools_page.dart';

class _ToolTypeChip extends StatelessWidget {
  const _ToolTypeChip({
    required this.label,
    required this.icon,
    required this.selected,
    required this.onTap,
  });

  final String label;
  final IconData icon;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return AppChoiceChip(
      label: Text(label),
      icon: icon,
      selected: selected,
      onSelected: onTap,
    );
  }
}

// ============================================================================
// TOOL FILTER MODEL
// ============================================================================
