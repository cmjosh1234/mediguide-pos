part of '../screens/help_center_page.dart';

class _StatusFilterChip extends StatelessWidget {
  const _StatusFilterChip({
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

// ===========================================================================
// TICKET SHELL
// ===========================================================================
