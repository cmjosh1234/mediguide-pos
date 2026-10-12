part of '../screens/publication_guideline_page.dart';

class _Overview extends ConsumerWidget {
  const _Overview({required this.content});

  final GuidelinePublicationContent content;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final publication = content.publication;

    final manifest = content.manifest;
    final copies =
        (ref.watch(guidelineDownloadsControllerProvider).valueOrNull ??
                <OfflineDownload>[])
            .where(
              (item) =>
                  item.guidelineId == publication.id &&
                  item.localPath.isNotEmpty &&
                  (item.status == OfflineDownloadStatus.ready ||
                      item.status == OfflineDownloadStatus.updateAvailable),
            );
    final saved = copies.firstOrNull;
    final outdated = saved != null && saved.version != manifest.version;

    return Padding(
      padding: EdgeInsets.fromLTRB(
        Responsive.horizontalPadding(context),
        AppSpacing.lg,
        Responsive.horizontalPadding(context),
        AppSpacing.md,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            publication.title,
            style: Theme.of(
              context,
            ).textTheme.headlineMedium?.copyWith(fontWeight: FontWeight.w800),
          ),

          if (publication.description.isNotEmpty) ...[
            AppSpacing.gapSm,

            AppMarkdownBody(data: publication.description, compact: true),
          ],

          if (saved != null)
            Card(
              child: ListTile(
                leading: Icon(
                  outdated
                      ? Icons.system_update_alt
                      : Icons.offline_pin_outlined,
                ),
                title: Text(
                  outdated ? 'Update your saved copy' : 'Available offline',
                ),
                subtitle: Text(
                  outdated
                      ? 'Saved version ${saved.version} · current version ${manifest.version}'
                      : 'Saved version ${saved.version}',
                ),
                trailing: outdated
                    ? TextButton(
                        child: const Text('Update'),
                        onPressed: () async {
                          try {
                            await ref
                                .read(
                                  guidelineDownloadsControllerProvider.notifier,
                                )
                                .download(
                                  content,
                                  originalDocument:
                                      saved.assetType == 'original_pdf',
                                );
                            if (context.mounted) {
                              AppMessage.success(
                                context,
                                'Offline copy updated.',
                              );
                            }
                          } catch (_) {
                            if (context.mounted) {
                              AppMessage.error(
                                context,
                                'Unable to update the saved copy. Check your connection.',
                              );
                            }
                          }
                        },
                      )
                    : null,
              ),
            ),
          AppSpacing.gapMd,

          Wrap(
            spacing: AppSpacing.sm,
            runSpacing: AppSpacing.sm,
            children: [
              if (publication.sourceOrganization.isNotEmpty)
                Chip(label: Text(publication.sourceOrganization)),

              if (publication.version.isNotEmpty)
                Chip(label: Text('Version ${publication.version}')),

              AppStatusBadge(
                icon: manifest.recommendedMode == GuidelineReaderMode.structured
                    ? LucideIcons.badgeCheck
                    : LucideIcons.fileWarning,
                tone: manifest.recommendedMode == GuidelineReaderMode.structured
                    ? AppStatusTone.success
                    : AppStatusTone.warning,
                label:
                    manifest.recommendedMode == GuidelineReaderMode.structured
                    ? 'Reviewed structured content'
                    : 'Partial reviewed content',
              ),
              Chip(label: Text('${manifest.sectionCount} sections')),
              Chip(
                label: Text(
                  '${manifest.reviewedSectionCount} sections with reviewed content',
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

// =============================================================================
// ORIGINAL DOCUMENT READER
// =============================================================================
