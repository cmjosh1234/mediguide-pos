import 'package:flutter_test/flutter_test.dart';
import 'package:user_app/core/network/api_client.dart';
import 'package:user_app/features/guidelines/data/repositories/guideline_publication_repository.dart';

import 'helpers/test_local_store.dart';

void main() {
  test('loads all chapter blocks through one content request', () async {
    final store = TestLocalStore();
    addTearDown(store.close);
    final api = _ContentBundleApi();
    final repository = GuidelinePublicationRepository(api, store.cache);

    final content = await repository.content('guideline-1');

    expect(content.sections, hasLength(2));
    expect(content.blocks, hasLength(1));
    expect(api.paths.where((path) => path.endsWith('/content')), hasLength(1));
    expect(api.paths.where((path) => path.contains('/sections')), isEmpty);
  });

  test(
    'reuses a version-matched cached projection on a subsequent open',
    () async {
      final store = TestLocalStore();
      addTearDown(store.close);
      final api = _ContentBundleApi();
      final repository = GuidelinePublicationRepository(api, store.cache);

      await repository.content('guideline-1');
      final cached = await repository.content('guideline-1');

      expect(cached.blocks, hasLength(1));
      expect(
        api.paths.where((path) => path.endsWith('/content')),
        hasLength(1),
      );
      expect(
        api.paths.where((path) => path.endsWith('/manifest')),
        hasLength(3),
      );
    },
  );

  test(
    'loads a lightweight overview without the structured content bundle',
    () async {
      final store = TestLocalStore();
      addTearDown(store.close);
      final api = _ContentBundleApi();
      final repository = GuidelinePublicationRepository(api, store.cache);

      final summary = await repository.summary('guideline-1');

      expect(summary.publication.title, 'Clinical guideline');
      expect(summary.manifest.sectionCount, 2);
      expect(summary.sections, isEmpty);
      expect(summary.blocks, isEmpty);
      expect(api.paths.where((path) => path.endsWith('/content')), isEmpty);
    },
  );

  test('rejects a stale content bundle from another version', () async {
    final store = TestLocalStore();
    addTearDown(store.close);
    final repository = GuidelinePublicationRepository(
      _ContentBundleApi(staleContent: true),
      store.cache,
    );

    await expectLater(
      repository.content('guideline-1'),
      throwsA(isA<GuidelinePublicationVersionMismatch>()),
    );
  });

  test('does not combine a new manifest with an older cached bundle', () async {
    final store = TestLocalStore();
    addTearDown(store.close);
    await GuidelinePublicationRepository(
      _ContentBundleApi(),
      store.cache,
    ).content('guideline-1');
    final repository = GuidelinePublicationRepository(
      _ContentBundleApi(
        manifestVersion: 'version-2',
        packageVersion: 3,
        checksum: 'checksum-2',
        contentError: true,
      ),
      store.cache,
    );

    await expectLater(
      repository.content('guideline-1'),
      throwsA(isA<GuidelinePublicationVersionMismatch>()),
    );
  });
}

final class _ContentBundleApi extends BackendApiService {
  _ContentBundleApi({
    this.staleContent = false,
    this.manifestVersion = 'version-1',
    this.packageVersion = 2,
    this.checksum = 'checksum-1',
    this.contentError = false,
  });

  final bool staleContent;
  final String manifestVersion;
  final int packageVersion;
  final String checksum;
  final bool contentError;
  final paths = <String>[];

  @override
  Future<Map<String, dynamic>> requestJson(
    String path, {
    required String method,
    Map<String, dynamic>? body,
    Map<String, String>? query,
    bool includeAuth = true,
  }) async {
    paths.add(path);
    if (path.endsWith('/manifest')) {
      return {
        'data': {
          'guideline_id': 'guideline-1',
          'version_id': manifestVersion,
          'version': '1',
          'recommended_mode': 'structured',
          'has_chapters': true,
          'section_count': 2,
          'block_count': 1,
          'schema_version': 2,
          'package_version': packageVersion,
          'checksum': checksum,
        },
      };
    }
    if (path.endsWith('/content')) {
      if (contentError) throw StateError('content unavailable');
      return {
        'data': {
          'guideline_id': 'guideline-1',
          'version_id': staleContent ? 'version-old' : manifestVersion,
          'package_version': staleContent ? 1 : packageVersion,
          'checksum': staleContent ? 'checksum-old' : checksum,
          'sections': [
            {
              'id': 'parent',
              'title': 'Clinical care',
              'level': 1,
              'sort_order': 1,
            },
            {
              'id': 'child',
              'parent_id': 'parent',
              'title': 'Assessment',
              'level': 2,
              'sort_order': 2,
            },
          ],
          'blocks': [
            {
              'id': 'block-1',
              'section_id': 'child',
              'type': 'paragraph',
              'sort_order': 1,
              'content': {'text': 'Assess the patient'},
            },
          ],
        },
      };
    }
    if (path.endsWith('/figures')) {
      return {
        'data': {'items': <Map<String, dynamic>>[]},
      };
    }
    return {
      'data': {
        'id': 'guideline-1',
        'title': 'Clinical guideline',
        'version': '1',
      },
    };
  }
}
