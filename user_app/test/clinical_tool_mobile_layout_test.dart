import 'dart:convert';
import 'dart:io';
import 'dart:ui' as ui;
import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:user_app/app/theme/app_theme.dart';
import 'package:user_app/features/calculators/data/models/clinical_tool_definition.dart';
import 'package:user_app/features/calculators/presentation/widgets/native_clinical_tool.dart';

ClinicalToolDefinition load(String name) => ClinicalToolDefinition.fromJson(
  (jsonDecode(
            File(
              '../clinical-tools/migrations/v1/definitions/$name.json',
            ).readAsStringSync(),
          )
          as Map<String, dynamic>)['definition']
      as Map<String, dynamic>,
);

Map<String, Object?> fixtureInputs(String name) {
  final envelope =
      jsonDecode(
            File(
              '../clinical-tools/migrations/v1/definitions/$name.json',
            ).readAsStringSync(),
          )
          as Map<String, dynamic>;
  return Map<String, Object?>.from(
    (envelope['definition']['test_cases'] as List).first['inputs'] as Map,
  );
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  setUpAll(() async {
    final font = FontLoader('Geist')
      ..addFont(rootBundle.load('assets/fonts/Geist-Regular.ttf'));
    await font.load();
    final icons = FontLoader('MaterialIcons')
      ..addFont(rootBundle.load('fonts/MaterialIcons-Regular.otf'));
    await icons.load();
  });
  final names =
      (jsonDecode(
                File(
                  '../clinical-tools/migrations/v1/catalog.json',
                ).readAsStringSync(),
              )
              as Map<String, dynamic>)['tools']
          as List;
  for (final scale in [1.0, 2.0]) {
    for (final item in names) {
      final name = (item as Map)['legacy_id'] as String;
      testWidgets('$name fits a 320px phone at text scale $scale', (
        tester,
      ) async {
        tester.view.physicalSize = const Size(320, 740);
        tester.view.devicePixelRatio = 1;
        addTearDown(tester.view.resetPhysicalSize);
        addTearDown(tester.view.resetDevicePixelRatio);
        await tester.pumpWidget(
          MaterialApp(
            theme: scale == 2 ? AppTheme.dark : AppTheme.light,
            home: Scaffold(
              body: MediaQuery(
                data: MediaQueryData(
                  size: const Size(320, 740),
                  textScaler: TextScaler.linear(scale),
                ),
                child: NativeClinicalTool(
                  definition: load(name),
                  initialValues: fixtureInputs(name),
                ),
              ),
            ),
          ),
        );
        await tester.pumpAndSettle();
        expect(tester.takeException(), isNull);
        final action = find.widgetWithText(FilledButton, 'Calculate');
        expect(tester.getRect(action).bottom, lessThanOrEqualTo(740));
        expect(tester.getSize(action).height, greaterThanOrEqualTo(48));
        await tester.tap(action);
        await tester.pumpAndSettle();
        expect(find.text('Result'), findsOneWidget);
        expect(tester.takeException(), isNull);
        final scroll = find.byType(SingleChildScrollView).first;
        await tester.drag(scroll, const Offset(0, -1200));
        await tester.pumpAndSettle();
        expect(tester.takeException(), isNull);
        expect(tester.getRect(action).bottom, lessThanOrEqualTo(740));
      });
    }
  }

  testWidgets(
    'long selections wrap in a mobile sheet and retain typed values',
    (tester) async {
      tester.view.physicalSize = const Size(320, 740);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final definition = load('pregnancy-due-date-calculator');
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.light,
          home: Scaffold(body: NativeClinicalTool(definition: definition)),
        ),
      );
      final method = definition.inputs.firstWhere(
        (input) => input.key == 'method',
      );
      await tester.tap(
        find.text(
          method.options.firstWhere((option) => option.value == 'lmp').label,
        ),
      );
      await tester.pumpAndSettle();
      final ultrasound = method.options.firstWhere(
        (option) => option.value == 'ultrasound',
      );
      await tester.tap(find.text(ultrasound.label).last);
      await tester.pumpAndSettle();
      expect(find.text('Ultrasound date *'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets('required fields below the fold are brought into view', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(320, 740);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final definition = load('apgar-score-calculator');
    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.light,
        home: Scaffold(body: NativeClinicalTool(definition: definition)),
      ),
    );
    await tester.tap(find.text('Calculate'));
    await tester.pumpAndSettle();
    final message = find.text(
      '${definition.inputs.firstWhere((input) => input.required).label} is required',
    );
    expect(message, findsOneWidget);
    expect(
      tester.getRect(message).bottom,
      lessThan(
        tester.getRect(find.widgetWithText(FilledButton, 'Calculate')).top,
      ),
    );
    expect(tester.takeException(), isNull);
  });

  testWidgets(
    'actions and results remain accessible with the phone keyboard open',
    (tester) async {
      tester.view.physicalSize = const Size(320, 740);
      tester.view.devicePixelRatio = 1;
      tester.view.viewInsets = const FakeViewPadding(bottom: 260);
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      addTearDown(tester.view.resetViewInsets);
      final definition = load('bmi-calculator');
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.light,
          home: Scaffold(
            body: NativeClinicalTool(
              definition: definition,
              initialValues: const {'weight': 70, 'height': 1.75},
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      final action = find.widgetWithText(FilledButton, 'Calculate');
      expect(tester.getRect(action).bottom, lessThanOrEqualTo(480));
      await tester.tap(action);
      await tester.pumpAndSettle();
      expect(find.textContaining('22.9'), findsOneWidget);
      expect(tester.getRect(find.text('Result')).top, greaterThanOrEqualTo(0));
      expect(
        tester.getRect(find.text('Result')).bottom,
        lessThan(tester.getRect(action).top),
      );
      expect(tester.takeException(), isNull);
    },
  );

  // Optional screenshots for visual review; no image baselines are required.
  if (const bool.fromEnvironment('CLINICAL_MOBILE_SCREENSHOTS')) {
    for (final dark in [false, true]) {
      testWidgets('export ${dark ? 'dark' : 'light'} mobile preview', (
        tester,
      ) async {
        tester.view.physicalSize = const Size(390, 844);
        tester.view.devicePixelRatio = 1;
        addTearDown(tester.view.resetPhysicalSize);
        addTearDown(tester.view.resetDevicePixelRatio);
        final key = GlobalKey();
        await tester.pumpWidget(
          MaterialApp(
            theme: dark ? AppTheme.dark : AppTheme.light,
            home: RepaintBoundary(
              key: key,
              child: Scaffold(
                body: SafeArea(
                  child: NativeClinicalTool(definition: load('bmi-calculator')),
                ),
              ),
            ),
          ),
        );
        await tester.pumpAndSettle();
        await tester.runAsync(() async {
          final boundary =
              key.currentContext!.findRenderObject()! as RenderRepaintBoundary;
          final image = await boundary.toImage(pixelRatio: 2);
          final data = await image.toByteData(format: ui.ImageByteFormat.png);
          final directory = Directory('../artifacts/clinical-tools-mobile')
            ..createSync(recursive: true);
          File(
            '${directory.path}/bmi-${dark ? 'dark' : 'light'}.png',
          ).writeAsBytesSync(data!.buffer.asUint8List());
          image.dispose();
        });
      });
    }
  }
}
