import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:user_app/features/discovery/presentation/widgets/discovery_widgets.dart';
import 'package:user_app/features/outbreaks/data/models/outbreak_models.dart';
import 'package:user_app/features/outbreaks/presentation/providers/outbreak_providers.dart';
import 'package:user_app/features/outbreaks/presentation/screens/situation_report_detail_page.dart';

void main() {
  testWidgets(
    'outbreak banner shows clinical indicators without database fields',
    (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: OutbreakBanner({
              'title': 'Regional outbreak update',
              'status': 'active',
              'metrics': [
                {
                  'key': 'cases',
                  'label': 'Confirmed cases',
                  'value': '36',
                  'unit': 'cases',
                  'numeric_value': 36,
                  'sort_order': 4,
                  'source_reference': 'WHO situation report 11',
                },
              ],
            }),
          ),
        ),
      );
      expect(find.text('36 cases'), findsOneWidget);
      expect(find.text('Confirmed cases'), findsOneWidget);
      expect(find.textContaining('sort order'), findsNothing);
      expect(find.textContaining('numeric value'), findsNothing);
      expect(find.textContaining('source reference'), findsNothing);
      expect(tester.takeException(), isNull);
    },
  );

  for (final scale in [1.0, 2.0]) {
    testWidgets('report is readable on a small phone at text scale $scale', (
      tester,
    ) async {
      tester.view.physicalSize = const Size(320, 720);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final report = PublicSituationReport(
        id: 'report',
        title: 'Bundibugyo virus disease weekly external situation report 11',
        geographicArea: 'Democratic Republic of the Congo and Uganda',
        sourceOrganization: 'WHO Regional Office for Africa',
        publicationDate: DateTime(2026, 7, 26),
        reportAssetUrl: 'https://example.org/report.pdf',
        summary: 'Regional response activity and reviewed updates. ' * 8,
        metrics: const [
          OutbreakMetric(
            label: 'Confirmed cases in Uganda',
            value: '20',
            unit: 'cases',
          ),
          OutbreakMetric(
            label: 'Contacts followed up',
            value: '836',
            unit: 'contacts',
          ),
        ],
      );
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            publicSituationReportProvider('report').overrideWith(
              (_) async => PublicContent(
                value: report,
                cache: PublicCacheMetadata(
                  cachedAt: DateTime(2026),
                  lastVerifiedAt: DateTime(2026),
                  isStale: false,
                  isWithdrawn: false,
                  isOffline: false,
                ),
              ),
            ),
          ],
          child: MaterialApp(
            home: MediaQuery(
              data: MediaQueryData(
                size: const Size(320, 720),
                textScaler: TextScaler.linear(scale),
              ),
              child: const SituationReportDetailPage(reportId: 'report'),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.scrollUntilVisible(
        find.text('Read full report'),
        250,
        scrollable: find.byType(Scrollable).first,
      );
      expect(find.text('Read full report'), findsOneWidget);
      await tester.scrollUntilVisible(
        find.text('20'),
        250,
        scrollable: find.byType(Scrollable).first,
      );
      expect(find.text('20'), findsOneWidget);
      await tester.scrollUntilVisible(
        find.text('Read summary'),
        250,
        scrollable: find.byType(Scrollable).first,
      );
      await tester.pumpAndSettle();
      await tester.tap(find.text('Read summary').hitTestable());
      await tester.pumpAndSettle();
      expect(tester.widget<Text>(find.text(report.summary)).maxLines, isNull);
      expect(tester.takeException(), isNull);
    });
  }
}
