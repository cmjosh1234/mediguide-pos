import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:user_app/features/downloads/data/models/offline_download.dart';
import 'package:user_app/features/downloads/presentation/controllers/guideline_downloads_controller.dart';
import 'package:user_app/features/notifications/presentation/widgets/notification_guideline_context.dart';
import 'package:user_app/shared/models/models.dart';

class _SavedCopy extends GuidelineDownloadsController {
  @override
  Future<List<OfflineDownload>> build() async => [
    OfflineDownload(
      id: 'saved',
      guidelineId: 'guideline',
      assetType: 'offline_package',
      version: '3.0',
      localPath: '/saved/copy',
      status: OfflineDownloadStatus.ready,
      scope: 'public',
    ),
  ];
}

void main() {
  testWidgets(
    'an older notification does not falsely label a newer saved copy outdated',
    (tester) async {
      const notification = MyNotification(
        id: 'notice',
        action: {
          'type': 'guideline',
          'resource_id': 'guideline',
          'parameters': {
            'version': '2.0',
            'change_summary': 'Reviewed changes',
          },
        },
      );
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            guidelineDownloadsControllerProvider.overrideWith(_SavedCopy.new),
          ],
          child: const MaterialApp(
            home: Scaffold(
              body: NotificationGuidelineContext(notification: notification),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.textContaining('Your saved copy is 3.0'), findsOneWidget);
      expect(find.textContaining('download the latest version'), findsNothing);
      expect(find.text('What changed: Reviewed changes'), findsOneWidget);
    },
  );
}
