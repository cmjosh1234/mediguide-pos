import 'package:flutter/material.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';
import 'package:url_launcher/url_launcher.dart';

import 'package:user_app/core/constants/app_dimensions.dart';
import 'package:user_app/core/constants/app_spacing.dart';
import 'package:user_app/core/utils/app_message.dart';

/// The single inline Markdown renderer for user-visible API and publication
/// content. Use this instead of passing Markdown-bearing content to [Text].
class AppMarkdownBody extends StatelessWidget {
  const AppMarkdownBody({
    super.key,
    required this.data,
    this.style,
    this.selectable = true,
    this.compact = false,
    this.justify = false,
  });

  final String data;
  final TextStyle? style;
  final bool selectable;
  final bool compact;
  final bool justify;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final textStyle = style ?? theme.textTheme.bodyMedium;
    final sheet = MarkdownStyleSheet.fromTheme(theme).copyWith(
      p: textStyle,
      textAlign: justify ? WrapAlignment.spaceBetween : WrapAlignment.start,
      unorderedListAlign: justify
          ? WrapAlignment.spaceBetween
          : WrapAlignment.start,
      orderedListAlign: justify
          ? WrapAlignment.spaceBetween
          : WrapAlignment.start,
      blockquoteAlign: justify
          ? WrapAlignment.spaceBetween
          : WrapAlignment.start,
      listBullet: textStyle,
      tableBody: textStyle,
      a: textStyle?.copyWith(
        color: theme.colorScheme.primary,
        decoration: TextDecoration.underline,
        decorationColor: theme.colorScheme.primary,
      ),
      blockSpacing: compact ? 4 : 8,
      pPadding: EdgeInsets.zero,
      // Markdown tables stay conventional tables: a fixed column width keeps
      // cells readable instead of squeezing every column into the screen, and
      // it is also what makes flutter_markdown_plus wrap the table in its own
      // horizontal scroll view, so a wide table scrolls rather than overflows.
      tableColumnWidth: const FixedColumnWidth(
        AppDimensions.tableMinColumnWidth,
      ),
      tableScrollbarThumbVisibility: true,
      tableBorder: TableBorder.all(color: theme.colorScheme.outlineVariant),
      tableCellsPadding: const EdgeInsets.all(AppSpacing.sm),
      tableHead: textStyle?.copyWith(fontWeight: FontWeight.w800),
      tableHeadCellsDecoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerHigh,
      ),
      // Leaves room under the table for the horizontal scrollbar thumb.
      tablePadding: const EdgeInsets.only(bottom: AppSpacing.sm),
    );

    final source = _normalizeMarkdown(data.trim());

    return MarkdownBody(
      // Known bold tags are normalized to Markdown. Other HTML remains visible
      // and inert so unsupported markup cannot silently hide clinical content.
      data: source,
      // Selectable text claims horizontal drags on mobile, so leaving it on
      // would swallow the sideways scroll a table needs to be readable.
      selectable: selectable && !_containsTable(source),
      fitContent: !justify,
      styleSheet: sheet,
      onTapLink: (_, href, _) => _openLink(context, href),
    );
  }
}

/// Matches the delimiter row that turns pipe-separated lines into a table,
/// such as `| --- | ---: |`.
final RegExp _tableDelimiterRow = RegExp(
  r'^[ \t]*\|?[ \t]*:?-+:?[ \t]*(\|[ \t]*:?-+:?[ \t]*)+\|?[ \t]*$',
  multiLine: true,
);

bool _containsTable(String value) => _tableDelimiterRow.hasMatch(value);

String _normalizeMarkdown(String value) {
  final withLineBreaks = value
      .replaceAllMapped(
        RegExp(
          r'(?:<br\s*/?>|&lt;br\s*/?&gt;)\s*[-–—]\s*',
          caseSensitive: false,
        ),
        (_) => '  \n• ',
      )
      .replaceAllMapped(
        RegExp(r'(?:<br\s*/?>|&lt;br\s*/?&gt;)', caseSensitive: false),
        (_) => '  \n',
      );
  return _escapeRawHtml(
    _normalizeBoldTags(withLineBreaks),
  ).replaceAll(RegExp(r';\s*[•·]\s*'), '  \n• ');
}

String _normalizeBoldTags(String value) {
  // Accept only these formatting tags, including entity-encoded source from
  // document extraction. Never enable general HTML rendering.
  final decoded = value.replaceAllMapped(
    RegExp(r'&lt;(/?(?:b|strong)\s*)&gt;', caseSensitive: false),
    (match) => '<${match.group(1)}>',
  );
  final formatted = decoded.replaceAllMapped(
    RegExp(r'<(b|strong)\s*>(.*?)</\1\s*>', caseSensitive: false, dotAll: true),
    (match) {
      final content = match.group(2)!;
      final text = content.trim();
      if (text.isEmpty) return content;
      final leading = content.substring(0, content.indexOf(text));
      final trailing = content.substring(content.indexOf(text) + text.length);
      return '$leading**$text**$trailing';
    },
  );
  // Incomplete extraction can leave a lone formatting tag. Keep its text.
  return formatted.replaceAll(
    RegExp(r'</?(?:b|strong)\s*>', caseSensitive: false),
    '',
  );
}

String _escapeRawHtml(String value) {
  return value.replaceAllMapped(
    RegExp(r'</?[A-Za-z][^>\n]*>'),
    (match) => match
        .group(0)!
        .replaceFirst('<', '&lt;')
        .replaceFirst(RegExp(r'>$'), '&gt;'),
  );
}

Future<void> _openLink(BuildContext context, String? value) async {
  final uri = Uri.tryParse(value?.trim() ?? '');
  if (uri == null || uri.scheme != 'https') {
    AppMessage.warning(context, 'This link is unavailable.');
    return;
  }

  try {
    final launched = await launchUrl(uri, mode: LaunchMode.externalApplication);
    if (!launched && context.mounted) {
      AppMessage.error(context, 'Unable to open this link.');
    }
  } catch (_) {
    if (context.mounted) {
      AppMessage.error(context, 'Unable to open this link.');
    }
  }
}
