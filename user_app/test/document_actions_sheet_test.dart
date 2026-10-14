import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:user_app/app/theme/app_theme.dart';
import 'package:user_app/features/documents/presentation/widgets/document_actions_sheet.dart';
import 'package:user_app/features/downloads/data/models/offline_download.dart';

void main() {
  WidgetController.hitTestWarningShouldBeFatal = true;
  const download = OfflineDownload(
    id: 'download',
    guidelineId: 'doc',
    assetType: 'original_pdf',
    scope: 'public',
  );

  Future<void> pumpMenu(
    WidgetTester tester,
    DocumentActionsSheet menu, {
    double scale = 1,
  }) async {
    tester.view.physicalSize = const Size(320, 800);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.dark,
        builder: (context, child) => MediaQuery(
          data: MediaQuery.of(
            context,
          ).copyWith(textScaler: TextScaler.linear(scale)),
          child: child!,
        ),
        home: Scaffold(
          body: Builder(
            builder: (context) => TextButton(
              onPressed: () => showModalBottomSheet<DocumentMenuAction>(
                context: context,
                isScrollControlled: true,
                useSafeArea: true,
                showDragHandle: true,
                builder: (_) => menu,
              ),
              child: const Text('Open menu'),
            ),
          ),
        ),
      ),
    );
    await tester.tap(find.text('Open menu'));
    await tester.pumpAndSettle();
  }

  testWidgets(
    'reading menu has grouped tools and a visible close button at large text',
    (tester) async {
      await pumpMenu(
        tester,
        const DocumentActionsSheet(
          title:
              'A long publication title with several words to verify text wrapping',
          kindLabel: 'Manual',
          version: '2026.01',
          showReadingTools: true,
          showSearch: true,
          showAi: true,
          hasOriginal: true,
          showDownload: true,
        ),
        scale: 2,
      );
      expect(find.text('Document actions'), findsOneWidget);
      expect(find.text('Reading tools'), findsOneWidget);
      expect(
        find.byTooltip('Close document actions').hitTestable(),
        findsOneWidget,
      );
      await tester.scrollUntilVisible(
        find.text('Links and source'),
        250,
        scrollable: find.byType(Scrollable).last,
      );
      expect(find.text('Copy document link'), findsOneWidget);
      expect(find.text('Share document'), findsNothing);
      expect(tester.takeException(), isNull);
      await tester.tap(find.byTooltip('Close document actions'));
      await tester.pumpAndSettle();
      expect(find.text('Document actions'), findsNothing);
    },
  );

  for (final status in OfflineDownloadStatus.values) {
    testWidgets(
      'offline status ${status.name} shows the right download action',
      (tester) async {
        await pumpMenu(
          tester,
          DocumentActionsSheet(
            title: 'Form',
            kindLabel: 'Form',
            showLibraryActions: false,
            showCopyLink: false,
            showDownload: true,
            offlineDownload: download.copyWith(status: status, progress: 0.4),
          ),
        );
        final tile = tester.widget<ListTile>(
          find.byKey(const ValueKey(DocumentMenuAction.download)),
        );
        final inProgress =
            status == OfflineDownloadStatus.queued ||
            status == OfflineDownloadStatus.downloading;
        expect(tile.onTap == null, inProgress);
        if (inProgress) expect(find.text('40% downloaded'), findsOneWidget);
        if (status == OfflineDownloadStatus.ready) {
          expect(find.text('Manage offline copy'), findsOneWidget);
        }
        if (status == OfflineDownloadStatus.updateAvailable) {
          expect(find.text('Update offline copy'), findsOneWidget);
        }
        if (status == OfflineDownloadStatus.failed ||
            status == OfflineDownloadStatus.corrupted) {
          expect(find.text('Retry download'), findsOneWidget);
        }
        expect(tester.takeException(), isNull);
      },
    );
  }

  testWidgets(
    'link and file menus omit irrelevant reading and download tools',
    (tester) async {
      await pumpMenu(
        tester,
        const DocumentActionsSheet(
          title: 'External resource',
          kindLabel: 'Link',
          isLink: true,
          websiteHost: 'example.org',
          isSignedIn: false,
          websiteAvailable: false,
        ),
      );
      expect(find.text('Reading settings'), findsNothing);
      expect(find.text('Search document'), findsNothing);
      expect(find.text('Ask AI'), findsNothing);
      expect(find.text('Download for offline use'), findsNothing);
      await tester.scrollUntilVisible(
        find.text('Open website'),
        250,
        scrollable: find.byType(Scrollable).last,
      );
      expect(
        tester
            .widget<ListTile>(
              find.byKey(const ValueKey(DocumentMenuAction.original)),
            )
            .onTap,
        isNull,
      );
      expect(find.text('The website link is unavailable'), findsOneWidget);
    },
  );

  testWidgets('PDF menu has page and source options without library actions', (
    tester,
  ) async {
    await pumpMenu(
      tester,
      const DocumentActionsSheet(
        title: 'original.pdf',
        kindLabel: 'PDF document',
        showPagePicker: true,
        hasOriginal: true,
        showLibraryActions: false,
        copyLabel: 'Copy file link',
      ),
    );
    expect(find.text('Go to page'), findsOneWidget);
    expect(find.text('Copy file link'), findsOneWidget);
    expect(find.text('Reading notes'), findsNothing);
    expect(find.text('Bookmark document'), findsNothing);
  });

  testWidgets('selection closes the sheet and returns the chosen action', (
    tester,
  ) async {
    DocumentMenuAction? chosen;
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: Builder(
            builder: (context) => TextButton(
              onPressed: () async {
                chosen = await showModalBottomSheet<DocumentMenuAction>(
                  context: context,
                  isScrollControlled: true,
                  builder: (_) => const DocumentActionsSheet(
                    title: 'Source',
                    kindLabel: 'Document',
                    showLibraryActions: false,
                  ),
                );
              },
              child: const Text('Open menu'),
            ),
          ),
        ),
      ),
    );
    await tester.tap(find.text('Open menu'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Copy document link'));
    await tester.pumpAndSettle();
    expect(chosen, DocumentMenuAction.copyLink);
    expect(find.text('Document actions'), findsNothing);
  });
}
