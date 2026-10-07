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

    await tester.tap(find.text('Source and review details'));
    await tester.pumpAndSettle();
    expect(find.textContaining('WHO publication catalogue'), findsOneWidget);
    await tester.tap(find.text('WHO technical guidance'));
    await tester.pumpAndSettle();
    expect(find.text('Open external resource?'), findsOneWidget);
    expect(find.textContaining('www.who.int'), findsOneWidget);

    await tester.tap(find.text('Cancel'));
    await tester.pumpAndSettle();
    expect(find.text('Open external resource?'), findsNothing);
  });
}
