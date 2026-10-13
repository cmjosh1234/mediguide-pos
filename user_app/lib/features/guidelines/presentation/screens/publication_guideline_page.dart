import 'dart:async';
import 'package:user_app/features/notifications/presentation/controllers/notification_preferences_controller.dart';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';
import 'package:url_launcher/url_launcher.dart';

import 'package:user_app/app/providers/app_providers.dart';
import 'package:user_app/app/router/route_names.dart';
import 'package:user_app/core/constants/app_spacing.dart';
import 'package:user_app/core/utils/app_message.dart';
import 'package:user_app/core/utils/responsive.dart';
import 'package:user_app/core/widgets/app_error_view.dart';
import 'package:user_app/core/widgets/app_loading_view.dart';
import 'package:user_app/core/widgets/app_status_badge.dart';

import 'package:user_app/features/ai_assistant/data/models/ai_context.dart';
import 'package:user_app/features/authentication/presentation/controllers/auth_controller.dart';
import 'package:user_app/features/documents/presentation/screens/document_reader_page.dart';
import 'package:user_app/features/documents/presentation/widgets/document_actions_sheet.dart';
import 'package:user_app/features/downloads/data/models/offline_download.dart';
import 'package:user_app/features/downloads/presentation/controllers/guideline_downloads_controller.dart';
import 'package:user_app/features/guidelines/data/models/guideline_publication.dart';
import 'package:user_app/features/guidelines/data/models/reading_progress.dart';
import 'package:user_app/features/guidelines/presentation/controllers/publication_guideline_controller.dart';
import 'package:user_app/features/guidelines/presentation/widgets/document_kind_reader.dart';
import 'package:user_app/features/guidelines/presentation/utils/document_reader_kind.dart';
import 'package:user_app/features/guidelines/presentation/utils/document_overview_presentation.dart';
import 'package:user_app/features/library/presentation/widgets/save_to_collection_sheet.dart';
import 'package:user_app/features/library/presentation/utils/collection_messages.dart';
import 'package:user_app/shared/widgets/app_markdown_body.dart';

part '../widgets/publication_guideline_page_guideline_content_search_delegate.dart';
part '../widgets/publication_guideline_page_reading_notes_sheet.dart';
part '../widgets/publication_guideline_page_reading_notes_sheet_state.dart';
part '../widgets/publication_guideline_page_guideline_overview.dart';
part '../widgets/publication_guideline_page_guideline_overview_state.dart';
part '../widgets/publication_guideline_page_guideline_tabs.dart';
part '../widgets/publication_guideline_page_guideline_about.dart';
part '../widgets/publication_guideline_page_chapter_list.dart';
part '../widgets/publication_guideline_page_key_point_list.dart';
part '../widgets/publication_guideline_page_table_list.dart';
part '../widgets/publication_guideline_page_review_status.dart';
part '../widgets/publication_guideline_page_reader_action_bar.dart';
part '../widgets/publication_guideline_page_overview_bottom_actions.dart';
part '../widgets/publication_guideline_page_reading_bottom_actions.dart';
part '../widgets/publication_guideline_page_bottom_icon_action.dart';
part '../widgets/publication_guideline_page_reader_bottom_action.dart';
part '../widgets/publication_guideline_page_original_document_reader.dart';
part '../widgets/publication_guideline_page_linked_document.dart';

class PublicationGuidelinePage extends ConsumerStatefulWidget {
  const PublicationGuidelinePage({
    super.key,
    required this.guidelineId,
    this.readerOnly = false,
  });

  final String guidelineId;
  final bool readerOnly;

  @override
  ConsumerState<PublicationGuidelinePage> createState() =>
      _PublicationGuidelinePageState();
}

class _PublicationGuidelinePageState
    extends ConsumerState<PublicationGuidelinePage> {
  String? _selectedSectionId;
  String? _readerIdentity;
  GlobalKey<DocumentKindReaderState> _readerKey =
      GlobalKey<DocumentKindReaderState>();
  String? _currentSectionId;

  bool _usageTracked = false;
  bool _deepLinkApplied = false;
  bool _initialProgressScheduled = false;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();

    if (_deepLinkApplied) {
      return;
    }

    _deepLinkApplied = true;

    final section = GoRouterState.of(context).uri.queryParameters['section'];

    if (section != null && section.trim().isNotEmpty) {
      _selectedSectionId = section.trim();
    }
  }

  @override
  Widget build(BuildContext context) {
    final summaryContent = ref.watch(
      publicationGuidelineSummaryProvider(widget.guidelineId),
    );
    final summary = summaryContent.valueOrNull;
    final kind = summary == null
        ? null
        : DocumentReaderKind.forPublication(summary.publication);
    // File and link readers need only metadata; their structured endpoints
    // may intentionally have no published content.
    final completeContent =
        summaryContent.hasError ||
            (summary != null &&
                kind != DocumentReaderKind.uploaded &&
                kind != DocumentReaderKind.link &&
                summary.manifest.recommendedMode !=
                    GuidelineReaderMode.originalDocument)
        ? ref.watch(publicationGuidelineProvider(widget.guidelineId))
        : summaryContent;
    final content = widget.readerOnly || completeContent.valueOrNull != null
        ? completeContent
        : summaryContent;

    if (!_usageTracked && content.valueOrNull != null) {
      _usageTracked = true;
      unawaited(
        ref.read(usageRepositoryProvider).guideline(widget.guidelineId),
      );
    }
    final progress = ref
        .watch(publicationReadingProgressProvider(widget.guidelineId))
        .valueOrNull;
    final downloadItems = ref
        .watch(guidelineDownloadsControllerProvider)
        .valueOrNull;
    final currentContent = content.valueOrNull;
    final overviewPresentation = currentContent == null
        ? null
        : DocumentOverviewPresentation(currentContent);
    final canResume =
        currentContent != null &&
        (progress?.progressPercentage ?? 0) > 0 &&
        (progress?.progressPercentage ?? 0) < 1 &&
        currentContent.readerSections.any(
          (section) => section.id == progress?.currentSection,
        ) &&
        !overviewPresentation!.isLink &&
        !overviewPresentation.isFile;
    final targetAssetType = currentContent?.manifest.hasOfflinePackage == true
        ? 'offline_package'
        : 'original_pdf';
    OfflineDownload? offlineDownload;
    for (final item in downloadItems ?? const <OfflineDownload>[]) {
      if (item.guidelineId == widget.guidelineId &&
          item.assetType == targetAssetType) {
        offlineDownload = item;
        break;
      }
    }

    return Scaffold(
      appBar: AppBar(
        title: Text(
          content.valueOrNull?.publication.title ??
              (widget.readerOnly ? 'Document reader' : 'Document overview'),
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
        ),
        actions: [
          if (!widget.readerOnly &&
              ref.watch(authControllerProvider).valueOrNull?.user != null)
            _FollowGuidelineButton(guidelineId: widget.guidelineId),
        ],
      ),

      // =====================================================================
      // BODY
      // =====================================================================
      body: content.when(
        loading: () => const AppLoadingView(message: 'Loading document...'),
        error: (error, _) => AppErrorView(
          error: error,
          onRetry: () {
            ref.invalidate(publicationGuidelineProvider(widget.guidelineId));
            ref.invalidate(
              publicationGuidelineSummaryProvider(widget.guidelineId),
            );
          },
        ),
        data: (value) {
          if (!widget.readerOnly) {
            return _overviewPage(
              context,
              value,
              progress,
              contentLoading: completeContent.isLoading,
              contentUnavailable:
                  completeContent.hasError && !summaryContent.hasError,
            );
          }
          if (DocumentReaderKind.forPublication(value.publication) ==
              DocumentReaderKind.link) {
            return _LinkedDocumentView(
              publication: value.publication,
              onOpen: () {
                _openWebsite(context, value.publication.externalUrl);
              },
            );
          }

          return _content(context, value);
        },
      ),

      // =====================================================================
      // BOTTOM ACTION BAR
      // =====================================================================
      // Keep document actions available from one consistent bottom menu.
      bottomNavigationBar: currentContent == null
          ? null
          : _ReaderActionBar(
              isBookmarked: progress?.isBookmarked == true,
              offlineDownload: offlineDownload,
              showRead: !widget.readerOnly,
              primaryLabel: canResume
                  ? 'Continue reading'
                  : overviewPresentation!.actionLabel,
              primaryIcon: overviewPresentation.isLink
                  ? Icons.open_in_new
                  : overviewPresentation.icon,
              primaryEnabled: overviewPresentation.canOpen,
              showDownload:
                  !overviewPresentation.isLink &&
                  (currentContent.manifest.hasOfflinePackage ||
                      currentContent.manifest.hasOriginalPdf),
              showAi:
                  !overviewPresentation.isLink &&
                  !overviewPresentation.isFile &&
                  currentContent.blocks.isNotEmpty,
              onMore: () => _showDocumentActions(
                context,
                currentContent,
                progress,
                offlineDownload,
              ),
              onRead: () {
                if (overviewPresentation.isLink) {
                  _openWebsite(context, currentContent.publication.externalUrl);
                } else if (overviewPresentation.isFile) {
                  _openOriginal(context);
                } else {
                  final route = AppRoutes.readPublicGuideline(
                    widget.guidelineId,
                  );
                  context.push(
                    canResume
                        ? '$route?section=${Uri.encodeQueryComponent(progress!.currentSection)}'
                        : route,
                  );
                }
              },
              onAskAi: () {
                _openAiAssistant(context, content.requireValue);
              },
              onBookmark: () {
                _toggleBookmark(context);
              },
              onNotes: () {
                _editNotes(context, progress?.notes ?? '');
              },
              onDownload: () {
                if (offlineDownload?.status == OfflineDownloadStatus.ready) {
                  context.push(AppRoutes.offlineContent);
                  return;
                }
                if (offlineDownload?.status ==
                        OfflineDownloadStatus.downloading ||
                    offlineDownload?.status == OfflineDownloadStatus.queued) {
                  return;
                }
                _download(context, content.requireValue);
              },
            ),
    );
  }

  // ===========================================================================
  // MENU
  // ===========================================================================

  Future<void> _showDocumentActions(
    BuildContext context,
    GuidelinePublicationContent content,
    ReadingProgress? progress,
    OfflineDownload? offlineDownload,
  ) async {
    final presentation = DocumentOverviewPresentation(content);
    OfflineDownload? matchingDownload(List<OfflineDownload>? items) {
      final assetType = content.manifest.hasOfflinePackage
          ? 'offline_package'
          : 'original_pdf';
      return items
          ?.where(
            (item) =>
                item.guidelineId == widget.guidelineId &&
                item.assetType == assetType,
          )
          .firstOrNull;
    }

    final action = await showModalBottomSheet<DocumentMenuAction>(
      context: context,
      useSafeArea: true,
      showDragHandle: true,
      isScrollControlled: true,
      constraints: const BoxConstraints(maxWidth: 640),
      builder: (_) => Consumer(
        builder: (context, menuRef, _) {
          final downloads = menuRef.watch(guidelineDownloadsControllerProvider);
          return DocumentActionsSheet(
            title: content.publication.title,
            kindLabel: presentation.name,
            version: content.publication.version,
            showReadingTools:
                widget.readerOnly && _readerKey.currentState != null,
            showSearch:
                !presentation.isFile &&
                !presentation.isLink &&
                content.blocks.isNotEmpty,
            showAi:
                !presentation.isFile &&
                !presentation.isLink &&
                content.blocks.isNotEmpty,
            showDownload:
                !presentation.isLink &&
                (content.manifest.hasOfflinePackage ||
                    content.manifest.hasOriginalPdf),
            hasOriginal: content.manifest.hasOriginalPdf,
            isLink: presentation.isLink,
            websiteAvailable: presentation.hasValidWebsite,
            websiteHost:
                Uri.tryParse(content.publication.externalUrl)?.host ?? '',
            isBookmarked: progress?.isBookmarked == true,
            hasNotes: progress?.notes.trim().isNotEmpty == true,
            isSignedIn:
                ref.read(authControllerProvider).valueOrNull?.user != null,
            offlineDownload: downloads.hasValue
                ? matchingDownload(downloads.valueOrNull)
                : offlineDownload,
          );
        },
      ),
    );
    if (!context.mounted || action == null) return;
    switch (action) {
      case DocumentMenuAction.contents:
        await _readerKey.currentState?.openContents();
      case DocumentMenuAction.settings:
        _readerKey.currentState?.openSettings();
      case DocumentMenuAction.search:
        await _searchWithin(context, content);
      case DocumentMenuAction.askAi:
        _openAiAssistant(context, content);
      case DocumentMenuAction.notes:
        await _editNotes(context, progress?.notes ?? '');
      case DocumentMenuAction.bookmark:
        await _toggleBookmark(context);
      case DocumentMenuAction.collection:
        await _saveToCollection(context);
      case DocumentMenuAction.copyLink:
        await _copyLink(context);
      case DocumentMenuAction.original:
        await _openOriginal(context);
      case DocumentMenuAction.download:
        if (matchingDownload(
              ref.read(guidelineDownloadsControllerProvider).valueOrNull,
            )?.status ==
            OfflineDownloadStatus.ready) {
          context.push(AppRoutes.offlineContent);
        } else {
          await _download(context, content);
        }
      case DocumentMenuAction.page:
        break;
    }
  }

  // ===========================================================================
  // COLLECTIONS
  // ===========================================================================

  Future<void> _saveToCollection(BuildContext context) async {
    final user = ref.read(authControllerProvider).valueOrNull?.user;
    if (user == null) {
      _requireSignIn(context, 'Sign in to save guidelines to collections.');
      return;
    }

    final result = await showSaveToCollectionSheet(
      context,
      userId: user.id,
      guidelineId: widget.guidelineId,
    );
    if (result != null && context.mounted) {
      if (result.alreadyPresent) {
        AppMessage.info(
          context,
          CollectionMessages.guidelineAlreadySaved(result.collectionName),
        );
      } else {
        AppMessage.success(
          context,
          CollectionMessages.guidelineSaved(result.collectionName),
        );
      }
    }
  }

  // ===========================================================================
  // OVERVIEW
  // ===========================================================================

  Widget _overviewPage(
    BuildContext context,
    GuidelinePublicationContent value,
    ReadingProgress? progress, {
    bool contentLoading = false,
    bool contentUnavailable = false,
  }) {
    return _GuidelineOverview(
      key: ValueKey(value.publication.id),
      contentLoading: contentLoading,
      contentUnavailable: contentUnavailable,
      onRetryContent: () =>
          ref.invalidate(publicationGuidelineProvider(widget.guidelineId)),
      onTable: (id) => context.push(
        AppRoutes.publicGuidelineTableView(widget.guidelineId, id),
      ),
      content: value,
      currentSectionId: progress?.currentSection,
      readingProgress: progress?.progressPercentage,
      onSection: (sectionId) {
        context.push(
          '${AppRoutes.readPublicGuideline(widget.guidelineId)}'
          '?section=${Uri.encodeQueryComponent(sectionId)}',
        );
      },
      onOriginal: () {
        _openOriginal(context);
      },
    );
  }

  // ===========================================================================
  // DOWNLOAD
  // ===========================================================================

  Future<void> _download(
    BuildContext context,
    GuidelinePublicationContent content,
  ) async {
    final original =
        !content.manifest.hasOfflinePackage && content.manifest.hasOriginalPdf;

    if (!content.manifest.hasOfflinePackage &&
        !content.manifest.hasOriginalPdf) {
      AppMessage.warning(context, 'This guideline has no downloadable asset.');

      return;
    }

    try {
      final result = await ref
          .read(guidelineDownloadsControllerProvider.notifier)
          .download(content, originalDocument: original);

      if (!context.mounted) {
        return;
      }

      final message = result.status == OfflineDownloadStatus.ready
          ? 'Verified offline copy is ready.'
          : 'Download ${result.status.name}.';

      if (result.status == OfflineDownloadStatus.ready) {
        AppMessage.success(context, message);
      } else {
        AppMessage.info(context, message);
      }
    } catch (_) {
      if (!context.mounted) {
        return;
      }

      AppMessage.error(
        context,
        'The offline copy could not be downloaded. '
        'Check your connection and try again.',
      );
    }
  }

  // ===========================================================================
  // AI
  // ===========================================================================

  void _openAiAssistant(
    BuildContext context,
    GuidelinePublicationContent content,
  ) {
    final publication = content.publication;

    final sectionId = _currentSectionId ?? _selectedSectionId;

    final selectedSection = sectionId == null
        ? null
        : content.sections
              .where((section) => section.id == sectionId)
              .firstOrNull;

    final relevantBlocks = selectedSection == null
        ? content.blocks
        : content.blocksFor(selectedSection.id);

    final referenceContent = <String>[
      publication.description,

      if (publication.sourceOrganization.isNotEmpty)
        'Source: ${publication.sourceOrganization}',

      if (publication.version.isNotEmpty) 'Version: ${publication.version}',

      if (selectedSection != null) 'Current section: ${selectedSection.title}',

      ...relevantBlocks.map(_searchableBlockText),
    ].where((value) => value.trim().isNotEmpty).join('\n\n');

    final aiContext = AiContext.guideline(
      title: selectedSection == null
          ? publication.title
          : '${publication.title} — ${selectedSection.title}',
      content: referenceContent.isEmpty
          ? 'Use approved MediGuide sources and cite the supporting guideline.'
          : referenceContent,
      guidelineId: widget.guidelineId,
      metadata: <String, dynamic>{
        'guideline_id': widget.guidelineId,
        'program_area': publication.programArea,
        'country': publication.country,
        'version': publication.version,
        if (selectedSection != null) 'section_id': selectedSection.id,
        'reviewed_content':
            content.manifest.recommendedMode == GuidelineReaderMode.structured,
      },
    );

    context.push(
      AppRoutes.aiAssistant,
      extra: <String, dynamic>{'aiContext': aiContext.toJson()},
    );
  }

  // ===========================================================================
  // READER
  // ===========================================================================

  Widget _content(BuildContext context, GuidelinePublicationContent value) {
    final mode = value.manifest.recommendedMode;
    final kind = DocumentReaderKind.forPublication(value.publication);

    if (kind == DocumentReaderKind.uploaded ||
        mode == GuidelineReaderMode.originalDocument) {
      final asset = ref
          .watch(guidelineOriginalDocumentProvider(widget.guidelineId))
          .valueOrNull;

      return _OriginalDocumentReader(
        publication: value.publication,
        manifest: value.manifest,
        asset: asset,
        onOpen: () {
          _openOriginal(context);
        },
      );
    }

    final sections = value.readerSections;

    _scheduleInitialProgress(sections);
    final identity = '${value.publication.id}:${value.manifest.checksum}';
    if (_readerIdentity != identity) {
      _readerIdentity = identity;
      _readerKey = GlobalKey<DocumentKindReaderState>();
    }
    return DocumentKindReader(
      key: _readerKey,
      content: value,
      kind: kind,
      initialSectionId: _selectedSectionId,
      onSection: (id) {
        setState(() {
          _selectedSectionId = id;
          _currentSectionId = id;
        });
        unawaited(_recordSectionProgress(sections, id));
      },
      onOpenOriginal: () => _openOriginal(context),
      onOpenSourcePage: (page) => _openOriginal(context, page: page),
    );
  }

  // ===========================================================================
  // READING PROGRESS
  // ===========================================================================

  void _scheduleInitialProgress(List<PublicationSection> sections) {
    if (_initialProgressScheduled || sections.isEmpty) {
      return;
    }

    final user = ref.read(authControllerProvider).valueOrNull?.user;

    if (user == null) {
      return;
    }

    final requested = _selectedSectionId;

    final sectionId =
        requested != null && sections.any((section) => section.id == requested)
        ? requested
        : sections.first.id;

    _initialProgressScheduled = true;

    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) {
        return;
      }

      unawaited(_recordSectionProgress(sections, sectionId));
    });
  }

  Future<void> _recordSectionProgress(
    List<PublicationSection> sections,
    String sectionId,
  ) async {
    final user = ref.read(authControllerProvider).valueOrNull?.user;

    if (user == null || sections.isEmpty) {
      return;
    }

    final index = sections.indexWhere((section) => section.id == sectionId);

    final progress = index < 0 ? 0.0 : (index + 1) / sections.length;

    await ref
        .read(readingProgressRepositoryProvider)
        .upsert(user.id, widget.guidelineId, {
          'current_section': sectionId,
          'total_sections': sections.length,
          'progress_percentage': progress.clamp(0.0, 1.0),
          'last_read_at': DateTime.now().toUtc().toIso8601String(),
        });

    if (!mounted) {
      return;
    }
    ref.invalidate(publicationReadingProgressProvider(widget.guidelineId));
  }

  // ===========================================================================
  // ORIGINAL DOCUMENT
  // ===========================================================================

  Future<void> _openOriginal(BuildContext context, {int? page}) async {
    final publication =
        (ref
                    .read(
                      publicationGuidelineSummaryProvider(widget.guidelineId),
                    )
                    .valueOrNull ??
                ref
                    .read(publicationGuidelineProvider(widget.guidelineId))
                    .valueOrNull)
            ?.publication;
    if (publication != null && publication.opensWebsite) {
      await _openWebsite(context, publication.externalUrl);
      return;
    }
    try {
      final asset = await ref.read(
        guidelineOriginalDocumentProvider(widget.guidelineId).future,
      );

      if (!context.mounted) {
        return;
      }

      if (asset == null || asset.url.isEmpty) {
        AppMessage.warning(context, 'Original document is unavailable.');

        return;
      }

      context.push(
        AppRoutes.documentReader,
        extra: DocumentReaderArgs(
          title: asset.originalFilename.isEmpty
              ? 'Original guideline'
              : asset.originalFilename,
          source: asset.url,
          initialPage: page,
        ),
      );
    } catch (_) {
      if (!context.mounted) {
        return;
      }

      AppMessage.error(
        context,
        'The original document could not be opened. '
        'Check your connection and try again.',
      );
    }
  }

  Future<void> _openWebsite(BuildContext context, String url) async {
    final uri = Uri.tryParse(url.trim());
    if (uri == null || uri.scheme != 'https') {
      AppMessage.warning(context, 'This link is unavailable.');
      return;
    }
    try {
      final launched = await launchUrl(
        uri,
        mode: LaunchMode.externalApplication,
      );
      if (!launched && context.mounted) {
        AppMessage.error(context, 'The website could not be opened.');
      }
    } catch (_) {
      if (context.mounted) {
        AppMessage.error(context, 'The website could not be opened.');
      }
    }
  }

  // ===========================================================================
  // BOOKMARK
  // ===========================================================================

  Future<void> _toggleBookmark(BuildContext context) async {
    final user = ref.read(authControllerProvider).valueOrNull?.user;

    if (user == null) {
      _requireSignIn(context, 'Sign in to save bookmarks and sync them.');

      return;
    }

    final provider = publicationReadingProgressProvider(widget.guidelineId);

    final current = ref.read(provider).valueOrNull;
    final willBookmark = !(current?.isBookmarked ?? false);

    try {
      await ref.read(readingProgressRepositoryProvider).upsert(
        user.id,
        widget.guidelineId,
        {'is_bookmarked': willBookmark},
      );

      ref.invalidate(provider);
      if (context.mounted) {
        AppMessage.success(
          context,
          willBookmark ? 'Guideline bookmarked.' : 'Bookmark removed.',
        );
      }
    } catch (_) {
      if (context.mounted) {
        AppMessage.error(
          context,
          'The bookmark could not be updated. Please try again.',
        );
      }
    }
  }

  // ===========================================================================
  // NOTES
  // ===========================================================================

  Future<void> _editNotes(BuildContext context, String current) async {
    final user = ref.read(authControllerProvider).valueOrNull?.user;

    if (user == null) {
      _requireSignIn(context, 'Sign in to create private reading notes.');

      return;
    }

    final note = await showModalBottomSheet<String>(
      context: context,
      isScrollControlled: true,
      showDragHandle: true,
      useSafeArea: true,
      builder: (_) {
        return _ReadingNotesSheet(initialValue: current);
      },
    );

    if (note == null || !mounted) {
      return;
    }

    final normalizedNote = note.trim();

    try {
      await ref.read(readingProgressRepositoryProvider).upsert(
        user.id,
        widget.guidelineId,
        {'notes': normalizedNote},
      );

      ref.invalidate(publicationReadingProgressProvider(widget.guidelineId));
      if (context.mounted) {
        AppMessage.success(
          context,
          normalizedNote.isEmpty
              ? 'Reading note removed.'
              : 'Reading note saved.',
        );
      }
    } catch (_) {
      if (context.mounted) {
        AppMessage.error(
          context,
          'The reading note could not be saved. Please try again.',
        );
      }
    }
  }

  // ===========================================================================
  // SHARE
  // ===========================================================================

  Future<void> _copyLink(BuildContext context) async {
    final link = AppRoutes.publicGuideline(widget.guidelineId);

    try {
      await Clipboard.setData(ClipboardData(text: link));
      if (context.mounted) {
        AppMessage.success(context, 'Document link copied.');
      }
    } catch (_) {
      if (context.mounted) {
        AppMessage.error(context, 'The document link could not be copied.');
      }
    }
  }

  // ===========================================================================
  // AUTH
  // ===========================================================================

  void _requireSignIn(BuildContext context, String message) {
    AppMessage.warning(context, message);

    final destination = Uri.encodeComponent(
      AppRoutes.publicGuideline(widget.guidelineId),
    );

    context.push(
      '${AppRoutes.login}'
      '?redirect=$destination',
    );
  }

  // ===========================================================================
  // SEARCH WITHIN
  // ===========================================================================

  Future<void> _searchWithin(
    BuildContext context,
    GuidelinePublicationContent content,
  ) async {
    final sectionId = await showSearch<String?>(
      context: context,
      delegate: _GuidelineContentSearchDelegate(content),
    );

    if (!mounted || sectionId == null) {
      return;
    }

    setState(() {
      _selectedSectionId = sectionId;
    });
  }
}

// =============================================================================
// MENU ACTIONS
// =============================================================================

class _FollowGuidelineButton extends ConsumerWidget {
  const _FollowGuidelineButton({required this.guidelineId});
  final String guidelineId;
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state = ref.watch(notificationPreferencesControllerProvider);
    final preferences = state.valueOrNull?.preferences;
    final following =
        preferences?.followedGuidelineIds.contains(guidelineId) ?? false;
    return IconButton(
      tooltip: following
          ? 'Unfollow guideline updates'
          : 'Follow guideline updates',
      icon: Icon(
        following ? Icons.notifications_active : Icons.notifications_none,
      ),
      onPressed: preferences == null || state.isLoading
          ? null
          : () async {
              final ids = preferences.followedGuidelineIds.toSet();
              if (following) {
                ids.remove(guidelineId);
              } else {
                ids.add(guidelineId);
              }
              await ref
                  .read(notificationPreferencesControllerProvider.notifier)
                  .updatePreferences({
                    'interests': {
                      ...preferences.interestsJson,
                      'followed_guideline_ids': ids.toList(),
                    },
                  });
              if (!context.mounted) return;
              if (ref
                  .read(notificationPreferencesControllerProvider)
                  .hasError) {
                AppMessage.error(
                  context,
                  'Unable to change followed guidelines. Try again when connected.',
                );
              } else {
                AppMessage.success(
                  context,
                  following
                      ? 'Guideline unfollowed.'
                      : 'Following guideline updates. Notification category and channel preferences still apply.',
                );
              }
            },
    );
  }
}
