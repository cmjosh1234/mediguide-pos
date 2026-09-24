import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:math';

import 'package:firebase_analytics/firebase_analytics.dart';
import 'package:firebase_core/firebase_core.dart';
import 'package:firebase_crashlytics/firebase_crashlytics.dart';
import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:firebase_remote_config/firebase_remote_config.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart' show AppLifecycleListener;
import 'package:flutter_local_notifications/flutter_local_notifications.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:user_app/core/config/app_config.dart';
import 'package:user_app/core/config/firebase_config.dart';
import 'package:user_app/core/network/api_client.dart';
import 'package:user_app/features/authentication/data/datasources/auth_remote_datasource.dart';

const _installationKey = 'firebase_installation_id';
const _deviceRecordKey = 'firebase_device_record_id';
const _permissionRequestedKey = 'firebase_notification_permission_requested';
const _outbreakTopicPreferenceKey = 'firebase_outbreak_topic_preference';
const _outbreakTopicSubscribedKey = 'firebase_outbreak_topic_subscribed';
const _publicOutbreakTopic = 'public-outbreaks';

// Channel IDs must match notificationAndroidChannel in the backend
// (internal/services/notification_outbox_service.go). A push naming a channel
// the app never created falls back to the default channel from the manifest.
const _urgentChannel = AndroidNotificationChannel(
  'mediguide_emergency',
  'Urgent alerts',
  description: 'Emergency and urgent clinical alerts',
  importance: Importance.high,
);
const _updatesChannel = AndroidNotificationChannel(
  'mediguide_updates',
  'MediGuide updates',
  description: 'Guideline updates, reminders, and general notifications',
  importance: Importance.defaultImportance,
);
// Created by earlier builds for in-app banners. Nothing is sent to it now.
const _retiredAlertsChannelId = 'mediguide_alerts';

enum AppNotificationPermissionState {
  notDetermined,
  provisional,
  authorized,
  denied,
  permanentlyDenied,
}

@pragma('vm:entry-point')
Future<void> firebaseBackgroundMessageHandler(RemoteMessage message) async {
  if (!MediGuideFirebaseConfig.isConfigured) return;
  await Firebase.initializeApp(
    options: MediGuideFirebaseConfig.currentPlatform,
  );
}

final class MediGuideFirebaseService {
  MediGuideFirebaseService(this._api, this._auth, this._preferences);

  final BackendApiService _api;
  final AuthService _auth;
  final SharedPreferences _preferences;
  final FlutterLocalNotificationsPlugin _localNotifications =
      FlutterLocalNotificationsPlugin();
  final StreamController<RemoteMessage> _openedMessages =
      StreamController.broadcast();
  final StreamController<RemoteMessage> _foregroundMessages =
      StreamController.broadcast();
  final StreamController<AppNotificationPermissionState> _permissionStates =
      StreamController.broadcast();
  final StreamController<bool> _outbreakBannerStates =
      StreamController<bool>.broadcast();

  StreamSubscription<String>? _tokenSubscription;
  StreamSubscription<RemoteMessage>? _foregroundSubscription;
  StreamSubscription<RemoteMessage>? _openedSubscription;
  StreamSubscription<RemoteConfigUpdate>? _remoteConfigSubscription;
  VoidCallback? _authListener;
  AppLifecycleListener? _lifecycleListener;
  RemoteMessage? _initialMessage;
  bool _enabled = false;
  bool _deviceRegistered = false;
  bool _crashReportingEnabled = false;
  AppNotificationPermissionState _permissionState =
      AppNotificationPermissionState.notDetermined;

  bool get enabled => _enabled;
  bool get crashReportingEnabled => _crashReportingEnabled;
  Stream<RemoteMessage> get openedMessages async* {
    final initial = _initialMessage;
    _initialMessage = null;
    if (initial != null) yield initial;
    yield* _openedMessages.stream;
  }

  Stream<RemoteMessage> get foregroundMessages => _foregroundMessages.stream;
  Stream<AppNotificationPermissionState> get permissionStates =>
      _permissionStates.stream;
  AppNotificationPermissionState get permissionState => _permissionState;
  Stream<bool> get outbreakBannerStates async* {
    yield outbreakBannerEnabled;
    yield* _outbreakBannerStates.stream;
  }

  FirebaseRemoteConfig? get remoteConfig =>
      _enabled ? FirebaseRemoteConfig.instance : null;
  bool get aiAssistantEnabled =>
      !_enabled || FirebaseRemoteConfig.instance.getBool('enable_ai_assistant');
  bool get outbreakBannerEnabled =>
      !_enabled ||
      FirebaseRemoteConfig.instance.getBool('outbreak_banner_enabled');
  bool get backendManagedOutbreakHubsEnabled =>
      _enabled &&
      FirebaseRemoteConfig.instance.getBool('api_driven_outbreak_pillars');
  bool get diseaseTaxonomyEnabled =>
      !_enabled ||
      FirebaseRemoteConfig.instance.getBool('disease_taxonomy_enabled');
  bool get diseaseContentAssignmentEnabled =>
      !_enabled ||
      FirebaseRemoteConfig.instance.getBool('disease_content_assignment');
  bool get diseaseHubsEnabled =>
      !_enabled ||
      FirebaseRemoteConfig.instance.getBool('disease_hubs_enabled');
  bool get genericHubsEnabled =>
      !_enabled ||
      FirebaseRemoteConfig.instance.getBool('generic_hubs_enabled');
  bool get guidelineCategoryAssignmentEnabled =>
      !_enabled ||
      FirebaseRemoteConfig.instance.getBool('guideline_category_assignment');
  bool get unifiedDocumentSearchEnabled =>
      !_enabled ||
      FirebaseRemoteConfig.instance.getBool('unified_document_search');
  bool get pillarRagMetadataEnabled =>
      !_enabled || FirebaseRemoteConfig.instance.getBool('pillar_rag_metadata');
  bool get pushNotificationsEnabled =>
      _enabled &&
      FirebaseRemoteConfig.instance.getBool('enable_push_notifications');
  bool get maintenanceMode =>
      _enabled && FirebaseRemoteConfig.instance.getBool('maintenance_mode');
  String get maintenanceMessage => _enabled
      ? FirebaseRemoteConfig.instance.getString('maintenance_message')
      : '';

  Map<String, RemoteConfigValue> get remoteConfigValues =>
      _enabled ? FirebaseRemoteConfig.instance.getAll() : const {};

  Future<void> recordOperationalEvent(
    String name,
    Map<String, Object> parameters,
  ) async {
    if (!_enabled) return;
    try {
      await FirebaseAnalytics.instance.logEvent(
        name: name,
        parameters: parameters,
      );
    } catch (_) {
      // Telemetry must never interrupt access to clinical content.
    }
  }

  DateTime? get remoteConfigLastFetchTime =>
      _enabled ? FirebaseRemoteConfig.instance.lastFetchTime : null;
  RemoteConfigFetchStatus? get remoteConfigLastFetchStatus =>
      _enabled ? FirebaseRemoteConfig.instance.lastFetchStatus : null;

  Future<bool> refreshRemoteConfig() async {
    if (!_enabled) return false;
    final changed = await FirebaseRemoteConfig.instance.fetchAndActivate();
    _outbreakBannerStates.add(outbreakBannerEnabled);
    await _syncPublicOutbreakTopic();
    return changed;
  }

  Future<MediGuideFirebaseService> init() async {
    if (!MediGuideFirebaseConfig.isConfigured ||
        !(Platform.isAndroid || Platform.isIOS)) {
      debugPrint('Firebase disabled: client configuration is absent.');
      return this;
    }

    try {
      await Firebase.initializeApp(
        options: MediGuideFirebaseConfig.currentPlatform,
      );
      _enabled = true;
      await _initializeCrashReporting();
      FirebaseMessaging.onBackgroundMessage(firebaseBackgroundMessageHandler);

      await _initializeRemoteConfig();
      await _initializeLocalNotifications();
      await _initializeMessaging();
      await _syncPublicOutbreakTopic();
    } catch (error, stack) {
      await _disableAfterFailedInit(error, stack);
      return this;
    }

    _auth.beforeLogout = _beforeLogout;
    _authListener = () {
      unawaited(_syncCrashReportingUser());
      if (_auth.currentUser.value != null) {
        unawaited(registerCurrentDevice());
        unawaited(syncOutbreakTopicPreferenceFromServer());
      }
    };
    _auth.currentUser.addListener(_authListener!);
    _lifecycleListener = AppLifecycleListener(
      onResume: () => unawaited(_refreshAfterResume()),
    );
    if (_auth.currentUser.value != null) {
      // Both calls reach the network. Startup must not wait on them, and an
      // offline launch must still reach the first screen.
      unawaited(registerCurrentDevice());
      unawaited(syncOutbreakTopicPreferenceFromServer());
    }
    return this;
  }

  /// Falls back to the same Firebase-disabled mode that builds without client
  /// configuration run in, so a failed setup never keeps the app from starting.
  Future<void> _disableAfterFailedInit(Object error, StackTrace stack) async {
    debugPrint('Firebase initialization failed; continuing without it: $error');
    _enabled = false;
    _initialMessage = null;
    await _tokenSubscription?.cancel();
    await _foregroundSubscription?.cancel();
    await _openedSubscription?.cancel();
    await _remoteConfigSubscription?.cancel();
    await _recordNonFatalQuietly(error, stack, 'Firebase initialization');
  }

  Future<void> _initializeCrashReporting() async {
    try {
      final crashlytics = FirebaseCrashlytics.instance;
      await crashlytics.setCrashlyticsCollectionEnabled(true);
      final package = await PackageInfo.fromPlatform();
      await crashlytics.setCustomKey(
        'environment',
        AppConfig.current.flavor.name,
      );
      await crashlytics.setCustomKey('platform', Platform.operatingSystem);
      await crashlytics.setCustomKey('app_version', package.version);
      await crashlytics.setCustomKey('build_number', package.buildNumber);

      FlutterError.onError = (details) {
        FlutterError.presentError(details);
        unawaited(crashlytics.recordFlutterFatalError(details));
      };
      PlatformDispatcher.instance.onError = (error, stack) {
        unawaited(crashlytics.recordError(error, stack, fatal: true));
        return true;
      };
      _crashReportingEnabled = true;
      await _syncCrashReportingUser();
    } catch (error) {
      debugPrint('Crashlytics initialization failed: $error');
    }
  }

  Future<void> _syncCrashReportingUser() async {
    if (!_crashReportingEnabled) return;
    final userID = _auth.currentUser.value?.id ?? '';
    try {
      await FirebaseCrashlytics.instance.setUserIdentifier(userID);
    } catch (_) {
      // Crash reporting must never interrupt access to clinical content.
    }
  }

  Future<void> recordNonFatalError(
    Object error,
    StackTrace stack, {
    String? reason,
  }) async {
    if (!_crashReportingEnabled) return;
    await FirebaseCrashlytics.instance.recordError(
      error,
      stack,
      reason: reason,
      fatal: false,
    );
  }

  Future<void> _beforeLogout() async {
    await unregisterCurrentDevice();
    if (_crashReportingEnabled) {
      await FirebaseCrashlytics.instance.setUserIdentifier('');
    }
  }

  Future<void> _initializeRemoteConfig() async {
    final config = FirebaseRemoteConfig.instance;
    await config.setDefaults(const {
      'maintenance_mode': false,
      'maintenance_message': '',
      'enable_ai_assistant': true,
      'enable_push_notifications': true,
      'minimum_supported_version': '',
      'outbreak_banner_enabled': true,
      // Enable after configured hubs are verified in production. Until then
      // every outbreak retains the existing compatibility presentation.
      'api_driven_outbreak_pillars': false,
      // Public disease and hub discovery is a stable reader capability. It
      // remains remotely reversible, but a missing/failing Remote Config
      // value must not hide already-published clinical content.
      'disease_taxonomy_enabled': true,
      'disease_content_assignment': false,
      'disease_hubs_enabled': true,
      'generic_hubs_enabled': true,
      'guideline_category_assignment': false,
      'unified_document_search': false,
      'pillar_rag_metadata': false,
    });
    await config.setConfigSettings(
      RemoteConfigSettings(
        fetchTimeout: const Duration(seconds: 30),
        minimumFetchInterval: kDebugMode
            ? const Duration(minutes: 5)
            : const Duration(hours: 1),
      ),
    );
    try {
      await config.fetchAndActivate();
    } catch (error) {
      debugPrint('Remote Config fetch failed; defaults remain active: $error');
    }
    _remoteConfigSubscription = config.onConfigUpdated.listen((_) async {
      await config.activate();
      _outbreakBannerStates.add(outbreakBannerEnabled);
      await _syncPublicOutbreakTopic();
    });
  }

  Future<void> _initializeLocalNotifications() async {
    const android = AndroidInitializationSettings('@drawable/ic_notification');
    // The plugin asks for permission on iOS by default. The app asks only
    // from its own prompt, after explaining why.
    const darwin = DarwinInitializationSettings(
      requestAlertPermission: false,
      requestBadgePermission: false,
      requestSoundPermission: false,
    );
    await _localNotifications.initialize(
      settings: const InitializationSettings(android: android, iOS: darwin),
      onDidReceiveNotificationResponse: (response) {
        final message = messageFromNotificationPayload(response.payload);
        if (message != null) _openedMessages.add(message);
      },
    );
    // A banner shown while the app was open can be tapped after the app has
    // been closed. That tap launches the app and arrives only here.
    final launch = await _localNotifications.getNotificationAppLaunchDetails();
    if (launch != null && launch.didNotificationLaunchApp) {
      _initialMessage = messageFromNotificationPayload(
        launch.notificationResponse?.payload,
      );
    }
    final androidNotifications = _localNotifications
        .resolvePlatformSpecificImplementation<
          AndroidFlutterLocalNotificationsPlugin
        >();
    await androidNotifications?.createNotificationChannel(_urgentChannel);
    await androidNotifications?.createNotificationChannel(_updatesChannel);
    await androidNotifications?.deleteNotificationChannel(
      channelId: _retiredAlertsChannelId,
    );
  }

  Future<void> _initializeMessaging() async {
    final messaging = FirebaseMessaging.instance;
    _setPermissionState(await _readPermissionState());
    await FirebaseAnalytics.instance.setAnalyticsCollectionEnabled(true);

    _tokenSubscription = messaging.onTokenRefresh.listen((_) {
      _deviceRegistered = false;
      unawaited(registerCurrentDevice());
    });
    _foregroundSubscription = FirebaseMessaging.onMessage.listen(
      _showForegroundMessage,
    );
    _openedSubscription = FirebaseMessaging.onMessageOpenedApp.listen(
      _openedMessages.add,
    );
    final initial = await messaging.getInitialMessage();
    if (initial != null) _initialMessage = initial;
  }

  Future<void> _showForegroundMessage(RemoteMessage message) async {
    _foregroundMessages.add(message);
    final notification = message.notification;
    if (notification == null) return;
    final channel = foregroundChannelFor(notification.android?.channelId);
    try {
      await _localNotifications.show(
        id: message.messageId.hashCode,
        title: notification.title,
        body: notification.body,
        notificationDetails: NotificationDetails(
          android: AndroidNotificationDetails(
            channel.id,
            channel.name,
            channelDescription: channel.description,
            importance: channel.importance,
            priority: channel.importance == Importance.high
                ? Priority.high
                : Priority.defaultPriority,
            visibility: NotificationVisibility.private,
          ),
          iOS: const DarwinNotificationDetails(),
        ),
        payload: jsonEncode(message.data),
      );
    } catch (error, stack) {
      // The inbox has already been told about the message, so a banner that
      // fails to show loses nothing the user cannot reach.
      debugPrint('Foreground notification could not be shown: $error');
      await _recordNonFatalQuietly(error, stack, 'foreground notification');
    }
  }

  Future<AppNotificationPermissionState> _readPermissionState() async {
    final settings = await FirebaseMessaging.instance.getNotificationSettings();
    return permissionStateFor(
      settings.authorizationStatus,
      previouslyRequested:
          _preferences.getBool(_permissionRequestedKey) ?? false,
    );
  }

  /// The user can change the permission in system Settings while the app is
  /// in the background, and a registration can fail while offline. Both are
  /// picked up here instead of waiting for the next cold start.
  Future<void> _refreshAfterResume() async {
    if (!_enabled) return;
    try {
      final state = await _readPermissionState();
      final changed = state != _permissionState;
      if (changed) {
        _setPermissionState(state);
        await _syncPublicOutbreakTopic();
      }
      if (changed || !_deviceRegistered) await registerCurrentDevice();
    } catch (error) {
      debugPrint('Notification permission refresh failed: $error');
    }
  }

  Future<AppNotificationPermissionState> requestNotificationPermission() async {
    if (!_enabled) return _permissionState;
    // Only called when the user asks for notifications, so iOS must show the
    // system prompt. A provisional request is granted silently and delivers
    // to Notification Center only, without banners or sounds.
    final settings = await FirebaseMessaging.instance.requestPermission(
      alert: true,
      badge: true,
      sound: true,
    );
    await _preferences.setBool(_permissionRequestedKey, true);
    final state = permissionStateFor(
      settings.authorizationStatus,
      previouslyRequested: true,
    );
    _setPermissionState(state);
    if (state == AppNotificationPermissionState.authorized ||
        state == AppNotificationPermissionState.provisional) {
      await registerCurrentDevice();
    }
    await _syncPublicOutbreakTopic();
    return state;
  }

  /// Persists the user's outbreak-alert preference and reconciles the public
  /// Firebase topic. Production never subscribes unless permission, Remote
  /// Config and both notification preferences permit it.
  Future<void> setOutbreakTopicPreference({
    required bool outbreakAlerts,
    required bool pushEnabled,
  }) async {
    await _preferences.setBool(
      _outbreakTopicPreferenceKey,
      outbreakAlerts && pushEnabled,
    );
    await _syncPublicOutbreakTopic();
  }

  Future<void> syncOutbreakTopicPreferenceFromServer() async {
    if (!_enabled || _auth.currentUser.value == null) return;
    try {
      final response = await _api.requestJson(
        '/api/v2/notification-preferences',
        method: 'GET',
      );
      final raw = response['data'];
      if (raw is! Map) return;
      final preferences = Map<String, dynamic>.from(raw);
      await setOutbreakTopicPreference(
        outbreakAlerts: preferences['outbreak_alerts'] != false,
        pushEnabled: preferences['push_enabled'] != false,
      );
    } catch (error) {
      debugPrint('Outbreak topic preference sync failed: $error');
    }
  }

  Future<void> _syncPublicOutbreakTopic() async {
    if (!_enabled) return;
    final shouldSubscribe = shouldSubscribeToPublicOutbreaks(
      remoteEnabled: outbreakBannerEnabled,
      pushEnabled: FirebaseRemoteConfig.instance.getBool(
        'enable_push_notifications',
      ),
      userOptedIn: _preferences.getBool(_outbreakTopicPreferenceKey) ?? false,
      permissionState: _permissionState,
    );
    final subscribed =
        _preferences.getBool(_outbreakTopicSubscribedKey) ?? false;
    if (shouldSubscribe == subscribed) return;
    try {
      if (shouldSubscribe) {
        await FirebaseMessaging.instance.subscribeToTopic(_publicOutbreakTopic);
      } else {
        await FirebaseMessaging.instance.unsubscribeFromTopic(
          _publicOutbreakTopic,
        );
      }
      await _preferences.setBool(_outbreakTopicSubscribedKey, shouldSubscribe);
    } catch (error) {
      debugPrint('Outbreak topic reconciliation failed: $error');
    }
  }

  Future<bool> openNotificationSettings() async {
    if (!_enabled) return false;
    try {
      return await const MethodChannel(
            'mediguide/app_settings',
          ).invokeMethod<bool>('openNotificationSettings') ??
          false;
    } on PlatformException {
      return false;
    }
  }

  static AppNotificationPermissionState permissionStateFor(
    AuthorizationStatus status, {
    required bool previouslyRequested,
  }) {
    return switch (status) {
      AuthorizationStatus.notDetermined =>
        AppNotificationPermissionState.notDetermined,
      AuthorizationStatus.provisional =>
        AppNotificationPermissionState.provisional,
      AuthorizationStatus.authorized =>
        AppNotificationPermissionState.authorized,
      AuthorizationStatus.denied =>
        previouslyRequested
            ? AppNotificationPermissionState.permanentlyDenied
            : AppNotificationPermissionState.denied,
    };
  }

  void _setPermissionState(AppNotificationPermissionState value) {
    _permissionState = value;
    _permissionStates.add(value);
  }

  /// Registers this device for pushes. It never throws: registration is
  /// retried on the next launch, login and token refresh, so a failure only
  /// delays delivery and must not block startup or surface as a crash.
  Future<void> registerCurrentDevice() async {
    try {
      await _registerCurrentDevice();
    } catch (error, stack) {
      debugPrint('Push device registration failed: $error');
      if (isReportableRegistrationFailure(error)) {
        await _recordNonFatalQuietly(error, stack, 'push device registration');
      }
    }
  }

  Future<void> _recordNonFatalQuietly(
    Object error,
    StackTrace stack,
    String reason,
  ) async {
    try {
      await recordNonFatalError(error, stack, reason: reason);
    } catch (_) {
      // Crash reporting must never interrupt access to clinical content.
    }
  }

  Future<void> _registerCurrentDevice() async {
    if (!_enabled || _auth.currentUser.value == null) return;
    if (_permissionState != AppNotificationPermissionState.authorized &&
        _permissionState != AppNotificationPermissionState.provisional) {
      return;
    }
    if (!FirebaseRemoteConfig.instance.getBool('enable_push_notifications')) {
      return;
    }
    if (Platform.isIOS) {
      final apns = await FirebaseMessaging.instance.getAPNSToken();
      if (apns == null) return;
    }
    final token = await FirebaseMessaging.instance.getToken();
    if (token == null || token.isEmpty) return;
    final package = await PackageInfo.fromPlatform();
    final installation = _installationID();
    final response = await _api.requestJson(
      '/api/v2/firebase/devices',
      method: 'POST',
      body: {
        'installation_id': installation,
        'registration_token': token,
        'platform': Platform.isIOS ? 'ios' : 'android',
        'app_version': '${package.version}+${package.buildNumber}',
        'locale': Platform.localeName,
      },
    );
    _deviceRegistered = true;
    final data = response['data'];
    if (data is Map && data['id'] != null) {
      await _preferences.setString(_deviceRecordKey, data['id'].toString());
    }
  }

  Future<void> unregisterCurrentDevice() async {
    if (!_enabled) return;
    _deviceRegistered = false;
    await _preferences.setBool(_outbreakTopicPreferenceKey, false);
    await _syncPublicOutbreakTopic();
    final id = _preferences.getString(_deviceRecordKey);
    if (id != null && id.isNotEmpty) {
      try {
        await _api.requestJson(
          '/api/v2/firebase/devices/${Uri.encodeComponent(id)}',
          method: 'DELETE',
        );
      } catch (_) {}
      await _preferences.remove(_deviceRecordKey);
    }
    try {
      await FirebaseMessaging.instance.deleteToken();
    } catch (_) {}
  }

  String _installationID() {
    final existing = _preferences.getString(_installationKey);
    if (existing != null && existing.isNotEmpty) return existing;
    final random = Random.secure();
    final bytes = List<int>.generate(24, (_) => random.nextInt(256));
    final value = base64UrlEncode(bytes).replaceAll('=', '');
    _preferences.setString(_installationKey, value);
    return value;
  }

  Future<void> dispose() async {
    if (_authListener != null) _auth.currentUser.removeListener(_authListener!);
    _lifecycleListener?.dispose();
    await _tokenSubscription?.cancel();
    await _foregroundSubscription?.cancel();
    await _openedSubscription?.cancel();
    await _remoteConfigSubscription?.cancel();
    await _openedMessages.close();
    await _foregroundMessages.close();
    await _permissionStates.close();
    await _outbreakBannerStates.close();
  }
}

@visibleForTesting
bool shouldSubscribeToPublicOutbreaks({
  required bool remoteEnabled,
  required bool pushEnabled,
  required bool userOptedIn,
  required AppNotificationPermissionState permissionState,
}) {
  final permitted =
      permissionState == AppNotificationPermissionState.authorized ||
      permissionState == AppNotificationPermissionState.provisional;
  return remoteEnabled && pushEnabled && userOptedIn && permitted;
}

/// The channel for a banner shown while the app is open, so it alerts the same
/// way the push would have with the app in the background.
@visibleForTesting
AndroidNotificationChannel foregroundChannelFor(String? requestedChannelId) =>
    requestedChannelId == _urgentChannel.id ? _urgentChannel : _updatesChannel;

/// Rebuilds the push data stored in a local notification's payload, or null
/// when there is nothing to open.
@visibleForTesting
RemoteMessage? messageFromNotificationPayload(String? payload) {
  if (payload == null || payload.isEmpty) return null;
  try {
    final decoded = jsonDecode(payload);
    if (decoded is! Map) return null;
    return RemoteMessage(data: Map<String, dynamic>.from(decoded));
  } on FormatException {
    return null;
  }
}

/// Whether a failed device registration is worth a Crashlytics report.
/// Being offline, timeouts, server outages, rate limits and expired sessions
/// clear up on their own; a rejected request or a platform error does not.
@visibleForTesting
bool isReportableRegistrationFailure(Object error) {
  if (error is! BackendApiException) return true;
  final status = error.statusCode;
  return status >= 400 &&
      status < 500 &&
      status != 401 &&
      status != 408 &&
      status != 429;
}
