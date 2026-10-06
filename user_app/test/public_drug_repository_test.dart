import 'dart:io';
import 'package:flutter_test/flutter_test.dart';
import 'package:user_app/core/network/api_client.dart';
import 'package:user_app/features/drugs/data/models/drug.dart';
import 'package:user_app/features/drugs/data/repositories/drug_repository.dart';
import 'package:user_app/features/drugs/data/repositories/drug_local_repository.dart';
import 'helpers/test_local_store.dart';

class PublicDrugApi extends BackendApiService {
  bool offline = false;
  @override
  Future<Map<String, dynamic>> requestJson(
    String path, {
    required String method,
    Map<String, dynamic>? body,
    Map<String, String>? query,
    bool includeAuth = true,
  }) async {
    expect(includeAuth, isFalse);
    expect(path.startsWith('/api/public/drugs'), isTrue);
    if (offline) throw const SocketException('offline');
    final item = {
      'id': 'approved',
      'name': 'Approved medicine',
      'status': 'active',
      'review_status': 'approved',
    };
    return {
      'data': path.endsWith('/approved')
          ? item
          : {
              'items': [item],
            },
    };
  }
}

void main() {
  test(
    'guest drug reads omit credentials and offline fallback excludes unapproved records',
    () async {
      final store = TestLocalStore();
      addTearDown(store.close);
      final api = PublicDrugApi();
      final local = DrugLocalRepository(store.cache);
      final repository = DrugRepository(api, local);
      expect((await repository.list()).items.single.id, 'approved');
      expect((await repository.get('approved'))?.id, 'approved');
      await local.saveDrug(
        Drug.fromJson({
          'id': 'pending',
          'name': 'Pending medicine',
          'status': 'active',
          'review_status': 'pending',
        }),
      );
      api.offline = true;
      expect((await repository.list()).items.map((d) => d.id), ['approved']);
      expect((await repository.get('approved'))?.id, 'approved');
      expect(await repository.get('pending'), isNull);
    },
  );
}
