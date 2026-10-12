import java.io.FileInputStream
import java.util.Properties
import java.util.Base64

plugins {
    id("com.android.application")
    id("com.google.gms.google-services")
    id("com.google.firebase.crashlytics")
    id("kotlin-android")
    // The Flutter Gradle Plugin must be applied after the Android and Kotlin Gradle plugins.
    id("dev.flutter.flutter-gradle-plugin")
}

val keystoreProperties = Properties()
val keystorePropertiesFile = rootProject.file("key.properties")
val hasReleaseSigning = keystorePropertiesFile.exists()

// The self-updating Firebase SDK must never enter a Play Store artifact.
val firebaseTesterBuild = (project.findProperty("dart-defines") as? String)
    .orEmpty().split(",").filter { it.isNotBlank() }.any {
        String(Base64.getDecoder().decode(it), Charsets.UTF_8) ==
            "MEDIGUIDE_DISTRIBUTION_CHANNEL=firebase"
    }

if (hasReleaseSigning) {
    FileInputStream(keystorePropertiesFile).use(keystoreProperties::load)
}

android {
    namespace = "com.mediguide.ug"
    compileSdk = maxOf(flutter.compileSdkVersion, 36)
    ndkVersion = flutter.ndkVersion

    compileOptions {
        isCoreLibraryDesugaringEnabled = true
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlinOptions {
        jvmTarget = JavaVersion.VERSION_17.toString()
    }

    defaultConfig {
        buildConfigField("boolean", "FIREBASE_TESTER_UPDATES", firebaseTesterBuild.toString())
        // TODO: Specify your own unique Application ID (https://developer.android.com/studio/build/application-id.html).
        applicationId = "com.mediguide.ug"
        // You can update the following values to match your application needs.
        // For more information, see: https://flutter.dev/to/review-gradle-config.
        minSdk = flutter.minSdkVersion
        targetSdk = maxOf(flutter.targetSdkVersion, 36)
        versionCode = flutter.versionCode
        versionName = flutter.versionName

    }

    flavorDimensions += "environment"
    buildFeatures { buildConfig = true }
    productFlavors {
        create("development") {
            dimension = "environment"
            applicationIdSuffix = ".dev"
            manifestPlaceholders["appName"] = "MediGuide Dev"
        }
        create("staging") {
            dimension = "environment"
            applicationIdSuffix = ".staging"
            manifestPlaceholders["appName"] = "MediGuide Staging"
        }
        create("production") {
            dimension = "environment"
            manifestPlaceholders["appName"] = "MediGuide"
        }
    }

    signingConfigs {
        if (hasReleaseSigning) {
            create("release") {
                keyAlias = keystoreProperties.getProperty("keyAlias")
                keyPassword = keystoreProperties.getProperty("keyPassword")
                storeFile = file(keystoreProperties.getProperty("storeFile"))
                storePassword = keystoreProperties.getProperty("storePassword")
            }
        }
    }

    buildTypes {
        release {
            // Release artifacts must never silently use the debug key.
            signingConfig = if (hasReleaseSigning) {
                signingConfigs.getByName("release")
            } else {
                null
            }
        }
    }
}

flutter {
    source = "../.."
}

dependencies {
    implementation("com.google.android.play:app-update:2.1.0")
    implementation("com.google.firebase:firebase-appdistribution-api:16.0.0-beta21")
    if (firebaseTesterBuild) {
        implementation("com.google.firebase:firebase-appdistribution:16.0.0-beta21")
    }
    coreLibraryDesugaring("com.android.tools:desugar_jdk_libs:2.1.5")
}

// Catch accidental transitive inclusion as well as direct dependency mistakes.
configurations.configureEach {
    if (!firebaseTesterBuild && name.endsWith("RuntimeClasspath")) {
        incoming.afterResolve {
            check(resolutionResult.allComponents.none {
                it.moduleVersion?.group == "com.google.firebase" &&
                    it.moduleVersion?.name == "firebase-appdistribution"
            }) { "Firebase self-update SDK is forbidden in Play builds" }
        }
    }
}
