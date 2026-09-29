import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'production Android releases stay device-specific and size-governed',
    () {
      final pubspec = File('pubspec.yaml').readAsStringSync();
      final workflow = File(
        '../.github/workflows/mobile-release.yml',
      ).readAsStringSync();

      expect(
        pubspec,
        isNot(contains('    - assets/\n')),
        reason: 'A broad asset declaration packages generator-only artwork.',
      );
      expect(workflow, contains('--split-per-abi'));
      expect(
        RegExp(
          r'--target-platform android-arm,android-arm64',
        ).allMatches(workflow).length,
        greaterThanOrEqualTo(2),
      );
      expect(workflow, contains('max_apk_bytes='));
      expect(workflow, contains('max_aab_bytes='));
    },
  );

  test('Google Play upload passes an absolute AAB path to Fastlane', () {
    final fastfile = File('fastlane/Fastfile').readAsStringSync();

    expect(fastfile, contains('aab = File.expand_path('));
    expect(fastfile, contains('aab: aab'));
  });

  test('production Android build disables advertising identifiers', () {
    final manifest = File(
      'android/app/src/production/AndroidManifest.xml',
    ).readAsStringSync();

    for (final setting in const [
      'google_analytics_adid_collection_enabled',
      'google_analytics_default_allow_ad_personalization_signals',
    ]) {
      expect(
        manifest,
        matches(
          RegExp(
            'android:name="$setting"\\s+android:value="false"',
            multiLine: true,
          ),
        ),
      );
    }
    for (final permission in const [
      'com.google.android.gms.permission.AD_ID',
      'android.permission.ACCESS_ADSERVICES_AD_ID',
      'android.permission.ACCESS_ADSERVICES_ATTRIBUTION',
    ]) {
      expect(
        manifest,
        contains(
          '<uses-permission android:name="$permission" tools:node="remove" />',
        ),
      );
    }
  });
}
