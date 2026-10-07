import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:responsive_framework/responsive_framework.dart';
import 'package:user_app/app/providers/app_providers.dart';
import 'package:user_app/app/theme/app_theme.dart';
import 'package:user_app/core/network/api_client.dart';
import 'package:user_app/features/downloads/data/models/offline_download.dart';
import 'package:user_app/features/downloads/presentation/controllers/guideline_downloads_controller.dart';
import 'package:user_app/features/guidelines/data/models/guideline_publication.dart';
import 'package:user_app/features/guidelines/data/repositories/guideline_publication_repository.dart';
import 'package:user_app/features/guidelines/data/repositories/progress_usage_repository.dart';
import 'package:user_app/features/guidelines/presentation/controllers/publication_guideline_controller.dart';
import 'package:user_app/features/guidelines/presentation/screens/publication_guideline_page.dart';

import 'helpers/test_local_store.dart';

final class _NoDownloads extends GuidelineDownloadsController {
  @override
  Future<List<OfflineDownload>> build() async => const [];
}

const _link = GuidelinePublication(
  id: 'ucg',
  title: 'UCG',
  sourceOrganization: 'MOH',
  publicationDate: '2026-10-06',
  version: '2026.01',
  language: 'en',
  documentKind: PublicDocumentKind(
    slug: 'link',
    name: 'Link',
    publishAsLink: true,
  ),
  externalUrl: 'https://who.int',
);

// Serves UCG, a Link document, as the public API does.
final class _LinkGuidelineApi extends BackendApiService {
  @override
  Future<Map<String, dynamic>> requestJson(
    String path, {
    required String method,
    Map<String, dynamic>? body,
    Map<String, String>? query,
    bool includeAuth = true,
  }) async => {
    'data': path.endsWith('/manifest')
        ? {
            'guideline_id': 'ucg',
            'version_id': 'v1',
            'version': '2026.01',
            'recommended_mode': 'original_document',
            'extraction_quality': 'markdown_fallback',
          }
        : {
            'id': 'ucg',
            'title': 'UCG',
            'source_org': 'MOH',
            'document_kind': {
              'slug': 'link',
              'name': 'Link',
              'publish_as_uploaded': false,
              'publish_as_link': true,
            },
            'external_url': 'https://who.int',
          },
  };
}

void main() {
  test('the repository keeps the kind and URL of a link document', () async {
    final store = TestLocalStore();
    addTearDown(store.close);
    final repository = GuidelinePublicationRepository(
      _LinkGuidelineApi(),
      store.cache,
    );

    final summary = await repository.summary('ucg');

    expect(summary.publication.opensWebsite, isTrue);
    expect(summary.publication.externalUrl, 'https://who.int');
    expect(summary.publication.documentKind?.name, 'Link');
  });

  test('a document published as a link opens a website', () {
    final publication = GuidelinePublication.fromJson({
      'id': 'ucg',
      'title': 'UCG',
      'document_kind': {
        'slug': 'link',
        'name': 'Link',
        'publish_as_uploaded': false,
        'publish_as_link': true,
      },
      'external_url': 'https://who.int',
    });
    expect(publication.opensWebsite, isTrue);
    expect(publication.externalUrl, 'https://who.int');
    // Other kinds, and links without a URL, are read in the app.
    expect(
      GuidelinePublication.fromJson({
        'id': 'g1',
        'title': 'Guideline',
      }).opensWebsite,
      isFalse,
    );
    expect(_link.copyWith(externalUrl: '').opensWebsite, isFalse);
  });

  testWidgets('a link document offers its website instead of a reader', (
    tester,
  ) async {
    final store = TestLocalStore();
    addTearDown(store.close);
    tester.view.physicalSize = const Size(430, 1400);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final content = GuidelinePublicationContent(
      publication: _link,
      manifest: const GuidelineManifest(guidelineId: 'ucg', versionId: 'v1'),
      sections: const [],
      blocks: const [],
    );
    final router = GoRouter(
      routes: [
        GoRoute(
          path: '/',
          builder: (_, _) => const PublicationGuidelinePage(guidelineId: 'ucg'),
        ),
      ],
    );
    addTearDown(router.dispose);

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          publicationGuidelineProvider(
            'ucg',
          ).overrideWith((_) async => content),
          publicationGuidelineSummaryProvider(
            'ucg',
          ).overrideWith((_) async => content),
          publicationReadingProgressProvider(
            'ucg',
          ).overrideWith((_) async => null),
          guidelineDownloadsControllerProvider.overrideWith(_NoDownloads.new),
          usageRepositoryProvider.overrideWithValue(
            UsageRepository(BackendApiService(), store.cache, () => null),
          ),
        ],
        child: MaterialApp.router(
          theme: AppTheme.light,
          routerConfig: router,
          builder: (context, child) => ResponsiveBreakpoints.builder(
            breakpoints: const [
              Breakpoint(start: 0, end: 450, name: MOBILE),
              Breakpoint(start: 451, end: 900, name: TABLET),
              Breakpoint(start: 901, end: double.infinity, name: DESKTOP),
            ],
            child: child!,
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('External website'), findsOneWidget);
    expect(find.text('who.int'), findsOneWidget);
    expect(find.text('Open website'), findsOneWidget);
    expect(find.text('Language'), findsOneWidget);
    // There are no chapters to read or a file to download.
    expect(find.text('Read guideline'), findsNothing);
    expect(find.byTooltip('Download for offline use'), findsNothing);
  });
}
