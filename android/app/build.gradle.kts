// build.gradle.kts du module app (Kotlin DSL).
// Consomme l'AAR produit par gomobile (libs/faillefox.aar) pour accéder
// au moteur Go depuis Kotlin.
plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
}

android {
    namespace = "com.dlnraja.faillefox"
    compileSdk = 34

    defaultConfig {
        applicationId = "com.dlnraja.faillefox"
        minSdk = 24
        targetSdk = 34
        versionCode = 2
        versionName = "0.2.0"
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
    kotlinOptions { jvmTarget = "17" }

    buildFeatures { viewBinding = true }

    // L'AAR Go est généré par `gomobile bind` (voir docs/android.md).
    // On le dépose dans app/libs/faillefox.aar.
}

dependencies {
    implementation("androidx.appcompat:appcompat:1.7.0")
    implementation("com.google.android.material:material:1.12.0")
    implementation(fileTree(mapOf("dir" to "libs", "include" to listOf("*.aar"))))
}
