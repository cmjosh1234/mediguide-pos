part of '../screens/publication_guideline_page.dart';

class _ChapterList extends StatefulWidget {
  const _ChapterList({
    this.contentsLabel = 'Chapters',
    required this.sections,
    required this.blocks,
    required this.currentSectionId,
    required this.readingProgress,
    required this.onSection,
  });

  final String contentsLabel;
  final List<PublicationSection> sections;
  final List<GuidelineBlock> blocks;
  final String? currentSectionId;
  final double? readingProgress;
  final ValueChanged<String> onSection;

  @override
  State<_ChapterList> createState() => _ChapterListState();
}

class _ChapterListState extends State<_ChapterList> {
  final TextEditingController _searchController = TextEditingController();
  final Set<String> _expanded = <String>{};
  String _query = '';

  @override
  void initState() {
    super.initState();
    _expandCurrentPath();
  }

  @override
  void didUpdateWidget(covariant _ChapterList oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.currentSectionId != widget.currentSectionId ||
        oldWidget.sections != widget.sections) {
      _expandCurrentPath();
    }
  }

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  void _expandCurrentPath() {
    var id = widget.currentSectionId;
    final byId = {for (final section in widget.sections) section.id: section};
    while (id != null && id.isNotEmpty) {
      _expanded.add(id);
      id = byId[id]?.parentId;
    }
  }

  @override
  Widget build(BuildContext context) {
    final sections = [...widget.sections]
      ..sort((a, b) => a.sortOrder.compareTo(b.sortOrder));
    final byId = {for (final section in sections) section.id: section};
    final children = <String, List<PublicationSection>>{};
    for (final section in sections) {
      final parentId = section.parentId;
      if (parentId != null && byId.containsKey(parentId)) {
        children.putIfAbsent(parentId, () => []).add(section);
      }
    }
    final roots = sections
        .where(
          (section) =>
              section.parentId == null || !byId.containsKey(section.parentId),
        )
        .toList(growable: false);
    final displayRoots = guidelineChapterDisplayRoots(roots, children);
    final blocksBySection = <String, List<GuidelineBlock>>{};
    for (final block in widget.blocks) {
      final sectionId = block.sectionId;
      if (sectionId != null) {
        blocksBySection.putIfAbsent(sectionId, () => []).add(block);
      }
    }
    final normalizedQuery = _query.trim().toLowerCase();
    final matches = normalizedQuery.isEmpty
        ? const <PublicationSection>[]
        : sections
              .where(
                (section) => _sectionSearchText(
                  section,
                  blocksBySection[section.id] ?? const [],
                ).contains(normalizedQuery),
              )
              .toList(growable: false);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Expanded(
              child: Text(
                widget.contentsLabel,
                style: Theme.of(
                  context,
                ).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w800),
              ),
            ),
            Text(
              '${sections.length} sections',
              style: Theme.of(context).textTheme.labelMedium?.copyWith(
                color: Theme.of(context).colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ),
        AppSpacing.gapSm,
        TextField(
          controller: _searchController,
          textInputAction: TextInputAction.search,
          onChanged: (value) => setState(() => _query = value),
          decoration: InputDecoration(
            hintText:
                'Search ${widget.contentsLabel.toLowerCase()} and content',
            prefixIcon: const Icon(LucideIcons.search),
            suffixIcon: _query.isEmpty
                ? null
                : IconButton(
                    tooltip: 'Clear content search',
                    onPressed: () {
                      _searchController.clear();
                      setState(() => _query = '');
                    },
                    icon: const Icon(LucideIcons.x),
                  ),
          ),
        ),
        AppSpacing.gapMd,
        if (sections.isEmpty)
          const Text('No structured chapters available.')
        else if (normalizedQuery.isNotEmpty)
          _ChapterSearchResults(
            sections: matches,
            blocksBySection: blocksBySection,
            currentSectionId: widget.currentSectionId,
            onSection: widget.onSection,
          )
        else
          LayoutBuilder(
            builder: (context, constraints) {
              final columns = constraints.maxWidth >= 700 ? 2 : 1;
              final spacing = columns == 1 ? 0.0 : AppSpacing.md;
              final width =
                  (constraints.maxWidth - spacing * (columns - 1)) / columns;
              return Wrap(
                spacing: spacing,
                runSpacing: AppSpacing.md,
                children: [
                  for (final root in displayRoots)
                    SizedBox(
                      width: width,
                      child: _ChapterCard(
                        section: root,
                        allSections: sections,
                        children: children,
                        blocksBySection: blocksBySection,
                        expanded: _expanded,
                        currentSectionId: widget.currentSectionId,
                        readingProgress: widget.readingProgress,
                        onExpansionChanged: (id, value) {
                          setState(() {
                            value ? _expanded.add(id) : _expanded.remove(id);
                          });
                        },
                        onSection: widget.onSection,
                      ),
                    ),
                ],
              );
            },
          ),
      ],
    );
  }
}

/// Returns the sections that should be rendered as chapter cards.
///
/// Markdown publications normally have a single H1 document-title wrapper,
/// with their actual chapters represented by its H2 children. Curated legacy
/// publications can instead expose chapters directly as roots. Supporting both
/// shapes keeps the chapter browser consistent without changing source truth.
List<PublicationSection> guidelineChapterDisplayRoots(
  List<PublicationSection> roots,
  Map<String, List<PublicationSection>> children,
) {
  if (roots.length != 1) return roots;
  final documentRoot = roots.single;
  final directChildren = children[documentRoot.id] ?? const [];
  if (documentRoot.level == 1 && directChildren.isNotEmpty) {
    return directChildren;
  }
  return roots;
}

class _ChapterCard extends StatelessWidget {
  const _ChapterCard({
    required this.section,
    required this.allSections,
    required this.children,
    required this.blocksBySection,
    required this.expanded,
    required this.currentSectionId,
    required this.readingProgress,
    required this.onExpansionChanged,
    required this.onSection,
  });

  final PublicationSection section;
  final List<PublicationSection> allSections;
  final Map<String, List<PublicationSection>> children;
  final Map<String, List<GuidelineBlock>> blocksBySection;
  final Set<String> expanded;
  final String? currentSectionId;
  final double? readingProgress;
  final void Function(String id, bool expanded) onExpansionChanged;
  final ValueChanged<String> onSection;

  @override
  Widget build(BuildContext context) {
    final descendants = _chapterDescendants(section.id, children);
    final sectionIds = <String>{
      section.id,
      ...descendants.map((item) => item.id),
    };
    final blockCount = sectionIds.fold<int>(
      0,
      (sum, id) => sum + (blocksBySection[id]?.length ?? 0),
    );
    final childSections = children[section.id] ?? const [];
    final description = _chapterDescription(sectionIds, blocksBySection);
    final progress = _chapterProgress(
      sectionIds,
      allSections,
      currentSectionId,
      readingProgress,
    );

    return Card(
      clipBehavior: Clip.antiAlias,
      margin: EdgeInsets.zero,
      child: Column(
        children: [
          InkWell(
            onTap: () => onSection(section.id),
            child: Padding(
              padding: const EdgeInsets.all(AppSpacing.md),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  DecoratedBox(
                    decoration: BoxDecoration(
                      color: Theme.of(
                        context,
                      ).colorScheme.primaryContainer.withValues(alpha: .6),
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child: Padding(
                      padding: const EdgeInsets.all(10),
                      child: Icon(
                        _chapterIcon(section.title),
                        color: Theme.of(context).colorScheme.primary,
                      ),
                    ),
                  ),
                  AppSpacing.hGapMd,
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          section.title,
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                          style: Theme.of(context).textTheme.titleSmall
                              ?.copyWith(fontWeight: FontWeight.w700),
                        ),
                        if (description.isNotEmpty) ...[
                          AppSpacing.gapXs,
                          Text(
                            description,
                            maxLines: 2,
                            overflow: TextOverflow.ellipsis,
                            style: Theme.of(context).textTheme.bodySmall
                                ?.copyWith(
                                  color: Theme.of(
                                    context,
                                  ).colorScheme.onSurfaceVariant,
                                ),
                          ),
                        ],
                        AppSpacing.gapSm,
                        Wrap(
                          spacing: AppSpacing.sm,
                          runSpacing: AppSpacing.xs,
                          children: [
                            _ChapterMetric(
                              icon: LucideIcons.layers3,
                              label: '${descendants.length + 1} sections',
                            ),
                            _ChapterMetric(
                              icon: LucideIcons.fileText,
                              label: '$blockCount reviewed blocks',
                            ),
                            if (section.pageLabel.isNotEmpty)
                              _ChapterMetric(
                                icon: LucideIcons.bookOpen,
                                label: section.pageLabel,
                              ),
                          ],
                        ),
                        if (progress != null) ...[
                          AppSpacing.gapSm,
                          LinearProgressIndicator(value: progress),
                          AppSpacing.gapXs,
                          Text(
                            progress >= 1
                                ? 'Completed'
                                : progress > 0
                                ? '${(progress * 100).round()}% read'
                                : 'Not started',
                            style: Theme.of(context).textTheme.labelSmall,
                          ),
                        ],
                      ],
                    ),
                  ),
                  const Icon(LucideIcons.chevronRight, size: 18),
                ],
              ),
            ),
          ),
          if (childSections.isNotEmpty) ...[
            const Divider(height: 1),
            InkWell(
              onTap: () => onExpansionChanged(
                section.id,
                !expanded.contains(section.id),
              ),
              child: Padding(
                padding: const EdgeInsets.symmetric(
                  horizontal: AppSpacing.md,
                  vertical: AppSpacing.sm,
                ),
                child: Row(
                  children: [
                    Expanded(
                      child: Text(
                        expanded.contains(section.id)
                            ? 'Hide sections'
                            : 'View ${childSections.length} sections',
                        style: Theme.of(context).textTheme.labelLarge,
                      ),
                    ),
                    Icon(
                      expanded.contains(section.id)
                          ? LucideIcons.chevronUp
                          : LucideIcons.chevronDown,
                      size: 18,
                    ),
                  ],
                ),
              ),
            ),
            if (expanded.contains(section.id))
              Padding(
                padding: const EdgeInsets.fromLTRB(
                  AppSpacing.md,
                  0,
                  AppSpacing.md,
                  AppSpacing.md,
                ),
                child: Column(
                  children: [
                    for (final child in childSections)
                      _ChapterBranch(
                        section: child,
                        children: children,
                        blocksBySection: blocksBySection,
                        expanded: expanded,
                        currentSectionId: currentSectionId,
                        onExpansionChanged: onExpansionChanged,
                        onSection: onSection,
                      ),
                  ],
                ),
              ),
          ],
        ],
      ),
    );
  }
}

class _ChapterBranch extends StatelessWidget {
  const _ChapterBranch({
    required this.section,
    required this.children,
    required this.blocksBySection,
    required this.expanded,
    required this.currentSectionId,
    required this.onExpansionChanged,
    required this.onSection,
  });

  final PublicationSection section;
  final Map<String, List<PublicationSection>> children;
  final Map<String, List<GuidelineBlock>> blocksBySection;
  final Set<String> expanded;
  final String? currentSectionId;
  final void Function(String id, bool expanded) onExpansionChanged;
  final ValueChanged<String> onSection;

  @override
  Widget build(BuildContext context) {
    final nested = children[section.id] ?? const [];
    final isExpanded = expanded.contains(section.id);
    final isCurrent = currentSectionId == section.id;
    return Column(
      children: [
        ListTile(
          dense: true,
          contentPadding: EdgeInsets.only(
            left: ((section.level - 2).clamp(0, 4)) * 10.0,
          ),
          leading: Icon(
            isCurrent ? LucideIcons.circleCheck : LucideIcons.fileText,
            size: 18,
            color: isCurrent ? Theme.of(context).colorScheme.primary : null,
          ),
          title: Text(section.title),
          subtitle: Text(
            '${blocksBySection[section.id]?.length ?? 0} reviewed blocks'
            '${section.pageLabel.isEmpty ? '' : ' • ${section.pageLabel}'}',
          ),
          trailing: nested.isEmpty
              ? const Icon(LucideIcons.chevronRight, size: 18)
              : IconButton(
                  tooltip: isExpanded ? 'Collapse section' : 'Expand section',
                  onPressed: () => onExpansionChanged(section.id, !isExpanded),
                  icon: Icon(
                    isExpanded
                        ? LucideIcons.chevronUp
                        : LucideIcons.chevronDown,
                    size: 18,
                  ),
                ),
          onTap: () => onSection(section.id),
        ),
        if (isExpanded)
          for (final child in nested)
            _ChapterBranch(
              section: child,
              children: children,
              blocksBySection: blocksBySection,
              expanded: expanded,
              currentSectionId: currentSectionId,
              onExpansionChanged: onExpansionChanged,
              onSection: onSection,
            ),
      ],
    );
  }
}

class _ChapterSearchResults extends StatelessWidget {
  const _ChapterSearchResults({
    required this.sections,
    required this.blocksBySection,
    required this.currentSectionId,
    required this.onSection,
  });

  final List<PublicationSection> sections;
  final Map<String, List<GuidelineBlock>> blocksBySection;
  final String? currentSectionId;
  final ValueChanged<String> onSection;

  @override
  Widget build(BuildContext context) {
    if (sections.isEmpty) {
      return const Padding(
        padding: EdgeInsets.symmetric(vertical: AppSpacing.lg),
        child: Center(child: Text('No matching chapters or content.')),
      );
    }
    return Column(
      children: [
        for (final section in sections)
          Card(
            margin: const EdgeInsets.only(bottom: AppSpacing.sm),
            child: ListTile(
              leading: Icon(
                _chapterIcon(section.title),
                color: Theme.of(context).colorScheme.primary,
              ),
              title: Text(section.title),
              subtitle: Text(
                '${blocksBySection[section.id]?.length ?? 0} reviewed blocks'
                '${section.pageLabel.isEmpty ? '' : ' • ${section.pageLabel}'}',
              ),
              trailing: currentSectionId == section.id
                  ? const Icon(LucideIcons.circleCheck)
                  : const Icon(LucideIcons.chevronRight),
              onTap: () => onSection(section.id),
            ),
          ),
      ],
    );
  }
}

class _ChapterMetric extends StatelessWidget {
  const _ChapterMetric({required this.icon, required this.label});

  final IconData icon;
  final String label;

  @override
  Widget build(BuildContext context) => Row(
    mainAxisSize: MainAxisSize.min,
    children: [
      Icon(icon, size: 14),
      const SizedBox(width: 4),
      Text(label, style: Theme.of(context).textTheme.labelSmall),
    ],
  );
}

List<PublicationSection> _chapterDescendants(
  String rootId,
  Map<String, List<PublicationSection>> children,
) {
  final result = <PublicationSection>[];
  final pending = <PublicationSection>[...(children[rootId] ?? const [])];
  while (pending.isNotEmpty) {
    final section = pending.removeAt(0);
    result.add(section);
    pending.addAll(children[section.id] ?? const []);
  }
  return result;
}

String _chapterDescription(
  Set<String> sectionIds,
  Map<String, List<GuidelineBlock>> blocksBySection,
) {
  for (final sectionId in sectionIds) {
    for (final block in blocksBySection[sectionId] ?? const []) {
      final text = _chapterBlockText(block).trim();
      if (text.isNotEmpty) {
        return text.length <= 150 ? text : '${text.substring(0, 147)}…';
      }
    }
  }
  return '';
}

String _sectionSearchText(
  PublicationSection section,
  List<GuidelineBlock> blocks,
) =>
    '${section.title} ${blocks.map(_chapterBlockText).join(' ')}'.toLowerCase();

String _chapterBlockText(GuidelineBlock block) => switch (block) {
  ParagraphGuidelineBlock(:final text) => text,
  HeadingGuidelineBlock(:final text) => text,
  OrderedListGuidelineBlock(:final items) => items.join(' '),
  UnorderedListGuidelineBlock(:final items) => items.join(' '),
  TableGuidelineBlock(:final payload) => payload.title,
  FigureGuidelineBlock(:final payload) => payload.caption,
  CalloutGuidelineBlock(:final payload) =>
    '${payload.title} ${payload.content}',
  AlgorithmGuidelineBlock(:final payload) => payload.title,
  ReferenceGuidelineBlock(:final citation) => citation,
  PageBreakGuidelineBlock() => '',
  UnknownGuidelineBlock() => '',
};

double? _chapterProgress(
  Set<String> sectionIds,
  List<PublicationSection> allSections,
  String? currentSectionId,
  double? overallProgress,
) {
  if (currentSectionId == null || currentSectionId.isEmpty) return null;
  final currentIndex = allSections.indexWhere(
    (section) => section.id == currentSectionId,
  );
  if (currentIndex < 0) return overallProgress?.clamp(0, 1);
  final indexes = <int>[
    for (var index = 0; index < allSections.length; index++)
      if (sectionIds.contains(allSections[index].id)) index,
  ];
  if (indexes.isEmpty) return null;
  if (currentIndex > indexes.last) return 1;
  if (currentIndex < indexes.first) return 0;
  final reached = indexes.where((index) => index <= currentIndex).length;
  return (reached / indexes.length).clamp(0, 1);
}

IconData _chapterIcon(String title) {
  final value = title.toLowerCase();
  if (value.contains('drug') || value.contains('medicine')) {
    return LucideIcons.pill;
  }
  if (value.contains('child') || value.contains('paediatric')) {
    return LucideIcons.baby;
  }
  if (value.contains('emerg') || value.contains('trauma')) {
    return LucideIcons.activity;
  }
  if (value.contains('prevent') || value.contains('infection')) {
    return LucideIcons.shieldCheck;
  }
  if (value.contains('heart') || value.contains('cardio')) {
    return LucideIcons.heartPulse;
  }
  if (value.contains('surgery') || value.contains('clinical')) {
    return LucideIcons.stethoscope;
  }
  return LucideIcons.bookOpenText;
}
