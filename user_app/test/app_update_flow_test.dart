import 'dart:async';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:in_app_update/in_app_update.dart' as play;
import 'package:shared_preferences/shared_preferences.dart';
import 'package:user_app/app/providers/app_providers.dart';
import 'package:user_app/features/settings/data/app_update_client.dart';
import 'package:user_app/features/settings/presentation/controllers/app_update_controller.dart';

import 'helpers/fake_app_update_client.dart';

void main() {
  late ProviderContainer container;
  late FakeAppUpdateClient client;
  late SharedPreferences prefs;
  late DateTime now;
  late AppUpdateController controller;
  setUp(() async {
    SharedPreferences.setMockInitialValues({});
    prefs = await SharedPreferences.getInstance();
    client = FakeAppUpdateClient();
    now = DateTime.utc(2026, 10, 12);
    container = ProviderContainer(
      overrides: [
        sharedPreferencesProvider.overrideWithValue(prefs),
        appUpdateClientProvider.overrideWithValue(client),
        appUpdateClockProvider.overrideWithValue(() => now),
      ],
    );
    await container.read(appUpdateControllerProvider.future);
    controller = container.read(appUpdateControllerProvider.notifier);
  });
  tearDown(() async {
    container.dispose();
    await client.events.close();
  });
  AppUpdateResult? result() =>
      container.read(appUpdateControllerProvider).valueOrNull;

  test('checks once daily; manual checks bypass the daily interval', () async {
    expect(await controller.check(automatic: true), AppUpdateResult.available);
    expect(
      client.flexibleStarts,
      0,
      reason: 'Discovery does not download without consent',
    );
    expect(await controller.check(automatic: true), isNull);
    await controller.check();
    expect(client.checks, 2);
    now = now.add(const Duration(days: 1));
    await controller.check(automatic: true);
    expect(client.checks, 3);
  });
  test('offline startup is silent and retries after 15 minutes', () async {
    client.offline = true;
    expect(await controller.check(automatic: true), isNull);
    expect(result(), isNull);
    now = now.add(const Duration(minutes: 10));
    await controller.check(automatic: true);
    expect(client.checks, 1);
    client.offline = false;
    now = now.add(const Duration(minutes: 5));
    expect(await controller.check(automatic: true), AppUpdateResult.available);
    expect(client.checks, 2);
  });
  test(
    'flexible update waits for explicit download and restart taps',
    () async {
      await controller.check();
      await controller.startUpdate();
      expect(result(), AppUpdateResult.readyToInstall);
      expect(client.installs, 0);
      expect(client.immediateStarts, 0);
      await controller.installUpdate();
      expect(client.installs, 1);
    },
  );
  test(
    'cancelled and failed flexible updates never escalate to immediate',
    () async {
      for (final outcome in [
        play.AppUpdateResult.userDeniedUpdate,
        play.AppUpdateResult.inAppUpdateFailed,
      ]) {
        client.outcome = outcome;
        await controller.check();
        await controller.startUpdate();
        expect(client.immediateStarts, 0);
        expect(
          result(),
          outcome == play.AppUpdateResult.userDeniedUpdate
              ? AppUpdateResult.cancelled
              : AppUpdateResult.failed,
        );
        expect(prefs.getBool('app_updates_pending_download'), false);
      }
    },
  );
  test(
    'critical updates use immediate flow only after an explicit tap',
    () async {
      client.info = updateInfo(priority: 4);
      await controller.check(automatic: true);
      expect(result(), AppUpdateResult.criticalAvailable);
      expect(client.immediateStarts, 0);
      await controller.startUpdate();
      expect(client.immediateStarts, 1);
      expect(client.flexibleStarts, 0);
    },
  );
  test(
    'pending download is recovered on resume despite the daily interval',
    () async {
      await controller.check(automatic: true);
      await prefs.setBool('app_updates_pending_download', true);
      client.info = updateInfo(status: play.InstallStatus.downloaded);
      await controller.check(automatic: true);
      expect(result(), AppUpdateResult.readyToInstall);
      expect(client.installs, 0);
      expect(client.checks, 2);
    },
  );
  test('failed installation leaves an explicit retry available', () async {
    client.info = updateInfo(status: play.InstallStatus.downloaded);
    await controller.check();
    client.failInstall = true;
    await controller.installUpdate();
    expect(result(), AppUpdateResult.readyToInstall);
    expect(prefs.getBool('app_updates_pending_download'), true);
  });
  test('live download and cancellation events are contained', () async {
    await controller.check();
    client.events.add(play.InstallStatus.downloaded);
    await Future<void>.delayed(Duration.zero);
    expect(result(), AppUpdateResult.readyToInstall);
    client.events.addError(StateError('Plugin stream disconnected'));
    client.events.add(play.InstallStatus.canceled);
    await Future<void>.delayed(Duration.zero);
    expect(result(), AppUpdateResult.cancelled);
    expect(prefs.getBool('app_updates_pending_download'), false);
  });
  test(
    'offline pending-download recovery also respects retry backoff',
    () async {
      await prefs.setBool('app_updates_pending_download', true);
      client.offline = true;
      await controller.check(automatic: true);
      now = now.add(const Duration(minutes: 10));
      await controller.check(automatic: true);
      expect(client.checks, 1);
      client.offline = false;
      client.info = updateInfo(status: play.InstallStatus.downloaded);
      now = now.add(const Duration(minutes: 5));
      expect(
        await controller.check(automatic: true),
        AppUpdateResult.readyToInstall,
      );
    },
  );
  test('sideloader never invokes Play update APIs', () async {
    client.playInstall = false;
    expect(await controller.check(), AppUpdateResult.unsupported);
    expect(client.checks, 0);
  });
  test('Firebase tester build stays on its own update channel', () async {
    client.firebaseTesterBuild = true;
    expect(
      await controller.check(automatic: true),
      AppUpdateResult.testerAlertsDisabled,
    );
    expect(client.lastAutomatic, true);
    expect(await controller.check(), AppUpdateResult.testerCheckComplete);
    expect(client.lastAutomatic, false);
    expect(client.checks, 0);
  });
  test('concurrent resume events share one check', () async {
    client.blockedCheck = Completer<play.AppUpdateInfo>();
    final first = controller.check(automatic: true);
    await Future<void>.delayed(Duration.zero);
    expect(await controller.check(automatic: true), isNull);
    client.blockedCheck!.complete(updateInfo());
    await first;
    expect(client.checks, 1);
  });
}
