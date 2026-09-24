import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';

import 'package:user_app/core/constants/app_spacing.dart';
import 'package:user_app/core/services/firebase_service.dart';
import 'package:user_app/core/utils/app_message.dart';
import 'package:user_app/features/notifications/presentation/controllers/notification_permission_prompt_controller.dart';

/// Explains why notifications matter before the system dialog is shown.
/// Renders nothing unless the signed-in user can still be asked.
class NotificationPermissionPromptCard extends ConsumerWidget {
  const NotificationPermissionPromptCard({
    super.key,
    this.padding = EdgeInsets.zero,
  });

  /// Applied only while the card is shown, so a hidden card leaves no gap.
  final EdgeInsetsGeometry padding;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    if (!ref.watch(notificationPermissionPromptProvider)) {
      return const SizedBox.shrink();
    }
    final controller = ref.read(notificationPermissionPromptProvider.notifier);
    final colors = Theme.of(context).colorScheme;
    final textTheme = Theme.of(context).textTheme;

    return Padding(
      padding: padding,
      child: Material(
        color: colors.secondaryContainer,
        borderRadius: BorderRadius.circular(16),
        child: Padding(
          padding: const EdgeInsets.fromLTRB(
            AppSpacing.md,
            AppSpacing.md,
            AppSpacing.md,
            AppSpacing.sm,
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Icon(
                    LucideIcons.bellRing,
                    color: colors.onSecondaryContainer,
                  ),
                  AppSpacing.md.gap,
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          'Turn on notifications',
                          style: textTheme.titleSmall?.copyWith(
                            color: colors.onSecondaryContainer,
                            fontWeight: FontWeight.w800,
                          ),
                        ),
                        const SizedBox(height: 4),
                        Text(
                          'Get approved outbreak alerts and guideline updates '
                          'as soon as they are published.',
                          style: textTheme.bodySmall?.copyWith(
                            color: colors.onSecondaryContainer,
                          ),
                        ),
                      ],
                    ),
                  ),
                ],
              ),
              Align(
                alignment: Alignment.centerRight,
                child: Wrap(
                  spacing: AppSpacing.sm,
                  children: [
                    TextButton(
                      onPressed: controller.dismiss,
                      child: const Text('Not now'),
                    ),
                    FilledButton(
                      onPressed: () async {
                        final result = await controller.turnOn();
                        if (!context.mounted) return;
                        if (result ==
                            AppNotificationPermissionState.authorized) {
                          AppMessage.success(
                            context,
                            'Notifications turned on',
                          );
                        }
                      },
                      child: const Text('Turn on'),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
