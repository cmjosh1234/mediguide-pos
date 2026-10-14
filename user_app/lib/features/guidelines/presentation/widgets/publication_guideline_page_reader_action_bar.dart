part of '../screens/publication_guideline_page.dart';

class _ReaderActionBar extends StatelessWidget {
  const _ReaderActionBar({
    required this.onMore,
    required this.primaryLabel,
    required this.primaryIcon,
    required this.primaryEnabled,
    required this.showDownload,
    required this.showAi,
    required this.isBookmarked,
    required this.offlineDownload,
    required this.showRead,
    required this.onRead,
    required this.onAskAi,
    required this.onBookmark,
    required this.onNotes,
    required this.onDownload,
  });

  final VoidCallback onMore;
  final String primaryLabel;
  final IconData primaryIcon;
  final bool primaryEnabled, showDownload, showAi;
  final bool isBookmarked;
  final OfflineDownload? offlineDownload;
  final bool showRead;

  final VoidCallback onRead;
  final VoidCallback onAskAi;
  final VoidCallback onBookmark;
  final VoidCallback onNotes;
  final VoidCallback onDownload;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;

    return Material(
      color: colors.surface,
      elevation: 8,
      child: SafeArea(
        top: false,
        child: Container(
          padding: const EdgeInsets.fromLTRB(
            AppSpacing.md,
            AppSpacing.sm,
            AppSpacing.md,
            AppSpacing.sm,
          ),
          decoration: BoxDecoration(
            border: Border(top: BorderSide(color: colors.outlineVariant)),
          ),
          child: showRead
              ? _OverviewBottomActions(
                  primaryLabel: primaryLabel,
                  primaryIcon: primaryIcon,
                  primaryEnabled: primaryEnabled,
                  showDownload: showDownload,
                  isBookmarked: isBookmarked,
                  offlineDownload: offlineDownload,
                  onRead: onRead,
                  onBookmark: onBookmark,
                  onDownload: onDownload,
                  onMore: onMore,
                )
              : _ReadingBottomActions(
                  showAi: showAi,
                  isBookmarked: isBookmarked,
                  onAskAi: onAskAi,
                  onBookmark: onBookmark,
                  onNotes: onNotes,
                  onMore: onMore,
                ),
        ),
      ),
    );
  }
}
