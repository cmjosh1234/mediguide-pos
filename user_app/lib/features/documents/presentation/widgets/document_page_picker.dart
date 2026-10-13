import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

class DocumentPagePicker extends StatefulWidget {
  const DocumentPagePicker({
    super.key,
    required this.pageCount,
    required this.currentPage,
  });
  final int pageCount;

  /// One-based page shown to readers.
  final int currentPage;
  @override
  State<DocumentPagePicker> createState() => _DocumentPagePickerState();
}

class _DocumentPagePickerState extends State<DocumentPagePicker> {
  final _form = GlobalKey<FormState>();
  late final _page = TextEditingController(text: '${widget.currentPage}');
  @override
  void dispose() {
    _page.dispose();
    super.dispose();
  }

  void _submit() {
    if (_form.currentState!.validate()) {
      Navigator.pop(context, int.parse(_page.text));
    }
  }

  @override
  Widget build(BuildContext context) => AlertDialog(
    title: const Text('Go to page'),
    scrollable: true,
    content: Form(
      key: _form,
      child: TextFormField(
        controller: _page,
        autofocus: true,
        keyboardType: TextInputType.number,
        inputFormatters: [FilteringTextInputFormatter.digitsOnly],
        textInputAction: TextInputAction.go,
        onFieldSubmitted: (_) => _submit(),
        decoration: InputDecoration(
          labelText: 'Page number',
          helperText: 'Choose a page from 1 to ${widget.pageCount}',
          border: const OutlineInputBorder(),
        ),
        validator: (value) {
          final page = int.tryParse(value ?? '');
          return page == null || page < 1 || page > widget.pageCount
              ? 'Enter a page from 1 to ${widget.pageCount}.'
              : null;
        },
      ),
    ),
    actions: [
      TextButton(
        onPressed: () => Navigator.pop(context),
        child: const Text('Cancel'),
      ),
      FilledButton(onPressed: _submit, child: const Text('Go')),
    ],
  );
}
