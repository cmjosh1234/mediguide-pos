import 'dart:async';
import 'dart:math';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:user_app/app/providers/app_providers.dart';
import 'package:user_app/features/authentication/presentation/controllers/auth_controller.dart';
import 'package:user_app/shared/providers/connectivity_provider.dart';
import 'calculator_local_repository.dart';
import 'calculator_repository.dart';

final calculatorUsageTrackerProvider = Provider<CalculatorUsageTracker>((ref) {
  var disposed = false;
  final tracker = CalculatorUsageTracker(
    ref.read(calculatorRepositoryProvider),
    ref.read(calculatorLocalRepositoryProvider),
    () => disposed
        ? null
        : ref.read(authControllerProvider).valueOrNull?.user?.id,
  );
  ref.onDispose(() => disposed = true);
  return tracker;
});

// The app root owns retries, even after the tool page has been disposed.
final calculatorUsageSyncProvider = Provider<void>((ref) {
  void sync() => unawaited(ref.read(calculatorUsageTrackerProvider).sync());
  ref.listen(authControllerProvider, (_, next) {
    if (next.valueOrNull?.user != null) sync();
  });
  ref.listen(connectivityProvider, (_, next) {
    if (next.valueOrNull == true) sync();
  });
  final timer = Timer.periodic(const Duration(minutes: 1), (_) => sync());
  ref.onDispose(timer.cancel);
  sync();
});

class CalculatorUsageTracker {
  CalculatorUsageTracker(this.repository, this.local, this.currentUserId);
  final CalculatorRepository repository;
  final CalculatorLocalRepository local;
  final String? Function() currentUserId;
  Future<void> _writes = Future.value();
  bool _syncing = false;
  Completer<void>? _completion;
  bool _syncAgain = false;

  // Serialize only local writes. A slow network must not delay persisting an end.
  Future<T> _write<T>(Future<T> Function() work) {
    final result = _writes.then((_) => work());
    _writes = result.then<void>(
      (_) {},
      onError: (Object error, StackTrace stack) {
        debugPrint('Clinical tool usage cache failed: ${error.runtimeType}');
      },
    );
    return result;
  }

  Future<String> begin({
    required String userId,
    required String calculatorId,
    required String versionId,
    required String calculatorType,
    required DateTime start,
  }) async {
    final random = Random.secure();
    final id = List.generate(
      16,
      (_) => random.nextInt(256).toRadixString(16).padLeft(2, '0'),
    ).join();
    await _write(
      () => local.saveUsage(userId, id, {
        'id': id,
        'calculator_id': calculatorId,
        'version_id': versionId,
        'calculator_type': calculatorType,
        'session_start': start.toUtc().toIso8601String(),
      }),
    );
    unawaited(sync());
    return id;
  }

  Future<void> end(String userId, String id, DateTime end) async {
    await _write(() async {
      final pending = await local.getUsage(userId, id);
      if (pending == null || pending['session_end'] != null) return;
      pending['session_end'] = end.toUtc().toIso8601String();
      await local.saveUsage(userId, id, pending);
    });
    unawaited(sync());
  }

  Future<void> sync() async {
    if (_syncing) {
      _syncAgain = true;
      return _completion!.future;
    }
    _syncing = true;
    _completion = Completer<void>();
    try {
      do {
        _syncAgain = false;
        final userId = currentUserId();
        if (userId == null) return;
        await _writes;
        final sessions = await local.pendingUsage(userId);
        for (final session in sessions) {
          if (currentUserId() != userId) return;
          try {
            final id = session['id'] as String;
            var usageId = session['server_usage_id'] as String?;
            if (usageId == null) {
              final record = await repository.startUsage(
                calculatorId: session['calculator_id'] as String,
                versionId: session['version_id'] as String,
                calculatorType: session['calculator_type'] as String,
                sessionStart: session['session_start'] as String,
                idempotencyKey: id,
              );
              usageId = record.id;
            }
            final pending = await _write(() async {
              final latest = await local.getUsage(userId, id);
              if (latest == null) return null;
              latest['server_usage_id'] = usageId;
              await local.saveUsage(userId, id, latest);
              return latest;
            });
            if (pending == null || pending['session_end'] == null) continue;
            if (currentUserId() != userId) return;
            await repository.finishUsage(
              usageId: usageId,
              sessionEnd: pending['session_end'] as String,
            );
            await _write(() => local.removeUsage(userId, id));
          } catch (error) {
            debugPrint(
              'Clinical tool usage session deferred: ${error.runtimeType}',
            );
          }
        }
      } while (_syncAgain);
    } catch (error) {
      // Retain the persisted payload and key for the next reconnect/timer retry.
      debugPrint('Clinical tool usage sync deferred: ${error.runtimeType}');
    } finally {
      _syncing = false;
      _completion!.complete();
      _completion = null;
    }
  }
}
