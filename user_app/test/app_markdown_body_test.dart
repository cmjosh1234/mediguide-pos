import 'package:flutter/material.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:user_app/shared/widgets/app_markdown_body.dart';

void main() {
  testWidgets('renders publication markdown instead of exposing its markers', (
    tester,
  ) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(
          body: AppMarkdownBody(
            data: '**Important** dose is `5 mg`\n\n- Take with water',
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
    expect(rendered, isNot(contains('**')));
    expect(rendered, isNot(contains('`')));
  });
}
