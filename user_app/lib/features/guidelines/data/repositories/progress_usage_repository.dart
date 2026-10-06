import 'dart:async';
import 'dart:math';
import 'package:flutter/foundation.dart';
import 'package:user_app/core/storage/local_cache_service.dart';
import 'package:user_app/shared/models/paginated_response.dart';
import 'package:user_app/core/network/api_client.dart';
import 'package:user_app/features/guidelines/data/models/reading_progress.dart';

final class ReadingProgressRepository {
  ReadingProgressRepository(this._api, this._localCacheService);

  final BackendApiService _api;
  final LocalCacheService _localCacheService;

  static const String _entityType = 'reading_progress';

  // =========================================================
  // SCOPE
  // =========================================================

  String _scope(String userId) {
    final normalized = userId.trim();

    if (normalized.isEmpty) {
      throw ArgumentError.value(userId, 'userId', 'User id is required');
    }

    return 'user:$normalized';
  }

  // =========================================================
  // ENTITY ID
  // =========================================================
  //
  // There should only ever be one reading progress record for
  // a user + guideline combination.
  //
  // Because the LocalCacheService already separates records by
  // scope, guidelineId is enough as the local entity id.
  // =========================================================

  String _entityId(String guidelineId) {
    final normalized = guidelineId.trim();

    if (normalized.isEmpty) {
      throw ArgumentError.value(
        guidelineId,
        'guidelineId',
        'Guideline id is required',
      );
    }

    return normalized;
  }

  // =========================================================
  // GET PROGRESS FOR GUIDELINE
  // =========================================================

  Future<ReadingProgress?> forGuideline(
    String userId,
    String guidelineId,
  ) async {
    final local = await _localFor(userId, guidelineId);

    // ---------------------------------------------------------
    // Try to push unsynced local changes first.
    // ---------------------------------------------------------

    if (local != null && local['pending_sync'] == true) {
      await _trySync(local);
    }

    try {
      final response = await _api.requestJson(
        '/api/v2/reading-progress/${Uri.encodeComponent(guidelineId)}',
        method: 'GET',
      );

      final record = _normalize(_data(response), userId: userId);

      await _saveRecord(record);

      return ReadingProgress.fromJson(record);
    } catch (_) {
      if (local == null) {
        return null;
      }

      return ReadingProgress.fromJson(local);
    }
  }

  // =========================================================
  // PROGRESS LIST
  // =========================================================

  Future<PaginatedResponse<ReadingProgress>> list(
    String userId, {
    int page = 1,
    int perPage = 20,
    bool? bookmarked,
    double? progressMin,
    double? progressMax,
  }) async {
    final safePage = page < 1 ? 1 : page;
    final safePerPage = perPage < 1 ? 20 : perPage;
    await syncPending(userId);

    try {
      final response = await _api.requestJson(
        '/api/v2/reading-progress',
        method: 'GET',
        query: {
          'page': '$safePage',
          'per_page': '$safePerPage',
          if (bookmarked != null) 'is_bookmarked': '$bookmarked',
          if (progressMin != null) 'progress_min': '$progressMin',
          if (progressMax != null) 'progress_max': '$progressMax',
          'sort': 'last_read_at',
          'order': 'desc',
        },
      );
      final data = _data(response);
      final rows = (data['items'] as List? ?? const [])
          .whereType<Map>()
          .map(
            (value) =>
                _normalize(Map<String, dynamic>.from(value), userId: userId),
          )
          .toList(growable: false);
      for (final row in rows) {
        try {
          await _saveRecord(row);
        } catch (_) {
          // A cache failure must not discard a valid server response.
        }
      }
      final items = rows.map(ReadingProgress.fromJson).toList(growable: false);
      return PaginatedResponse<ReadingProgress>(
        page: (data['page'] as num?)?.toInt() ?? safePage,
        perPage: (data['per_page'] as num?)?.toInt() ?? safePerPage,
        totalItems: (data['total_items'] as num?)?.toInt() ?? items.length,
        totalPages:
            (data['total_pages'] as num?)?.toInt() ??
            _totalPages(items.length, safePerPage),
        items: items,
      );
    } catch (_) {
      return _localList(
        userId,
        page: safePage,
        perPage: safePerPage,
        bookmarked: bookmarked,
        progressMin: progressMin,
        progressMax: progressMax,
      );
    }
  }

  // =========================================================
  // IN-PROGRESS GUIDELINES
  // =========================================================

  Future<PaginatedResponse<ReadingProgress>> inProgress(
    String userId, {
    int page = 1,
    int perPage = 20,
  }) async {
    return list(
      userId,
      page: page,
      perPage: perPage,
      progressMin: 0.000001,
      progressMax: 0.999999,
    );
  }

  // =========================================================
  // UPSERT
  // =========================================================
  //
  // IMPORTANT:
  //
  // Local persistence happens FIRST.
  //
  // This means:
  //
  // - scrolling progress works offline
  // - bookmark changes work offline
  // - completion works offline
  //
  // The record is marked pending_sync until the API confirms it.
  // =========================================================

  Future<ReadingProgress> upsert(
    String userId,
    String guidelineId,
    Map<String, dynamic> values,
  ) async {
    final normalizedGuidelineId = _entityId(guidelineId);

    final existing = await _localFor(userId, normalizedGuidelineId);

    final now = DateTime.now().toUtc().toIso8601String();

    final local = <String, dynamic>{
      ...?existing,
      ...values,

      'id': existing?['id'] ?? 'local-$normalizedGuidelineId',

      'user_id': userId,

      'guideline_id': normalizedGuidelineId,

      'guideline_document_id': normalizedGuidelineId,

      'created_at': existing?['created_at'] ?? now,

      'updated_at': now,

      'pending_sync': true,
    };

    // ---------------------------------------------------------
    // Save immediately before trying the network.
    // ---------------------------------------------------------

    await _saveRecord(local);

    // ---------------------------------------------------------
    // Best-effort remote synchronization.
    // ---------------------------------------------------------

    final synced = await _trySync(local);

    return ReadingProgress.fromJson(synced ?? local);
  }

  // =========================================================
  // SYNC PENDING
  // =========================================================

  Future<void> syncPending(String userId) async {
    final rows = await _readUserRecords(userId);

    for (final row in rows) {
      if (row['pending_sync'] == true) {
        await _trySync(row);
      }
    }
  }

  // =========================================================
  // SYNC SINGLE RECORD
  // =========================================================

  Future<Map<String, dynamic>?> _trySync(Map<String, dynamic> record) async {
    final guidelineId =
        record['guideline_document_id']?.toString().trim() ??
        record['guideline_id']?.toString().trim() ??
        '';

    final userId = record['user_id']?.toString().trim() ?? '';

    if (guidelineId.isEmpty || userId.isEmpty) {
      return null;
    }

    try {
      final payload = Map<String, dynamic>.from(record)
        ..removeWhere(
          (key, _) => {
            'id',
            'user_id',
            'guideline_id',
            'created_at',
            'updated_at',
            'pending_sync',
          }.contains(key),
        );

      final response = await _api.requestJson(
        '/api/v2/reading-progress/${Uri.encodeComponent(guidelineId)}',
        method: 'PUT',
        body: payload,
      );

      final synced = _normalize(_data(response), userId: userId);

      await _saveRecord(synced);

      return synced;
    } catch (_) {
      // -------------------------------------------------------
      // Keep pending_sync=true.
      //
      // The next:
      //
      // - app startup
      // - home load
      // - guideline open
      // - explicit sync
      //
      // can retry this record.
      // -------------------------------------------------------

      return null;
    }
  }

  // =========================================================
  // LOCAL SINGLE RECORD
  // =========================================================

  Future<Map<String, dynamic>?> _localFor(String userId, String guidelineId) {
    return _localCacheService.get(
      type: _entityType,
      id: _entityId(guidelineId),
      scope: _scope(userId),
    );
  }

  Future<PaginatedResponse<ReadingProgress>> _localList(
    String userId, {
    required int page,
    required int perPage,
    bool? bookmarked,
    double? progressMin,
    double? progressMax,
  }) async {
    final rows = await _readUserRecords(userId);
    final filtered =
        rows.where((row) {
          final progress =
              (row['progress_percentage'] as num?)?.toDouble() ?? 0;
          if (bookmarked != null &&
              (row['is_bookmarked'] == true) != bookmarked) {
            return false;
          }
          if (progressMin != null && progress < progressMin) return false;
          if (progressMax != null && progress > progressMax) return false;
          return true;
        }).toList()..sort(
          (a, b) =>
              _date(b['last_read_at']).compareTo(_date(a['last_read_at'])),
        );
    final items = _paginate(
      filtered,
      page: page,
      perPage: perPage,
    ).map(ReadingProgress.fromJson).toList(growable: false);
    return PaginatedResponse<ReadingProgress>(
      page: page,
      perPage: perPage,
      totalItems: filtered.length,
      totalPages: _totalPages(filtered.length, perPage),
      items: items,
    );
  }

  // =========================================================
  // READ USER RECORDS
  // =========================================================

  Future<List<Map<String, dynamic>>> _readUserRecords(String userId) {
    return _localCacheService.list(
      type: _entityType,
      scope: _scope(userId),
      limit: 1000,
      offset: 0,
    );
  }

  // =========================================================
  // SAVE RECORD
  // =========================================================

  Future<void> _saveRecord(Map<String, dynamic> record) async {
    final userId = record['user_id']?.toString().trim() ?? '';

    final guidelineId =
        record['guideline_document_id']?.toString().trim() ??
        record['guideline_id']?.toString().trim() ??
        '';

    if (userId.isEmpty || guidelineId.isEmpty) {
      return;
    }

    await _localCacheService.put(
      type: _entityType,
      id: guidelineId,
      scope: _scope(userId),
      data: record,

      searchableText: [
        guidelineId,
        record['current_section']?.toString() ?? '',
      ].where((value) => value.trim().isNotEmpty).join(' ').toLowerCase(),

      metadata: {
        'guidelineId': guidelineId,

        'currentSection': record['current_section'],

        'progress': (record['progress_percentage'] as num?)?.toDouble(),

        'completed': record['is_completed'] == true,

        'bookmarked': record['is_bookmarked'] == true,

        'pendingSync': record['pending_sync'] == true,

        'lastReadAt': record['last_read_at'],
      },

      remoteUpdatedAt: _dateOrNull(record['updated_at']),
    );
  }

  // =========================================================
  // NORMALIZE
  // =========================================================

  Map<String, dynamic> _normalize(
    Map<String, dynamic> value, {
    required String userId,
  }) {
    final guidelineId =
        value['guideline_document_id'] ?? value['guideline_id'] ?? '';

    return {
      ...value,

      'user_id': userId,

      'guideline_id': guidelineId,

      'guideline_document_id': guidelineId,

      'created_at': value['created_at'] ?? value['created'] ?? '',

      'updated_at': value['updated_at'] ?? value['updated'] ?? '',

      'pending_sync': false,
    };
  }

  // =========================================================
  // CLEAR USER DATA
  // =========================================================

  Future<void> clearUserProgress(String userId) {
    return _localCacheService.clearType(
      type: _entityType,
      scope: _scope(userId),
    );
  }

  // =========================================================
  // CACHE STATUS
  // =========================================================

  Future<bool> hasCachedProgress(String userId) {
    return _localCacheService.hasData(type: _entityType, scope: _scope(userId));
  }

  // =========================================================
  // WATCH
  // =========================================================

  Stream<List<ReadingProgress>> watchProgress(String userId) {
    return _localCacheService
        .watch(type: _entityType, scope: _scope(userId))
        .map((rows) {
          final result = <ReadingProgress>[];

          for (final row in rows) {
            try {
              result.add(ReadingProgress.fromJson(row));
            } catch (_) {
              // Ignore malformed cached rows.
            }
          }

          result.sort((a, b) => b.lastReadAt.compareTo(a.lastReadAt));

          return List<ReadingProgress>.unmodifiable(result);
        });
  }

  // =========================================================
  // HELPERS
  // =========================================================

  List<T> _paginate<T>(
    List<T> items, {
    required int page,
    required int perPage,
  }) {
    final start = (page - 1) * perPage;

    if (start >= items.length) {
      return <T>[];
    }

    final end = (start + perPage).clamp(0, items.length);

    return items.sublist(start, end);
  }

  int _totalPages(int totalItems, int perPage) {
    if (totalItems <= 0 || perPage <= 0) {
      return 0;
    }

    return (totalItems / perPage).ceil();
  }

  DateTime _date(dynamic value) {
    return _dateOrNull(value) ??
        DateTime.fromMillisecondsSinceEpoch(0, isUtc: true);
  }

  DateTime? _dateOrNull(dynamic value) {
    if (value == null) {
      return null;
    }

    if (value is DateTime) {
      return value;
    }

    return DateTime.tryParse(value.toString());
  }
}

// ===========================================================
// USAGE / ANALYTICS
// ===========================================================
// Events are persisted before sending; analytics never block clinical content.

final class UsageRepository {
  UsageRepository(this._api, this._cache, this.currentUserId);
  final BackendApiService _api;
  final LocalCacheService _cache;
  final String? Function() currentUserId;
  Future<void> _writes = Future.value();
  Completer<void>? _syncing;
  bool _syncAgain = false;
  static const _type = 'usage_pending';

  Future<void> guideline(String id) =>
      _record('guidelines', resourceId: id, resourceType: 'guideline_document');
  Future<void> abbreviation(String id) =>
      _record('abbreviations', resourceId: id);
  Future<void> ai({String? ownerId}) {
    if (ownerId != null && currentUserId() != ownerId) return Future.value();
    return _record('ai');
  }

  Future<void> drug(String id) => _record('drugs', resourceId: id);
  Future<void> facility(String id) => _record('facilities', resourceId: id);
  Future<void> feature(String feature) => _record('features', feature: feature);

  Future<T> _write<T>(Future<T> Function() work) {
    final result = _writes.then((_) => work());
    _writes = result.then<void>(
      (_) {},
      onError: (Object error) {
        debugPrint('Usage cache failed: ${error.runtimeType}');
      },
    );
    return result;
  }

  Future<void> _record(
    String type, {
    String? resourceId,
    String? resourceType,
    String? feature,
  }) async {
    final userId = currentUserId();
    if (userId == null || userId.isEmpty) return;
    final random = Random.secure();
    final key = List.generate(
      16,
      (_) => random.nextInt(256).toRadixString(16).padLeft(2, '0'),
    ).join();
    final path = type == 'drugs' || type == 'facilities'
        ? '/api/v2/$type/${Uri.encodeComponent(resourceId!)}/usage'
        : '/api/v2/usage/$type';
    await _enqueue(userId, key, path, {
      'idempotency_key': key,
      if (resourceId != null) 'resource_id': resourceId,
      if (resourceType != null) 'resource_type': resourceType,
      if (feature != null) 'feature': feature,
    });
  }

  Future<void> notificationDelivery({
    required String ownerId,
    required String deliveryId,
    required String eventType,
    required String eventId,
  }) async {
    if (currentUserId() != ownerId ||
        ownerId.isEmpty ||
        deliveryId.trim().isEmpty) {
      return;
    }
    if (eventType != 'open' && eventType != 'click') return;
    // The server deduplicates by the supplied event ID, which survives retries.
    final key = 'notification-$eventType-$eventId';
    await _enqueue(
      ownerId,
      key,
      '/api/v2/notification-deliveries/${Uri.encodeComponent(deliveryId)}/$eventType',
      {
        'event_id': eventId,
        'occurred_at': DateTime.now().toUtc().toIso8601String(),
      },
    );
  }

  Future<void> _enqueue(
    String userId,
    String key,
    String path,
    Map<String, dynamic> body,
  ) async {
    try {
      await _write(() async {
        if (await _cache.get(type: _type, id: key, scope: 'user:$userId') !=
            null) {
          return;
        }
        await _cache.put(
          type: _type,
          id: key,
          scope: 'user:$userId',
          data: {'id': key, 'path': path, 'body': body},
        );
      });
      unawaited(sync());
    } catch (error) {
      debugPrint('Usage event deferred: ${error.runtimeType}');
    }
  }

  Future<void> sync() async {
    if (_syncing != null) {
      _syncAgain = true;
      return _syncing!.future;
    }
    final done = Completer<void>();
    _syncing = done;
    try {
      do {
        _syncAgain = false;
        final userId = currentUserId();
        if (userId == null || userId.isEmpty) return;
        await _writes;
        final events = await _cache.list(
          type: _type,
          scope: 'user:$userId',
          limit: 10000,
        );
        for (final event in events) {
          if (currentUserId() != userId) return;
          // Queued events from the retired condition-card reader cannot be
          // assigned to a document ID. Remove them during the client upgrade.
          if ((event['body'] as Map?)?['resource_type'] ==
              'medical_guideline') {
            await _write(
              () => _cache.tombstone(
                type: _type,
                id: event['id'] as String,
                scope: 'user:$userId',
              ),
            );
            continue;
          }
          try {
            await _api.requestJson(
              event['path'] as String,
              method: 'POST',
              body: Map<String, dynamic>.from(event['body'] as Map),
            );
            await _write(
              () => _cache.tombstone(
                type: _type,
                id: event['id'] as String,
                scope: 'user:$userId',
              ),
            );
          } catch (error) {
            debugPrint('Usage sync deferred: ${error.runtimeType}');
            if (error is BackendApiException &&
                (error.statusCode == 0 ||
                    error.statusCode == 401 ||
                    error.statusCode == 429 ||
                    error.statusCode >= 500)) {
              return;
            }
          }
        }
      } while (_syncAgain);
    } catch (error) {
      debugPrint('Usage queue unavailable: ${error.runtimeType}');
    } finally {
      _syncing = null;
      done.complete();
    }
  }
}

// ===========================================================
// API RESPONSE
// ===========================================================

Map<String, dynamic> _data(Map<String, dynamic> response) {
  final data = response['data'];

  return data is Map ? Map<String, dynamic>.from(data) : response;
}
