// Simulator-only entry point. Launch with the development flavor; no sample
// publication is created or changed on the backend.
import 'package:drift/native.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:responsive_framework/responsive_framework.dart';
import 'package:user_app/app/providers/app_providers.dart';
import 'package:user_app/app/router/route_names.dart';
import 'package:user_app/app/theme/app_theme.dart';
import 'package:user_app/core/network/api_client.dart';
import 'package:user_app/core/storage/database/app_database.dart';
import 'package:user_app/core/storage/local_cache_service.dart';
import 'package:user_app/features/authentication/presentation/controllers/auth_controller.dart';
import 'package:user_app/features/authentication/presentation/controllers/auth_state.dart';
import 'package:user_app/features/downloads/data/models/offline_download.dart';
import 'package:user_app/features/downloads/presentation/controllers/guideline_downloads_controller.dart';
import 'package:user_app/features/guidelines/data/models/guideline_publication.dart';
import 'package:user_app/features/guidelines/data/repositories/progress_usage_repository.dart';
import 'package:user_app/features/guidelines/presentation/controllers/publication_guideline_controller.dart';
import 'package:user_app/features/guidelines/presentation/screens/publication_guideline_page.dart';
import 'package:user_app/features/guidelines/presentation/utils/document_reader_kind.dart';

const _kinds = <String, String>{
  'guideline': 'Guideline',
  'manual': 'Manual',
  'framework': 'Framework',
  'sop': 'SOP',
  'ipc_protocol': 'IPC Protocol',
  'laboratory_protocol': 'Laboratory Protocol',
  'surveillance_protocol': 'Surveillance Protocol',
  'contact_tracing_guide': 'Contact Tracing Guide',
  'treatment_protocol': 'Treatment Protocol',
  'referral_protocol': 'Referral Protocol',
  'checklist': 'Checklist',
  'case_definition': 'Case Definition',
  'training_material': 'Training Material',
  'communication_material': 'Communication Material',
  'policy': 'Policy',
  'situation_report_attachment': 'Situation Report Attachment',
  'form': 'Form',
  'link': 'Link',
  'custom': 'Custom kind (fallback)',
};

class _Guest extends AuthController {
  @override
  Future<AuthState> build() async => const AuthState.unauthenticated();
}

class _NoDownloads extends GuidelineDownloadsController {
  @override
  Future<List<OfflineDownload>> build() async => const [];
}

GuidelinePublicationContent _sample(String id) {
  final slug = id.replaceFirst('partial-', '');
  final partial = id.startsWith('partial-');
  final name = _kinds[slug] ?? slug;
  final titles = switch (slug) {
    'case_definition' => [
      'Suspected category',
      'Probable category',
      'Confirmed category',
    ],
    'training_material' => ['Introduction', 'Practice activity', 'Review'],
    'checklist' => ['Preparation', 'Review', 'Follow-up'],
    _ => ['Purpose and scope', 'Published procedure', 'References'],
  };
  return GuidelinePublicationContent(
    publication: GuidelinePublication(
      id: id,
      title: 'Sample $name',
      description:
          'Preview data for testing the reader layout. This is not clinical guidance.',
      sourceOrganization: 'Reader preview',
      version: '1.0',
      language: 'en',
      documentKind: PublicDocumentKind(
        slug: slug,
        name: name,
        publishAsUploaded: slug == 'form',
        publishAsLink: slug == 'link',
      ),
      externalUrl: slug == 'link' ? 'https://example.org' : '',
    ),
    manifest: GuidelineManifest(
      guidelineId: id,
      versionId: 'preview-v1',
      version: '1.0',
      checksum: id,
      recommendedMode: partial
          ? GuidelineReaderMode.partial
          : GuidelineReaderMode.structured,
      hasChapters: true,
      sectionCount: 3,
      reviewedSectionCount: partial ? 2 : 3,
      extractionQuality: partial ? 'partially_reviewed' : 'fully_reviewed',
    ),
    sections: [
      for (var i = 0; i < titles.length; i++)
        PublicationSection(id: 'section-$i', title: titles[i], sortOrder: i),
    ],
    blocks: [
      const GuidelineBlock.paragraph(
        id: 'intro',
        sectionId: 'section-0',
        sortOrder: 0,
        text:
            'Use this sample to check text scaling, section navigation, and reading marks.',
      ),
      const GuidelineBlock.orderedList(
        id: 'list',
        sectionId: 'section-1',
        sortOrder: 1,
        items: [
          'Open a publication section.',
          'Read the sample content.',
          'Continue to the next section.',
        ],
      ),
      if (!partial)
        const GuidelineBlock.paragraph(
          id: 'end',
          sectionId: 'section-2',
          sortOrder: 2,
          text:
              'This final section uses the same published-block renderer as the real app.',
        ),
    ],
  );
}

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  final db = AppDatabase.forTesting(NativeDatabase.memory());
  runApp(
    ProviderScope(
      overrides: [
        authControllerProvider.overrideWith(_Guest.new),
        guidelineDownloadsControllerProvider.overrideWith(_NoDownloads.new),
        usageRepositoryProvider.overrideWithValue(
          UsageRepository(
            BackendApiService(),
            LocalCacheService(db),
            () => null,
          ),
        ),
        for (final slug in _kinds.keys)
          for (final id in [slug, 'partial-$slug']) ...[
            publicationGuidelineProvider(
              id,
            ).overrideWith((_) async => _sample(id)),
            publicationGuidelineSummaryProvider(
              id,
            ).overrideWith((_) async => _sample(id)),
            publicationReadingProgressProvider(
              id,
            ).overrideWith((_) async => null),
            guidelineOriginalDocumentProvider(
              id,
            ).overrideWith((_) async => null),
          ],
      ],
      child: const _PreviewApp(),
    ),
  );
}

class _PreviewApp extends StatefulWidget {
  const _PreviewApp();
  @override
  State<_PreviewApp> createState() => _PreviewAppState();
}

class _PreviewAppState extends State<_PreviewApp> {
  bool _dark = false;
  bool _partial = false;
  late final _router = GoRouter(
    routes: [
      GoRoute(
        path: '/',
        builder: (context, _) => StatefulBuilder(
          builder: (context, setGalleryState) => Scaffold(
            appBar: AppBar(
              title: const Text('Document reader previews'),
              actions: [
                IconButton(
                  tooltip: 'Toggle dark mode',
                  icon: Icon(_dark ? Icons.light_mode : Icons.dark_mode),
                  onPressed: () => setState(() => _dark = !_dark),
                ),
              ],
            ),
            body: ListView(
              children: [
                const Padding(
                  padding: EdgeInsets.all(16),
                  child: Text(
                    'Sample documents only. Choose a kind to test its reader.',
                  ),
                ),
                SwitchListTile(
                  title: const Text('Partially reviewed sample'),
                  value: _partial,
                  onChanged: (value) => setGalleryState(() => _partial = value),
                ),
                for (final entry in _kinds.entries)
                  ListTile(
                    title: Text(entry.value),
                    subtitle: Text(
                      DocumentReaderKind.forPublication(
                        _sample(entry.key).publication,
                      ).label,
                    ),
                    trailing: const Icon(Icons.chevron_right),
                    onTap: () => context.push(
                      AppRoutes.readPublicGuideline(
                        '${_partial ? 'partial-' : ''}${entry.key}',
                      ),
                    ),
                  ),
              ],
            ),
          ),
        ),
      ),
      for (final path in [AppRoutes.login, AppRoutes.aiAssistant])
        GoRoute(
          path: path,
          builder: (_, _) => Scaffold(
            appBar: AppBar(title: const Text('Reader preview')),
            body: const Center(
              child: Padding(
                padding: EdgeInsets.all(24),
                child: Text(
                  'This gallery previews document readers. Sign-in, AI, and saved-library actions use the regular development app.',
                ),
              ),
            ),
          ),
        ),
      GoRoute(
        path: AppRoutes.publicGuidelineReader,
        builder: (_, state) => PublicationGuidelinePage(
          guidelineId: state.pathParameters['guidelineId']!,
          readerOnly: true,
        ),
      ),
    ],
  );

  @override
  void dispose() {
    _router.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => MaterialApp.router(
    debugShowCheckedModeBanner: false,
    theme: AppTheme.light,
    darkTheme: AppTheme.dark,
    themeMode: _dark ? ThemeMode.dark : ThemeMode.light,
    routerConfig: _router,
    builder: (context, child) => ResponsiveBreakpoints.builder(
      breakpoints: const [
        Breakpoint(start: 0, end: 450, name: MOBILE),
        Breakpoint(start: 451, end: 900, name: TABLET),
        Breakpoint(start: 901, end: double.infinity, name: DESKTOP),
      ],
      child: child!,
    ),
  );
}
