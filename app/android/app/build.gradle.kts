import java.time.LocalDate
import java.time.format.DateTimeFormatter

plugins {
    id("com.android.application")
    kotlin("android")
    kotlin("plugin.serialization")
}

fun generateVersionName(): String {
    val today = LocalDate.now()
    val dateStr = today.format(DateTimeFormatter.ofPattern("yyyy.MM.dd"))
    val commitsToday = try {
        val process = ProcessBuilder("git", "log", "--oneline", "--since=midnight")
            .directory(rootProject.projectDir)
            .redirectErrorStream(true)
            .start()
        val output = process.inputStream.bufferedReader().readLines()
        process.waitFor()
        output.size
    } catch (_: Exception) { 0 }
    return "$dateStr.${commitsToday + 1}"
}

fun generateVersionCode(): Int {
    val today = LocalDate.now()
    // YYYYMMDD * 10 + sequence (capped at 9 per day, plenty)
    val commitsToday = try {
        val process = ProcessBuilder("git", "log", "--oneline", "--since=midnight")
            .directory(rootProject.projectDir)
            .redirectErrorStream(true)
            .start()
        val output = process.inputStream.bufferedReader().readLines()
        process.waitFor()
        output.size
    } catch (_: Exception) { 0 }
    return today.year * 10000 + today.monthValue * 100 + today.dayOfMonth
}

android {
    namespace = "com.bartmann.noteseditor"
    compileSdk = 35

    defaultConfig {
        applicationId = "com.bartmann.noteseditor"
        minSdk = 31
        targetSdk = 35
        versionCode = generateVersionCode()
        versionName = generateVersionName()
    }

    buildTypes {
        release {
            isMinifyEnabled = false
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro"
            )
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlinOptions {
        jvmTarget = "17"
    }

    buildFeatures {
        compose = true
        buildConfig = true
    }

    composeOptions {
        kotlinCompilerExtensionVersion = "1.5.14"
    }
}

dependencies {
    implementation("androidx.core:core-ktx:1.13.1")
    implementation("androidx.activity:activity-compose:1.9.2")
    implementation("androidx.compose.ui:ui:1.6.8")
    implementation("androidx.compose.ui:ui-tooling-preview:1.6.8")
    implementation("androidx.compose.foundation:foundation:1.6.8")
    implementation("androidx.compose.material:material-icons-extended:1.6.8")
    implementation("androidx.compose.material3:material3:1.3.0")
    implementation("androidx.navigation:navigation-compose:2.8.0")
    implementation("androidx.lifecycle:lifecycle-runtime-ktx:2.8.4")
    implementation("androidx.lifecycle:lifecycle-runtime-compose:2.8.4")
    implementation("androidx.media:media:1.7.0")
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.6.3")
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-android:1.8.1")
    implementation("com.squareup.okhttp3:okhttp:4.12.0")

    debugImplementation("androidx.compose.ui:ui-tooling:1.6.8")
    testImplementation("junit:junit:4.13.2")
}
