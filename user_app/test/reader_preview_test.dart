import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import '../tool/reader_preview.dart' as preview;

void main() {
  testWidgets('gallery opens a sample procedure and its partial variant', (
    tester,
  ) async {
    preview.main();
    await tester.pumpAndSettle();
    expect(find.text('Document reader previews'), findsOneWidget);
    await tester.tap(find.text('SOP'));
    await tester.pumpAndSettle();
    expect(find.text('Sample SOP'), findsWidgets);
    expect(find.text('Procedure reader'), findsOneWidget);
    await tester.pageBack();
    await tester.pumpAndSettle();
    await tester.tap(find.byType(SwitchListTile));
    await tester.pumpAndSettle();
    expect(
      tester.widget<SwitchListTile>(find.byType(SwitchListTile)).value,
      isTrue,
    );
    await tester.tap(find.text('SOP'));
    await tester.pumpAndSettle();
    expect(find.textContaining('Partially reviewed content.'), findsOneWidget);
    await tester.pumpWidget(const SizedBox.shrink());
  });
}
