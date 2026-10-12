final class NotificationPreferences {
  const NotificationPreferences({
    required this.clinicalContentUpdates,
    required this.outbreakAlerts,
    required this.emergencyAlerts,
    required this.reminders,
    required this.systemNotices,
    required this.productAnnouncements,
    required this.quietHoursEnabled,
    required this.quietHoursTimezone,
    required this.preferredLanguage,
    required this.pushEnabled,
    required this.inAppEnabled,
    this.programAreas = const [],
    this.regionIds = const [],
    this.followedGuidelineIds = const [],
    this.followedOnly = false,
    this.routineDigest = false,
    this.digestTime = '18:00',
    this.quietHoursStart,
    this.quietHoursEnd,
  });

  final List<String> programAreas;
  final List<String> regionIds;
  final List<String> followedGuidelineIds;
  final bool followedOnly;
  final bool routineDigest;
  final String digestTime;
  Map<String, dynamic> get interestsJson => {
    'program_areas': programAreas,
    'region_ids': regionIds,
    'followed_guideline_ids': followedGuidelineIds,
    'followed_only': followedOnly,
  };

  final bool clinicalContentUpdates;
  final bool outbreakAlerts;
  final bool emergencyAlerts;
  final bool reminders;
  final bool systemNotices;
  final bool productAnnouncements;
  final bool quietHoursEnabled;
  final String? quietHoursStart;
  final String? quietHoursEnd;
  final String quietHoursTimezone;
  final String preferredLanguage;
  final bool pushEnabled;
  final bool inAppEnabled;

  factory NotificationPreferences.fromJson(Map<String, dynamic> json) {
    bool flag(String key, {bool fallback = true}) =>
        json[key] is bool ? json[key] as bool : fallback;
    final interests = json['interests'] is Map
        ? json['interests'] as Map
        : const {};
    List<String> values(String key) => (interests[key] as List? ?? const [])
        .whereType<String>()
        .toList(growable: false);
    return NotificationPreferences(
      programAreas: values('program_areas'),
      regionIds: values('region_ids'),
      followedGuidelineIds: values('followed_guideline_ids'),
      followedOnly: interests['followed_only'] == true,
      routineDigest: flag('routine_digest', fallback: false),
      digestTime: json['digest_time']?.toString() ?? '18:00',
      clinicalContentUpdates: flag('clinical_content_updates'),
      outbreakAlerts: flag('outbreak_alerts'),
      emergencyAlerts: flag('emergency_alerts'),
      reminders: flag('reminders'),
      systemNotices: flag('system_notices'),
      productAnnouncements: flag('product_announcements'),
      quietHoursEnabled: flag('quiet_hours_enabled', fallback: false),
      quietHoursStart: json['quiet_hours_start']?.toString(),
      quietHoursEnd: json['quiet_hours_end']?.toString(),
      quietHoursTimezone: json['quiet_hours_timezone']?.toString() ?? 'UTC',
      preferredLanguage: json['preferred_language']?.toString() ?? 'en',
      pushEnabled: flag('push_enabled'),
      inAppEnabled: flag('in_app_enabled'),
    );
  }
}

final class NotificationDevice {
  const NotificationDevice({
    required this.id,
    required this.installationId,
    required this.platform,
    required this.notificationsEnabled,
    required this.lastSeenAt,
    this.appVersion,
    this.locale,
  });

  final String id;
  final String installationId;
  final String platform;
  final String? appVersion;
  final String? locale;
  final bool notificationsEnabled;
  final DateTime lastSeenAt;

  factory NotificationDevice.fromJson(Map<String, dynamic> json) {
    return NotificationDevice(
      id: json['id']?.toString() ?? '',
      installationId: json['installation_id']?.toString() ?? '',
      platform: json['platform']?.toString() ?? '',
      appVersion: json['app_version']?.toString(),
      locale: json['locale']?.toString(),
      notificationsEnabled: json['notifications_enabled'] == true,
      lastSeenAt:
          DateTime.tryParse(json['last_seen_at']?.toString() ?? '') ??
          DateTime.fromMillisecondsSinceEpoch(0, isUtc: true),
    );
  }
}
