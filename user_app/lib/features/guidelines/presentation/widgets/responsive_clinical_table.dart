import 'dart:math' as math;

import 'package:flutter/material.dart';

import 'package:user_app/core/constants/app_dimensions.dart';
import 'package:user_app/core/constants/app_spacing.dart';
import 'package:user_app/features/guidelines/data/models/guideline_publication.dart';
import 'package:user_app/shared/widgets/app_markdown_body.dart';

/// Renders reviewed clinical tables as conventional tables on every screen.
///
/// Rows keep their header row, so a value is always read against its column.
/// When the columns cannot fit the available width — the common case for wide
/// tables on a phone held vertically — the table scrolls horizontally inside
/// its own viewport instead of overflowing: the widget never reports a width
/// larger than its constraints, so the page around it stays within the window.
class ResponsiveClinicalTable extends StatelessWidget {
  const ResponsiveClinicalTable({
    super.key,
    required this.payload,
    this.showTitle = false,
  });

  final GuidelineTablePayload payload;
  final bool showTitle;

  @override
  Widget build(BuildContext context) {
    final columns = payload.columns;

    if (columns.isEmpty) {
      return const Text('This table has no columns.');
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (showTitle && payload.title.trim().isNotEmpty) ...[
          AppMarkdownBody(
            data: payload.title.trim(),
            style: Theme.of(context).textTheme.titleMedium,
            compact: true,
          ),
          AppSpacing.gapSm,
        ],
        _ScrollableTable(columns: columns, rows: payload.rows),
      ],
    );
  }
}

class _ScrollableTable extends StatefulWidget {
  const _ScrollableTable({required this.columns, required this.rows});

  final List<String> columns;
  final List<List<String>> rows;

  @override
  State<_ScrollableTable> createState() => _ScrollableTableState();
}

class _ScrollableTableState extends State<_ScrollableTable> {
  final ScrollController _controller = ScrollController();

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;

    // Large accessibility text needs proportionally more room before a column
    // collapses to one word per line. Cap the growth so a single column can
    // still never take the whole viewport.
    final textScale = math.min(
      1.6,
      math.max(1.0, MediaQuery.textScalerOf(context).scale(1)),
    );
    final columnWidth = AppDimensions.tableMinColumnWidth * textScale;

    return LayoutBuilder(
      builder: (context, constraints) {
        final available = constraints.maxWidth;
        final naturalWidth = columnWidth * widget.columns.length;
        // Unbounded width (a horizontally scrolling ancestor) has no viewport
        // to fit, so fall back to the natural width.
        final fits = available.isFinite && naturalWidth <= available;

        return ClipRRect(
          borderRadius: BorderRadius.circular(AppDimensions.radiusMedium),
          child: Scrollbar(
            controller: _controller,
            thumbVisibility: !fits,
            child: SingleChildScrollView(
              controller: _controller,
              scrollDirection: Axis.horizontal,
              physics: fits ? const NeverScrollableScrollPhysics() : null,
              // Keeps the scrollbar thumb clear of the last row.
              padding: EdgeInsets.only(bottom: fits ? 0 : AppSpacing.sm),
              child: SizedBox(
                width: fits ? available : naturalWidth,
                child: _ClinicalTable(
                  columns: widget.columns,
                  rows: widget.rows,
                  colors: colors,
                  // Selectable text claims horizontal drags on mobile, which
                  // would swallow the sideways scroll. Cells stay selectable
                  // whenever there is nothing to scroll.
                  selectable: fits,
                ),
              ),
            ),
          ),
        );
      },
    );
  }
}

class _ClinicalTable extends StatelessWidget {
  const _ClinicalTable({
    required this.columns,
    required this.rows,
    required this.colors,
    required this.selectable,
  });

  final List<String> columns;
  final List<List<String>> rows;
  final ColorScheme colors;
  final bool selectable;

  @override
  Widget build(BuildContext context) => Table(
    border: TableBorder.all(color: colors.outlineVariant),
    columnWidths: {
      for (var index = 0; index < columns.length; index++)
        index: const FlexColumnWidth(),
    },
    defaultVerticalAlignment: TableCellVerticalAlignment.top,
    children: [
      TableRow(
        decoration: BoxDecoration(color: colors.surfaceContainerHigh),
        children: [
          for (final column in columns)
            _TableCell(text: column, selectable: selectable, isHeader: true),
        ],
      ),
      for (final row in rows)
        TableRow(
          children: [
            for (var index = 0; index < columns.length; index++)
              _TableCell(
                text: index < row.length ? row[index] : '',
                selectable: selectable,
              ),
          ],
        ),
    ],
  );
}

class _TableCell extends StatelessWidget {
  const _TableCell({
    required this.text,
    required this.selectable,
    this.isHeader = false,
  });

  final String text;
  final bool selectable;
  final bool isHeader;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.all(AppSpacing.sm),
    child: AppMarkdownBody(
      data: text,
      selectable: selectable,
      style: isHeader
          ? Theme.of(
              context,
            ).textTheme.labelMedium?.copyWith(fontWeight: FontWeight.w800)
          : Theme.of(context).textTheme.bodySmall,
      compact: true,
    ),
  );
}
