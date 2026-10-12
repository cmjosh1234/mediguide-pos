import 'dart:io';
import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:in_app_update/in_app_update.dart' as play;

const firebaseTesterUpdates =
    String.fromEnvironment('MEDIGUIDE_DISTRIBUTION_CHANNEL') == 'firebase';
final appUpdateClientProvider = Provider<AppUpdateClient>(
  (_) => const PlatformAppUpdateClient(),
);
final appUpdateClockProvider = Provider<DateTime Function()>(
  (_) => DateTime.now,
);

abstract interface class AppUpdateClient {
  bool get supported;
  bool get firebaseTesterBuild;
  Future<bool> isPlayInstall();
  Future<play.AppUpdateInfo> check();
  Future<play.AppUpdateResult> startFlexible();
  Future<play.AppUpdateResult> startImmediate();
  Future<void> completeFlexible();
  Stream<play.InstallStatus> get installStatus;
  Future<bool> checkTesterUpdates({required bool automatic});
}

class PlatformAppUpdateClient implements AppUpdateClient {
  const PlatformAppUpdateClient();
  static const _channel = MethodChannel('mediguide/app_updates');
  @override
  bool get supported => !kIsWeb && Platform.isAndroid;
  @override
  bool get firebaseTesterBuild => firebaseTesterUpdates;
  @override
  Future<bool> isPlayInstall() async =>
      await _channel.invokeMethod<String>('installerPackage') ==
      'com.android.vending';
  @override
  Future<play.AppUpdateInfo> check() => play.InAppUpdate.checkForUpdate();
  @override
  Future<play.AppUpdateResult> startFlexible() =>
      play.InAppUpdate.startFlexibleUpdate();
  @override
  Future<play.AppUpdateResult> startImmediate() =>
      play.InAppUpdate.performImmediateUpdate();
  @override
  Future<void> completeFlexible() =>
      _channel.invokeMethod<void>('completePlayUpdate');
  @override
  Stream<play.InstallStatus> get installStatus =>
      play.InAppUpdate.installUpdateListener;
  @override
  Future<bool> checkTesterUpdates({required bool automatic}) async =>
      await _channel.invokeMethod<bool>('checkTesterUpdates', {
        'automatic': automatic,
      }) ??
      false;
}
