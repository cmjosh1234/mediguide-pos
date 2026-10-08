import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:user_app/features/discovery/presentation/widgets/discovery_widgets.dart';

Future<void> _pumpBanner(WidgetTester tester, Map<String, dynamic> outbreak) =>
    tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: SingleChildScrollView(child: OutbreakBanner(outbreak)),
        ),
      ),
    );

const _outbreak = {
  'title': 'Bundibugyo virus disease response — Uganda',
  'status': 'monitoring',
  'geographic_area': 'Uganda and the Democratic Republic of the Congo',
};

void main() {
  testWidgets('hub banner shows one tile per metric, not raw fields', (
    tester,
  ) async {
    await _pumpBanner(tester, {
      ..._outbreak,
      'metrics': [
        {
          'key': 'uganda_confirmed',
          'label': 'Confirmed cases in Uganda',
          'value': '20',
        },
        {'key': 'uganda_deaths', 'label': 'Deaths in Uganda', 'value': 2},
      ],
    });

    expect(find.text('Confirmed cases in Uganda'), findsOneWidget);
    expect(find.text('20'), findsOneWidget);
    expect(find.text('Deaths in Uganda'), findsOneWidget);
    expect(find.text('2'), findsOneWidget);
    expect(find.byType(Chip), findsNothing);
    expect(find.textContaining('uganda_confirmed'), findsNothing);
    expect(find.textContaining(' label'), findsNothing);
    expect(find.textContaining(' value'), findsNothing);
  });

  testWidgets('hub banner shows the same metric detail as the web hub', (
    tester,
  ) async {
    await _pumpBanner(tester, {
      ..._outbreak,
      'last_verified_at': '2026-07-26T12:00:00Z',
      'metrics': [
        {
          'key': 'contacts_followed',
          'label': 'Contacts followed up',
          'value': '1234',
          'numeric_value': 1234,
          'unit': 'contacts',
          'as_of': '2026-07-26T12:00:00Z',
          'source_reference': 'WHO situation report 11',
          'sort_order': 2,
        },
        {
          'key': 'uganda_confirmed',
          'label': 'Confirmed cases in Uganda',
          'value': '20',
          'numeric_value': 20,
          'unit': 'cases',
          'as_of': 'not-a-date',
          'source_reference': 'WHO situation report 11',
          'sort_order': 1,
        },
      ],
    });

    expect(find.text('Verified 26 Jul 2026'), findsOneWidget);
    expect(find.text('20 cases'), findsOneWidget);
    expect(find.text('1,234 contacts'), findsOneWidget);
    expect(
      find.text('WHO situation report 11 · as of 26 Jul 2026'),
      findsOneWidget,
    );
    // An unparseable as_of drops the date but keeps the source.
    expect(find.text('WHO situation report 11'), findsOneWidget);
    expect(find.textContaining('sort order'), findsNothing);
    expect(find.textContaining('source reference'), findsNothing);
    expect(
      tester.getTopLeft(find.text('20 cases')).dx,
      lessThan(tester.getTopLeft(find.text('1,234 contacts')).dx),
    );
  });
}
