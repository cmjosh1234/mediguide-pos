import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:user_app/features/documents/presentation/widgets/document_page_picker.dart';

void main() {
  testWidgets(
    'page picker validates page bounds and returns a one-based page',
    (tester) async {
      int? selected;
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: Builder(
              builder: (context) => TextButton(
                onPressed: () async {
                  selected = await showDialog<int>(
                    context: context,
                    builder: (_) =>
                        const DocumentPagePicker(pageCount: 5, currentPage: 2),
                  );
                },
                child: const Text('Choose page'),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('Choose page'));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(TextFormField), '6');
      await tester.tap(find.text('Go'));
      await tester.pumpAndSettle();
      expect(find.text('Enter a page from 1 to 5.'), findsOneWidget);
      expect(selected, isNull);
      await tester.enterText(find.byType(TextFormField), '4');
      await tester.tap(find.text('Go'));
      await tester.pumpAndSettle();
      expect(selected, 4);
    },
  );
}
