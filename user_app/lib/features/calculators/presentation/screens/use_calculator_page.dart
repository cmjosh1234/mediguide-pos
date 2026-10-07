import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';
import 'package:user_app/core/constants/app_spacing.dart';
import 'package:user_app/core/widgets/app_error_view.dart';
import 'package:user_app/core/widgets/app_loading_view.dart';
import 'package:user_app/features/calculators/presentation/controllers/use_calculator_controller.dart';
import 'package:user_app/features/calculators/presentation/widgets/native_clinical_tool.dart';
import 'package:user_app/shared/models/models.dart';

part '../widgets/use_calculator_page_calculator_context_bar.dart';

class UseCalculatorPage extends ConsumerStatefulWidget {
  const UseCalculatorPage({super.key, this.arguments});
  final Object? arguments;
  @override
  ConsumerState<UseCalculatorPage> createState() => _UseCalculatorPageState();
}

class _UseCalculatorPageState extends ConsumerState<UseCalculatorPage> {
  late final UseCalculatorRequest _request;
  @override
  void initState() {
    super.initState();
    _request = _requestFromArguments(widget.arguments);
  }

  UseCalculatorRequest _requestFromArguments(Object? arguments) {
    if (arguments is Calculator) {
      return UseCalculatorRequest(id: arguments.id, calculator: arguments);
    }
    if (arguments is Map) {
      return UseCalculatorRequest(
        id:
            arguments['calculatorId']?.toString().trim() ??
            arguments['id']?.toString().trim() ??
            '',
      );
    }
    return UseCalculatorRequest(id: arguments?.toString().trim() ?? '');
  }

  @override
  Widget build(BuildContext context) {
    final provider = useCalculatorControllerProvider(_request);
    final asyncState = ref.watch(provider);
    final calculator =
        asyncState.valueOrNull?.calculator ?? _request.calculator;
    return Scaffold(
      appBar: AppBar(
        titleSpacing: AppSpacing.sm,
        toolbarHeight: calculator == null
            ? kToolbarHeight
            : _CalculatorContextBar.height(context, calculator),
        title: calculator == null
            ? const Text('Clinical tool')
            : _CalculatorContextBar(calculator: calculator),
        bottom: PreferredSize(
          preferredSize: const Size.fromHeight(1),
          child: Divider(
            height: 1,
            color: Theme.of(context).colorScheme.outlineVariant,
          ),
        ),
        actions: [
          IconButton(
            tooltip: 'Reload tool',
            icon: const Icon(LucideIcons.refreshCw),
            onPressed: () => ref.invalidate(provider),
          ),
          AppSpacing.xs.gap,
        ],
      ),
      body: asyncState.when(
        loading: () =>
            const AppLoadingView(message: 'Loading clinical tool...'),
        error: (error, stackTrace) => AppErrorView(
          error: error,
          title: 'Clinical tool unavailable',
          message:
              'A clinically reviewed JSON-schema version has not been published for this tool, or its cached schema is unavailable.',
          onRetry: () => ref.invalidate(provider),
        ),
        data: (state) => NativeClinicalTool(
          definition: state.definition.definition,
          initialValues: state.responses,
          onChanged: ref.read(provider.notifier).saveResponses,
        ),
      ),
    );
  }
}
