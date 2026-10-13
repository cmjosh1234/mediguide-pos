part of '../screens/publication_guideline_page.dart';

class _ReadingBottomActions extends StatelessWidget {
  const _ReadingBottomActions({
    required this.showAi,
    required this.isBookmarked,
    required this.onAskAi,
    required this.onBookmark,
    required this.onNotes,
    required this.onMore,
  });

  final bool isBookmarked;
  final bool showAi;

  final VoidCallback onAskAi;
  final VoidCallback onBookmark;
  final VoidCallback onNotes;
  final VoidCallback onMore;

  @override
  Widget build(BuildContext context) {
    final actions = <Widget>[
      if (showAi)
        _ReaderBottomAction(
          icon: LucideIcons.sparkles,
          label: 'Ask AI',
          onTap: onAskAi,
        ),
      _ReaderBottomAction(
        icon: LucideIcons.notebookPen,
        label: 'Notes',
        onTap: onNotes,
      ),
      _ReaderBottomAction(
        icon: isBookmarked ? LucideIcons.bookmarkCheck : LucideIcons.bookmark,
        label: isBookmarked ? 'Saved' : 'Bookmark',
        onTap: onBookmark,
      ),
      _ReaderBottomAction(
        icon: LucideIcons.ellipsis,
        label: 'More',
        onTap: onMore,
      ),
    ];
    return LayoutBuilder(
      builder: (context, constraints) {
        final columns = MediaQuery.textScalerOf(context).scale(1) > 1.3
            ? 2
            : actions.length;
        return Wrap(
          spacing: 8,
          runSpacing: 4,
          alignment: WrapAlignment.center,
          children: [
            for (final action in actions)
              SizedBox(
                width: (constraints.maxWidth - 8 * (columns - 1)) / columns,
                child: action,
              ),
          ],
        );
      },
    );
  }
}
