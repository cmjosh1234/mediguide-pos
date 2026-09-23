import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:user_app/features/guidelines/data/models/guideline_publication.dart';
import 'package:user_app/features/guidelines/presentation/widgets/responsive_clinical_table.dart';

final Finder _horizontalScrollView = find.byWidgetPredicate(
  (widget) =>
      widget is SingleChildScrollView &&
      widget.scrollDirection == Axis.horizontal,
);

Widget _host({required double width, required GuidelineTablePayload payload}) =>
    MaterialApp(
      home: Scaffold(
        body: SizedBox(
          width: width,
          child: ResponsiveClinicalTable(payload: payload),
        ),
      ),
    );

void main() {
  testWidgets(
    'wide clinical tables stay a table and scroll sideways within the window',
    (tester) async {
      await tester.pumpWidget(
        _host(
          width: 320,
          payload: const GuidelineTablePayload(
            title: 'Diagnostic tests',
            columns: ['Test', 'Typical setting', 'Result'],
            rows: [
              ['RDT', 'Point of care', 'Antigen detection'],
              ['Microscopy', 'Laboratory', 'Parasite count'],
            ],
          ),
        ),
      );

      expect(find.byType(Table), findsOneWidget);
      expect(_horizontalScrollView, findsOneWidget);

      // The scrolling viewport never grows past the space it was given, while
      // the table inside it keeps full-width columns to scroll through.
      expect(tester.getSize(_horizontalScrollView).width, 320);
      expect(tester.getSize(find.byType(Table)).width, greaterThan(320));

      expect(find.text('Typical setting'), findsOneWidget);
      expect(find.text('Point of care'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets('clinical tables that fit fill the width without scrolling', (
    tester,
  ) async {
    await tester.pumpWidget(
      _host(
        width: 390,
        payload: const GuidelineTablePayload(
          columns: ['Abbreviation', 'Meaning'],
          rows: [
            ['HbA1c', 'Glycated haemoglobin'],
          ],
        ),
      ),
    );

    expect(find.byType(Table), findsOneWidget);
    expect(tester.getSize(find.byType(Table)).width, 390);
    expect(
      tester.widget<SingleChildScrollView>(_horizontalScrollView).physics,
      isA<NeverScrollableScrollPhysics>(),
    );
    expect(find.text('Glycated haemoglobin'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('short rows are padded so every column keeps a cell', (
    tester,
  ) async {
    await tester.pumpWidget(
      _host(
        width: 320,
        payload: const GuidelineTablePayload(
          columns: ['Drug', 'Dose', 'Route'],
          rows: [
            ['Artemether'],
          ],
        ),
      ),
    );

    final table = tester.widget<Table>(find.byType(Table));
    expect(table.children, hasLength(2));
    expect(table.children.last.children, hasLength(3));
    expect(find.text('Artemether'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
}
