part of '../screens/publication_guideline_page.dart';

/// A library document published as a link: there is nothing to read in the
/// app, so the page explains where the link goes and opens it in the browser.
class _LinkedDocumentView extends StatelessWidget {
  const _LinkedDocumentView({required this.publication, required this.onOpen});

  final GuidelinePublication publication;
  final VoidCallback onOpen;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    final host = Uri.tryParse(publication.externalUrl)?.host ?? '';
    final meta = [
      publication.sourceOrganization,
      publication.publicationDate,
      if (publication.version.isNotEmpty) 'v${publication.version}',
    ].where((value) => value.trim().isNotEmpty).join(' • ');

    return ListView(
      padding: const EdgeInsets.fromLTRB(
        AppSpacing.md,
        AppSpacing.lg,
        AppSpacing.md,
        AppSpacing.xl,
      ),
      children: [
        Text(
          publication.title,
          style: theme.textTheme.headlineMedium?.copyWith(
            fontWeight: FontWeight.w800,
          ),
        ),
        if (meta.isNotEmpty) ...[
          AppSpacing.gapXs,
          Text(
            meta,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: colors.onSurfaceVariant,
            ),
          ),
        ],
        AppSpacing.gapLg,
        Card(
          child: Padding(
            padding: const EdgeInsets.all(AppSpacing.md),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Container(
                      width: 40,
                      height: 40,
                      alignment: Alignment.center,
                      decoration: BoxDecoration(
                        color: colors.primaryContainer,
                        borderRadius: BorderRadius.circular(10),
                      ),
                      child: Icon(
                        LucideIcons.globe,
                        size: 20,
                        color: colors.onPrimaryContainer,
                      ),
                    ),
                    AppSpacing.hGapSm,
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            'External website',
                            style: theme.textTheme.titleMedium?.copyWith(
                              fontWeight: FontWeight.w700,
                            ),
                          ),
                          if (host.isNotEmpty)
                            Text(
                              host,
                              style: theme.textTheme.bodySmall?.copyWith(
                                color: colors.onSurfaceVariant,
                              ),
                            ),
                        ],
                      ),
                    ),
                  ],
                ),
                AppSpacing.gapSm,
                Text(
                  'This resource is published as a link. It opens in your '
                  'browser, and its content is maintained by the website '
                  'that publishes it.',
                  style: theme.textTheme.bodyMedium,
                ),
                AppSpacing.gapMd,
                SizedBox(
                  width: double.infinity,
                  child: FilledButton.icon(
                    onPressed: onOpen,
                    icon: const Icon(LucideIcons.externalLink, size: 18),
                    label: const Text('Open website'),
                    style: FilledButton.styleFrom(
                      minimumSize: const Size.fromHeight(48),
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
        AppSpacing.gapLg,
        _GuidelineAbout(publication: publication, recommendations: const []),
      ],
    );
  }
}
