import 'package:flutter/material.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:user_app/shared/widgets/app_markdown_body.dart';

void main() {
  testWidgets('renders extracted HTML bold tags as bold reader text', (
    tester,
  ) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(
          body: AppMarkdownBody(
            selectable: false,
            data:
                '<b>Important</b> and &lt;B&gt;Encoded&lt;/B&gt;\n\n'
                '- <strong> Warning </strong>\n\n'
                '| Note |\n| --- |\n| <b>Table note</b> |\n\n'
                '<b>Unclosed\n\n<script>unsupported</script>',
          ),
        ),
      ),
    );

    final spans = tester
        .widgetList<RichText>(find.byType(RichText))
        .map((widget) => widget.text)
        .toList();
    final rendered = spans.map((span) => span.toPlainText()).join(' ');
    for (final text in ['Important', 'Encoded', 'Warning', 'Table note']) {
      expect(rendered, contains(text));
      expect(spans.any((span) => _containsBoldText(span, text)), isTrue);
    }
    expect(rendered, contains('Unclosed'));
    expect(rendered, isNot(contains('<b>')));
    expect(rendered, isNot(contains('&lt;B&gt;')));
    expect(rendered, isNot(contains('<strong>')));
    expect(rendered, contains('<script>unsupported</script>'));
    expect(tester.takeException(), isNull);
  });

  testWidgets('renders publication markdown instead of exposing its markers', (
    tester,
  ) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(
          body: AppMarkdownBody(
            data:
                '**Important** dose is `5 mg`<br>- Take with water<br/>- Verify the dose',
          ),
        ),
      ),
    );

    expect(find.byType(MarkdownBody), findsOneWidget);

    final rendered = <String>[
      ...tester
          .widgetList<RichText>(find.byType(RichText))
          .map((widget) => widget.text.toPlainText()),
      ...tester
          .widgetList<SelectableText>(find.byType(SelectableText))
          .map((widget) => widget.data ?? widget.textSpan?.toPlainText() ?? ''),
    ].join(' ');
    expect(rendered, contains('Important'));
    expect(rendered, contains('5 mg'));
    expect(rendered, contains('Take with water'));
    expect(rendered, contains('Verify the dose'));
    expect(rendered, isNot(contains('**')));
    expect(rendered, isNot(contains('`')));
    expect(rendered, isNot(contains('<br>')));

    final markdown = tester.widget<MarkdownBody>(find.byType(MarkdownBody));
    expect(markdown.data, contains('  \n• Take with water'));
    expect(markdown.data, isNot(contains('<br')));
  });

  testWidgets('markdown tables scroll sideways instead of overflowing', (
    tester,
  ) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(
          body: SizedBox(
            width: 320,
            child: AppMarkdownBody(
              data:
                  '| Drug | Dose | Route |\n'
                  '| --- | --- | --- |\n'
                  '| Artemether | 3.2 mg/kg | Intramuscular |',
            ),
          ),
        ),
      ),
    );

    final scrollView = find.byWidgetPredicate(
      (widget) =>
          widget is SingleChildScrollView &&
          widget.scrollDirection == Axis.horizontal,
    );

    expect(find.byType(Table), findsOneWidget);
    expect(scrollView, findsOneWidget);
    expect(tester.getSize(scrollView).width, lessThanOrEqualTo(320));
    expect(tester.getSize(find.byType(Table)).width, greaterThan(320));
    expect(tester.takeException(), isNull);
  });
}

bool _containsBoldText(InlineSpan span, String text) {
  if (span is! TextSpan) return false;
  if (span.style?.fontWeight == FontWeight.bold &&
      span.toPlainText().contains(text)) {
    return true;
  }
  return span.children?.any((child) => _containsBoldText(child, text)) ?? false;
}
