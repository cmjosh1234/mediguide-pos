import 'dart:async';
import 'package:in_app_update/in_app_update.dart' as play;
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:user_app/app/providers/app_providers.dart';
import 'package:user_app/features/settings/data/app_update_client.dart';

part 'app_update_controller.g.dart';

enum AppUpdateResult {
  unsupported,
  upToDate,
  available,
  criticalAvailable,
  storeUpdateAvailable,
  downloading,
  readyToInstall,
  installing,
  cancelled,
  failed,
  testerAlertsDisabled,
  testerCheckComplete,
}

const _lastCheckKey = 'app_updates_last_success';
const _lastAttemptKey = 'app_updates_last_attempt';
const _pendingKey = 'app_updates_pending_download';
const _lastFailureKey = 'app_updates_last_failure';

@Riverpod(keepAlive: true)
class AppUpdateController extends _$AppUpdateController {
  bool _busy = false;
  bool _disposed = false;
  play.AppUpdateInfo? _info;
  StreamSubscription<play.InstallStatus>? _subscription;

  @override
  FutureOr<AppUpdateResult?> build() {
    _disposed = false;
    ref.onDispose(() {
      _disposed = true;
      unawaited(_subscription?.cancel());
    });
    return null;
  }

  void _set(AppUpdateResult value) {
    if (!_disposed) state = AsyncData(value);
  }

  Future<AppUpdateResult?> check({bool automatic = false}) async {
    if (_busy) return null;
    final client = ref.read(appUpdateClientProvider);
    if (!client.supported) {
      _set(AppUpdateResult.unsupported);
      return AppUpdateResult.unsupported;
    }
    _busy = true;
    SharedPreferences? preferences;
    try {
      final prefs = ref.read(sharedPreferencesProvider);
      preferences = prefs;
      final now = ref.read(appUpdateClockProvider)().millisecondsSinceEpoch;
      final pending = prefs.getBool(_pendingKey) ?? false;
      bool recent(String key, Duration interval) {
        final saved = prefs.getInt(key);
        return saved != null &&
            now - saved >= 0 &&
            now - saved < interval.inMilliseconds;
      }

      if (automatic &&
          (recent(_lastFailureKey, const Duration(minutes: 15)) ||
              (!pending &&
                  (recent(_lastCheckKey, const Duration(days: 1)) ||
                      recent(_lastAttemptKey, const Duration(minutes: 15)))))) {
        return null;
      }
      await prefs.setInt(_lastAttemptKey, now);
      if (_disposed) return null;
      if (!automatic) state = const AsyncLoading();
      if (client.firebaseTesterBuild) {
        final checked = await client.checkTesterUpdates(automatic: automatic);
        await prefs.setInt(_lastCheckKey, now);
        final result = checked
            ? AppUpdateResult.testerCheckComplete
            : AppUpdateResult.testerAlertsDisabled;
        _set(result);
        return result;
      }
      if (!await client.isPlayInstall().timeout(const Duration(seconds: 5))) {
        _set(AppUpdateResult.unsupported);
        return AppUpdateResult.unsupported;
      }
      final info = await client.check().timeout(const Duration(seconds: 25));
      _info = info;
      await prefs.remove(_lastFailureKey);
      await prefs.setInt(_lastCheckKey, now);
      if (_disposed) return null;
      _subscription ??= client.installStatus.listen(
        _onInstallStatus,
        onError: (Object error, StackTrace stack) {
          // Optional updater failures must not reach the global fatal handler.
        },
      );
      final result = _resultFor(info);
      await prefs.setBool(
        _pendingKey,
        result == AppUpdateResult.downloading ||
            result == AppUpdateResult.readyToInstall ||
            result == AppUpdateResult.installing,
      );
      _set(result);
      return result;
    } catch (_) {
      // Automatic failures stay silent and retry after a short backoff.
      if (!_disposed && preferences != null) {
        try {
          await preferences.setInt(
            _lastFailureKey,
            ref.read(appUpdateClockProvider)().millisecondsSinceEpoch,
          );
        } catch (_) {
          /* Optional local bookkeeping. */
        }
      }
      if (!automatic) _set(AppUpdateResult.failed);
      return automatic ? null : AppUpdateResult.failed;
    } finally {
      _busy = false;
    }
  }

  AppUpdateResult _resultFor(play.AppUpdateInfo info) {
    switch (info.installStatus) {
      case play.InstallStatus.downloaded:
        return AppUpdateResult.readyToInstall;
      case play.InstallStatus.downloading:
      case play.InstallStatus.pending:
        return AppUpdateResult.downloading;
      case play.InstallStatus.installing:
        return AppUpdateResult.installing;
      default:
        break;
    }
    if (info.updateAvailability ==
            play.UpdateAvailability.developerTriggeredUpdateInProgress &&
        info.immediateUpdateAllowed) {
      return AppUpdateResult.criticalAvailable;
    }
    if (info.updateAvailability != play.UpdateAvailability.updateAvailable) {
      return AppUpdateResult.upToDate;
    }
    if (info.updatePriority >= 4 && info.immediateUpdateAllowed) {
      return AppUpdateResult.criticalAvailable;
    }
    return info.flexibleUpdateAllowed
        ? AppUpdateResult.available
        : AppUpdateResult.storeUpdateAvailable;
  }

  void _onInstallStatus(play.InstallStatus status) {
    final result = switch (status) {
      play.InstallStatus.downloaded => AppUpdateResult.readyToInstall,
      play.InstallStatus.downloading ||
      play.InstallStatus.pending => AppUpdateResult.downloading,
      play.InstallStatus.installing => AppUpdateResult.installing,
      play.InstallStatus.installed => AppUpdateResult.upToDate,
      play.InstallStatus.canceled => AppUpdateResult.cancelled,
      play.InstallStatus.failed => AppUpdateResult.failed,
      play.InstallStatus.unknown => null,
    };
    if (result != null) {
      _set(result);
      if (result == AppUpdateResult.cancelled ||
          result == AppUpdateResult.failed ||
          result == AppUpdateResult.upToDate) {
        unawaited(_clearPendingQuietly());
      }
    }
  }

  Future<void> _clearPendingQuietly() async {
    if (_disposed) return;
    try {
      await ref.read(sharedPreferencesProvider).setBool(_pendingKey, false);
    } catch (_) {
      /* Best-effort update bookkeeping. */
    }
  }

  /// Start only after an explicit tap; never escalate a failed/declined
  /// flexible update into an immediate update.
  Future<void> startUpdate() async {
    if (_busy || _disposed || _info == null) return;
    final immediate = state.valueOrNull == AppUpdateResult.criticalAvailable;
    if (!immediate && state.valueOrNull != AppUpdateResult.available) return;
    _busy = true;
    final prefs = ref.read(sharedPreferencesProvider);
    try {
      await prefs.setBool(_pendingKey, true);
      _set(AppUpdateResult.downloading);
      final client = ref.read(appUpdateClientProvider);
      final result = immediate
          ? await client.startImmediate()
          : await client.startFlexible();
      switch (result) {
        case play.AppUpdateResult.success:
          _set(
            immediate
                ? AppUpdateResult.upToDate
                : AppUpdateResult.readyToInstall,
          );
          if (immediate) await prefs.setBool(_pendingKey, false);
        case play.AppUpdateResult.userDeniedUpdate:
          await prefs.setBool(_pendingKey, false);
          _set(AppUpdateResult.cancelled);
        case play.AppUpdateResult.inAppUpdateFailed:
          await prefs.setBool(_pendingKey, false);
          _set(AppUpdateResult.failed);
      }
    } catch (_) {
      _set(AppUpdateResult.failed);
    } finally {
      _busy = false;
    }
  }

  Future<void> installUpdate() async {
    if (_busy || state.valueOrNull != AppUpdateResult.readyToInstall) return;
    _busy = true;
    _set(AppUpdateResult.installing);
    try {
      await ref.read(appUpdateClientProvider).completeFlexible();
      await ref.read(sharedPreferencesProvider).setBool(_pendingKey, false);
    } catch (_) {
      // Keep the downloaded update available for another explicit attempt.
      _set(AppUpdateResult.readyToInstall);
    } finally {
      _busy = false;
    }
  }

  void reset() {
    if (!_busy) state = const AsyncData(null);
  }
}
