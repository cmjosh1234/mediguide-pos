part of '../screens/publication_guideline_page.dart';

class _GuidelineOverviewState extends State<_GuidelineOverview> {
  int _selectedTab = 0;

  @override
  Widget build(BuildContext context) {
    final content = widget.content;
    final publication = content.publication;
    final manifest = content.manifest;
    final presentation = DocumentOverviewPresentation(content);
    final colors = Theme.of(context).colorScheme;
    final text = Theme.of(context).textTheme;
    final sections = content.readerSections;
    final keyPoints = content.blocks
        .whereType<CalloutGuidelineBlock>()
        .where(
          (block) => const {
            'key_point',
            'recommendation',
            'important',
          }.contains(block.blockType),
        )
        .take(6)
        .toList(growable: false);
    final tables = content.blocks.whereType<TableGuidelineBlock>().toList(
      growable: false,
    );
    final hasContents =
        !presentation.isFile && !presentation.isLink && sections.isNotEmpty;
    final hasKeyPoints =
        !presentation.isFile && !presentation.isLink && keyPoints.isNotEmpty;
    final hasTables =
        !presentation.isFile && !presentation.isLink && tables.isNotEmpty;
    final tabs = <int>{
      0,
      if (hasContents) 1,
      if (hasKeyPoints) 2,
      if (hasTables) 3,
    };
    final selected = tabs.contains(_selectedTab) ? _selectedTab : 0;
    final currentSection = sections
        .where((s) => s.id == widget.currentSectionId)
        .firstOrNull;
    final progress = widget.readingProgress?.clamp(0.0, 1.0) ?? 0.0;
    final resumable =
        currentSection != null &&
        progress > 0 &&
        progress < 1 &&
        !presentation.isFile &&
        !presentation.isLink;

    return ListView(
      padding: EdgeInsets.symmetric(
        horizontal: Responsive.horizontalPadding(context),
        vertical: AppSpacing.md,
      ),
      children: [
        Container(
          padding: const EdgeInsets.all(AppSpacing.lg),
          decoration: BoxDecoration(
            color: colors.primaryContainer.withValues(alpha: 0.45),
            borderRadius: BorderRadius.circular(24),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Wrap(
                spacing: 10,
                runSpacing: 8,
                crossAxisAlignment: WrapCrossAlignment.center,
                children: [
                  Icon(presentation.icon, size: 28, color: colors.primary),
                  Text(
                    presentation.name,
                    style: text.labelLarge?.copyWith(
                      color: colors.primary,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  if (publication.programArea.isNotEmpty)
                    Text(
                      publication.programArea,
                      style: text.labelMedium?.copyWith(
                        color: colors.onSurfaceVariant,
                      ),
                    ),
                ],
              ),
              AppSpacing.gapMd,
              Semantics(
                header: true,
                child: Text(
                  publication.title,
                  style: text.headlineMedium?.copyWith(
                    fontWeight: FontWeight.w800,
                    height: 1.15,
                  ),
                ),
              ),
              if (publication.sourceOrganization.isNotEmpty) ...[
                AppSpacing.gapSm,
                Text(
                  publication.sourceOrganization,
                  style: text.bodyMedium?.copyWith(
                    color: colors.onSurfaceVariant,
                  ),
                ),
              ],
              AppSpacing.gapMd,
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: [
                  if (publication.version.isNotEmpty)
                    Chip(
                      label: Text('Version ${publication.version}'),
                      visualDensity: VisualDensity.compact,
                    ),
                  _ReviewStatus(manifest: manifest, presentation: presentation),
                ],
              ),
              if (publication.description.isNotEmpty) ...[
                AppSpacing.gapMd,
                AppMarkdownBody(
                  data: publication.description,
                  compact: true,
                  style: text.bodyLarge?.copyWith(height: 1.5),
                ),
              ],
            ],
          ),
        ),
        AppSpacing.gapLg,
        if (resumable) ...[
          Card(
            margin: EdgeInsets.zero,
            child: Padding(
              padding: const EdgeInsets.all(AppSpacing.md),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    'Pick up where you left off',
                    style: text.titleMedium?.copyWith(
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  AppSpacing.gapSm,
                  Text(currentSection.title),
                  AppSpacing.gapMd,
                  LinearProgressIndicator(
                    value: progress,
                    semanticsLabel: 'Reading progress',
                    semanticsValue: '${(progress * 100).round()}',
                  ),
                  AppSpacing.gapSm,
                  Text(
                    '${(progress * 100).round()}% read',
                    style: text.labelMedium,
                  ),
                  TextButton.icon(
                    onPressed: () => widget.onSection(currentSection.id),
                    icon: const Icon(Icons.arrow_forward),
                    label: const Text('Continue reading'),
                  ),
                ],
              ),
            ),
          ),
          AppSpacing.gapLg,
        ],
        Text(
          presentation.isLink
              ? 'Visit the resource'
              : presentation.isFile
              ? 'About the file'
              : 'Inside this ${presentation.name.toLowerCase()}',
          style: text.titleLarge?.copyWith(fontWeight: FontWeight.w700),
        ),
        AppSpacing.gapSm,
        Text(
          presentation.introduction,
          style: text.bodyMedium?.copyWith(
            color: colors.onSurfaceVariant,
            height: 1.5,
          ),
        ),
        if (presentation.isLink) ...[
          AppSpacing.gapMd,
          Card(
            margin: EdgeInsets.zero,
            child: Padding(
              padding: const EdgeInsets.all(AppSpacing.md),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    Uri.tryParse(publication.externalUrl)?.host ?? '',
                    style: text.titleMedium?.copyWith(
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  AppSpacing.gapXs,
                  Text(
                    presentation.hasValidWebsite
                        ? publication.externalUrl
                        : 'Website link unavailable',
                    style: text.bodySmall,
                  ),
                ],
              ),
            ),
          ),
        ],
        if (presentation.isFile && !manifest.hasOriginalPdf) ...[
          AppSpacing.gapMd,
          const Text('The original file is currently unavailable.'),
        ],
        if (!presentation.isFile &&
            !presentation.isLink &&
            manifest.recommendedMode == GuidelineReaderMode.partial) ...[
          AppSpacing.gapMd,
          Card(
            color: colors.secondaryContainer,
            margin: EdgeInsets.zero,
            child: Padding(
              padding: const EdgeInsets.all(AppSpacing.md),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    'Some sections have not been reviewed',
                    style: text.titleSmall?.copyWith(
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  AppSpacing.gapSm,
                  Text(
                    '${manifest.reviewedSectionCount} of ${manifest.sectionCount} sections have reviewed content.',
                  ),
                  if (manifest.hasOriginalPdf)
                    TextButton.icon(
                      onPressed: widget.onOriginal,
                      icon: const Icon(Icons.description_outlined),
                      label: const Text('Check the original document'),
                    ),
                ],
              ),
            ),
          ),
        ],
        if (widget.contentLoading) ...[
          AppSpacing.gapMd,
          const LinearProgressIndicator(
            semanticsLabel: 'Loading document contents',
          ),
          AppSpacing.gapSm,
          const Text('Loading contents…'),
        ],
        if (widget.contentUnavailable) ...[
          AppSpacing.gapMd,
          const Text(
            'The contents could not be loaded. You can retry or open the original document.',
          ),
          TextButton.icon(
            onPressed: widget.onRetryContent,
            icon: const Icon(Icons.refresh),
            label: const Text('Retry contents'),
          ),
          if (manifest.hasOriginalPdf)
            TextButton(
              onPressed: widget.onOriginal,
              child: const Text('Open original document'),
            ),
        ],
        if (tabs.length > 1) ...[
          AppSpacing.gapLg,
          _GuidelineTabs(
            selectedIndex: selected,
            hasChapters: hasContents,
            hasKeyPoints: hasKeyPoints,
            hasTables: hasTables,
            contentsLabel: presentation.contentsLabel,
            onSelected: (index) => setState(() => _selectedTab = index),
          ),
        ],
        AppSpacing.gapLg,
        switch (selected) {
          1 => _ChapterList(
            sections: sections,
            blocks: content.blocks,
            currentSectionId: widget.currentSectionId,
            readingProgress: widget.readingProgress,
            onSection: widget.onSection,
            contentsLabel: presentation.contentsLabel,
          ),
          2 => _KeyPointList(blocks: keyPoints),
          3 => _TableList(tables: tables, onTable: widget.onTable),
          _ => _GuidelineAbout(publication: publication),
        },
        const SizedBox(height: AppSpacing.lg),
      ],
    );
  }
}
