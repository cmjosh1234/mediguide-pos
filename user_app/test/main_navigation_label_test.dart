import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';
import 'package:user_app/app/theme/app_theme.dart';
import 'package:user_app/features/navigation/presentation/widgets/main_navigation_bar.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  setUpAll(() async {
    final font = FontLoader('Geist');
    for (final weight in ['Regular', 'Medium', 'SemiBold', 'Bold']) {
      font.addFont(rootBundle.load('assets/fonts/Geist-$weight.ttf'));
    }
    await font.load();
  });

  Future<void> pumpBar(WidgetTester tester, double width, int selected) async {
    tester.view.devicePixelRatio = 3;
    tester.view.physicalSize = Size(width * 3, 800 * 3);
    addTearDown(tester.view.reset);
    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.light,
        // Large accessibility text size.
        builder: (context, child) => MediaQuery(
          data: MediaQuery.of(
            context,
          ).copyWith(textScaler: const TextScaler.linear(2)),
          child: child!,
        ),
        home: Scaffold(
          bottomNavigationBar: MainNavigationBar(
            selectedIndex: selected,
            onDestinationSelected: (_) {},
            destinations: const [
              (LucideIcons.house, 'Home'),
              (LucideIcons.search, 'Search'),
              (LucideIcons.library, 'My Library'),
              (LucideIcons.grid2x2, 'Tools'),
              (LucideIcons.user, 'Profile'),
            ],
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
  }

  for (final width in [360.0, 393.0]) {
    for (final selected in [0, 2]) {
      testWidgets(
        'My Library label stays on one line at ${width}dp, tab $selected selected',
        (tester) async {
          await pumpBar(tester, width, selected);
          final singleLine = tester.getSize(find.text('Home')).height;
          expect(tester.getSize(find.text('My Library')).height, singleLine);
        },
      );
    }
  }

  testWidgets('selecting a tab does not change its label width', (
    tester,
  ) async {
    await pumpBar(tester, 393, 0);
    final unselected = tester.getSize(find.text('My Library')).width;
    await pumpBar(tester, 393, 2);
    expect(tester.getSize(find.text('My Library')).width, unselected);
  });
}
