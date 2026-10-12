import 'dart:async';
import 'package:flutter_test/flutter_test.dart';
import 'package:user_app/features/notifications/presentation/controllers/notification_sync.dart';

void main() {
  testWidgets(
    'a throttled resume gets one delayed catch-up without continuous polling',
    (tester) async {
      var calls = 0;
      final sync = NotificationSync(
        refresh: () async {
          calls++;
          return true;
        },
      );
      addTearDown(sync.dispose);
      await sync.sync();
      await sync.sync();
      expect(calls, 1);
      await tester.pump(const Duration(minutes: 1));
      await tester.pump();
      expect(calls, 2);
      await tester.pump(const Duration(minutes: 1));
      expect(calls, 2);
    },
  );
  testWidgets(
    'failed catch-up retries automatically while active and pauses in the background',
    (tester) async {
      var calls = 0;
      final sync = NotificationSync(
        refresh: () async {
          calls++;
          return calls >= 2;
        },
      );
      addTearDown(sync.dispose);
      await sync.sync();
      sync.pause();
      await tester.pump(const Duration(seconds: 15));
      expect(calls, 1);
      await sync.resume();
      await tester.pump(const Duration(seconds: 15));
      await tester.pump();
      expect(calls, 2);
    },
  );
  test(
    'resume checks are throttled while manual checks bypass the interval',
    () async {
      var now = DateTime.utc(2026, 10, 12);
      var calls = 0;
      final sync = NotificationSync(
        now: () => now,
        refresh: () async {
          calls++;
          return true;
        },
      );
      addTearDown(sync.dispose);
      await sync.sync();
      await sync.sync();
      expect(calls, 1);
      await sync.sync(force: true);
      expect(calls, 2);
      now = now.add(const Duration(minutes: 1));
      await sync.sync();
      expect(calls, 3);
    },
  );
  test(
    'offline and synchronous failures are contained and can be retried',
    () async {
      var now = DateTime.utc(2026, 10, 12);
      var calls = 0;
      final sync = NotificationSync(
        now: () => now,
        refresh: () {
          calls++;
          throw StateError('offline');
        },
      );
      addTearDown(sync.dispose);
      await sync.sync();
      await sync.sync();
      expect(calls, 1);
      now = now.add(const Duration(seconds: 15));
      await sync.sync();
      expect(calls, 2);
    },
  );
  test(
    'a push arriving during catch-up schedules a follow-up refresh',
    () async {
      final firstDone = Completer<bool>();
      var calls = 0;
      final sync = NotificationSync(
        refresh: () {
          calls++;
          return calls == 1 ? firstDone.future : Future.value(true);
        },
      );
      addTearDown(sync.dispose);
      final pending = sync.sync();
      sync.sync(force: true);
      firstDone.complete(true);
      await pending;
      expect(calls, 2);
    },
  );
  test(
    'overlapping automatic checks share one request and disposal stops new work',
    () async {
      final completed = Completer<bool>();
      var calls = 0;
      final sync = NotificationSync(
        refresh: () {
          calls++;
          return completed.future;
        },
      );
      addTearDown(sync.dispose);
      final first = sync.sync();
      final second = sync.sync();
      expect(identical(first, second), true);
      expect(calls, 1);
      completed.complete(true);
      await first;
      sync.dispose();
      await sync.sync(force: true);
      expect(calls, 1);
    },
  );
}
