part of '../screens/help_center_page.dart';

/// Support view for visitors who are not signed in.
///
/// Guests can submit a request (their contact details are collected in the
/// create dialog) but have no account to list tickets under, so this view
/// explains that and points them to sign in for tracking.
class _GuestSupportView extends StatelessWidget {
  const _GuestSupportView({required this.onCreateTicket});

  final VoidCallback onCreateTicket;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;

    final textTheme = Theme.of(context).textTheme;

    return ListView(
      physics: const AlwaysScrollableScrollPhysics(),
      padding: const EdgeInsets.fromLTRB(
        AppSpacing.md,
        AppSpacing.md,
        AppSpacing.md,
        AppSpacing.xxxl,
      ),
      children: [
        // =====================================================================
        // SUBMIT A REQUEST
        // =====================================================================
        Container(
          padding: const EdgeInsets.all(AppSpacing.md),
          decoration: BoxDecoration(
            color: colors.surfaceContainerLow,
            borderRadius: BorderRadius.circular(16),
            border: Border.all(color: colors.outlineVariant),
          ),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Container(
                width: 42,
                height: 42,
                decoration: BoxDecoration(
                  color: colors.primaryContainer,
                  borderRadius: BorderRadius.circular(12),
                ),
                child: Icon(
                  LucideIcons.headphones,
                  color: colors.primary,
                  size: 21,
                ),
              ),

              AppSpacing.hGapMd,

              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'How can we help?',
                      style: textTheme.titleSmall?.copyWith(
                        fontWeight: FontWeight.w800,
                      ),
                    ),

                    const SizedBox(height: 3),

                    Text(
                      'You can request support without an account. '
                      'Share your name and email and our team will reply '
                      'to you directly.',
                      style: textTheme.bodySmall?.copyWith(
                        color: colors.onSurfaceVariant,
                        height: 1.4,
                      ),
                    ),

                    AppSpacing.gapMd,

                    FilledButton.tonalIcon(
                      onPressed: onCreateTicket,
                      icon: const Icon(LucideIcons.plus, size: 18),
                      label: const Text('Request support'),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),

        AppSpacing.gapLg,

        // =====================================================================
        // SIGN IN TO TRACK
        // =====================================================================
        Text(
          'Track your requests',
          style: textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w800),
        ),

        const SizedBox(height: 3),

        Text(
          'Sign in to see the status of your tickets, read replies from the '
          'support team and continue the conversation in the app.',
          style: textTheme.bodySmall?.copyWith(color: colors.onSurfaceVariant),
        ),

        AppSpacing.gapSm,

        Container(
          padding: const EdgeInsets.all(AppSpacing.md),
          decoration: BoxDecoration(
            color: colors.surfaceContainerLowest,
            borderRadius: BorderRadius.circular(16),
            border: Border.all(color: colors.outlineVariant),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Icon(
                    LucideIcons.lockKeyhole,
                    size: 18,
                    color: colors.primary,
                  ),
                  AppSpacing.hGapSm,
                  Expanded(
                    child: Text(
                      'Ticket history is only available to signed-in users.',
                      style: textTheme.bodyMedium?.copyWith(
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
                ],
              ),

              AppSpacing.gapMd,

              Wrap(
                spacing: AppSpacing.sm,
                runSpacing: AppSpacing.sm,
                children: [
                  FilledButton.icon(
                    onPressed: () {
                      context.push(
                        '${AppRoutes.login}?redirect=${Uri.encodeComponent(AppRoutes.helpCenter)}',
                      );
                    },
                    icon: const Icon(LucideIcons.logIn, size: 18),
                    label: const Text('Sign in'),
                  ),

                  OutlinedButton.icon(
                    onPressed: () {
                      context.push(AppRoutes.register);
                    },
                    icon: const Icon(LucideIcons.userPlus, size: 18),
                    label: const Text('Create account'),
                  ),
                ],
              ),
            ],
          ),
        ),
      ],
    );
  }
}
