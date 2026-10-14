import 'package:flutter/material.dart';
import 'package:user_app/features/guidelines/data/models/guideline_publication.dart';
import 'document_reader_kind.dart';

/// User-facing overview language follows the publication kind and delivery mode.
class DocumentOverviewPresentation {
  DocumentOverviewPresentation(this.content);
  final GuidelinePublicationContent content;
  GuidelinePublication get publication => content.publication;
  DocumentReaderKind get kind => DocumentReaderKind.forPublication(publication);
  String get slug => publication.documentKind?.slug ?? 'guideline';
  String get name => publication.documentKind?.name.trim().isNotEmpty == true
      ? publication.documentKind!.name.trim()
      : 'Guideline';
  bool get isLink => kind == DocumentReaderKind.link;
  bool get isFile =>
      !isLink &&
      (kind == DocumentReaderKind.uploaded ||
          content.manifest.recommendedMode ==
              GuidelineReaderMode.originalDocument);
  bool get hasValidWebsite {
    final uri = Uri.tryParse(publication.externalUrl.trim());
    return uri?.scheme == 'https' && uri!.host.isNotEmpty;
  }

  bool get canOpen => isLink
      ? hasValidWebsite
      : isFile
      ? content.manifest.hasOriginalPdf
      : content.readerSections.isNotEmpty || content.manifest.hasChapters;
  IconData get icon => switch (kind) {
    DocumentReaderKind.link => Icons.language_outlined,
    DocumentReaderKind.uploaded => Icons.description_outlined,
    DocumentReaderKind.procedure => Icons.route_outlined,
    DocumentReaderKind.checklist => Icons.checklist_outlined,
    DocumentReaderKind.caseDefinition => Icons.fact_check_outlined,
    DocumentReaderKind.training => Icons.school_outlined,
    DocumentReaderKind.reference =>
      slug == 'framework' ? Icons.layers_outlined : Icons.article_outlined,
    _ => Icons.menu_book_outlined,
  };
  String get actionLabel {
    if (isLink) return 'Open website';
    if (isFile) {
      return kind == DocumentReaderKind.uploaded
          ? (slug == 'form' ? 'Open form' : 'Open file')
          : 'Open original document';
    }
    return switch (kind) {
      DocumentReaderKind.procedure => 'View procedure',
      DocumentReaderKind.checklist => 'View checklist',
      DocumentReaderKind.caseDefinition => 'View definitions',
      DocumentReaderKind.training => 'Start reading',
      _ =>
        slug == 'framework'
            ? 'Explore framework'
            : 'Read ${name.toLowerCase()}',
    };
  }

  String get contentsLabel => switch (kind) {
    DocumentReaderKind.procedure => 'Sections',
    DocumentReaderKind.checklist => 'Checklist sections',
    DocumentReaderKind.caseDefinition => 'Definitions',
    DocumentReaderKind.training => 'Topics',
    DocumentReaderKind.reference =>
      slug == 'framework' ? 'Framework areas' : 'Sections',
    _ => 'Chapters',
  };
  String get introduction {
    if (isLink) {
      return 'This resource opens in your browser. Its content is maintained by the publishing website.';
    }
    if (isFile) {
      return 'View the original document with its formatting and page layout preserved.';
    }
    return switch (kind) {
      DocumentReaderKind.procedure =>
        'Browse the published procedure by section, or read it from the beginning.',
      DocumentReaderKind.checklist =>
        'Review each section and keep track of your reading during this session.',
      DocumentReaderKind.caseDefinition =>
        'Compare the published definitions in separate, easy-to-scan sections.',
      DocumentReaderKind.training =>
        'Work through the publication topic by topic at your own pace.',
      DocumentReaderKind.reference =>
        slug == 'framework'
            ? 'Explore the framework’s principles, implementation areas and published reference material.'
            : 'Read the publication or jump directly to a section.',
      _ =>
        'Read at your own pace, browse chapters and return to where you left off.',
    };
  }
}
