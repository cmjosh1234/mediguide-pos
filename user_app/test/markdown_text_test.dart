import 'package:flutter_test/flutter_test.dart';
import 'package:user_app/core/utils/markdown_text.dart';

void main() {
  test('converts structural Markdown labels to plain text', () {
    expect(markdownLabel('## **Clinical care**'), 'Clinical care');
    expect(
      markdownLabel('[Malaria diagnosis](https://example.test/guidance)'),
      'Malaria diagnosis',
    );
    expect(markdownLabel('Use `ACT` for *malaria*'), 'Use ACT for malaria');
    expect(
      markdownLabel(r'HIV_AIDS and escaped \*value\*'),
      'HIV_AIDS and escaped *value*',
    );
  });
}
