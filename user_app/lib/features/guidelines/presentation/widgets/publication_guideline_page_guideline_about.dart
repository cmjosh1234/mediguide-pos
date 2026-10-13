part of '../screens/publication_guideline_page.dart';

class _GuidelineAbout extends StatelessWidget {
  const _GuidelineAbout({required this.publication});
  final GuidelinePublication publication;

  @override
  Widget build(BuildContext context) {
    String date(String value) {
      final parsed = DateTime.tryParse(value);
      return parsed == null
          ? value
          : MaterialLocalizations.of(context).formatMediumDate(parsed);
    }

    final language = switch (publication.language.toLowerCase()) {
      'en' => 'English',
      'sw' => 'Swahili',
      _ => publication.language,
    };
    final facts = <(String, String)>[
      ('Published', date(publication.publicationDate)),
      ('Review date', date(publication.reviewDate)),
      ('Target users', publication.intendedPopulation),
      ('Applies to', publication.healthcareLevel),
      ('Country', publication.country),
      ('Language', language),
    ].where((item) => item.$2.trim().isNotEmpty).toList(growable: false);
    if (facts.isEmpty) return const SizedBox.shrink();
    return Card(
      margin: EdgeInsets.zero,
      child: Padding(
        padding: const EdgeInsets.all(AppSpacing.lg),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              'Document details',
              style: Theme.of(
                context,
              ).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700),
            ),
            AppSpacing.gapLg,
            LayoutBuilder(
              builder: (context, constraints) {
                final columns =
                    constraints.maxWidth >= 520 &&
                        MediaQuery.textScalerOf(context).scale(1) <= 1.3
                    ? 2
                    : 1;
                final width =
                    (constraints.maxWidth - AppSpacing.lg * (columns - 1)) /
                    columns;
                return Wrap(
                  spacing: AppSpacing.lg,
                  runSpacing: AppSpacing.lg,
                  children: [
                    for (final fact in facts)
                      SizedBox(
                        width: width,
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              fact.$1,
                              style: Theme.of(context).textTheme.labelMedium
                                  ?.copyWith(
                                    color: Theme.of(
                                      context,
                                    ).colorScheme.onSurfaceVariant,
                                  ),
                            ),
                            AppSpacing.gapXs,
                            Text(
                              fact.$2,
                              style: Theme.of(
                                context,
                              ).textTheme.bodyLarge?.copyWith(height: 1.4),
                            ),
                          ],
                        ),
                      ),
                  ],
                );
              },
            ),
          ],
        ),
      ),
    );
  }
}
