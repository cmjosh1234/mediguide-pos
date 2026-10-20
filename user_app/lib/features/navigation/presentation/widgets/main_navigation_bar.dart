import 'package:flutter/material.dart';

/// Labels are capped below NavigationBar's own 1.3x so five tabs, including
/// "My Library", stay on one line on 360dp phones at large text sizes.
const double mainNavigationMaxTextScale = 1.15;

/// Bottom navigation bar for compact layouts.
class MainNavigationBar extends StatelessWidget {
  const MainNavigationBar({
    super.key,
    required this.destinations,
    required this.selectedIndex,
    required this.onDestinationSelected,
  });

  final List<(IconData, String)> destinations;
  final int selectedIndex;
  final ValueChanged<int> onDestinationSelected;

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: BoxDecoration(
        border: Border(
          top: BorderSide(color: Theme.of(context).colorScheme.outlineVariant),
        ),
      ),
      child: MediaQuery.withClampedTextScaling(
        maxScaleFactor: mainNavigationMaxTextScale,
        child: NavigationBar(
          selectedIndex: selectedIndex,
          onDestinationSelected: onDestinationSelected,
          destinations: [
            for (final destination in destinations)
              NavigationDestination(
                icon: Icon(destination.$1),
                selectedIcon: Icon(destination.$1, fill: 1),
                label: destination.$2,
              ),
          ],
        ),
      ),
    );
  }
}
