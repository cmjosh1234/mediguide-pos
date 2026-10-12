import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:user_app/features/downloads/data/models/offline_download.dart';
import 'package:user_app/features/downloads/presentation/controllers/guideline_downloads_controller.dart';
import 'package:user_app/shared/models/models.dart';

class NotificationGuidelineContext extends ConsumerWidget {
  const NotificationGuidelineContext({super.key, required this.notification});
  final MyNotification notification;
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final action = notification.action;
    if (action?['type'] != 'guideline') return const SizedBox.shrink();
    final id = action?['resource_id'];
    final parameters = action?['parameters'] is Map
        ? action!['parameters'] as Map
        : const {};
    final version = parameters['version']?.toString() ?? '';
    final summary = parameters['change_summary']?.toString() ?? '';
    final downloads =
        ref.watch(guidelineDownloadsControllerProvider).valueOrNull ?? [];
    final copies = downloads.where(
      (item) =>
          item.guidelineId == id &&
          item.localPath.isNotEmpty &&
          (item.status == OfflineDownloadStatus.ready ||
              item.status == OfflineDownloadStatus.updateAvailable),
    );
    final copy = copies.firstOrNull;
    final outdated =
        copy != null && copy.status == OfflineDownloadStatus.updateAvailable;
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 0, 16, 12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (version.isNotEmpty)
            Text(
              'Publication version $version',
              style: Theme.of(context).textTheme.labelMedium,
            ),
          if (summary.isNotEmpty)
            Text(
              'What changed: $summary',
              maxLines: 4,
              overflow: TextOverflow.ellipsis,
            ),
          Text(
            copy == null
                ? 'Download this guideline to read it offline.'
                : outdated
                ? 'Your saved copy is version ${copy.version}. Open the guideline to download the latest version.'
                : version.isNotEmpty && copy.version != version
                ? 'This alert refers to version $version. Your saved copy is ${copy.version}; open the guideline to check the current version.'
                : 'Saved for offline reading · version ${copy.version}',
            style: Theme.of(context).textTheme.bodySmall,
          ),
        ],
      ),
    );
  }
}
