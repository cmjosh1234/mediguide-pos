import Flutter
import UIKit
import UserNotifications
import FirebaseMessaging

@main
@objc class AppDelegate: FlutterAppDelegate, FlutterImplicitEngineDelegate {
  override func application(
    _ application: UIApplication,
    didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?
  ) -> Bool {
    // Lets notifications shown while the app is open appear as banners and
    // report their taps back to Flutter.
    UNUserNotificationCenter.current().delegate = self as? UNUserNotificationCenterDelegate
    return super.application(application, didFinishLaunchingWithOptions: launchOptions)
  }

  override func application(
    _ application: UIApplication,
    didReceiveRemoteNotification userInfo: [AnyHashable: Any],
    fetchCompletionHandler completionHandler: @escaping (UIBackgroundFetchResult) -> Void
  ) {
    if UserDefaults.standard.bool(forKey: "notificationDeliveryDiagnostics") {
      Messaging.serviceExtension().exportDeliveryMetricsToBigQuery(withMessageInfo: userInfo)
    }
    super.application(application, didReceiveRemoteNotification: userInfo,
                      fetchCompletionHandler: completionHandler)
  }

  func didInitializeImplicitFlutterEngine(_ engineBridge: FlutterImplicitEngineBridge) {
    GeneratedPluginRegistrant.register(with: engineBridge.pluginRegistry)
    let channel = FlutterMethodChannel(
      name: "mediguide/app_settings",
      binaryMessenger: engineBridge.applicationRegistrar.messenger()
    )
    channel.setMethodCallHandler { call, result in
      if call.method == "setDeliveryDiagnostics", let enabled = call.arguments as? Bool {
        UserDefaults.standard.set(enabled, forKey: "notificationDeliveryDiagnostics")
        result(nil)
        return
      }
      if call.method == "setBadgeCount", let count = call.arguments as? Int {
        DispatchQueue.main.async {
          if #available(iOS 16.0, *) {
            UNUserNotificationCenter.current().setBadgeCount(max(0, count)) { error in
              result(error == nil)
            }
          } else {
            UIApplication.shared.applicationIconBadgeNumber = max(0, count)
            result(true)
          }
        }
        return
      }
      guard call.method == "openNotificationSettings",
            let url = URL(string: UIApplication.openSettingsURLString) else {
        result(FlutterMethodNotImplemented)
        return
      }
      UIApplication.shared.open(url, options: [:]) { opened in
        result(opened)
      }
    }
  }
}
