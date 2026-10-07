import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:responsive_framework/responsive_framework.dart';
import 'package:user_app/features/content/data/models/generic_page.dart';
import 'package:user_app/features/discovery/presentation/widgets/discovery_widgets.dart';
import 'package:user_app/features/guidelines/data/models/guideline_publication.dart';
import 'package:user_app/features/guidelines/presentation/widgets/responsive_clinical_table.dart';

Widget responsiveBuilder(BuildContext context, Widget? child) =>
    ResponsiveBreakpoints.builder(
      child: child!,
      breakpoints: const [
        Breakpoint(start: 0, end: 450, name: MOBILE),
        Breakpoint(start: 451, end: 800, name: TABLET),
        Breakpoint(start: 801, end: double.infinity, name: DESKTOP),
      ],
    );

void main() {
  GenericPage page(Map<String, dynamic> content) => GenericPage(
    id: 'test',
    title: 'Test',
    key: 'test',
    content: content,
    created: DateTime(2026),
    updated: DateTime(2026),
  );

  test('blank HTML and empty sections show the page empty state', () {
    expect(page({'body': '<p>&nbsp; &#160; &#xA0;</p>'}).hasContent, isFalse);
    expect(
      page({
        'a': {'title': 'Section', 'content': '  '},
      }).hasContent,
      isFalse,
    );
    expect(page({'body': '<p>Useful content</p>'}).hasContent, isTrue);
    expect(page({'body': '<img src="figure.png">'}).hasContent, isTrue);
    expect(
      page({
        'a': {'title': 'Section', 'content': 'Reviewed content'},
      }).hasContent,
      isTrue,
    );
  });

  testWidgets('an empty clinical table does not render a header-only table', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        builder: responsiveBuilder,
        home: const Scaffold(
          body: ResponsiveClinicalTable(
            payload: GuidelineTablePayload(columns: ['Treatment', 'Dose']),
          ),
        ),
      ),
    );
    expect(find.text('No table content available'), findsOneWidget);
    expect(find.text('Dose'), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets(
    'discovery empty states fit a small phone with large text and keep the action reachable',
    (tester) async {
      tester.view.physicalSize = const Size(320, 480);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      var cleared = false;
      await tester.pumpWidget(
        MaterialApp(
          builder: responsiveBuilder,
          home: MediaQuery(
            data: const MediaQueryData(
              size: Size(320, 480),
              textScaler: TextScaler.linear(2),
            ),
            child: Scaffold(
              body: EmptyState(
                'Try another search or clear your filters to see all resources.',
                title: 'No matching resources',
                isSearch: true,
                actionLabel: 'Clear filters',
                onAction: () => cleared = true,
              ),
            ),
          ),
        ),
      );
      await tester.ensureVisible(find.text('Clear filters'));
      await tester.tap(find.text('Clear filters'));
      expect(cleared, isTrue);
      expect(tester.takeException(), isNull);
    },
  );
}
