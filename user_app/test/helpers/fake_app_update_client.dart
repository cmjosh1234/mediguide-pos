import 'dart:async';
import 'package:in_app_update/in_app_update.dart' as play;
import 'package:user_app/features/settings/data/app_update_client.dart';

play.AppUpdateInfo updateInfo({
  play.UpdateAvailability availability =
      play.UpdateAvailability.updateAvailable,
  play.InstallStatus status = play.InstallStatus.unknown,
  int priority = 0,
  bool flexible = true,
}) => play.AppUpdateInfo(
  updateAvailability: availability,
  immediateUpdateAllowed: true,
  immediateAllowedPreconditions: null,
  flexibleUpdateAllowed: flexible,
  flexibleAllowedPreconditions: null,
  availableVersionCode: 70,
  installStatus: status,
  packageName: 'com.mediguide.ug',
  updatePriority: priority,
  clientVersionStalenessDays: 1,
);

class FakeAppUpdateClient implements AppUpdateClient {
  @override
  bool supported = true;
  @override
  bool firebaseTesterBuild = false;
  bool playInstall = true;
  bool offline = false;
  bool failInstall = false;
  int checks = 0,
      flexibleStarts = 0,
      immediateStarts = 0,
      installs = 0,
      testerChecks = 0;
  bool? lastAutomatic;
  play.AppUpdateInfo info = updateInfo();
  play.AppUpdateResult outcome = play.AppUpdateResult.success;
  Completer<play.AppUpdateInfo>? blockedCheck;
  final events = StreamController<play.InstallStatus>.broadcast();
  @override
  Future<bool> isPlayInstall() async => playInstall;
  @override
  Future<play.AppUpdateInfo> check() async {
    checks++;
    if (offline) throw StateError('Offline');
    return blockedCheck?.future ?? info;
  }

  @override
  Future<play.AppUpdateResult> startFlexible() async {
    flexibleStarts++;
    return outcome;
  }

  @override
  Future<play.AppUpdateResult> startImmediate() async {
    immediateStarts++;
    return outcome;
  }

  @override
  Future<void> completeFlexible() async {
    installs++;
    if (failInstall) throw StateError('Install failed');
  }

  @override
  Stream<play.InstallStatus> get installStatus => events.stream;
  @override
  Future<bool> checkTesterUpdates({required bool automatic}) async {
    testerChecks++;
    lastAutomatic = automatic;
    return !automatic;
  }
}
