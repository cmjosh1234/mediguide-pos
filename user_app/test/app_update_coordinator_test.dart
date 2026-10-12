import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:in_app_update/in_app_update.dart' as play;
import 'package:shared_preferences/shared_preferences.dart';
import 'package:user_app/app/providers/app_providers.dart';
import 'package:user_app/core/config/app_keys.dart';
import 'package:user_app/features/settings/data/app_update_client.dart';
import 'package:user_app/features/settings/presentation/widgets/app_update_coordinator.dart';
import 'package:user_app/features/settings/presentation/controllers/app_update_controller.dart';
import 'helpers/fake_app_update_client.dart';

Future<void> mount(WidgetTester tester, FakeAppUpdateClient client) async {
  SharedPreferences.setMockInitialValues({});
  final prefs = await SharedPreferences.getInstance();
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        sharedPreferencesProvider.overrideWithValue(prefs),
        appUpdateClientProvider.overrideWithValue(client),
        automaticAppUpdatesEnabledProvider.overrideWithValue(true),
      ],
      child: MaterialApp(
        scaffoldMessengerKey: AppKeys.scaffoldMessengerKey,
        builder: (_, child) => AppUpdateCoordinator(child: child!),
        home: const Scaffold(body: Text('Clinical content')),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('startup notice needs download consent and explicit restart', (
    tester,
  ) async {
    final client = FakeAppUpdateClient();
    await mount(tester, client);
    expect(find.text('Clinical content'), findsOneWidget);
    expect(find.text('Update'), findsOneWidget);
    expect(client.flexibleStarts, 0);
    await tester.tap(find.text('Update'));
    await tester.pumpAndSettle();
    expect(client.flexibleStarts, 1);
    expect(find.text('Restart'), findsOneWidget);
    expect(client.installs, 0);
    await tester.tap(find.text('Restart'));
    await tester.pumpAndSettle();
    expect(client.installs, 1);
    await tester.pumpWidget(const SizedBox());
    await client.events.close();
  });
  testWidgets('dismissed update notice is not repeated on every resume', (
    tester,
  ) async {
    final client = FakeAppUpdateClient();
    await mount(tester, client);
    await tester.tap(find.byIcon(Icons.close));
    await tester.pumpAndSettle();
    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.inactive);
    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
    await tester.pumpAndSettle();
    expect(find.text('Update'), findsNothing);
    expect(client.checks, 1);
    expect(client.flexibleStarts, 0);
    await tester.pumpWidget(const SizedBox());
    await client.events.close();
  });
  testWidgets('download completed in background prompts after resume', (
    tester,
  ) async {
    final client = FakeAppUpdateClient();
    await mount(tester, client);
    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.inactive);
    client.events.add(play.InstallStatus.downloaded);
    await tester.pump();
    await tester.pumpAndSettle();
    expect(
      ProviderScope.containerOf(
        tester.element(find.text('Clinical content')),
      ).read(appUpdateControllerProvider).valueOrNull,
      AppUpdateResult.readyToInstall,
    );
    expect(find.text('Restart'), findsNothing);
    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
    await tester.pumpAndSettle();
    expect(find.text('Restart'), findsOneWidget);
    expect(client.installs, 0);
    await tester.pumpWidget(const SizedBox());
    await client.events.close();
  });
}
