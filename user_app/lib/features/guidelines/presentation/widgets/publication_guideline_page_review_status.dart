part of '../screens/publication_guideline_page.dart';

class _ReviewStatus extends StatelessWidget {
  const _ReviewStatus({required this.manifest, required this.presentation});
  final GuidelineManifest manifest;
  final DocumentOverviewPresentation presentation;

  @override
  Widget build(BuildContext context) => AppStatusBadge(
    label: presentation.isLink
        ? 'External website'
        : presentation.isFile
        ? 'Original document'
        : manifest.recommendedMode == GuidelineReaderMode.partial
        ? 'Partially reviewed'
        : 'Reviewed content',
    icon: presentation.isLink
        ? Icons.open_in_new
        : presentation.isFile
        ? Icons.description_outlined
        : Icons.verified_outlined,
    tone: presentation.isLink || presentation.isFile
        ? AppStatusTone.information
        : manifest.recommendedMode == GuidelineReaderMode.partial
        ? AppStatusTone.warning
        : AppStatusTone.success,
  );
}
