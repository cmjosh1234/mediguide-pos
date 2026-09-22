/// Converts Markdown used as a structural label into plain display text.
///
/// Body content should be rendered with `AppMarkdownBody`. This helper is for
/// titles, breadcrumbs, chips, search labels, and accessibility strings where
/// a full Markdown widget would be inappropriate.
String markdownLabel(String source) {
  var value = source.trim();
  final escaped = <String>[];
  value = value.replaceAllMapped(RegExp(r'\\([\\`*_{}\[\]()#+.!~-])'), (match) {
    escaped.add(match.group(1) ?? '');
    return '\u{E000}${escaped.length - 1}\u{E001}';
  });
  value = value.replaceAllMapped(
    RegExp(r'!\[([^\]]*)\]\([^)]*\)'),
    (match) => match.group(1) ?? '',
  );
  value = value.replaceAllMapped(
    RegExp(r'\[([^\]]+)\]\([^)]*\)'),
    (match) => match.group(1) ?? '',
  );
  value = value.replaceAll(RegExp(r'<[^>]+>'), '');

  final formatting = <RegExp>[
    RegExp(r'\*\*\*([^*]+)\*\*\*'),
    RegExp(r'___([^_]+)___'),
    RegExp(r'\*\*([^*]+)\*\*'),
    RegExp(r'__([^_]+)__'),
    RegExp(r'~~([^~]+)~~'),
    RegExp(r'\*([^*]+)\*'),
    RegExp(r'(?<!\w)_([^_]+)_(?!\w)'),
    RegExp(r'`([^`]+)`'),
  ];
  for (final pattern in formatting) {
    value = value.replaceAllMapped(pattern, (match) => match.group(1) ?? '');
  }

  value = value
      .replaceFirst(RegExp(r'^#{1,6}\s+'), '')
      .replaceAll(RegExp(r'\s+'), ' ')
      .trim();
  value = value.replaceAllMapped(
    RegExp('\u{E000}(\\d+)\u{E001}', unicode: true),
    (match) => escaped[int.parse(match.group(1)!)],
  );
  return value;
}
