import 'dart:async';

import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:flutter/foundation.dart';
import 'package:user_app/features/calculators/data/repositories/calculator_usage_tracker.dart';
import 'package:user_app/app/providers/app_providers.dart';
import 'package:user_app/features/authentication/presentation/controllers/auth_controller.dart';
import 'package:user_app/features/calculators/data/models/clinical_tool_definition.dart';
import 'package:user_app/features/calculators/data/repositories/calculator_repository.dart';
import 'package:user_app/shared/models/models.dart';

part 'use_calculator_controller.g.dart';

final class UseCalculatorRequest {
  const UseCalculatorRequest({required this.id, this.calculator});
  final String id;
  final Calculator? calculator;
  @override
  bool operator ==(Object other) =>
      other is UseCalculatorRequest && other.id == id;
  @override
  int get hashCode => id.hashCode;
}

final class UseCalculatorState {
  const UseCalculatorState({
    required this.calculator,
    required this.definition,
    this.responses = const {},
  });
  final Calculator calculator;
  final ClinicalToolDefinitionEnvelope definition;
  final Map<String, Object?> responses;
  UseCalculatorState copyWith({Map<String, Object?>? responses}) =>
      UseCalculatorState(
        calculator: calculator,
        definition: definition,
        responses: responses ?? this.responses,
      );
}

@riverpod
class UseCalculatorController extends _$UseCalculatorController {
  CalculatorRepository get _repository =>
      ref.read(calculatorRepositoryProvider);

  @override
  Future<UseCalculatorState> build(UseCalculatorRequest request) async {
    if (request.id.trim().isEmpty) {
      throw ArgumentError.value(request.id, 'calculatorId', 'is required');
    }
    var disposed = false;
    ref.onDispose(() => disposed = true);
    final repository = _repository;
    final calculator = request.calculator ?? await repository.get(request.id);

    // A published schema is now the only executable clinical-tool runtime.
    // The repository may return a checksum-validated cached definition offline.
    final definition = await repository.definition(calculator.id);
    if (disposed) throw StateError('Tool page closed during loading');
    final userId = ref.read(authControllerProvider).valueOrNull?.user?.id;
    final responses =
        userId == null || !definition.definition.completion.allowResume
        ? const <String, Object?>{}
        : await _repository.workflow(
            calculatorId: calculator.id,
            userId: userId,
            definition: definition,
          );
    if (!disposed && userId != null) {
      final tracker = ref.read(calculatorUsageTrackerProvider);
      final started = tracker.begin(
        userId: userId,
        calculatorId: calculator.id,
        versionId: definition.versionId,
        calculatorType: _calculatorTypeValue(calculator.type),
        start: DateTime.now().toUtc(),
      );
      // Observe errors without making the clinical form depend on analytics.
      unawaited(
        started.then<void>(
          (_) {},
          onError: (Object error) {
            debugPrint(
              'Clinical tool usage start failed: ${error.runtimeType}',
            );
          },
        ),
      );
      ref.onDispose(() {
        final end = DateTime.now().toUtc();
        unawaited(
          started.then((id) => tracker.end(userId, id, end)).catchError((
            Object error,
          ) {
            debugPrint('Clinical tool usage end failed: ${error.runtimeType}');
          }),
        );
      });
    }
    return UseCalculatorState(
      calculator: calculator,
      definition: definition,
      responses: responses,
    );
  }

  Future<void> saveResponses(Map<String, Object?> responses) async {
    final current = state.valueOrNull;
    final userId = ref.read(authControllerProvider).valueOrNull?.user?.id;
    if (current == null ||
        userId == null ||
        !current.definition.definition.completion.allowResume) {
      return;
    }
    await _repository.saveWorkflow(
      calculatorId: current.calculator.id,
      userId: userId,
      definition: current.definition,
      responses: responses,
    );
    state = AsyncData(
      current.copyWith(responses: Map<String, Object?>.from(responses)),
    );
  }

  Future<void> reload() async {
    ref.invalidateSelf();
    await future;
  }

  String _calculatorTypeValue(CalculatorType type) => switch (type) {
    CalculatorType.calculator => 'calculator',
    CalculatorType.decisionTool => 'decision_tool',
    CalculatorType.checklist => 'checklist',
  };
}
