import 'dart:async';

import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:user_app/core/services/firebase_error_handling.dart';

void main() {
  test(
    'Remote Config network errors retain values and allow later updates',
    () async {
      final updates = StreamController<int>();
      var active = 'cached';
      final recovered = Completer<void>();
      final subscription = listenRemoteConfigUpdates(
        updates.stream,
        onUpdate: () async {
          active = 'updated';
          recovered.complete();
        },
      );
      updates.addError(
        PlatformException(
          code: 'firebase_remote_config',
          message: 'Unable to connect to the server.',
        ),
      );
      await Future<void>.delayed(Duration.zero);
      expect(active, 'cached');
      updates.add(1);
      await recovered.future;
      expect(active, 'updated');
      await subscription.cancel();
      await updates.close();
    },
  );

  test(
    'failed async activation is contained and the listener recovers',
    () async {
      final updates = StreamController<int>();
      var attempts = 0;
      var active = 'cached';
      final failed = Completer<void>();
      final recovered = Completer<void>();
      final subscription = listenRemoteConfigUpdates(
        updates.stream,
        onUpdate: () async {
          if (++attempts == 1) {
            await Future<void>.delayed(Duration.zero);
            failed.complete();
            throw PlatformException(code: 'firebase_remote_config');
          }
          active = 'updated';
          recovered.complete();
        },
      );
      updates.add(1);
      await failed.future;
      expect(active, 'cached');
      updates.add(2);
      await recovered.future;
      expect(active, 'updated');
      await subscription.cancel();
      await updates.close();
    },
  );

  test(
    'Crashlytics synchronous and asynchronous failures do not escape',
    () async {
      await reportCrashQuietly(
        () => throw PlatformException(code: 'crashlytics'),
      );
      await reportCrashQuietly(() async {
        await Future<void>.delayed(Duration.zero);
        throw PlatformException(code: 'crashlytics');
      });
      var recorded = false;
      await reportCrashQuietly(() async {
        recorded = true;
      });
      expect(recorded, isTrue);
    },
  );
}
