import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:user_app/features/discovery/data/models/discovery_models.dart';
import 'package:user_app/features/discovery/presentation/widgets/discovery_widgets.dart';

void main() {
  test('nested pillars retain their hierarchy and aggregate resources', () {
    const resource = DiscoveryResource(
      id: 'case-definition',
      contentType: 'outbreak_document',
      title: 'Case definition',
    );
    const pillar = DiscoveryPillar(
      id: 'clinical-care',
      name: 'Clinical care',
      slug: 'clinical-care',
      children: [
        DiscoveryPillar(
          id: 'screening',
          name: 'Screening',
          slug: 'screening',
          items: [resource],
        ),
      ],
    );

    expect(flatten([pillar]).map((item) => item.slug), [
      'clinical-care',
      'screening',
    ]);
    expect(pillar.resourceCount, 1);
    expect(resources(pillar), [resource]);
  });

  testWidgets('external resources show provenance and require confirmation', (
    tester,
  ) async {
    const resource = DiscoveryResource(
      id: 'who-1',
      contentType: 'approved_external_url',
      title: 'WHO technical guidance',
      route: 'https://www.who.int/publications/example',
      source: 'World Health Organization',
      provenance: 'WHO publication catalogue',
    );

    await tester.pumpWidget(
      const MaterialApp(home: Scaffold(body: ResourceTile(resource))),
    );

    expect(find.textContaining('WHO publication catalogue'), findsOneWidget);
    expect(find.text('External resource'), findsOneWidget);
    expect(find.text('Visit website'), findsOneWidget);
    await tester.tap(find.text('WHO technical guidance'));
    await tester.pumpAndSettle();
    expect(find.text('Open external resource?'), findsOneWidget);
    expect(find.textContaining('www.who.int'), findsOneWidget);

    await tester.tap(find.text('Cancel'));
    await tester.pumpAndSettle();
    expect(find.text('Open external resource?'), findsNothing);
  });

  testWidgets('guidelines show their type, issuer, dates and action', (
    tester,
  ) async {
    const resource = DiscoveryResource(
      id: 'ebola-guideline',
      contentType: 'guideline',
      title: 'Ebola and Marburg Disease Preparedness',
      description: 'Recognition, isolation and safe initial management.',
      route: '/guidelines/ebola',
      source: 'Ministry of Health Uganda',
      version: '2.0',
      publicationDate: '2026-05-28T00:00:00Z',
      reviewAt: '2027-05-28T00:00:00Z',
    );

    await tester.pumpWidget(
      const MaterialApp(home: Scaffold(body: ResourceTile(resource))),
    );

    expect(find.text('Guideline'), findsOneWidget);
    expect(find.text('v2.0'), findsOneWidget);
    expect(find.text('Ministry of Health Uganda'), findsOneWidget);
    expect(
      find.text('Published 28 May 2026', findRichText: true),
      findsOneWidget,
    );
    expect(
      find.text('Next review 28 May 2027', findRichText: true),
      findsOneWidget,
    );
    expect(find.text('Read guideline'), findsOneWidget);
    expect(find.textContaining(' · '), findsNothing);
  });

  testWidgets('unknown types fall back to a readable label without an action', (
    tester,
  ) async {
    const resource = DiscoveryResource(
      id: 'field-guide',
      contentType: 'field_guide',
      title: 'Community field guide',
    );

    await tester.pumpWidget(
      const MaterialApp(home: Scaffold(body: ResourceTile(resource))),
    );

    expect(find.text('Field guide'), findsOneWidget);
    expect(find.text('Open'), findsNothing);
  });
}
