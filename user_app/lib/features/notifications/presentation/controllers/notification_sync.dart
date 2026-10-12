import 'dart:async';

import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:user_app/app/providers/app_providers.dart';
import 'package:user_app/shared/providers/connectivity_provider.dart';

/// Serializes catch-up requests and limits network traffic on repeated resumes.
/// A failed request gets a short backoff; an explicit refresh bypasses it.
class NotificationSync {
  NotificationSync({required this.refresh, DateTime Function()? now})
    : now = now ?? DateTime.now;
  final Future<bool> Function() refresh;
  final DateTime Function() now;
  Future<void>? _pending;
  DateTime? _nextCheck;
  bool _disposed = false;
  bool _refreshAgain = false;
  bool _paused = false;
  int _failures = 0;
  Timer? _retryTimer;
  bool get active => !_paused && !_disposed;

  Future<void> sync({bool force = false}) {
    if (!active) return Future.value();
    if (_pending != null) {
      if (force) _refreshAgain = true;
      return _pending!;
    }
    if (!force && _nextCheck != null && now().isBefore(_nextCheck!)) {
      if (_retryTimer == null) _scheduleRetry(_nextCheck!.difference(now()));
      return Future.value();
    }
    _retryTimer?.cancel();
    _retryTimer = null;
    return _pending = Future<void>.sync(
      _runLoop,
    ).whenComplete(() => _pending = null);
  }

  Future<void> _runLoop() async {
    do {
      _refreshAgain = false;
      await _run();
    } while (_refreshAgain && active);
  }

  Future<void> _run() async {
    var success = false;
    try {
      success = await refresh();
    } catch (_) {
      // Notification catch-up must never interrupt clinical work.
    } finally {
      if (success) {
        _failures = 0;
        _nextCheck = now().add(const Duration(minutes: 1));
      } else {
        final seconds = 15 << (_failures > 5 ? 5 : _failures);
        _failures++;
        final delay = Duration(seconds: seconds > 300 ? 300 : seconds);
        _nextCheck = now().add(delay);
        _scheduleRetry(delay);
      }
    }
  }

  void _scheduleRetry(Duration delay) {
    if (!active) return;
    _retryTimer?.cancel();
    _retryTimer = Timer(delay, () {
      _retryTimer = null;
      unawaited(sync(force: true));
    });
  }

  void pause() {
    _paused = true;
    _retryTimer?.cancel();
    _retryTimer = null;
  }

  Future<void> resume() {
    _paused = false;
    return sync();
  }

  void dispose() {
    _disposed = true;
    _retryTimer?.cancel();
  }
}

final notificationSyncProvider = Provider<NotificationSync>((ref) {
  final repository = ref.watch(notificationRepositoryProvider);
  final firebase = ref.read(firebaseServiceProvider);
  var disposed = false;
  final sync = NotificationSync(
    refresh: () async {
      if (repository.userId.trim().isEmpty) return true;
      await repository.list(page: 1, perPage: 30);
      if (disposed || repository.status.value.offline) return false;
      ref.read(notificationInboxRefreshProvider.notifier).state++;
      ref.invalidate(notificationUnreadCountProvider);
      await firebase.updateBadgeCount(await repository.unreadCount());
      return !repository.status.value.offline;
    },
  );
  final lifecycle = AppLifecycleListener(
    onPause: sync.pause,
    onResume: () {
      unawaited(sync.resume());
      unawaited(firebase.registerCurrentDevice());
      unawaited(firebase.syncOutbreakTopicPreferenceFromServer());
    },
  );
  ref.listen(connectivityProvider, (previous, next) {
    if (sync.active &&
        previous?.valueOrNull == false &&
        next.valueOrNull == true) {
      unawaited(sync.sync(force: true));
      unawaited(firebase.registerCurrentDevice());
      unawaited(firebase.syncOutbreakTopicPreferenceFromServer());
    }
  });
  ref.onDispose(() {
    disposed = true;
    lifecycle.dispose();
    sync.dispose();
  });
  // Let provider construction finish before updating other providers.
  scheduleMicrotask(() => unawaited(sync.sync()));
  return sync;
});
