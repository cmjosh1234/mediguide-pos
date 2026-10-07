import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:responsive_framework/responsive_framework.dart';
import 'package:user_app/app/theme/app_theme.dart';
import 'package:user_app/features/outbreaks/data/models/outbreak_models.dart';
import 'package:user_app/features/outbreaks/presentation/screens/outbreak_screens.dart';

Future<void> _pumpReport(
  WidgetTester tester,
  PublicSituationReport report,
) async {
  tester.view.physicalSize = const Size(800, 3000);
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);
  final router = GoRouter(
    routes: [
      GoRoute(
        path: '/',
        builder: (_, _) => SituationReportDetailPage(reportId: report.id),
      ),
    ],
  );
  addTearDown(router.dispose);
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        publicSituationReportProvider(report.id).overrideWith(
          (_) async => PublicContent(
            value: report,
            cache: const PublicCacheMetadata.online(),
          ),
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
}

void main() {
  test('a situation report parses its attachments', () {
    final report = PublicSituationReport.fromJson({
      'id': 'report-1',
      'title': 'Week 12',
      'attachments': [
        {
          'id': 'attachment-1',
          'title': 'Full report',
          'resource_type': 'guideline',
          'target_type': 'guideline',
          'url': '/public/guidelines/doc-1',
        },
      ],
    });

    final attachment = report.attachments.single;
    expect(attachment.title, 'Full report');
    expect(attachment.resourceType, 'guideline');
    expect(attachment.url, '/public/guidelines/doc-1');
    // Attachments belong to a report, not an outbreak.
    expect(attachment.outbreakId, '');
    expect(
      PublicSituationReport.fromJson({'id': 'report-2'}).attachments,
      isEmpty,
    );
  });

  testWidgets('a report lists its attached documents', (tester) async {
    await _pumpReport(
      tester,
      const PublicSituationReport(
        id: 'report-1',
        title: 'Week 12',
        reportAssetUrl: '/api/public/guidelines/doc-1/original/download',
        attachments: [
          PublicOutbreakResource(
            id: 'attachment-1',
            title: 'Full report',
            resourceType: 'guideline',
            url: '/public/guidelines/doc-1',
          ),
          PublicOutbreakResource(
            id: 'attachment-2',
            title: 'District annex',
            resourceType: 'guideline',
            url: '/public/guidelines/doc-2',
          ),
        ],
      ),
    );

    expect(find.text('Report documents'), findsOneWidget);
    expect(find.text('Full report'), findsOneWidget);
    expect(find.text('District annex'), findsOneWidget);
    // The fallback link for older app versions isn't offered twice.
    expect(find.text('Read full report'), findsNothing);
  });

  testWidgets('an older report still opens its uploaded PDF', (tester) async {
    await _pumpReport(
      tester,
      const PublicSituationReport(
        id: 'report-2',
        title: 'Week 11',
        reportAssetUrl: '/api/public/situation-reports/report-2/asset',
      ),
    );

    expect(find.text('Read full report'), findsOneWidget);
    expect(find.text('Report documents'), findsNothing);
  });
}
