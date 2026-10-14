part of '../screens/publication_guideline_page.dart';

class _OverviewBottomActions extends StatelessWidget {
  const _OverviewBottomActions({
    required this.isBookmarked,
    required this.offlineDownload,
    required this.onRead,
    required this.onBookmark,
    required this.onDownload,
    required this.onMore,
    required this.primaryLabel,
    required this.primaryIcon,
    required this.primaryEnabled,
    required this.showDownload,
  });
  final bool isBookmarked;
  final OfflineDownload? offlineDownload;
  final VoidCallback onRead, onBookmark, onDownload, onMore;
  final String primaryLabel;
  final IconData primaryIcon;
  final bool primaryEnabled, showDownload;

  @override
  Widget build(BuildContext context) {
    final primary = FilledButton.icon(
      key: const Key('document-primary-action'),
      onPressed: primaryEnabled ? onRead : null,
      icon: Icon(primaryIcon, size: 20),
      label: Text(primaryLabel, textAlign: TextAlign.center),
      style: FilledButton.styleFrom(
        minimumSize: const Size.fromHeight(48),
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      ),
    );
    final secondary = <Widget>[
      _BottomIconAction(
        icon: isBookmarked ? LucideIcons.bookmarkCheck : LucideIcons.bookmark,
        tooltip: isBookmarked ? 'Remove bookmark' : 'Bookmark',
        onTap: onBookmark,
      ),
      if (showDownload)
        _BottomIconAction(
          icon: offlineDownload?.status == OfflineDownloadStatus.ready
              ? LucideIcons.cloudCheck
              : LucideIcons.download,
          tooltip: offlineDownload?.status == OfflineDownloadStatus.ready
              ? 'Manage offline copy'
              : 'Download for offline use',
          onTap: onDownload,
        ),
      _BottomIconAction(
        icon: LucideIcons.ellipsis,
        tooltip: 'More',
        onTap: onMore,
      ),
    ];
    return LayoutBuilder(
      builder: (context, constraints) {
        if (constraints.maxWidth < 380 ||
            MediaQuery.textScalerOf(context).scale(1) > 1.3) {
          return Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              SizedBox(width: double.infinity, child: primary),
              AppSpacing.gapSm,
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceEvenly,
                children: secondary,
              ),
            ],
          );
        }
        return Row(
          children: [
            Expanded(child: primary),
            for (final action in secondary) ...[AppSpacing.hGapSm, action],
          ],
        );
      },
    );
  }
}
