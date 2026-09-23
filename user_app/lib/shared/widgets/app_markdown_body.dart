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
  });

  final String data;
  final TextStyle? style;
  final bool selectable;
  final bool compact;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final textStyle = style ?? theme.textTheme.bodyMedium;
    final sheet = MarkdownStyleSheet.fromTheme(theme).copyWith(
      p: textStyle,
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

    final source = _escapeRawHtml(data.trim());

    return MarkdownBody(
      // flutter_markdown deliberately drops raw HTML nodes. Preserve them as
      // visible, inert source text instead: clinical content must never execute
      // embedded HTML, but silently hiding it can change the meaning of the
      // reviewed source and makes unsafe markup impossible to spot.
      data: source,
      // Selectable text claims horizontal drags on mobile, so leaving it on
      // would swallow the sideways scroll a table needs to be readable.
      selectable: selectable && !_containsTable(source),
      fitContent: true,
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
