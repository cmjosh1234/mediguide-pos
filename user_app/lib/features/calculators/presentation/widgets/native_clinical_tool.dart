import 'package:flutter/material.dart';
import 'package:user_app/features/calculators/data/models/clinical_tool_definition.dart';
import 'package:user_app/features/calculators/domain/clinical_tool_evaluator.dart';

class NativeClinicalTool extends StatefulWidget {
  const NativeClinicalTool({
    super.key,
    required this.definition,
    this.initialValues = const {},
    this.onChanged,
  });
  final ClinicalToolDefinition definition;
  final Map<String, Object?> initialValues;
  final ValueChanged<Map<String, Object?>>? onChanged;
  @override
  State<NativeClinicalTool> createState() => _NativeClinicalToolState();
}

class _NativeClinicalToolState extends State<NativeClinicalTool> {
  final _formKey = GlobalKey<FormState>();
  final _feedbackKey = GlobalKey();
  final Map<String, Object?> _values = {};
  final Map<String, String> _units = {};
  int _formRevision = 0;
  ClinicalToolResult? _result;
  String? _error;

  @override
  void initState() {
    super.initState();
    _restoreInputs(widget.initialValues);
  }

  void _restoreInputs(Map<String, Object?> restoredValues) {
    _values.clear();
    for (final input in widget.definition.inputs) {
      if (input.defaultValue != null) _values[input.key] = input.defaultValue;
    }
    _values.addAll(restoredValues);
    _units.clear();
    for (final input in widget.definition.inputs) {
      final restored = _values[input.key];
      _units[input.key] = restored is Map
          ? restored['unit']?.toString() ?? input.defaultUnit
          : input.defaultUnit.isNotEmpty
          ? input.defaultUnit
          : input.allowedUnits.firstOrNull ?? '';
    }
  }

  @override
  void didUpdateWidget(covariant NativeClinicalTool oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.definition != widget.definition) {
      _restoreInputs(widget.initialValues);
      _formRevision++;
      _result = null;
      _error = null;
    }
  }

  @override
  Widget build(BuildContext context) {
    final definition = widget.definition;
    final visible = definition.inputs
        .where(
          (input) => const ClinicalToolEvaluator().inputVisible(
            definition,
            input,
            _values,
          ),
        )
        .toList();
    final sections = [...definition.sections]
      ..sort((a, b) => a.order.compareTo(b.order));
    final colors = Theme.of(context).colorScheme;
    return Form(
      key: _formKey,
      child: Column(
        children: [
          Expanded(
            child: SingleChildScrollView(
              keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
              child: Center(
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 720),
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        Text(
                          definition.title,
                          style: Theme.of(context).textTheme.headlineSmall
                              ?.copyWith(fontWeight: FontWeight.w800),
                        ),
                        if (definition.description.isNotEmpty) ...[
                          const SizedBox(height: 8),
                          Text(definition.description),
                        ],
                        const SizedBox(height: 12),
                        Text(
                          'Required fields are marked *',
                          style: Theme.of(context).textTheme.bodySmall
                              ?.copyWith(color: colors.onSurfaceVariant),
                        ),
                        const SizedBox(height: 16),
                        ...definition.warnings
                            .where((item) => item.when == null)
                            .map((item) => _MessageCard(message: item)),
                        for (final section in sections)
                          if (visible.any(
                            (input) => input.sectionKey == section.key,
                          ))
                            _section(
                              section.title,
                              visible
                                  .where(
                                    (input) => input.sectionKey == section.key,
                                  )
                                  .toList(),
                              description: section.description,
                            ),
                        if (visible.any(
                          (input) => !sections.any(
                            (section) => section.key == input.sectionKey,
                          ),
                        ))
                          _section(
                            'Assessment',
                            visible
                                .where(
                                  (input) => !sections.any(
                                    (section) =>
                                        section.key == input.sectionKey,
                                  ),
                                )
                                .toList(),
                          ),
                        Container(
                          key: _feedbackKey,
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.stretch,
                            children: [
                              if (_error != null)
                                Semantics(
                                  liveRegion: true,
                                  child: Container(
                                    padding: const EdgeInsets.all(16),
                                    decoration: BoxDecoration(
                                      color: colors.errorContainer,
                                      borderRadius: BorderRadius.circular(16),
                                    ),
                                    child: Text(
                                      _error!,
                                      style: TextStyle(
                                        color: colors.onErrorContainer,
                                      ),
                                    ),
                                  ),
                                ),
                              if (_result != null)
                                _ResultView(
                                  definition: definition,
                                  result: _result!,
                                ),
                            ],
                          ),
                        ),
                        if (definition.citations.isNotEmpty) ...[
                          const SizedBox(height: 24),
                          Text(
                            'Clinical sources',
                            style: Theme.of(context).textTheme.titleMedium,
                          ),
                          ...definition.citations.map(
                            (item) => ListTile(
                              contentPadding: EdgeInsets.zero,
                              leading: const Icon(Icons.menu_book_outlined),
                              title: Text(item.title),
                              subtitle: Text(item.organization),
                            ),
                          ),
                        ],
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ),
          Container(
            decoration: BoxDecoration(
              color: colors.surface,
              border: Border(top: BorderSide(color: colors.outlineVariant)),
            ),
            child: SafeArea(
              top: false,
              child: Padding(
                padding: const EdgeInsets.fromLTRB(16, 12, 16, 12),
                child: LayoutBuilder(
                  builder: (context, constraints) {
                    final calculate = FilledButton.icon(
                      style: FilledButton.styleFrom(
                        minimumSize: const Size(double.infinity, 52),
                        padding: const EdgeInsets.symmetric(
                          horizontal: 16,
                          vertical: 14,
                        ),
                      ),
                      onPressed: _calculate,
                      icon: const Icon(Icons.calculate_outlined),
                      label: Text(
                        definition.toolType == 'checklist'
                            ? 'Review checklist'
                            : 'Calculate',
                      ),
                    );
                    final reset = TextButton(
                      onPressed: _reset,
                      style: TextButton.styleFrom(
                        minimumSize: const Size(64, 48),
                      ),
                      child: const Text('Reset'),
                    );
                    if (constraints.maxWidth < 360 ||
                        MediaQuery.textScalerOf(context).scale(16) > 22) {
                      return Column(
                        mainAxisSize: MainAxisSize.min,
                        crossAxisAlignment: CrossAxisAlignment.stretch,
                        children: [calculate, reset],
                      );
                    }
                    return Row(
                      children: [
                        Expanded(child: calculate),
                        const SizedBox(width: 12),
                        reset,
                      ],
                    );
                  },
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _section(
    String title,
    List<ClinicalToolInput> inputs, {
    String description = '',
  }) {
    final colors = Theme.of(context).colorScheme;
    return Container(
      margin: const EdgeInsets.only(bottom: 16),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: colors.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(18),
        border: Border.all(color: colors.outlineVariant),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            title,
            style: Theme.of(
              context,
            ).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700),
          ),
          if (description.isNotEmpty) ...[
            const SizedBox(height: 6),
            Text(description),
          ],
          ...inputs.map(_keyedInput),
        ],
      ),
    );
  }

  Widget _keyedInput(ClinicalToolInput input) => KeyedSubtree(
    key: ValueKey('$_formRevision-${input.key}'),
    child: Padding(
      padding: const EdgeInsets.only(top: 16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (input.type != 'boolean' && input.type != 'checklist_item') ...[
            Text(
              '${input.label}${input.required ? ' *' : ''}',
              style: Theme.of(
                context,
              ).textTheme.bodyMedium?.copyWith(fontWeight: FontWeight.w600),
            ),
            const SizedBox(height: 8),
          ],
          _input(input),
          if (input.helpText.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(top: 6),
              child: Text(
                input.helpText,
                style: Theme.of(context).textTheme.bodySmall,
              ),
            ),
          if (input.clinicalWarning.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(top: 6),
              child: Text(
                input.clinicalWarning,
                style: TextStyle(color: Theme.of(context).colorScheme.error),
              ),
            ),
        ],
      ),
    ),
  );

  InputDecoration _decoration({String? hint, String? error}) => InputDecoration(
    hintText: hint,
    errorText: error,
    errorMaxLines: 4,
    filled: true,
    fillColor: Theme.of(context).colorScheme.surfaceContainerLow,
    contentPadding: const EdgeInsets.symmetric(horizontal: 14, vertical: 16),
    border: OutlineInputBorder(borderRadius: BorderRadius.circular(12)),
  );

  Widget _input(ClinicalToolInput input) {
    final colors = Theme.of(context).colorScheme;
    if (input.control == 'radio' && input.options.isNotEmpty) {
      return FormField<Object?>(
        initialValue: _values[input.key],
        validator: (_) => input.required && _values[input.key] == null
            ? '${input.label} is required'
            : null,
        builder: (state) => RadioGroup<Object?>(
          groupValue: _values[input.key],
          onChanged: (value) {
            _values[input.key] = value;
            state.didChange(value);
            _changed();
          },
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              ...input.options.map(
                (option) => Padding(
                  padding: const EdgeInsets.only(bottom: 8),
                  child: DecoratedBox(
                    decoration: BoxDecoration(
                      color: _values[input.key] == option.value
                          ? colors.primaryContainer
                          : colors.surfaceContainerLow,
                      borderRadius: BorderRadius.circular(12),
                      border: Border.all(
                        color: _values[input.key] == option.value
                            ? colors.primary
                            : colors.outlineVariant,
                      ),
                    ),
                    child: Material(
                      type: MaterialType.transparency,
                      child: RadioListTile<Object?>(
                        contentPadding: const EdgeInsets.symmetric(
                          horizontal: 8,
                          vertical: 4,
                        ),
                        shape: RoundedRectangleBorder(
                          borderRadius: BorderRadius.circular(12),
                        ),
                        title: Text(option.label),
                        value: option.value,
                      ),
                    ),
                  ),
                ),
              ),
              if (state.hasError)
                Text(state.errorText!, style: TextStyle(color: colors.error)),
            ],
          ),
        ),
      );
    }
    if (input.type == 'boolean' || input.type == 'checklist_item') {
      return DecoratedBox(
        decoration: BoxDecoration(
          color: _values[input.key] == true
              ? colors.primaryContainer
              : colors.surfaceContainerLow,
          borderRadius: BorderRadius.circular(12),
          border: Border.all(
            color: _values[input.key] == true
                ? colors.primary
                : colors.outlineVariant,
          ),
        ),
        child: Material(
          type: MaterialType.transparency,
          child: CheckboxListTile(
            controlAffinity: ListTileControlAffinity.leading,
            contentPadding: const EdgeInsets.symmetric(
              horizontal: 8,
              vertical: 4,
            ),
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(12),
            ),
            title: Text('${input.label}${input.required ? ' *' : ''}'),
            value: _values[input.key] == true,
            onChanged: (value) {
              _values[input.key] = value ?? false;
              _changed();
            },
          ),
        ),
      );
    }
    if (input.options.isNotEmpty) {
      return FormField<Object?>(
        initialValue: _values[input.key],
        validator: (_) => input.required && _values[input.key] == null
            ? '${input.label} is required'
            : null,
        builder: (state) {
          final selected = input.options
              .where((option) => option.value == _values[input.key])
              .firstOrNull;
          return Semantics(
            button: true,
            label: input.label,
            child: InkWell(
              borderRadius: BorderRadius.circular(12),
              onTap: () async {
                FocusManager.instance.primaryFocus?.unfocus();
                final option = await _chooseOption(input);
                if (!mounted || !state.mounted || option == null) return;
                _values[input.key] = option.value;
                state.didChange(option.value);
                _changed();
              },
              child: InputDecorator(
                decoration: _decoration(error: state.errorText),
                child: Row(
                  children: [
                    Expanded(
                      child: Text(
                        selected?.label ?? 'Select an option',
                        style: TextStyle(
                          color: selected == null
                              ? colors.onSurfaceVariant
                              : colors.onSurface,
                        ),
                      ),
                    ),
                    const SizedBox(width: 8),
                    const Icon(Icons.expand_more),
                  ],
                ),
              ),
            ),
          );
        },
      );
    }
    final numeric = ['number', 'integer', 'measurement'].contains(input.type);
    final calendar = input.type == 'date' || input.type == 'time';
    final textField = TextFormField(
      initialValue: _displayValue(_values[input.key]),
      decoration:
          _decoration(
            hint: calendar
                ? (input.type == 'date' ? 'Choose date' : 'Choose time')
                : null,
          ).copyWith(
            suffixText:
                input.allowedUnits.isEmpty && input.defaultUnit.isNotEmpty
                ? input.defaultUnit
                : null,
            suffixIcon: calendar
                ? Icon(
                    input.type == 'date'
                        ? Icons.calendar_today
                        : Icons.access_time,
                  )
                : null,
          ),
      readOnly: calendar,
      onTap: calendar ? () => _pickDateOrTime(input) : null,
      minLines: input.control == 'textarea' ? 3 : 1,
      maxLines: input.control == 'textarea' ? 6 : 1,
      textInputAction: input.control == 'textarea'
          ? TextInputAction.newline
          : TextInputAction.next,
      keyboardType: numeric
          ? TextInputType.numberWithOptions(decimal: input.type != 'integer')
          : input.control == 'textarea'
          ? TextInputType.multiline
          : TextInputType.text,
      validator: (value) =>
          input.required && (value == null || value.trim().isEmpty)
          ? '${input.label} is required'
          : null,
      onChanged: (value) {
        final Object? parsed = value.isEmpty
            ? null
            : numeric
            ? (double.tryParse(value) ?? value)
            : value;
        _values[input.key] = parsed == null || input.allowedUnits.isEmpty
            ? parsed
            : {'value': parsed, 'unit': _units[input.key]};
        _changed();
      },
    );
    if (input.allowedUnits.isEmpty) return textField;
    final unitField = DropdownButtonFormField<String>(
      isExpanded: true,
      initialValue: _units[input.key],
      decoration: _decoration().copyWith(labelText: 'Unit'),
      items: input.allowedUnits
          .map((unit) => DropdownMenuItem(value: unit, child: Text(unit)))
          .toList(),
      onChanged: (unit) {
        if (unit == null) return;
        _units[input.key] = unit;
        final current = _values[input.key],
            raw = current is Map ? current['value'] : current;
        if (raw != null) _values[input.key] = {'value': raw, 'unit': unit};
        _changed();
      },
    );
    return LayoutBuilder(
      builder: (context, constraints) {
        if (constraints.maxWidth < 300 ||
            MediaQuery.textScalerOf(context).scale(16) > 22) {
          return Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [textField, const SizedBox(height: 12), unitField],
          );
        }
        return Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Expanded(child: textField),
            const SizedBox(width: 12),
            SizedBox(width: 128, child: unitField),
          ],
        );
      },
    );
  }

  Future<ClinicalToolOption?> _chooseOption(ClinicalToolInput input) =>
      showModalBottomSheet<ClinicalToolOption>(
        context: context,
        useSafeArea: true,
        isScrollControlled: true,
        showDragHandle: true,
        builder: (context) => DraggableScrollableSheet(
          expand: false,
          initialChildSize: .65,
          minChildSize: .35,
          maxChildSize: .9,
          builder: (context, controller) => Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Padding(
                padding: const EdgeInsets.fromLTRB(20, 0, 12, 12),
                child: Row(
                  children: [
                    Expanded(
                      child: Text(
                        input.label,
                        style: Theme.of(context).textTheme.titleMedium,
                      ),
                    ),
                    IconButton(
                      tooltip: 'Close choices',
                      onPressed: () => Navigator.pop(context),
                      icon: const Icon(Icons.close),
                    ),
                  ],
                ),
              ),
              Expanded(
                child: ListView(
                  controller: controller,
                  padding: const EdgeInsets.fromLTRB(12, 0, 12, 24),
                  children: [
                    if (!input.required)
                      ListTile(
                        title: const Text('Clear selection'),
                        leading: const Icon(Icons.clear),
                        onTap: () => Navigator.pop(
                          context,
                          const ClinicalToolOption(value: null, label: ''),
                        ),
                      ),
                    ...input.options.map(
                      (option) => ListTile(
                        minVerticalPadding: 16,
                        shape: RoundedRectangleBorder(
                          borderRadius: BorderRadius.circular(12),
                        ),
                        selected: _values[input.key] == option.value,
                        leading: Icon(
                          _values[input.key] == option.value
                              ? Icons.radio_button_checked
                              : Icons.radio_button_unchecked,
                        ),
                        title: Text(option.label),
                        onTap: () => Navigator.pop(context, option),
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      );

  String? _displayValue(Object? value) =>
      value is Map ? value['value']?.toString() : value?.toString();

  Future<void> _pickDateOrTime(ClinicalToolInput input) async {
    String? selected;
    if (input.type == 'date') {
      final first = DateTime(1900), last = DateTime(2100, 12, 31);
      final restored =
          DateTime.tryParse(_values[input.key]?.toString() ?? '') ??
          DateTime.now();
      final date = await showDatePicker(
        context: context,
        initialDate: restored.isBefore(first)
            ? first
            : restored.isAfter(last)
            ? last
            : restored,
        firstDate: first,
        lastDate: last,
      );
      selected = date?.toIso8601String().substring(0, 10);
    } else {
      final parts = (_values[input.key]?.toString() ?? '').split(':');
      final hour = parts.length == 2 ? int.tryParse(parts[0]) : null;
      final minute = parts.length == 2 ? int.tryParse(parts[1]) : null;
      final time = await showTimePicker(
        context: context,
        initialTime:
            hour != null &&
                minute != null &&
                hour >= 0 &&
                hour < 24 &&
                minute >= 0 &&
                minute < 60
            ? TimeOfDay(hour: hour, minute: minute)
            : TimeOfDay.now(),
      );
      if (time != null) {
        selected =
            '${time.hour.toString().padLeft(2, '0')}:${time.minute.toString().padLeft(2, '0')}';
      }
    }
    if (!mounted || selected == null) return;
    setState(() {
      _values[input.key] = selected;
      _formRevision++;
    });
    _changed();
  }

  void _calculate() {
    FocusManager.instance.primaryFocus?.unfocus();
    final invalid = _formKey.currentState?.validateGranularly();
    if (invalid == null) return;
    if (invalid.isNotEmpty) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted && invalid.first.mounted) {
          Scrollable.ensureVisible(
            invalid.first.context,
            duration: const Duration(milliseconds: 250),
            alignment: .15,
          );
        }
      });
      return;
    }
    try {
      final value = const ClinicalToolEvaluator().evaluate(
        widget.definition,
        _values,
      );
      setState(() {
        _result = value;
        _error = null;
      });
    } catch (error) {
      setState(() {
        _result = null;
        _error = error.toString();
      });
    }
    _showFeedback();
  }

  void _showFeedback() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final feedback = _feedbackKey.currentContext;
      if (mounted && feedback != null) {
        Scrollable.ensureVisible(
          feedback,
          duration: const Duration(milliseconds: 250),
          alignment: .1,
        );
      }
    });
  }

  Future<void> _reset() async {
    if (widget.definition.completion.resetConfirmation) {
      final confirmed = await showDialog<bool>(
        context: context,
        builder: (context) => AlertDialog(
          title: const Text('Reset all responses?'),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: const Text('Cancel'),
            ),
            TextButton(
              onPressed: () => Navigator.pop(context, true),
              child: const Text('Reset'),
            ),
          ],
        ),
      );
      if (!mounted || confirmed != true) return;
    }
    setState(() {
      _restoreInputs(const {});
      _formRevision++;
      _result = null;
      _error = null;
    });
    _changed();
  }

  void _changed() {
    setState(() {
      _result = null;
      _error = null;
    });
    widget.onChanged?.call(Map<String, Object?>.from(_values));
  }
}

class _MessageCard extends StatelessWidget {
  const _MessageCard({required this.message});
  final ClinicalToolMessage message;
  @override
  Widget build(BuildContext context) => Card(
    color: Theme.of(context).colorScheme.errorContainer,
    child: Padding(
      padding: const EdgeInsets.all(12),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Icon(Icons.warning_amber_rounded),
          const SizedBox(width: 10),
          Expanded(child: Text(message.text)),
        ],
      ),
    ),
  );
}

class _ResultView extends StatelessWidget {
  const _ResultView({required this.definition, required this.result});
  final ClinicalToolDefinition definition;
  final ClinicalToolResult result;
  String _formatted(ClinicalToolOutput output, Object? value) =>
      value is num && output.precision != null
      ? value.toStringAsFixed(output.precision!)
      : (value?.toString() ?? '—');
  @override
  Widget build(BuildContext context) => Semantics(
    liveRegion: true,
    child: Card(
      margin: EdgeInsets.zero,
      color: Theme.of(context).colorScheme.surfaceContainerLowest,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(18),
        side: BorderSide(color: Theme.of(context).colorScheme.outlineVariant),
      ),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Result', style: Theme.of(context).textTheme.titleLarge),
            ...definition.outputs.map(
              (output) => Container(
                width: double.infinity,
                margin: const EdgeInsets.only(top: 12),
                padding: const EdgeInsets.all(14),
                decoration: BoxDecoration(
                  color: Theme.of(context).colorScheme.surfaceContainerLow,
                  borderRadius: BorderRadius.circular(12),
                ),
                child: Text.rich(
                  TextSpan(
                    children: [
                      TextSpan(text: '${output.label}: '),
                      TextSpan(
                        text: _formatted(output, result.values[output.key]),
                        style: const TextStyle(
                          fontWeight: FontWeight.w700,
                          fontSize: 22,
                        ),
                      ),
                      TextSpan(text: ' ${output.unit}'),
                    ],
                  ),
                ),
              ),
            ),
            if (result.interpretation != null) ...[
              const Divider(),
              Text(
                result.interpretation!.label,
                style: Theme.of(context).textTheme.titleMedium,
              ),
              if (result.interpretation!.description.isNotEmpty)
                Text(result.interpretation!.description),
            ],
            ...result.recommendations.map(
              (item) => ListTile(
                contentPadding: EdgeInsets.zero,
                leading: const Icon(Icons.check_circle_outline),
                title: Text(item),
              ),
            ),
            ...result.warnings.map((item) => _MessageCard(message: item)),
          ],
        ),
      ),
    ),
  );
}
