part of '../screens/publication_guideline_page.dart';

class _GuidelineOverview extends StatefulWidget {
  const _GuidelineOverview({
    super.key,
    this.contentLoading = false,
    this.contentUnavailable = false,
    this.onRetryContent,
    required this.onTable,
    required this.content,
    required this.currentSectionId,
    required this.readingProgress,
    required this.onSection,
    required this.onOriginal,
  });

  final GuidelinePublicationContent content;
  final bool contentLoading;
  final bool contentUnavailable;
  final VoidCallback? onRetryContent;
  final ValueChanged<String> onTable;

  final String? currentSectionId;

  final double? readingProgress;

  final ValueChanged<String> onSection;

  final VoidCallback onOriginal;

  @override
  State<_GuidelineOverview> createState() => _GuidelineOverviewState();
}
