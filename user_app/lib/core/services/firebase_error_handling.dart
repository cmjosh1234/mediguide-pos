import 'dart:async';

import 'package:flutter/foundation.dart';

/// Remote Config is optional: retain active values when its update channel or
/// activation fails, and keep listening so later updates can recover.
StreamSubscription<T> listenRemoteConfigUpdates<T>(
  Stream<T> updates, {
  required Future<void> Function() onUpdate,
}) {
  return updates.listen(
    (_) async {
      try {
        await onUpdate();
      } catch (error) {
        debugPrint(
          'Remote Config update failed; active values retained: $error',
        );
      }
    },
    onError: (Object error, StackTrace stack) {
      debugPrint(
        'Remote Config connection failed; active values retained: $error',
      );
    },
  );
}

/// A reporting failure must not re-enter the global fatal-error handler.
Future<void> reportCrashQuietly(Future<void> Function() report) async {
  try {
    await report();
  } catch (_) {
    debugPrint('Crashlytics could not record an error');
  }
}
