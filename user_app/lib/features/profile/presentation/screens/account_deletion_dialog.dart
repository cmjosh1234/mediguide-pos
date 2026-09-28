import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:user_app/app/providers/app_providers.dart';

class AccountDeletionDialog extends ConsumerStatefulWidget {
  const AccountDeletionDialog({super.key});

  @override
  ConsumerState<AccountDeletionDialog> createState() =>
      _AccountDeletionDialogState();
}

class _AccountDeletionDialogState extends ConsumerState<AccountDeletionDialog> {
  final _formKey = GlobalKey<FormState>();
  String _password = '';
  bool _confirmed = false;
  bool _busy = false;
  String? _requestId;
  String? _error;

  Future<void> _submit() async {
    if (_busy || !_confirmed || !_formKey.currentState!.validate()) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final response = await ref
          .read(backendApiServiceProvider)
          .requestJson(
            '/api/v2/me/deletion-request',
            method: 'POST',
            body: {'current_password': _password, 'confirm': true},
          );
      final data = response['data'];
      if (data is! Map || data['request_id'] is! String) {
        throw const FormatException('Missing deletion request reference');
      }
      if (!mounted) return;
      setState(() {
        _requestId = data['request_id'] as String;
        _password = '';
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _error =
            'Unable to submit. Check your current password and connection. '
            'If you have tried several times, wait before retrying.';
      });
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return PopScope(
      canPop: !_busy,
      child: AlertDialog(
        title: const Text('Request account deletion'),
        content: SingleChildScrollView(
          child: _requestId != null
              ? SelectableText(
                  'Your request has been recorded. Reference: $_requestId\n\n'
                  'Your account has not yet been deleted. Support will process '
                  'the request and confirm the outcome. You can follow up '
                  'through Help & Support.',
                )
              : Form(
                  key: _formKey,
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      const Text(
                        'Request removal of your MediGuide account and associated '
                        'personal data, including saved notes and reading history. '
                        'Support reviews any required retention of shared clinical '
                        'or security records. Submission does not immediately '
                        'erase data or sign you out.',
                      ),
                      const SizedBox(height: 16),
                      TextFormField(
                        obscureText: true,
                        autocorrect: false,
                        enableSuggestions: false,
                        enabled: !_busy,
                        decoration: const InputDecoration(
                          labelText: 'Current password',
                        ),
                        onChanged: (value) => _password = value,
                        validator: (value) => value == null || value.isEmpty
                            ? 'Enter your current password'
                            : null,
                      ),
                      CheckboxListTile(
                        contentPadding: EdgeInsets.zero,
                        title: const Text(
                          'I want to request account and associated data deletion.',
                        ),
                        value: _confirmed,
                        onChanged: _busy
                            ? null
                            : (value) =>
                                  setState(() => _confirmed = value ?? false),
                      ),
                      if (_error != null)
                        Text(
                          _error!,
                          style: TextStyle(
                            color: Theme.of(context).colorScheme.error,
                          ),
                        ),
                    ],
                  ),
                ),
        ),
        actions: [
          TextButton(
            onPressed: _busy ? null : () => Navigator.of(context).pop(),
            child: Text(_requestId != null ? 'Close' : 'Cancel'),
          ),
          if (_requestId == null)
            FilledButton(
              onPressed: _busy || !_confirmed ? null : _submit,
              child: Text(_busy ? 'Submitting…' : 'Submit request'),
            ),
        ],
      ),
    );
  }
}
