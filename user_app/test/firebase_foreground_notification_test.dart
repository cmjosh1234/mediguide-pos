import 'package:flutter_local_notifications/flutter_local_notifications.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:user_app/core/services/firebase_service.dart';

void main() {
  test('urgent pushes keep their high-importance channel in the app', () {
    final channel = foregroundChannelFor('mediguide_emergency');
    expect(channel.id, 'mediguide_emergency');
    expect(channel.importance, Importance.high);
  });

  test('every other push uses the updates channel', () {
    for (final requested in [
      'mediguide_updates',
      'mediguide_alerts',
      'unknown',
      null,
    ]) {
      final channel = foregroundChannelFor(requested);
      expect(channel.id, 'mediguide_updates', reason: '$requested');
      expect(channel.importance, Importance.defaultImportance);
    }
  });

  test('a tapped banner reopens the push data it was shown for', () {
    final message = messageFromNotificationPayload(
      '{"action_type":"open_guideline","resource_id":"g-1"}',
    );
    expect(message?.data, {
      'action_type': 'open_guideline',
      'resource_id': 'g-1',
    });
  });

  test('a banner without usable data opens nothing', () {
    for (final payload in [null, '', 'not json', '["a"]', '"text"']) {
      expect(messageFromNotificationPayload(payload), isNull, reason: payload);
    }
  });
}
