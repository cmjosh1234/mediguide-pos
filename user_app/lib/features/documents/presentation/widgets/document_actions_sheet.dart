import 'package:flutter/material.dart';
import 'package:user_app/features/downloads/data/models/offline_download.dart';

enum DocumentMenuAction {
  contents,
  settings,
  search,
  askAi,
  notes,
  bookmark,
  collection,
  copyLink,
  original,
  download,
  page,
}

/// One menu for reader and overview actions, with only applicable options shown.
class DocumentActionsSheet extends StatelessWidget {
  const DocumentActionsSheet({
    super.key,
    required this.title,
    required this.kindLabel,
    this.version = '',
    this.showReadingTools = false,
    this.showSearch = false,
    this.showAi = false,
    this.showLibraryActions = true,
    this.showCopyLink = true,
    this.showPagePicker = false,
    this.showDownload = false,
    this.hasOriginal = false,
    this.isLink = false,
    this.websiteAvailable = true,
    this.websiteHost = '',
    this.isBookmarked = false,
    this.isSignedIn = true,
    this.hasNotes = false,
    this.offlineDownload,
    this.copyLabel = 'Copy document link',
  });
  final String title, kindLabel, version, websiteHost, copyLabel;
  final bool showReadingTools,
      showSearch,
      showAi,
      showLibraryActions,
      showCopyLink,
      showPagePicker,
      showDownload,
      hasOriginal,
      isLink,
      websiteAvailable,
      isBookmarked,
      isSignedIn,
      hasNotes;
  final OfflineDownload? offlineDownload;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    final status = offlineDownload?.status;
    final downloading =
        status == OfflineDownloadStatus.downloading ||
        status == OfflineDownloadStatus.queued;
    final downloadTitle = switch (status) {
      OfflineDownloadStatus.ready => 'Manage offline copy',
      OfflineDownloadStatus.updateAvailable => 'Update offline copy',
      OfflineDownloadStatus.failed ||
      OfflineDownloadStatus.corrupted => 'Retry download',
      OfflineDownloadStatus.downloading ||
      OfflineDownloadStatus.queued => 'Downloading offline copy',
      _ => 'Download for offline use',
    };
    final downloadSubtitle = switch (status) {
      OfflineDownloadStatus.ready => 'Available without an internet connection',
      OfflineDownloadStatus.updateAvailable =>
        'A newer published version is available',
      OfflineDownloadStatus.failed ||
      OfflineDownloadStatus.corrupted => 'The last download did not complete',
      OfflineDownloadStatus.downloading ||
      OfflineDownloadStatus.queued => 'Your verified copy is being saved',
      _ => 'Save a verified copy on this device',
    };
    Widget action(
      DocumentMenuAction value,
      IconData icon,
      String label,
      String subtitle, {
      bool enabled = true,
      String? badge,
      Widget? extra,
    }) => Padding(
      padding: const EdgeInsets.symmetric(vertical: 2),
      child: Material(
        color: colors.surfaceContainerLow,
        borderRadius: BorderRadius.circular(16),
        child: ListTile(
          key: ValueKey(value),
          enabled: enabled,
          isThreeLine: false,
          contentPadding: const EdgeInsets.symmetric(
            horizontal: 16,
            vertical: 8,
          ),
          leading: Container(
            width: 40,
            height: 40,
            alignment: Alignment.center,
            decoration: BoxDecoration(
              color: enabled
                  ? colors.primaryContainer
                  : colors.surfaceContainerHighest,
              borderRadius: BorderRadius.circular(12),
            ),
            child: Icon(
              icon,
              size: 21,
              color: enabled
                  ? colors.onPrimaryContainer
                  : colors.onSurfaceVariant,
            ),
          ),
          title: Text(
            label,
            style: theme.textTheme.titleSmall?.copyWith(
              fontWeight: FontWeight.w700,
            ),
          ),
          subtitle: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const SizedBox(height: 4),
              Text(subtitle),
              if (badge != null)
                Padding(
                  padding: const EdgeInsets.only(top: 8),
                  child: Text(
                    badge,
                    style: theme.textTheme.labelMedium?.copyWith(
                      color: colors.primary,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                ),
              if (extra != null) ...[const SizedBox(height: 8), extra],
            ],
          ),
          trailing: enabled ? const Icon(Icons.chevron_right, size: 20) : null,
          onTap: enabled ? () => Navigator.pop(context, value) : null,
        ),
      ),
    );
    Widget group(String label, List<Widget> actions) => Padding(
      padding: const EdgeInsets.only(bottom: 20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.only(bottom: 10, top: 4),
            child: Text(
              label,
              style: theme.textTheme.labelLarge?.copyWith(
                color: colors.onSurfaceVariant,
                fontWeight: FontWeight.w700,
              ),
            ),
          ),
          ...actions,
        ],
      ),
    );
    return SizedBox(
      height: MediaQuery.sizeOf(context).height * 0.85,
      child: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(24, 4, 8, 16),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Document actions',
                        style: theme.textTheme.titleLarge?.copyWith(
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                      const SizedBox(height: 8),
                      Text(
                        title,
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                        style: theme.textTheme.bodyMedium,
                      ),
                      const SizedBox(height: 4),
                      Text(
                        [
                          kindLabel,
                          if (version.isNotEmpty) 'Version $version',
                        ].join(' · '),
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                        style: theme.textTheme.labelMedium?.copyWith(
                          color: colors.onSurfaceVariant,
                        ),
                      ),
                    ],
                  ),
                ),
                IconButton(
                  tooltip: 'Close document actions',
                  onPressed: () => Navigator.pop(context),
                  icon: const Icon(Icons.close),
                ),
              ],
            ),
          ),
          const Divider(height: 1),
          Expanded(
            child: ListView(
              padding: const EdgeInsets.fromLTRB(16, 16, 16, 24),
              children: [
                if (showReadingTools ||
                    showSearch ||
                    showAi ||
                    showPagePicker ||
                    showLibraryActions)
                  group('Reading tools', [
                    if (showReadingTools)
                      action(
                        DocumentMenuAction.contents,
                        Icons.list_alt,
                        'Contents',
                        'Jump to a chapter or section',
                      ),
                    if (showSearch)
                      action(
                        DocumentMenuAction.search,
                        Icons.search,
                        'Search document',
                        'Find words in the published content',
                      ),
                    if (showReadingTools)
                      action(
                        DocumentMenuAction.settings,
                        Icons.text_fields,
                        'Reading settings',
                        'Adjust text size and continuous reading',
                      ),
                    if (showPagePicker)
                      action(
                        DocumentMenuAction.page,
                        Icons.find_in_page_outlined,
                        'Go to page',
                        'Choose a page number in this PDF',
                      ),
                    if (showAi)
                      action(
                        DocumentMenuAction.askAi,
                        Icons.auto_awesome_outlined,
                        'Ask AI',
                        'Ask questions using this document as context',
                      ),
                    if (showLibraryActions)
                      action(
                        DocumentMenuAction.notes,
                        Icons.edit_note,
                        'Reading notes',
                        !isSignedIn
                            ? 'Sign in to keep private reading notes'
                            : hasNotes
                            ? 'View or edit your saved notes'
                            : 'Keep private notes while you read',
                      ),
                  ]),
                if (showLibraryActions || showDownload)
                  group('Save and offline', [
                    if (showLibraryActions)
                      action(
                        DocumentMenuAction.bookmark,
                        isBookmarked ? Icons.bookmark : Icons.bookmark_outline,
                        isBookmarked ? 'Remove bookmark' : 'Bookmark document',
                        isSignedIn
                            ? 'Return to this document from your library'
                            : 'Sign in to sync your bookmarks',
                        badge: isBookmarked ? 'Bookmarked' : null,
                      ),
                    if (showLibraryActions)
                      action(
                        DocumentMenuAction.collection,
                        Icons.create_new_folder_outlined,
                        'Save to collection',
                        isSignedIn
                            ? 'Organize this document in your library'
                            : 'Sign in to organize your library',
                      ),
                    if (showDownload)
                      action(
                        DocumentMenuAction.download,
                        status == OfflineDownloadStatus.ready
                            ? Icons.offline_pin_outlined
                            : status == OfflineDownloadStatus.updateAvailable
                            ? Icons.refresh
                            : Icons.download_outlined,
                        downloadTitle,
                        downloadSubtitle,
                        enabled: !downloading,
                        badge: downloading
                            ? '${((offlineDownload?.progress ?? 0).clamp(0.0, 1.0) * 100).round()}% downloaded'
                            : null,
                        extra: downloading
                            ? LinearProgressIndicator(
                                value: (offlineDownload?.progress ?? 0) > 0
                                    ? offlineDownload!.progress.clamp(0.0, 1.0)
                                    : null,
                                semanticsLabel: 'Download progress',
                              )
                            : null,
                      ),
                  ]),
                if (showCopyLink || hasOriginal || isLink)
                  group('Links and source', [
                    if (showCopyLink)
                      action(
                        DocumentMenuAction.copyLink,
                        Icons.link,
                        copyLabel,
                        'Copy a link to your clipboard',
                      ),
                    if (hasOriginal || isLink)
                      action(
                        DocumentMenuAction.original,
                        isLink
                            ? Icons.language_outlined
                            : Icons.description_outlined,
                        isLink ? 'Open website' : 'Open original document',
                        isLink
                            ? websiteAvailable
                                  ? websiteHost.isEmpty
                                        ? 'Visit the publishing website'
                                        : 'Visit $websiteHost'
                                  : 'The website link is unavailable'
                            : 'View the file as it was published',
                        enabled: !isLink || websiteAvailable,
                      ),
                  ]),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
