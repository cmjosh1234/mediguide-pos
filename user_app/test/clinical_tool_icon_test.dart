import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';
import 'package:user_app/features/calculators/presentation/widgets/clinical_tool_icon.dart';

Future<void> pumpIcon(WidgetTester tester, String icon) => tester.pumpWidget(
  MaterialApp(
    home: ClinicalToolIcon(
      icon: icon,
      fallback: LucideIcons.calculator,
      size: 22,
      color: Colors.black,
    ),
  ),
);

void main() {
  testWidgets('renders the stored Lucide icon name', (tester) async {
    await pumpIcon(tester, 'baby');
    expect(find.byIcon(LucideIcons.baby), findsOneWidget);
    expect(find.byIcon(LucideIcons.calculator), findsNothing);
  });

  testWidgets('renders emoji icons as text', (tester) async {
    await pumpIcon(tester, '🩺');
    expect(find.text('🩺'), findsOneWidget);
  });

  testWidgets('falls back to the type icon for empty or unknown names', (
    tester,
  ) async {
    await pumpIcon(tester, '');
    expect(find.byIcon(LucideIcons.calculator), findsOneWidget);
    await pumpIcon(tester, 'not-a-real-icon');
    expect(find.byIcon(LucideIcons.calculator), findsOneWidget);
    expect(find.text('not-a-real-icon'), findsNothing);
  });

  test('covers every icon the seeded tools use', () {
    const seeded = [
      'calculator',
      'baby',
      'activity',
      'droplets',
      'calendar-heart',
      'heart-pulse',
      'heart',
      'droplet',
      'siren',
      'brain',
      'badge-alert',
      'thermometer',
      'bandage',
      'list-checks',
    ];
    for (final name in seeded) {
      expect(clinicalToolIcons, contains(name));
    }
  });
}
