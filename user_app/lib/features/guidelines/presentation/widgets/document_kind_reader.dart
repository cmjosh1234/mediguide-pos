import 'dart:async';
import 'package:flutter/material.dart';
import 'package:scroll_to_index/scroll_to_index.dart';
import 'package:user_app/features/guidelines/data/models/guideline_publication.dart';
import 'package:user_app/features/guidelines/presentation/utils/document_reader_kind.dart';
import 'package:user_app/features/guidelines/presentation/utils/document_overview_presentation.dart';
import 'package:user_app/features/guidelines/presentation/widgets/document_contents_sheet.dart';
import 'package:user_app/features/guidelines/presentation/widgets/publication_block_view.dart';
import 'package:user_app/features/guidelines/presentation/widgets/publication_guideline_page_empty_section.dart';

/// Shared reading tools around kind-specific, approved publication content.
class DocumentKindReader extends StatefulWidget {
  const DocumentKindReader({
    super.key,
    required this.content,
    required this.kind,
    required this.onSection,
    required this.onOpenOriginal,
    this.initialSectionId,
    this.onOpenSourcePage,
  });
  final GuidelinePublicationContent content;
  final DocumentReaderKind kind;
  final String? initialSectionId;
  final ValueChanged<String> onSection;
  final VoidCallback onOpenOriginal;
  final ValueChanged<int>? onOpenSourcePage;
  @override
  State<DocumentKindReader> createState() => DocumentKindReaderState();
}

class DocumentKindReaderState extends State<DocumentKindReader> {
  final _readSections = <String>{};
  final _scroll = AutoScrollController(suggestedRowHeight: 360);
  final _viewport = GlobalKey();
  late bool _continuous = const {
    DocumentReaderKind.reference,
    DocumentReaderKind.checklist,
    DocumentReaderKind.caseDefinition,
  }.contains(widget.kind);
  double _textSize = 1;
  String? _selectedId;
  bool _navigating = false;
  bool _trackingScheduled = false;

  GuidelinePublicationContent? _cachedContent;
  List<PublicationSection> _cachedSections = const [];
  List<PublicationSection> get _sections {
    if (!identical(_cachedContent, widget.content)) {
      _cachedContent = widget.content;
      _cachedSections = [...widget.content.readerSections]
        ..sort((a, b) => a.sortOrder.compareTo(b.sortOrder));
    }
    return _cachedSections;
  }

  @override
  void initState() {
    super.initState();
    _selectedId = _validId(widget.initialSectionId);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted && _selectedId != null) {
        if (widget.initialSectionId != null) {
          unawaited(_select(_selectedId!));
        } else {
          widget.onSection(_selectedId!);
        }
      }
    });
  }

  String? _validId(String? id) {
    final sections = _sections;
    return sections.any((section) => section.id == id)
        ? id
        : sections.firstOrNull?.id;
  }

  @override
  void didUpdateWidget(DocumentKindReader oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.initialSectionId != oldWidget.initialSectionId &&
        widget.initialSectionId != _selectedId) {
      final id = _validId(widget.initialSectionId);
      if (id != null) {
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (mounted) unawaited(_select(id));
        });
      }
    }
  }

  @override
  void dispose() {
    _scroll.dispose();
    super.dispose();
  }

  Future<void> _select(String id) async {
    final index = _sections.indexWhere((section) => section.id == id);
    if (index < 0) return;
    setState(() {
      _selectedId = id;
      _navigating = true;
    });
    widget.onSection(id);
    try {
      if (_continuous && _scroll.hasClients) {
        await _scroll.scrollToIndex(
          index + 1,
          preferPosition: AutoScrollPosition.begin,
          duration: const Duration(milliseconds: 220),
        );
      } else if (_scroll.hasClients) {
        _scroll.jumpTo(0);
      }
    } finally {
      if (mounted) setState(() => _navigating = false);
    }
  }

  void _trackVisibleSection() {
    if (!mounted || !_continuous || _navigating) return;
    final viewport = _viewport.currentContext?.findRenderObject();
    if (viewport is! RenderBox || !viewport.attached) return;
    final top = viewport.localToGlobal(Offset.zero).dy;
    final sections = _sections;
    int? nearest;
    var distance = double.infinity;
    for (final entry in _scroll.tagMap.entries) {
      if (entry.key < 1 || entry.key > sections.length) continue;
      final box = entry.value.context.findRenderObject();
      if (box is! RenderBox || !box.attached) continue;
      final position = box.localToGlobal(Offset.zero).dy;
      final candidate = (position - top).abs();
      if (position <= top + 48 && position + box.size.height > top) {
        nearest = entry.key - 1;
        break;
      }
      if (candidate < distance) {
        distance = candidate;
        nearest = entry.key - 1;
      }
    }
    if (nearest != null && sections[nearest].id != _selectedId) {
      final id = sections[nearest].id;
      setState(() => _selectedId = id);
      widget.onSection(id);
    }
  }

  Future<void> openContents() => _showContents();
  void openSettings() => _settings();

  Future<void> _showContents() async {
    final id = await showModalBottomSheet<String>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      showDragHandle: true,
      builder: (_) => DocumentContentsSheet(
        sections: _sections,
        label: DocumentOverviewPresentation(widget.content).contentsLabel,
        selectedId: _selectedId,
      ),
    );
    if (mounted && id != null) await _select(id);
  }

  void _settings() {
    showModalBottomSheet<void>(
      context: context,
      useSafeArea: true,
      showDragHandle: true,
      isScrollControlled: true,
      builder: (_) => StatefulBuilder(
        builder: (context, updateSheet) => SizedBox(
          height: MediaQuery.sizeOf(context).height * 0.8,
          child: Column(
            children: [
              Expanded(
                child: SingleChildScrollView(
                  padding: const EdgeInsets.fromLTRB(24, 8, 24, 24),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Reading settings',
                        style: Theme.of(context).textTheme.titleLarge,
                      ),
                      const SizedBox(height: 16),
                      Text('Text size · ${(_textSize * 100).round()}%'),
                      Slider(
                        value: _textSize,
                        min: 1,
                        max: 1.6,
                        divisions: 6,
                        label: '${(_textSize * 100).round()}%',
                        onChanged: (value) {
                          setState(() => _textSize = value);
                          updateSheet(() {});
                        },
                      ),
                      const Text(
                        'Your device’s text size remains the starting size.',
                      ),
                      SwitchListTile(
                        contentPadding: EdgeInsets.zero,
                        title: const Text('Continuous reading'),
                        subtitle: const Text(
                          'Show all sections in one scrollable document.',
                        ),
                        value: _continuous,
                        onChanged: (value) {
                          setState(() => _continuous = value);
                          updateSheet(() {});
                          WidgetsBinding.instance.addPostFrameCallback((_) {
                            if (mounted && _selectedId != null) {
                              unawaited(_select(_selectedId!));
                            }
                          });
                        },
                      ),
                      if (widget.kind == DocumentReaderKind.checklist &&
                          _readSections.isNotEmpty)
                        TextButton(
                          onPressed: () {
                            setState(_readSections.clear);
                            updateSheet(() {});
                          },
                          child: const Text('Reset reading marks'),
                        ),
                    ],
                  ),
                ),
              ),
              const Divider(height: 1),
              Padding(
                padding: const EdgeInsets.all(16),
                child: SizedBox(
                  width: double.infinity,
                  child: FilledButton(
                    onPressed: () => Navigator.pop(context),
                    child: const Text('Done'),
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final sections = _sections;
    final index = sections.indexWhere((section) => section.id == _selectedId);
    final selected = index < 0 ? 0 : index;
    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    final textTheme = theme.textTheme.apply(fontSizeFactor: _textSize);
    final readerTheme = theme.copyWith(
      textTheme: textTheme.copyWith(
        bodyMedium: textTheme.bodyMedium?.copyWith(height: 1.6),
        bodyLarge: textTheme.bodyLarge?.copyWith(height: 1.6),
      ),
    );
    final toolbar = Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Row(
        children: [
          Expanded(
            child: TextButton.icon(
              onPressed: sections.isEmpty ? null : _showContents,
              icon: const Icon(Icons.list_alt),
              label: const Text('Contents'),
            ),
          ),
          if (sections.isNotEmpty)
            Flexible(
              child: Text(
                '${selected + 1} / ${sections.length}',
                textAlign: TextAlign.center,
                semanticsLabel: 'Section ${selected + 1} of ${sections.length}',
                style: theme.textTheme.labelLarge,
              ),
            ),
          IconButton(
            tooltip: 'Reading settings',
            onPressed: _settings,
            icon: const Icon(Icons.text_fields),
          ),
        ],
      ),
    );
    return Column(
      children: [
        toolbar,
        if (sections.isNotEmpty)
          LinearProgressIndicator(
            value: (selected + 1) / sections.length,
            minHeight: 2,
            semanticsLabel: 'Section position',
          ),
        Expanded(
          key: _viewport,
          child: Theme(
            data: readerTheme,
            child: NotificationListener<ScrollNotification>(
              onNotification: (notification) {
                if (_continuous &&
                    !_trackingScheduled &&
                    notification is ScrollEndNotification) {
                  _trackingScheduled = true;
                  WidgetsBinding.instance.addPostFrameCallback((_) {
                    _trackingScheduled = false;
                    _trackVisibleSection();
                  });
                }
                return false;
              },
              child: _continuous && sections.isNotEmpty
                  ? ListView.builder(
                      controller: _scroll,
                      padding: const EdgeInsets.only(bottom: 32),
                      itemCount: sections.length + 1,
                      itemBuilder: (context, position) => AutoScrollTag(
                        key: ValueKey('reader-$position'),
                        controller: _scroll,
                        index: position,
                        child: _measure(
                          context,
                          position == 0
                              ? _header(context)
                              : _section(context, sections[position - 1]),
                        ),
                      ),
                    )
                  : ListView(
                      controller: _scroll,
                      padding: const EdgeInsets.only(bottom: 32),
                      children: [
                        _measure(
                          context,
                          Builder(builder: (context) => _header(context)),
                        ),
                        if (sections.isEmpty)
                          _measure(context, _empty(const []))
                        else
                          _measure(
                            context,
                            Builder(
                              builder: (context) =>
                                  _section(context, sections[selected]),
                            ),
                          ),
                      ],
                    ),
            ),
          ),
        ),
        if (sections.isNotEmpty)
          Container(
            decoration: BoxDecoration(
              color: colors.surface,
              border: Border(top: BorderSide(color: colors.outlineVariant)),
            ),
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
            child: Wrap(
              spacing: 16,
              runSpacing: 4,
              alignment: WrapAlignment.spaceBetween,
              children: [
                TextButton.icon(
                  onPressed: selected > 0 && !_navigating
                      ? () => _select(sections[selected - 1].id)
                      : null,
                  icon: const Icon(Icons.chevron_left),
                  label: const Text('Previous section'),
                ),
                TextButton.icon(
                  onPressed: selected + 1 < sections.length && !_navigating
                      ? () => _select(sections[selected + 1].id)
                      : null,
                  icon: const Icon(Icons.chevron_right),
                  label: const Text('Next section'),
                ),
              ],
            ),
          ),
      ],
    );
  }

  Widget _measure(BuildContext context, Widget child) => Center(
    child: ConstrainedBox(
      constraints: const BoxConstraints(maxWidth: 760),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 20),
        child: SizedBox(width: double.infinity, child: child),
      ),
    ),
  );

  Widget _header(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 20),
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          widget.kind.label,
          style: Theme.of(context).textTheme.labelLarge?.copyWith(
            color: Theme.of(context).colorScheme.primary,
          ),
        ),
        if (widget.content.publication.sourceOrganization.isNotEmpty) ...[
          const SizedBox(height: 4),
          Text(
            widget.content.publication.sourceOrganization,
            style: Theme.of(context).textTheme.bodySmall,
          ),
        ],
        if (widget.content.manifest.recommendedMode ==
            GuidelineReaderMode.partial) ...[
          const SizedBox(height: 12),
          const Text(
            'Partially reviewed content. Verify the original document for sections without approved content.',
          ),
          if (widget.content.manifest.hasOriginalPdf)
            TextButton.icon(
              onPressed: widget.onOpenOriginal,
              icon: const Icon(Icons.description_outlined),
              label: const Text('Open original document'),
            ),
        ],
        if (widget.kind == DocumentReaderKind.checklist) ...[
          const SizedBox(height: 16),
          Text(
            '${_readSections.length} of ${_sections.where((s) => widget.content.displayBlocksFor(s).isNotEmpty).length} sections read',
          ),
          const Text(
            'Reading marks last for this session and do not record clinical task completion.',
          ),
          if (_readSections.isNotEmpty)
            TextButton(
              onPressed: () => setState(_readSections.clear),
              child: const Text('Reset reading marks'),
            ),
        ],
      ],
    ),
  );

  Widget _section(BuildContext context, PublicationSection section) {
    final content = widget.content;
    final blocks = content.displayBlocksFor(section);
    final body = Padding(
      padding: const EdgeInsets.symmetric(vertical: 16, horizontal: 16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (!content.isDocumentTitleWrapper(section))
            Semantics(
              header: true,
              child: Text(
                section.title,
                style: Theme.of(context).textTheme.headlineSmall?.copyWith(
                  fontWeight: FontWeight.w700,
                  height: 1.25,
                ),
              ),
            ),
          if (section.pageLabel.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child:
                  content.manifest.hasOriginalPdf &&
                      widget.onOpenSourcePage != null
                  ? TextButton.icon(
                      onPressed: () =>
                          widget.onOpenSourcePage!(section.pageStart!),
                      icon: const Icon(Icons.open_in_new, size: 16),
                      label: Text(section.pageLabel),
                    )
                  : Text(
                      section.pageLabel,
                      style: Theme.of(context).textTheme.labelMedium,
                    ),
            ),
          if (widget.kind == DocumentReaderKind.checklist && blocks.isNotEmpty)
            CheckboxListTile(
              contentPadding: EdgeInsets.zero,
              title: const Text('Section read'),
              value: _readSections.contains(section.id),
              onChanged: (checked) {
                setState(() {
                  _selectedId = section.id;
                  checked == true
                      ? _readSections.add(section.id)
                      : _readSections.remove(section.id);
                });
                widget.onSection(section.id);
              },
            ),
          const SizedBox(height: 20),
          if (blocks.isEmpty) _empty(_descendants(section.id)),
          for (final block in blocks) ...[
            PublicationBlockView(
              block: block,
              guidelineId: content.publication.id,
            ),
            const SizedBox(height: 20),
          ],
        ],
      ),
    );
    return Padding(
      padding: const EdgeInsets.only(bottom: 16),
      child:
          const {
            DocumentReaderKind.checklist,
            DocumentReaderKind.caseDefinition,
          }.contains(widget.kind)
          ? Card(margin: EdgeInsets.zero, child: body)
          : body,
    );
  }

  List<PublicationSection> _descendants(String id) {
    final ids = <String>{id};
    var changed = true;
    while (changed) {
      changed = false;
      for (final section in widget.content.sections) {
        if (ids.contains(section.parentId) && ids.add(section.id)) {
          changed = true;
        }
      }
    }
    return _sections
        .where(
          (section) =>
              section.id != id &&
              ids.contains(section.id) &&
              widget.content.displayBlocksFor(section).isNotEmpty,
        )
        .toList();
  }

  Widget _empty(List<PublicationSection> descendants) =>
      PublicationEmptyReviewedSection(
        hasOriginalDocument: widget.content.manifest.hasOriginalPdf,
        reviewedDescendants: descendants,
        onSection: (id) => _select(id),
        onOpenOriginal: widget.onOpenOriginal,
      );
}
