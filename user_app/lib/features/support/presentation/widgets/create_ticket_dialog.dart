import 'package:flutter/material.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:form_builder_validators/form_builder_validators.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';

import 'package:user_app/core/constants/app_spacing.dart';
import 'package:user_app/features/authentication/presentation/controllers/auth_controller.dart';
import 'package:user_app/features/support/presentation/controllers/help_center_controller.dart';
import 'package:user_app/shared/models/models.dart';

/// Full-screen dialog for creating a new support ticket.
///
/// Signed-in users submit tickets against their account. Guests are asked for
/// a name and email address so support staff can follow up with them.
class CreateTicketDialog extends ConsumerStatefulWidget {
  const CreateTicketDialog({super.key});

  static Future<void> show(BuildContext context) {
    return showDialog<void>(
      context: context,
      barrierDismissible: false,
      useSafeArea: false,
      builder: (_) {
        return const CreateTicketDialog();
      },
    );
  }

  @override
  ConsumerState<CreateTicketDialog> createState() => _CreateTicketDialogState();
}

class _CreateTicketDialogState extends ConsumerState<CreateTicketDialog> {
  final _formKey = GlobalKey<FormBuilderState>();

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(helpCenterControllerProvider);

    final controller = ref.read(helpCenterControllerProvider.notifier);

    final isGuest = !ref.watch(
      authControllerProvider.select(
        (value) => value.valueOrNull?.isAuthenticated ?? false,
      ),
    );

    final colors = Theme.of(context).colorScheme;

    return PopScope(
      canPop: !state.isCreatingTicket,
      child: Scaffold(
        appBar: AppBar(
          title: Text(isGuest ? 'Request Support' : 'Create Support Ticket'),
          leading: IconButton(
            tooltip: 'Close',
            icon: const Icon(LucideIcons.x),
            onPressed: state.isCreatingTicket
                ? null
                : () {
                    Navigator.of(context).pop();
                  },
          ),
          actions: [
            TextButton.icon(
              onPressed: state.isCreatingTicket
                  ? null
                  : () {
                      _submit(controller);
                    },
              icon: state.isCreatingTicket
                  ? const SizedBox(
                      width: 18,
                      height: 18,
                      child: CircularProgressIndicator.adaptive(strokeWidth: 2),
                    )
                  : const Icon(LucideIcons.save, size: 18),
              label: Text(
                state.isCreatingTicket
                    ? (isGuest ? 'Sending...' : 'Creating...')
                    : (isGuest ? 'Send' : 'Create'),
              ),
            ),
          ],
        ),
        body: SafeArea(
          child: FormBuilder(
            key: _formKey,
            autovalidateMode: AutovalidateMode.onUserInteraction,
            child: ListView(
              keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
              padding: AppSpacing.pagePadding,
              children: [
                AppSpacing.contentGap,

                // ============================================
                // GUEST CONTACT DETAILS
                // ============================================
                if (isGuest) ...[
                  Container(
                    padding: const EdgeInsets.all(AppSpacing.md),
                    decoration: BoxDecoration(
                      color: colors.surfaceContainerLow,
                      borderRadius: BorderRadius.circular(12),
                      border: Border.all(color: colors.outlineVariant),
                    ),
                    child: Row(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Icon(LucideIcons.info, size: 18, color: colors.primary),
                        AppSpacing.hGapSm,
                        Expanded(
                          child: Text(
                            'You are not signed in. Tell us how to reach you '
                            'and our team will reply by email. Sign in to '
                            'track your requests in the app.',
                            style: Theme.of(context).textTheme.bodySmall
                                ?.copyWith(
                                  color: colors.onSurfaceVariant,
                                  height: 1.4,
                                ),
                          ),
                        ),
                      ],
                    ),
                  ),

                  AppSpacing.fieldGap,

                  FormBuilderTextField(
                    name: 'requester_name',
                    enabled: !state.isCreatingTicket,
                    decoration: const InputDecoration(
                      labelText: 'Your name *',
                      hintText: 'Full name',
                      prefixIcon: Icon(LucideIcons.user),
                    ),
                    textCapitalization: TextCapitalization.words,
                    textInputAction: TextInputAction.next,
                    validator: FormBuilderValidators.compose([
                      FormBuilderValidators.required(),
                      FormBuilderValidators.minLength(2),
                      FormBuilderValidators.maxLength(120),
                    ]),
                  ),

                  AppSpacing.fieldGap,

                  FormBuilderTextField(
                    name: 'requester_email',
                    enabled: !state.isCreatingTicket,
                    decoration: const InputDecoration(
                      labelText: 'Email address *',
                      hintText: 'name@example.com',
                      prefixIcon: Icon(LucideIcons.mail),
                    ),
                    keyboardType: TextInputType.emailAddress,
                    autocorrect: false,
                    textInputAction: TextInputAction.next,
                    validator: FormBuilderValidators.compose([
                      FormBuilderValidators.required(),
                      FormBuilderValidators.email(),
                    ]),
                  ),

                  AppSpacing.fieldGap,
                ],

                // ============================================
                // SUBJECT
                // ============================================
                FormBuilderTextField(
                  name: 'subject',
                  enabled: !state.isCreatingTicket,
                  decoration: const InputDecoration(
                    labelText: 'Subject *',
                    hintText: 'Brief description of your issue',
                    prefixIcon: Icon(LucideIcons.type),
                  ),
                  textInputAction: TextInputAction.next,
                  validator: FormBuilderValidators.compose([
                    FormBuilderValidators.required(),
                    FormBuilderValidators.minLength(5),
                  ]),
                ),

                AppSpacing.fieldGap,

                // ============================================
                // CATEGORY
                // ============================================
                FormBuilderDropdown<String>(
                  name: 'category',
                  enabled: !state.isCreatingTicket,
                  decoration: const InputDecoration(
                    labelText: 'Category',
                    prefixIcon: Icon(LucideIcons.tag),
                  ),
                  initialValue: 'General Question',
                  items: HelpCenterController.availableCategories.map((
                    category,
                  ) {
                    return DropdownMenuItem<String>(
                      value: category,
                      child: Text(category),
                    );
                  }).toList(),
                  validator: FormBuilderValidators.required(),
                ),

                AppSpacing.fieldGap,

                // ============================================
                // PRIORITY
                // ============================================
                FormBuilderDropdown<TicketPriority>(
                  name: 'priority',
                  enabled: !state.isCreatingTicket,
                  decoration: const InputDecoration(
                    labelText: 'Priority',
                    prefixIcon: Icon(LucideIcons.flag),
                  ),
                  initialValue: TicketPriority.normal,
                  items: TicketPriority.values.map((priority) {
                    return DropdownMenuItem<TicketPriority>(
                      value: priority,
                      child: Row(
                        children: [
                          Container(
                            width: 12,
                            height: 12,
                            decoration: BoxDecoration(
                              color: priority.color,
                              shape: BoxShape.circle,
                            ),
                          ),
                          AppSpacing.gapSm,
                          Text(priority.label),
                        ],
                      ),
                    );
                  }).toList(),
                  validator: FormBuilderValidators.required(),
                ),

                AppSpacing.fieldGap,

                // ============================================
                // DESCRIPTION
                // ============================================
                FormBuilderTextField(
                  name: 'description',
                  enabled: !state.isCreatingTicket,
                  decoration: const InputDecoration(
                    labelText: 'Description *',
                    hintText: 'Provide detailed information about your issue',
                    prefixIcon: Icon(LucideIcons.fileText),
                    alignLabelWithHint: true,
                  ),
                  minLines: 5,
                  maxLines: 8,
                  textCapitalization: TextCapitalization.sentences,
                  textInputAction: TextInputAction.newline,
                  validator: FormBuilderValidators.compose([
                    FormBuilderValidators.required(),
                    FormBuilderValidators.minLength(10),
                  ]),
                ),

                AppSpacing.sectionGap,

                // ============================================
                // SUBMIT BUTTON
                // ============================================
                FilledButton.icon(
                  onPressed: state.isCreatingTicket
                      ? null
                      : () {
                          _submit(controller);
                        },
                  icon: state.isCreatingTicket
                      ? const SizedBox(
                          width: 18,
                          height: 18,
                          child: CircularProgressIndicator.adaptive(
                            strokeWidth: 2,
                          ),
                        )
                      : const Icon(LucideIcons.send),
                  label: Text(
                    state.isCreatingTicket
                        ? (isGuest
                              ? 'Sending request...'
                              : 'Creating ticket...')
                        : (isGuest ? 'Send Request' : 'Create Ticket'),
                  ),
                ),

                AppSpacing.gapLg,
              ],
            ),
          ),
        ),
      ),
    );
  }

  Future<void> _submit(HelpCenterController controller) async {
    if (!(_formKey.currentState?.saveAndValidate() ?? false)) {
      return;
    }

    final values = _formKey.currentState!.value;

    final subject = values['subject']?.toString().trim() ?? '';

    final description = values['description']?.toString().trim() ?? '';

    final category = values['category']?.toString() ?? 'General Question';

    final priority =
        values['priority'] as TicketPriority? ?? TicketPriority.normal;

    final requesterName = values['requester_name']?.toString().trim();

    final requesterEmail = values['requester_email']?.toString().trim();

    final created = await controller.createTicket(
      subject: subject,
      description: description,
      category: category,
      priority: priority,
      requesterName: requesterName,
      requesterEmail: requesterEmail,
    );

    if (!mounted || !created) {
      return;
    }

    Navigator.of(context).pop();
  }
}
