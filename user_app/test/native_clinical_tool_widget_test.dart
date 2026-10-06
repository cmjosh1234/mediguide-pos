import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:user_app/features/calculators/data/models/clinical_tool_definition.dart';
import 'package:user_app/features/calculators/presentation/widgets/native_clinical_tool.dart';

void main() {
  const definition = ClinicalToolDefinition(
    schemaVersion: '1.0',
    toolType: 'calculator',
    title: 'Dose helper',
    description: 'A native clinical tool.',
    version: '1.0.0',
    warnings: [
      ClinicalToolMessage(
        key: 'warning',
        text: 'Confirm the patient details.',
        severity: 'warning',
      ),
    ],
    inputs: [
      ClinicalToolInput(
        key: 'weight',
        type: 'number',
        label: 'Weight',
        required: true,
        defaultUnit: 'kg',
      ),
    ],
    outputs: [
      ClinicalToolOutput(
        key: 'result',
        label: 'Dose',
        unit: 'mg',
        value: ClinicalToolExpression(
          op: 'multiply',
          args: [
            ClinicalToolExpression(op: 'field', field: 'weight'),
            ClinicalToolExpression(op: 'literal', value: 2),
          ],
        ),
      ),
    ],
    completion: ClinicalToolCompletion(mode: 'none'),
  );

  testWidgets(
    'renders and calculates natively at large text scale in dark mode',
    (tester) async {
      await tester.pumpWidget(
        MediaQuery(
          data: const MediaQueryData(textScaler: TextScaler.linear(2)),
          child: MaterialApp(
            theme: ThemeData.dark().copyWith(
              splashFactory: NoSplash.splashFactory,
            ),
            home: const Scaffold(
              body: NativeClinicalTool(definition: definition),
            ),
          ),
        ),
      );
      expect(find.text('Confirm the patient details.'), findsOneWidget);
      await tester.enterText(find.byType(TextFormField), '30');
      await tester.tap(find.text('Calculate'));
      await tester.pump();
      expect(find.textContaining('Dose: 60'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets('restores workflow values and reports changes', (tester) async {
    Map<String, Object?>? saved;
    await tester.pumpWidget(
      MaterialApp(
        theme: ThemeData(splashFactory: NoSplash.splashFactory),
        home: Scaffold(
          body: NativeClinicalTool(
            definition: definition,
            initialValues: const {'weight': 45},
            onChanged: (value) => saved = value,
          ),
        ),
      ),
    );
    expect(find.text('45'), findsOneWidget);
    await tester.enterText(find.byType(TextFormField), '50');
    expect(saved?['weight'], 50);
  });

  testWidgets('reset clears inputs and computed results', (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        home: const Scaffold(body: NativeClinicalTool(definition: definition)),
      ),
    );
    await tester.enterText(find.byType(TextFormField), '30');
    await tester.tap(find.text('Calculate'));
    await tester.pump();
    expect(find.textContaining('Dose: 60'), findsOneWidget);
    await tester.tap(find.text('Reset'));
    await tester.pumpAndSettle();
    expect(find.text('Reset all responses?'), findsOneWidget);
    await tester.tap(find.widgetWithText(TextButton, 'Reset').last);
    await tester.pumpAndSettle();
    expect(find.textContaining('Dose: 60'), findsNothing);
    expect(find.text('30'), findsNothing);
  });
  testWidgets('measurement defaults render and edits invalidate results', (
    tester,
  ) async {
    final measured = definition.copyWith(
      inputs: const [
        ClinicalToolInput(
          key: 'weight',
          type: 'measurement',
          label: 'Weight',
          required: true,
          defaultValue: 30,
          defaultUnit: 'kg',
          allowedUnits: ['kg', 'lb'],
          minimum: 1,
          maximum: 100,
        ),
      ],
    );
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(body: NativeClinicalTool(definition: measured)),
      ),
    );
    expect(find.text('30'), findsOneWidget);
    await tester.tap(find.text('Calculate'));
    await tester.pump();
    expect(find.textContaining('Dose: 60'), findsOneWidget);
    await tester.enterText(find.byType(TextFormField), '40');
    await tester.pump();
    expect(find.textContaining('Dose: 60'), findsNothing);
    await tester.tap(find.text('Calculate'));
    await tester.pump();
    expect(find.textContaining('Dose: 80'), findsOneWidget);
  });

  testWidgets('conditional required fields follow selected method', (
    tester,
  ) async {
    final conditional = definition.copyWith(
      inputs: const [
        ClinicalToolInput(
          key: 'method',
          type: 'single_selection',
          label: 'Method',
          defaultValue: 'preset',
          options: [
            ClinicalToolOption(value: 'preset', label: 'Preset'),
            ClinicalToolOption(value: 'custom', label: 'Custom'),
          ],
        ),
        ClinicalToolInput(
          key: 'weight',
          type: 'number',
          label: 'Weight',
          defaultValue: 30,
          required: true,
          visibleWhen: ClinicalToolExpression(
            op: 'equal',
            args: [
              ClinicalToolExpression(op: 'field', field: 'method'),
              ClinicalToolExpression(op: 'literal', value: 'custom'),
            ],
          ),
        ),
      ],
    );
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(body: NativeClinicalTool(definition: conditional)),
      ),
    );
    expect(find.byType(TextFormField), findsNothing);
    await tester.tap(find.text('Preset'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Custom').last);
    await tester.pumpAndSettle();
    expect(find.byType(TextFormField), findsOneWidget);
    expect(find.text('30'), findsOneWidget);
  });

  testWidgets('uses radio, multiline and calendar controls from the schema', (
    tester,
  ) async {
    Map<String, Object?>? saved;
    final controls = definition.copyWith(
      inputs: const [
        ClinicalToolInput(
          key: 'score',
          type: 'single_selection',
          control: 'radio',
          label: 'Score',
          options: [
            ClinicalToolOption(value: 0, label: 'Absent'),
            ClinicalToolOption(value: 2, label: 'Normal'),
          ],
        ),
        ClinicalToolInput(
          key: 'notes',
          type: 'text',
          control: 'textarea',
          label: 'Notes',
        ),
        ClinicalToolInput(
          key: 'date',
          type: 'date',
          label: 'Assessment date',
          defaultValue: '2026-10-01',
        ),
      ],
    );
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: NativeClinicalTool(
            definition: controls,
            onChanged: (value) => saved = value,
          ),
        ),
      ),
    );
    expect(find.byType(RadioListTile<Object?>), findsNWidgets(2));
    await tester.tap(find.text('Normal'));
    await tester.pump();
    expect(saved?['score'], 2);
    final fields = find.byType(TextFormField);
    final notes = tester.widget<TextField>(
      find.descendant(of: fields.first, matching: find.byType(TextField)),
    );
    expect(notes.minLines, 3);
    expect(notes.maxLines, 6);
    await tester.enterText(fields.first, 'Movement worsens pain\nRest helps');
    expect(saved?['notes'], 'Movement worsens pain\nRest helps');
    await tester.ensureVisible(fields.last);
    await tester.pumpAndSettle();
    await tester.tap(fields.last);
    await tester.pumpAndSettle();
    expect(find.byType(DatePickerDialog), findsOneWidget);
    await tester.tap(find.text('OK'));
    await tester.pumpAndSettle();
    expect(saved?['date'], '2026-10-01');
    expect(saved?['notes'], 'Movement worsens pain\nRest helps');
    expect(saved?['score'], 2);
    expect(tester.takeException(), isNull);
  });
}
