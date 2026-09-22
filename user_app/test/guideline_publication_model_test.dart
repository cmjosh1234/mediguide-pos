import 'package:flutter_test/flutter_test.dart';
import 'package:user_app/features/guidelines/data/models/guideline_publication.dart';
import 'package:user_app/features/guidelines/presentation/screens/publication_guideline_page.dart';

void main() {
  group('Guideline publication models', () {
    test('manifest uses the server-provided reader mode', () {
      final manifest = GuidelineManifest.fromJson(const {
        'guideline_id': 'guideline-1',
        'version_id': 'version-1',
        'recommended_mode': 'partial',
      });

      expect(manifest.recommendedMode, GuidelineReaderMode.partial);
      expect(manifest.reviewedSectionCount, 0);
      expect(manifest.emptyLeafSectionCount, 0);
    });

    test('manifest exposes schema 2 completeness counts', () {
      final manifest = GuidelineManifest.fromJson(const {
        'guideline_id': 'guideline-1',
        'version_id': 'version-1',
        'schema_version': 2,
        'section_count': 310,
        'reviewed_section_count': 15,
        'leaf_section_count': 220,
        'reviewed_leaf_section_count': 12,
        'empty_leaf_section_count': 208,
        'block_count': 16,
        'reviewed_paragraph_count': 0,
      });

      expect(manifest.sectionCount, 310);
      expect(manifest.reviewedSectionCount, 15);
      expect(manifest.leafSectionCount, 220);
      expect(manifest.reviewedLeafSectionCount, 12);
      expect(manifest.emptyLeafSectionCount, 208);
      expect(manifest.blockCount, 16);
      expect(manifest.reviewedParagraphCount, 0);
    });

    test('unknown future block types use the safe fallback', () {
      final block = GuidelineBlock.fromJson(const {
        'kind': 'future_clinical_widget',
        'id': 'block-1',
        'sortOrder': 3,
        'rawType': 'future_clinical_widget',
        'raw': {'value': 'not rendered as clinical guidance'},
      });

      expect(block, isA<UnknownGuidelineBlock>());
      final unknown = block as UnknownGuidelineBlock;
      expect(unknown.rawType, 'future_clinical_widget');
      expect(unknown.raw['value'], 'not rendered as clinical guidance');
    });

    test('section blocks are returned in deterministic order', () {
      final content = GuidelinePublicationContent(
        publication: const GuidelinePublication(id: 'guideline-1'),
        manifest: const GuidelineManifest(
          guidelineId: 'guideline-1',
          versionId: 'version-1',
        ),
        sections: const [],
        blocks: const [
          GuidelineBlock.paragraph(
            id: 'second',
            sectionId: 'section-1',
            sortOrder: 2,
            text: 'Second',
          ),
          GuidelineBlock.paragraph(
            id: 'first',
            sectionId: 'section-1',
            sortOrder: 1,
            text: 'First',
          ),
        ],
      );

      expect(content.blocksFor('section-1').map((block) => block.id), [
        'first',
        'second',
      ]);
    });

    test(
      'reader removes only the leading heading represented by its section',
      () {
        const section = PublicationSection(
          id: 'section-1',
          title: 'Clinical assessment',
          level: 2,
        );
        final content = GuidelinePublicationContent(
          publication: const GuidelinePublication(id: 'guideline-1'),
          manifest: const GuidelineManifest(
            guidelineId: 'guideline-1',
            versionId: 'version-1',
          ),
          sections: const [section],
          blocks: const [
            GuidelineBlock.heading(
              id: 'section-heading',
              sectionId: 'section-1',
              sortOrder: 0,
              text: 'Clinical assessment',
              level: 2,
            ),
            GuidelineBlock.paragraph(
              id: 'body',
              sectionId: 'section-1',
              sortOrder: 1,
              text: 'Reviewed body.',
            ),
            GuidelineBlock.heading(
              id: 'nested-heading',
              sectionId: 'section-1',
              sortOrder: 2,
              text: 'Clinical assessment',
              level: 3,
            ),
          ],
        );

        expect(content.displayBlocksFor(section).map((block) => block.id), [
          'body',
          'nested-heading',
        ]);
      },
    );

    test(
      'reader removes an empty H1 document wrapper but keeps its chapters',
      () {
        const title = PublicationSection(
          id: 'title',
          title: 'Diabetes Guideline 2026',
          level: 1,
        );
        const chapter = PublicationSection(
          id: 'chapter-1',
          parentId: 'title',
          title: 'Introduction',
          level: 2,
        );
        final content = GuidelinePublicationContent(
          publication: const GuidelinePublication(
            id: 'guideline-1',
            title: 'Diabetes Guideline 2026',
          ),
          manifest: const GuidelineManifest(
            guidelineId: 'guideline-1',
            versionId: 'version-1',
          ),
          sections: const [title, chapter],
          blocks: const [
            GuidelineBlock.heading(
              id: 'title-heading',
              sectionId: 'title',
              sortOrder: 0,
              text: '# Diabetes Guideline 2026',
              level: 1,
            ),
          ],
        );

        expect(content.isDocumentTitleWrapper(title), isTrue);
        expect(content.readerSections, [chapter]);
      },
    );

    test('chapter cards promote children of a single document-title root', () {
      const title = PublicationSection(
        id: 'title',
        title: 'Diabetes',
        level: 1,
      );
      const chapters = [
        PublicationSection(
          id: 'chapter-1',
          parentId: 'title',
          title: 'Chapter 1',
          level: 2,
        ),
        PublicationSection(
          id: 'chapter-2',
          parentId: 'title',
          title: 'Chapter 2',
          level: 2,
        ),
      ];

      expect(
        guidelineChapterDisplayRoots(const [title], const {'title': chapters}),
        chapters,
      );
    });

    test('chapter cards preserve curated direct-root chapters', () {
      const roots = [
        PublicationSection(id: 'overview', title: '1. Overview', level: 1),
        PublicationSection(id: 'treatment', title: '2. Treatment', level: 1),
      ];

      expect(guidelineChapterDisplayRoots(roots, const {}), roots);
    });
  });
}
