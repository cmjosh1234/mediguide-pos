part of '../screens/guidelines_indexer_page.dart';

class _BrowseChip extends StatelessWidget {
  const _BrowseChip({
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
// ACTIVE FILTERS
// ===========================================================================
