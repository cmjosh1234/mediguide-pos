package com.mediguide.ug

import android.content.Intent
import android.net.Uri
import android.provider.Settings
import android.os.Build
import com.google.firebase.appdistribution.FirebaseAppDistribution
import com.google.android.play.core.appupdate.AppUpdateManagerFactory
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel

class MainActivity : FlutterActivity() {
    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, "mediguide/app_updates")
            .setMethodCallHandler { call, result ->
                try {
                    when (call.method) {
                        "completePlayUpdate" -> {
                            AppUpdateManagerFactory.create(this).completeUpdate()
                                .addOnSuccessListener { result.success(null) }
                                .addOnFailureListener {
                                    result.error("play_update_install", "Unable to install the downloaded update. Try again later.", null)
                                }
                        }
                        "installerPackage" -> {
                            val installer = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
                                packageManager.getInstallSourceInfo(packageName).installingPackageName
                            } else {
                                @Suppress("DEPRECATION")
                                packageManager.getInstallerPackageName(packageName)
                            }
                            result.success(installer)
                        }
                        "checkTesterUpdates" -> {
                            if (!BuildConfig.FIREBASE_TESTER_UPDATES) {
                                result.success(false)
                            } else {
                                val distribution = FirebaseAppDistribution.getInstance()
                                val automatic = call.argument<Boolean>("automatic") ?: false
                                if (automatic && !distribution.isTesterSignedIn) {
                                    // Sign-in is offered only after the tester explicitly checks.
                                    result.success(false)
                                } else {
                                    distribution.updateIfNewReleaseAvailable()
                                        .addOnSuccessListener { result.success(true) }
                                        .addOnFailureListener {
                                            result.error("firebase_app_distribution", "Unable to check for tester updates. Try again later.", null)
                                        }
                                }
                            }
                        }
                        else -> result.notImplemented()
                    }
                } catch (_: Exception) {
                    result.error("app_updates", "Update service is unavailable.", null)
                }
            }
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, "mediguide/app_settings")
            .setMethodCallHandler { call, result ->
                if (call.method != "openNotificationSettings") {
                    result.notImplemented()
                    return@setMethodCallHandler
                }
                val intent = Intent(Settings.ACTION_APPLICATION_DETAILS_SETTINGS).apply {
                    data = Uri.fromParts("package", packageName, null)
                }
                startActivity(intent)
                result.success(true)
            }
    }
}
