import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:user_app/app/providers/app_providers.dart';
import 'package:user_app/features/notifications/data/repositories/notification_repository.dart';
import 'package:user_app/features/notifications/presentation/controllers/notification_sync.dart';
import 'package:user_app/shared/providers/connectivity_provider.dart';

class NotificationInboxStatusCard extends ConsumerWidget {
  const NotificationInboxStatusCard({super.key});
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final repository = ref.watch(notificationRepositoryProvider);
    final connected = ref.watch(connectivityProvider).valueOrNull;
    return ValueListenableBuilder<NotificationInboxStatus>(
      valueListenable: repository.status,
      builder: (context, status, _) {
        final offline = connected == false || status.offline;
        final synced = status.lastSynced?.toLocal();
        return ListTile(
          contentPadding: EdgeInsets.zero,
          leading: Icon(
            offline ? Icons.cloud_off_outlined : Icons.cloud_done_outlined,
          ),
          title: Text(
            offline ? 'Viewing saved notifications' : 'Notification inbox',
          ),
          subtitle: Text(
            synced == null
                ? 'Pull down to check for updates.'
                : 'Last synced ${synced.toString().split('.').first}${offline ? '. Read changes will sync when connected.' : ''}',
          ),
          trailing: IconButton(
            tooltip: 'Sync notifications',
            icon: const Icon(Icons.refresh),
            onPressed: () =>
                ref.read(notificationSyncProvider).sync(force: true),
          ),
        );
      },
    );
  }
}
