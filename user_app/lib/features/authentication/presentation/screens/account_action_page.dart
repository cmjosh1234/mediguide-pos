import 'dart:convert';
import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:user_app/app/providers/app_providers.dart';
import 'package:user_app/app/router/route_names.dart';
import 'package:user_app/features/authentication/presentation/controllers/auth_controller.dart';

class AccountActionPage extends ConsumerStatefulWidget {
  const AccountActionPage({
    super.key,
    required this.verification,
    this.token = '',
  });
  final bool verification;
  final String token;
  @override
  ConsumerState<AccountActionPage> createState() => _AccountActionPageState();
}

class _AccountActionPageState extends ConsumerState<AccountActionPage> {
  final form = GlobalKey<FormState>();
  final email = TextEditingController();
  final password = TextEditingController();
  final confirm = TextEditingController();
  bool busy = false, complete = false, resent = false;
  bool emailInitialized = false;
  String? error;
  int cooldown = 0;
  Timer? timer;
  @override
  void initState() {
    super.initState();
    email.text =
        ref.read(authControllerProvider).valueOrNull?.user?.email ?? '';
  }

  @override
  void dispose() {
    timer?.cancel();
    email.dispose();
    password.dispose();
    confirm.dispose();
    super.dispose();
  }

  Future<void> submit({bool resend = false}) async {
    if (busy || (resend && cooldown > 0)) return;
    if ((resend || !widget.verification) &&
        !(form.currentState?.validate() ?? false)) {
      return;
    }
    setState(() {
      busy = true;
      error = null;
    });
    try {
      final repository = ref.read(userRepositoryProvider);
      if (resend) {
        await repository.requestEmailVerification(email.text.trim());
        if (!mounted) return;
        setState(() {
          resent = true;
          cooldown = 60;
        });
        timer?.cancel();
        timer = Timer.periodic(const Duration(seconds: 1), (t) {
          if (!mounted) {
            t.cancel();
            return;
          }
          setState(() => cooldown--);
          if (cooldown <= 0) t.cancel();
        });
      } else if (widget.verification) {
        await repository.confirmEmailVerification(widget.token);
        await ref.read(authControllerProvider.notifier).refreshProfile();
        if (mounted) {
          setState(() => complete = true);
        }
      } else {
        await repository.confirmPasswordReset(
          token: widget.token,
          password: password.text,
          passwordConfirm: confirm.text,
        );
        await ref.read(authControllerProvider.notifier).logout();
        if (mounted) {
          setState(() => complete = true);
        }
      }
    } catch (_) {
      if (mounted) {
        setState(
          () => error = resend
              ? 'Unable to request a link right now. Please wait and try again.'
              : 'This link may be expired or already used. Request a new link and try again.',
        );
      }
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final verification = widget.verification;
    final hasToken = widget.token.trim().isNotEmpty;
    final user = ref.watch(authControllerProvider).valueOrNull?.user;
    if (!emailInitialized && user != null) {
      email.text = user.email;
      emailInitialized = true;
    }
    final verified =
        verification && !hasToken && (user?.emailVerified ?? false);
    return Scaffold(
      appBar: AppBar(
        title: Text(verification ? 'Verify your email' : 'Reset password'),
      ),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 480),
              child: Form(
                key: form,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    Icon(
                      complete || verified
                          ? Icons.check_circle_outline
                          : verification
                          ? Icons.mark_email_unread_outlined
                          : Icons.lock_reset,
                      size: 48,
                      color: Theme.of(context).colorScheme.primary,
                    ),
                    const SizedBox(height: 20),
                    Text(
                      complete || verified
                          ? verification
                                ? 'Your email is verified'
                                : 'Password updated'
                          : verification
                          ? 'Confirm your email address'
                          : 'Choose a new password',
                      style: Theme.of(context).textTheme.titleLarge,
                    ),
                    const SizedBox(height: 12),
                    Text(
                      complete || verified
                          ? verification
                                ? 'Your email ownership has been confirmed. Administrative account approval is separate.'
                                : 'Your previous sessions have been signed out. Sign in with your new password.'
                          : verification
                          ? 'We sent a verification link when you created your account. Open the link to confirm ownership of your email.'
                          : 'Use at least 8 characters, including a letter and a number.',
                    ),
                    const SizedBox(height: 20),
                    if (error != null) ...[
                      Text(
                        error!,
                        style: TextStyle(
                          color: Theme.of(context).colorScheme.error,
                        ),
                      ),
                      const SizedBox(height: 16),
                    ],
                    if (complete || verified)
                      FilledButton(
                        onPressed: () => context.go(
                          user == null ? AppRoutes.login : AppRoutes.profile,
                        ),
                        child: Text(
                          user == null ? 'Sign in' : 'Back to profile',
                        ),
                      )
                    else if (verification) ...[
                      if (hasToken) ...[
                        FilledButton(
                          onPressed: busy ? null : () => submit(),
                          child: Text(busy ? 'Verifying…' : 'Verify email'),
                        ),
                        const SizedBox(height: 24),
                      ],
                      TextFormField(
                        controller: email,
                        keyboardType: TextInputType.emailAddress,
                        autofillHints: const [AutofillHints.email],
                        decoration: const InputDecoration(
                          labelText: 'Email address',
                        ),
                        validator: (value) =>
                            RegExp(
                              r'^[^\s@]+@[^\s@]+\.[^\s@]+$',
                            ).hasMatch(value?.trim() ?? '')
                            ? null
                            : 'Enter a valid email address',
                      ),
                      const SizedBox(height: 12),
                      if (resent)
                        const Text(
                          'If this account needs verification, a new link will be emailed. Check your inbox and spam folder.',
                        ),
                      const SizedBox(height: 12),
                      OutlinedButton(
                        onPressed: busy || cooldown > 0
                            ? null
                            : () => submit(resend: true),
                        child: Text(
                          cooldown > 0
                              ? 'Request again in ${cooldown}s'
                              : 'Send verification link',
                        ),
                      ),
                      if (user != null)
                        TextButton(
                          onPressed: busy
                              ? null
                              : () async {
                                  await ref
                                      .read(authControllerProvider.notifier)
                                      .refreshProfile();
                                  if (mounted &&
                                      !(ref
                                              .read(authControllerProvider)
                                              .valueOrNull
                                              ?.user
                                              ?.emailVerified ??
                                          false)) {
                                    setState(
                                      () => error =
                                          'Email is not verified yet. Open the link in your email, then check again.',
                                    );
                                  }
                                },
                          child: const Text(
                            'I verified in my browser — check status',
                          ),
                        ),
                    ] else if (hasToken) ...[
                      TextFormField(
                        controller: password,
                        obscureText: true,
                        autofillHints: const [AutofillHints.newPassword],
                        decoration: const InputDecoration(
                          labelText: 'New password',
                        ),
                        validator: (value) {
                          final password = value ?? '';
                          if (utf8.encode(password).length > 72) {
                            return 'Password is too long';
                          }
                          if (password.length < 8 ||
                              !RegExp('[A-Za-z]').hasMatch(password) ||
                              !RegExp('[0-9]').hasMatch(password)) {
                            return 'Use at least 8 characters, a letter and number';
                          }
                          return null;
                        },
                      ),
                      const SizedBox(height: 16),
                      TextFormField(
                        controller: confirm,
                        obscureText: true,
                        decoration: const InputDecoration(
                          labelText: 'Confirm password',
                        ),
                        validator: (v) => v == password.text
                            ? null
                            : 'Passwords do not match',
                      ),
                      const SizedBox(height: 20),
                      FilledButton(
                        onPressed: busy ? null : () => submit(),
                        child: Text(busy ? 'Updating…' : 'Update password'),
                      ),
                      TextButton(
                        onPressed: busy
                            ? null
                            : () => context.go(AppRoutes.forgotPassword),
                        child: const Text('Request a new reset link'),
                      ),
                    ] else ...[
                      const Text(
                        'A reset link is required. Request one using your account email.',
                      ),
                      const SizedBox(height: 16),
                      FilledButton(
                        onPressed: () => context.go(AppRoutes.forgotPassword),
                        child: const Text('Request reset link'),
                      ),
                    ],
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
