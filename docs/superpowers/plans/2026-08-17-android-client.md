# Androidクライアントアプリ(v1) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** ISP1807マルチセンサーボード用の、初めての恒久的なAndroidクライアントアプリ（v1: センサー表示＋LED/Buzzer制御）を実装する。

**Architecture:** `client/android/`に新規Gradleプロジェクト（単一`app`モジュール、Kotlin + Jetpack Compose）を作成する。BLE通信は独立クラス`BleClient`に閉じ込め、接続状態とセンサー値を`StateFlow`で公開する。センサーのraw→物理値変換は`SensorConversions.kt`の純粋関数群に分離し、ファームウェア側`convert/`パッケージと同じ入出力ペアでJVMユニットテストする。

**Tech Stack:** Kotlin 2.2.10, Jetpack Compose (BOM 2026.02.01), AGP 9.3.1, Gradle 9.5.0, `android.bluetooth`/`android.bluetooth.le`（外部BLEライブラリ不使用）, JUnit4（`app/src/test`）。

**Spec:** `docs/superpowers/specs/2026-08-17-android-client-design.md`

## Global Constraints

- minSdk 33 / targetSdk 37 / compileSdk 37
- Kotlin 2.2.10, Compose BOM 2026.02.01, AGP 9.3.1, Gradle 9.5.0（`client/sample/test01`と同一バージョン）
- パッケージ名 `com.osaboh.multisensor`、アプリ名「MultiSensor」
- v1スコープ: センサー5種のNotify表示 + LED1/LED2トグル + Buzzer固定300ms。SW_TOP/SW_SIDE表示、NUS、複数デバイス選択UI、自動再接続、Buzzer時間入力は対象外
- 新規プロジェクトは`client/android/`に作成し、メインの`claude/`リポジトリでバージョン管理する（`client/sample/test01`とは別の独立プロジェクト。UWB/wear関連コードはコピーしない）
- BLE UUID・変換式は`docs/ble-protocol-reference.md`を正本とする

---

### Task 1: プロジェクト scaffold（ビルド可能な空アプリ）

**Files:**
- Create: `client/android/settings.gradle.kts`
- Create: `client/android/build.gradle.kts`
- Create: `client/android/gradle.properties`
- Create: `client/android/gradle/libs.versions.toml`
- Create: `client/android/gradle/wrapper/gradle-wrapper.jar`（`client/sample/test01`からコピー）
- Create: `client/android/gradle/wrapper/gradle-wrapper.properties`（`client/sample/test01`からコピー）
- Create: `client/android/gradlew`, `client/android/gradlew.bat`（`client/sample/test01`からコピー）
- Create: `client/android/.gitignore`
- Create: `client/android/local.properties`（gitignore対象、コミットしない）
- Create: `client/android/app/build.gradle.kts`
- Create: `client/android/app/src/main/AndroidManifest.xml`
- Create: `client/android/app/src/main/java/com/osaboh/multisensor/MainActivity.kt`（プレースホルダー版）
- Create: `client/android/app/src/main/java/com/osaboh/multisensor/ui/theme/Color.kt`
- Create: `client/android/app/src/main/java/com/osaboh/multisensor/ui/theme/Theme.kt`
- Create: `client/android/app/src/main/java/com/osaboh/multisensor/ui/theme/Type.kt`
- Create: `client/android/app/src/main/res/values/strings.xml`
- Create: `client/android/app/src/main/res/values/themes.xml`
- Create: `client/android/app/src/main/res/values/colors.xml`
- Create: `client/android/app/src/main/res/xml/backup_rules.xml`
- Create: `client/android/app/src/main/res/xml/data_extraction_rules.xml`
- Create: `client/android/app/src/main/res/mipmap-*/*`（`client/sample/test01/app/src/main/res/mipmap-*`からコピー）
- Create: `client/android/app/src/main/res/drawable/ic_launcher_background.xml`, `ic_launcher_foreground.xml`（コピー）

**Interfaces:**
- Consumes: なし（最初のタスク）
- Produces: `./gradlew :app:assembleDebug`が通るビルド可能な空プロジェクト。以降のタスクはこの上にファイルを追加していく

- [ ] **Step 1: ディレクトリを作り、gradle wrapperとlauncherアイコン一式をsampleからコピーする**

```bash
mkdir -p /Users/osanai/Proj/multi_sensor/claude/client/android
cd /Users/osanai/Proj/multi_sensor/claude/client/android

cp -R /Users/osanai/Proj/multi_sensor/claude/client/sample/test01/gradle/wrapper .tmp_wrapper
mkdir -p gradle
mv .tmp_wrapper gradle/wrapper
cp /Users/osanai/Proj/multi_sensor/claude/client/sample/test01/gradlew .
cp /Users/osanai/Proj/multi_sensor/claude/client/sample/test01/gradlew.bat .
chmod +x gradlew

mkdir -p app/src/main/res
cp -R /Users/osanai/Proj/multi_sensor/claude/client/sample/test01/app/src/main/res/mipmap-anydpi app/src/main/res/
cp -R /Users/osanai/Proj/multi_sensor/claude/client/sample/test01/app/src/main/res/mipmap-hdpi app/src/main/res/
cp -R /Users/osanai/Proj/multi_sensor/claude/client/sample/test01/app/src/main/res/mipmap-mdpi app/src/main/res/
cp -R /Users/osanai/Proj/multi_sensor/claude/client/sample/test01/app/src/main/res/mipmap-xhdpi app/src/main/res/
cp -R /Users/osanai/Proj/multi_sensor/claude/client/sample/test01/app/src/main/res/mipmap-xxhdpi app/src/main/res/
cp -R /Users/osanai/Proj/multi_sensor/claude/client/sample/test01/app/src/main/res/mipmap-xxxhdpi app/src/main/res/
mkdir -p app/src/main/res/drawable
cp /Users/osanai/Proj/multi_sensor/claude/client/sample/test01/app/src/main/res/drawable/ic_launcher_background.xml app/src/main/res/drawable/
cp /Users/osanai/Proj/multi_sensor/claude/client/sample/test01/app/src/main/res/drawable/ic_launcher_foreground.xml app/src/main/res/drawable/
mkdir -p app/src/main/res/xml
cp /Users/osanai/Proj/multi_sensor/claude/client/sample/test01/app/src/main/res/xml/backup_rules.xml app/src/main/res/xml/
cp /Users/osanai/Proj/multi_sensor/claude/client/sample/test01/app/src/main/res/xml/data_extraction_rules.xml app/src/main/res/xml/
```

- [ ] **Step 2: ルートのGradle設定ファイルを作成する**

`client/android/settings.gradle.kts`:

```kotlin
pluginManagement {
    repositories {
        google {
            content {
                includeGroupByRegex("com\\.android.*")
                includeGroupByRegex("com\\.google.*")
                includeGroupByRegex("androidx.*")
            }
        }
        mavenCentral()
        gradlePluginPortal()
    }
}
plugins {
    id("org.gradle.toolchains.foojay-resolver-convention") version "1.0.0"
}
dependencyResolutionManagement {
    repositoriesMode.set(RepositoriesMode.FAIL_ON_PROJECT_REPOS)
    repositories {
        google()
        mavenCentral()
    }
}

rootProject.name = "MultiSensor"
include(":app")
```

`client/android/build.gradle.kts`:

```kotlin
// Top-level build file where you can add configuration options common to all sub-projects/modules.
plugins {
    alias(libs.plugins.android.application) apply false
    alias(libs.plugins.kotlin.compose) apply false
}
```

`client/android/gradle.properties`:

```properties
org.gradle.jvmargs=-Xmx2048m -Dfile.encoding=UTF-8
org.gradle.configuration-cache=true
kotlin.code.style=official
```

`client/android/gradle/libs.versions.toml`:

```toml
[versions]
agp = "9.3.1"
coreKtx = "1.10.1"
junit = "4.13.2"
lifecycleRuntimeKtx = "2.6.1"
activityCompose = "1.8.0"
kotlin = "2.2.10"
composeBom = "2026.02.01"
coroutines = "1.9.0"

[libraries]
androidx-core-ktx = { group = "androidx.core", name = "core-ktx", version.ref = "coreKtx" }
junit = { group = "junit", name = "junit", version.ref = "junit" }
androidx-lifecycle-runtime-ktx = { group = "androidx.lifecycle", name = "lifecycle-runtime-ktx", version.ref = "lifecycleRuntimeKtx" }
androidx-activity-compose = { group = "androidx.activity", name = "activity-compose", version.ref = "activityCompose" }
androidx-compose-bom = { group = "androidx.compose", name = "compose-bom", version.ref = "composeBom" }
androidx-compose-ui = { group = "androidx.compose.ui", name = "ui" }
androidx-compose-ui-graphics = { group = "androidx.compose.ui", name = "ui-graphics" }
androidx-compose-ui-tooling = { group = "androidx.compose.ui", name = "ui-tooling" }
androidx-compose-ui-tooling-preview = { group = "androidx.compose.ui", name = "ui-tooling-preview" }
androidx-compose-material3 = { group = "androidx.compose.material3", name = "material3" }
kotlinx-coroutines-android = { group = "org.jetbrains.kotlinx", name = "kotlinx-coroutines-android", version.ref = "coroutines" }

[plugins]
android-application = { id = "com.android.application", version.ref = "agp" }
kotlin-compose = { id = "org.jetbrains.kotlin.plugin.compose", version.ref = "kotlin" }
```

- [ ] **Step 3: `client/android/.gitignore`を作成する**

```gitignore
*.iml
.gradle
/local.properties
/.idea
.DS_Store
/build
/app/build
.kotlin
```

- [ ] **Step 4: `client/android/local.properties`を作成する（gitignore対象、コミットしない）**

```bash
echo "sdk.dir=$(sed -n 's/^sdk.dir=//p' /Users/osanai/Proj/multi_sensor/claude/client/sample/test01/local.properties)" > /Users/osanai/Proj/multi_sensor/claude/client/android/local.properties
cat /Users/osanai/Proj/multi_sensor/claude/client/android/local.properties
```

- [ ] **Step 5: `app/build.gradle.kts`を作成する**

```kotlin
plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.compose)
}

android {
    namespace = "com.osaboh.multisensor"
    compileSdk {
        version = release(37)
    }

    defaultConfig {
        applicationId = "com.osaboh.multisensor"
        minSdk = 33
        targetSdk = 37
        versionCode = 1
        versionName = "1.0"
    }

    buildTypes {
        release {
            optimization {
                enable = false
            }
        }
    }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_11
        targetCompatibility = JavaVersion.VERSION_11
    }
    buildFeatures {
        compose = true
    }
}

dependencies {
    implementation(platform(libs.androidx.compose.bom))
    implementation(libs.androidx.activity.compose)
    implementation(libs.androidx.compose.material3)
    implementation(libs.androidx.compose.ui)
    implementation(libs.androidx.compose.ui.graphics)
    implementation(libs.androidx.compose.ui.tooling.preview)
    implementation(libs.androidx.core.ktx)
    implementation(libs.androidx.lifecycle.runtime.ktx)
    implementation(libs.kotlinx.coroutines.android)
    testImplementation(libs.junit)
    debugImplementation(libs.androidx.compose.ui.tooling)
}
```

- [ ] **Step 6: `AndroidManifest.xml`を作成する**

`client/android/app/src/main/AndroidManifest.xml`:

```xml
<?xml version="1.0" encoding="utf-8"?>
<manifest xmlns:android="http://schemas.android.com/apk/res/android"
    xmlns:tools="http://schemas.android.com/tools">

    <uses-permission
        android:name="android.permission.BLUETOOTH_SCAN"
        android:usesPermissionFlags="neverForLocation" />
    <uses-permission android:name="android.permission.BLUETOOTH_CONNECT" />
    <uses-feature
        android:name="android.hardware.bluetooth_le"
        android:required="true" />

    <application
        android:allowBackup="true"
        android:dataExtractionRules="@xml/data_extraction_rules"
        android:fullBackupContent="@xml/backup_rules"
        android:icon="@mipmap/ic_launcher"
        android:label="@string/app_name"
        android:roundIcon="@mipmap/ic_launcher_round"
        android:supportsRtl="true"
        android:theme="@style/Theme.MultiSensor">
        <activity
            android:name=".MainActivity"
            android:exported="true"
            android:label="@string/app_name"
            android:theme="@style/Theme.MultiSensor"
            android:windowSoftInputMode="adjustResize">
            <intent-filter>
                <action android:name="android.intent.action.MAIN" />

                <category android:name="android.intent.category.LAUNCHER" />
            </intent-filter>
        </activity>
    </application>

</manifest>
```

- [ ] **Step 7: resファイルを作成する**

`client/android/app/src/main/res/values/strings.xml`:

```xml
<resources>
    <string name="app_name">MultiSensor</string>
</resources>
```

`client/android/app/src/main/res/values/themes.xml`:

```xml
<?xml version="1.0" encoding="utf-8"?>
<resources>

    <style name="Theme.MultiSensor" parent="android:Theme.Material.Light.NoActionBar" />
</resources>
```

`client/android/app/src/main/res/values/colors.xml`:

```xml
<?xml version="1.0" encoding="utf-8"?>
<resources>
    <color name="purple_200">#FFBB86FC</color>
    <color name="purple_500">#FF6200EE</color>
    <color name="purple_700">#FF3700B3</color>
    <color name="teal_200">#FF03DAC5</color>
    <color name="teal_700">#FF018786</color>
    <color name="black">#FF000000</color>
    <color name="white">#FFFFFFFF</color>
</resources>
```

- [ ] **Step 8: テーマ用Kotlinファイルを作成する**

`client/android/app/src/main/java/com/osaboh/multisensor/ui/theme/Color.kt`:

```kotlin
package com.osaboh.multisensor.ui.theme

import androidx.compose.ui.graphics.Color

val Purple80 = Color(0xFFD0BCFF)
val PurpleGrey80 = Color(0xFFCCC2DC)
val Pink80 = Color(0xFFEFB8C8)

val Purple40 = Color(0xFF6650a4)
val PurpleGrey40 = Color(0xFF625b71)
val Pink40 = Color(0xFF7D5260)
```

`client/android/app/src/main/java/com/osaboh/multisensor/ui/theme/Type.kt`:

```kotlin
package com.osaboh.multisensor.ui.theme

import androidx.compose.material3.Typography
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.sp

val Typography = Typography(
    bodyLarge = TextStyle(
        fontFamily = FontFamily.Default,
        fontWeight = FontWeight.Normal,
        fontSize = 16.sp,
        lineHeight = 24.sp,
        letterSpacing = 0.5.sp
    )
)
```

`client/android/app/src/main/java/com/osaboh/multisensor/ui/theme/Theme.kt`:

```kotlin
package com.osaboh.multisensor.ui.theme

import android.app.Activity
import android.os.Build
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.dynamicDarkColorScheme
import androidx.compose.material3.dynamicLightColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.platform.LocalContext

private val DarkColorScheme = darkColorScheme(
    primary = Purple80,
    secondary = PurpleGrey80,
    tertiary = Pink80
)

private val LightColorScheme = lightColorScheme(
    primary = Purple40,
    secondary = PurpleGrey40,
    tertiary = Pink40
)

@Composable
fun MultiSensorTheme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    dynamicColor: Boolean = true,
    content: @Composable () -> Unit
) {
    val colorScheme = when {
        dynamicColor && Build.VERSION.SDK_INT >= Build.VERSION_CODES.S -> {
            val context = LocalContext.current
            if (darkTheme) dynamicDarkColorScheme(context) else dynamicLightColorScheme(context)
        }

        darkTheme -> DarkColorScheme
        else -> LightColorScheme
    }

    MaterialTheme(
        colorScheme = colorScheme,
        typography = Typography,
        content = content
    )
}
```

- [ ] **Step 9: プレースホルダー版`MainActivity.kt`を作成する（Task 4で本実装に置き換える）**

`client/android/app/src/main/java/com/osaboh/multisensor/MainActivity.kt`:

```kotlin
package com.osaboh.multisensor

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.ui.Modifier
import com.osaboh.multisensor.ui.theme.MultiSensorTheme

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            MultiSensorTheme {
                Scaffold(modifier = Modifier.fillMaxSize()) { innerPadding ->
                    Text(text = "MultiSensor", modifier = Modifier.padding(innerPadding))
                }
            }
        }
    }
}
```

- [ ] **Step 10: ビルドを確認する**

```bash
cd /Users/osanai/Proj/multi_sensor/claude/client/android
./gradlew :app:assembleDebug
```

Expected: `BUILD SUCCESSFUL`。生成物は`app/build/outputs/apk/debug/app-debug.apk`。

- [ ] **Step 11: コミットする**

```bash
cd /Users/osanai/Proj/multi_sensor/claude
git add client/android
git status
git commit -m "$(cat <<'EOF'
Androidクライアント(v1)のプロジェクトscaffoldを追加

client/android/に新規Gradleプロジェクトを作成。client/sample/test01の
gradle wrapper/アイコン一式を流用しつつ、UWB/wear関連は含めない。
現時点ではプレースホルダー画面のみ表示するビルド可能な空アプリ。

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: センサー変換関数（`SensorConversions.kt`、TDD）

**Files:**
- Create: `client/android/app/src/main/java/com/osaboh/multisensor/SensorConversions.kt`
- Test: `client/android/app/src/test/java/com/osaboh/multisensor/SensorConversionsTest.kt`

**Interfaces:**
- Consumes: なし（純粋関数、Android APIに依存しない）
- Produces: 後続タスク（`BleClient`）が使うデータクラスと変換関数
  - `data class Lps22hbReading(val pressureHPa: Double, val temperatureC: Double)`
  - `data class Hdc2010Reading(val temperatureC: Double, val humidityPct: Double)`
  - `data class AccelReading(val xMg: Double, val yMg: Double, val zMg: Double)`
  - `data class GyroReading(val xDps: Double, val yDps: Double, val zDps: Double)`
  - `data class MagReading(val xUt: Float, val yUt: Float, val zUt: Float)`
  - `fun decodeLps22hb(bytes: ByteArray): Lps22hbReading`
  - `fun decodeHdc2010(bytes: ByteArray): Hdc2010Reading`
  - `fun decodeAccel(bytes: ByteArray): AccelReading`
  - `fun decodeGyro(bytes: ByteArray): GyroReading`
  - `fun decodeMag(bytes: ByteArray): MagReading`

テストの入出力ペアは、ファームウェア側`convert/lps22hb_test.go`・`convert/hdc2010_test.go`・`convert/bmx055_accel_test.go`・`convert/bmx055_gyro_test.go`が検証しているワイヤーフォーマットのバイト列と同じ値を使う（`docs/ble-protocol-reference.md`の worked example とも一致）。

- [ ] **Step 1: 失敗するテストを書く**

`client/android/app/src/test/java/com/osaboh/multisensor/SensorConversionsTest.kt`:

```kotlin
package com.osaboh.multisensor

import org.junit.Assert.assertEquals
import org.junit.Test

class SensorConversionsTest {

    @Test
    fun decodeLps22hb_matchesProtocolExample() {
        // docs/ble-protocol-reference.md 3節の例: 1006.00hPa, 22.50℃
        val bytes = byteArrayOf(0x2B, 0xFD.toByte(), 0xCA.toByte(), 0x08)
        val reading = decodeLps22hb(bytes)
        assertEquals(1006.00, reading.pressureHPa, 0.001)
        assertEquals(22.50, reading.temperatureC, 0.001)
    }

    @Test
    fun decodeHdc2010_matchesProtocolExample() {
        // docs/ble-protocol-reference.md 4節の例: 23.48℃, 55.20%RH
        val bytes = byteArrayOf(0x2C, 0x09, 0x90.toByte(), 0x15)
        val reading = decodeHdc2010(bytes)
        assertEquals(23.48, reading.temperatureC, 0.001)
        assertEquals(55.20, reading.humidityPct, 0.001)
    }

    @Test
    fun decodeAccel_matchesProtocolExample() {
        // docs/ble-protocol-reference.md 5.1節の例: raw X=51, Y=-20, Z=1020
        val bytes = byteArrayOf(0x33, 0x00, 0xEC.toByte(), 0xFF.toByte(), 0xFC.toByte(), 0x03)
        val reading = decodeAccel(bytes)
        assertEquals(51 * 0.98, reading.xMg, 0.001)
        assertEquals(-20 * 0.98, reading.yMg, 0.001)
        assertEquals(1020 * 0.98, reading.zMg, 0.001)
    }

    @Test
    fun decodeGyro_matchesProtocolExample() {
        // docs/ble-protocol-reference.md 5.2節の例: X=+10°/s, Y=-5°/s, Z=0°/s
        val bytes = byteArrayOf(0xA4.toByte(), 0x00, 0xAE.toByte(), 0xFF.toByte(), 0x00, 0x00)
        val reading = decodeGyro(bytes)
        assertEquals(10.0, reading.xDps, 0.001)
        assertEquals(-5.0, reading.yDps, 0.001)
        assertEquals(0.0, reading.zDps, 0.001)
    }

    @Test
    fun decodeMag_parsesLittleEndianFloat32() {
        fun floatBytesLE(v: Float): ByteArray {
            val bits = v.toRawBits()
            return byteArrayOf(
                (bits and 0xFF).toByte(),
                ((bits shr 8) and 0xFF).toByte(),
                ((bits shr 16) and 0xFF).toByte(),
                ((bits shr 24) and 0xFF).toByte(),
            )
        }
        val bytes = floatBytesLE(12.5f) + floatBytesLE(-3.25f) + floatBytesLE(0.0f)
        val reading = decodeMag(bytes)
        assertEquals(12.5f, reading.xUt, 0.0f)
        assertEquals(-3.25f, reading.yUt, 0.0f)
        assertEquals(0.0f, reading.zUt, 0.0f)
    }
}
```

- [ ] **Step 2: テストが失敗することを確認する**

```bash
cd /Users/osanai/Proj/multi_sensor/claude/client/android
./gradlew :app:testDebugUnitTest --tests "com.osaboh.multisensor.SensorConversionsTest"
```

Expected: FAIL（`decodeLps22hb`等が未定義でコンパイルエラー）。

- [ ] **Step 3: `SensorConversions.kt`を実装する**

`client/android/app/src/main/java/com/osaboh/multisensor/SensorConversions.kt`:

```kotlin
package com.osaboh.multisensor

data class Lps22hbReading(val pressureHPa: Double, val temperatureC: Double)
data class Hdc2010Reading(val temperatureC: Double, val humidityPct: Double)
data class AccelReading(val xMg: Double, val yMg: Double, val zMg: Double)
data class GyroReading(val xDps: Double, val yDps: Double, val zDps: Double)
data class MagReading(val xUt: Float, val yUt: Float, val zUt: Float)

private fun ByteArray.int16LE(offset: Int): Int {
    val raw = (this[offset].toInt() and 0xFF) or ((this[offset + 1].toInt() and 0xFF) shl 8)
    return raw.toShort().toInt()
}

private fun ByteArray.float32LE(offset: Int): Float {
    val bits = (this[offset].toInt() and 0xFF) or
        ((this[offset + 1].toInt() and 0xFF) shl 8) or
        ((this[offset + 2].toInt() and 0xFF) shl 16) or
        ((this[offset + 3].toInt() and 0xFF) shl 24)
    return Float.fromBits(bits)
}

// LPS22HB characteristic (a0b40121): int16 LE pressure-deviation-from-1013.25hPa
// x100, int16 LE temperature x100. See docs/ble-protocol-reference.md 3節.
fun decodeLps22hb(bytes: ByteArray): Lps22hbReading {
    val pressureDevRaw = bytes.int16LE(0)
    val tempRaw = bytes.int16LE(2)
    return Lps22hbReading(
        pressureHPa = pressureDevRaw / 100.0 + 1013.25,
        temperatureC = tempRaw / 100.0,
    )
}

// HDC2010 characteristic (a0b40131): int16 LE temperature x100, int16 LE
// humidity x100. See docs/ble-protocol-reference.md 4節.
fun decodeHdc2010(bytes: ByteArray): Hdc2010Reading {
    val tempRaw = bytes.int16LE(0)
    val humidityRaw = bytes.int16LE(2)
    return Hdc2010Reading(
        temperatureC = tempRaw / 100.0,
        humidityPct = humidityRaw / 100.0,
    )
}

// BMX055 accelerometer (a0b40141): int16 LE raw x,y,z counts, ±2g range,
// 0.98 mg/LSB. See docs/ble-protocol-reference.md 5.1節.
fun decodeAccel(bytes: ByteArray): AccelReading {
    val x = bytes.int16LE(0)
    val y = bytes.int16LE(2)
    val z = bytes.int16LE(4)
    return AccelReading(x * 0.98, y * 0.98, z * 0.98)
}

// BMX055 gyroscope (a0b40142): int16 LE raw x,y,z counts, ±2000°/s range,
// 16.4 LSB/(°/s). See docs/ble-protocol-reference.md 5.2節.
fun decodeGyro(bytes: ByteArray): GyroReading {
    val x = bytes.int16LE(0)
    val y = bytes.int16LE(2)
    val z = bytes.int16LE(4)
    return GyroReading(x / 16.4, y / 16.4, z / 16.4)
}

// BMX055 magnetometer (a0b40143): float32 LE x,y,z in µT, already calibrated
// firmware-side. See docs/ble-protocol-reference.md 5.3節.
fun decodeMag(bytes: ByteArray): MagReading {
    return MagReading(bytes.float32LE(0), bytes.float32LE(4), bytes.float32LE(8))
}
```

- [ ] **Step 4: テストが通ることを確認する**

```bash
cd /Users/osanai/Proj/multi_sensor/claude/client/android
./gradlew :app:testDebugUnitTest --tests "com.osaboh.multisensor.SensorConversionsTest"
```

Expected: `BUILD SUCCESSFUL`、5テストすべてPASS。

- [ ] **Step 5: コミットする**

```bash
cd /Users/osanai/Proj/multi_sensor/claude
git add client/android/app/src/main/java/com/osaboh/multisensor/SensorConversions.kt
git add client/android/app/src/test/java/com/osaboh/multisensor/SensorConversionsTest.kt
git commit -m "$(cat <<'EOF'
Androidクライアント: センサーraw値→物理値の変換関数を追加

docs/ble-protocol-reference.mdの変換式をKotlinに移植。ファームウェア側
convert/パッケージのGoテストと同じワイヤーフォーマットの入出力ペアで検証。

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: BLE UUID定義と`BleClient`（スキャン・接続・Notify購読・Write）

**Files:**
- Create: `client/android/app/src/main/java/com/osaboh/multisensor/BleUuids.kt`
- Create: `client/android/app/src/main/java/com/osaboh/multisensor/BleClient.kt`

**Interfaces:**
- Consumes: Task 2の`decodeLps22hb`/`decodeHdc2010`/`decodeAccel`/`decodeGyro`/`decodeMag`と対応するデータクラス
- Produces: 後続タスク（UI）が使う公開API
  - `sealed interface ConnectionState`（`Idle`, `Scanning`, `Connecting`, `DiscoveringServices`, `Connected`, `Disconnected(reason: String)`）
  - `data class SensorReadings(val lps22hb: Lps22hbReading?, val hdc2010: Hdc2010Reading?, val accel: AccelReading?, val gyro: GyroReading?, val mag: MagReading?)`（全フィールドdefault null）
  - `class BleClient(context: Context)`
    - `val connectionState: StateFlow<ConnectionState>`
    - `val readings: StateFlow<SensorReadings>`
    - `fun startScan()`
    - `fun setLed1(on: Boolean)`
    - `fun setLed2(on: Boolean)`
    - `fun triggerBuzzer(durationMs: Int)`
    - `fun close()`

このタスクはAndroid実機のBLEスタックに依存するため自動テスト対象外（`docs/superpowers/specs/2026-08-17-android-client-design.md`のテスト方針どおり）。ビルドが通ることをこのタスクの完了条件とし、実機での動作確認はTask 5で行う。

- [ ] **Step 1: UUID定義を作成する**

`client/android/app/src/main/java/com/osaboh/multisensor/BleUuids.kt`:

```kotlin
package com.osaboh.multisensor

import java.util.UUID

// docs/ble-protocol-reference.md の UUID一覧に準拠。
object BleUuids {
    val CCCD: UUID = UUID.fromString("00002902-0000-1000-8000-00805f9b34fb")

    val IO_SERVICE: UUID = UUID.fromString("a0b40100-926d-4d61-98df-8c5c62ee53b3")
    val LED1: UUID = UUID.fromString("a0b40101-926d-4d61-98df-8c5c62ee53b3")
    val LED2: UUID = UUID.fromString("a0b40102-926d-4d61-98df-8c5c62ee53b3")
    val BUZZER: UUID = UUID.fromString("a0b40103-926d-4d61-98df-8c5c62ee53b3")

    val LPS22HB_SERVICE: UUID = UUID.fromString("a0b40120-926d-4d61-98df-8c5c62ee53b3")
    val LPS22HB_CHAR: UUID = UUID.fromString("a0b40121-926d-4d61-98df-8c5c62ee53b3")

    val HDC2010_SERVICE: UUID = UUID.fromString("a0b40130-926d-4d61-98df-8c5c62ee53b3")
    val HDC2010_CHAR: UUID = UUID.fromString("a0b40131-926d-4d61-98df-8c5c62ee53b3")

    val BMX055_SERVICE: UUID = UUID.fromString("a0b40140-926d-4d61-98df-8c5c62ee53b3")
    val ACCEL_CHAR: UUID = UUID.fromString("a0b40141-926d-4d61-98df-8c5c62ee53b3")
    val GYRO_CHAR: UUID = UUID.fromString("a0b40142-926d-4d61-98df-8c5c62ee53b3")
    val MAG_CHAR: UUID = UUID.fromString("a0b40143-926d-4d61-98df-8c5c62ee53b3")

    const val DEVICE_NAME_FILTER = "MultiSenser"
}
```

- [ ] **Step 2: `BleClient.kt`を実装する**

`client/android/app/src/main/java/com/osaboh/multisensor/BleClient.kt`:

```kotlin
package com.osaboh.multisensor

import android.annotation.SuppressLint
import android.bluetooth.BluetoothDevice
import android.bluetooth.BluetoothGatt
import android.bluetooth.BluetoothGattCallback
import android.bluetooth.BluetoothGattCharacteristic
import android.bluetooth.BluetoothGattDescriptor
import android.bluetooth.BluetoothManager
import android.bluetooth.BluetoothProfile
import android.bluetooth.le.ScanCallback
import android.bluetooth.le.ScanResult
import android.content.Context
import java.util.UUID
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update

sealed interface ConnectionState {
    data object Idle : ConnectionState
    data object Scanning : ConnectionState
    data object Connecting : ConnectionState
    data object DiscoveringServices : ConnectionState
    data object Connected : ConnectionState
    data class Disconnected(val reason: String) : ConnectionState
}

data class SensorReadings(
    val lps22hb: Lps22hbReading? = null,
    val hdc2010: Hdc2010Reading? = null,
    val accel: AccelReading? = null,
    val gyro: GyroReading? = null,
    val mag: MagReading? = null,
)

private val NOTIFY_CHARACTERISTICS: List<Pair<UUID, UUID>> = listOf(
    BleUuids.LPS22HB_SERVICE to BleUuids.LPS22HB_CHAR,
    BleUuids.HDC2010_SERVICE to BleUuids.HDC2010_CHAR,
    BleUuids.BMX055_SERVICE to BleUuids.ACCEL_CHAR,
    BleUuids.BMX055_SERVICE to BleUuids.GYRO_CHAR,
    BleUuids.BMX055_SERVICE to BleUuids.MAG_CHAR,
)

// BLEスキャン・GATT接続・Notify購読・Write を担当する。UIはconnectionState/
// readingsのStateFlowを購読するだけでよく、BluetoothGattCallbackの詳細を
// 知る必要はない。呼び出し側はBLUETOOTH_SCAN/BLUETOOTH_CONNECT権限を事前に
// 取得しておくこと（権限確認自体はこのクラスの責務ではない）。
@SuppressLint("MissingPermission")
class BleClient(private val context: Context) {

    private val _connectionState = MutableStateFlow<ConnectionState>(ConnectionState.Idle)
    val connectionState: StateFlow<ConnectionState> = _connectionState.asStateFlow()

    private val _readings = MutableStateFlow(SensorReadings())
    val readings: StateFlow<SensorReadings> = _readings.asStateFlow()

    private val bluetoothManager: BluetoothManager? =
        context.getSystemService(Context.BLUETOOTH_SERVICE) as? BluetoothManager

    private var gatt: BluetoothGatt? = null
    private var ledChar1: BluetoothGattCharacteristic? = null
    private var ledChar2: BluetoothGattCharacteristic? = null
    private var buzzerChar: BluetoothGattCharacteristic? = null
    private var notifyIndex = 0

    fun startScan() {
        val adapter = bluetoothManager?.adapter
        val scanner = adapter?.bluetoothLeScanner
        if (adapter == null || !adapter.isEnabled || scanner == null) {
            _connectionState.value = ConnectionState.Disconnected("Bluetoothが無効です")
            return
        }
        _connectionState.value = ConnectionState.Scanning
        scanner.startScan(scanCallback)
    }

    fun setLed1(on: Boolean) = writeByte(ledChar1, if (on) 0x01 else 0x00)
    fun setLed2(on: Boolean) = writeByte(ledChar2, if (on) 0x01 else 0x00)

    fun triggerBuzzer(durationMs: Int) {
        val characteristic = buzzerChar ?: return
        val value = byteArrayOf((durationMs and 0xFF).toByte(), ((durationMs shr 8) and 0xFF).toByte())
        gatt?.writeCharacteristic(characteristic, value, BluetoothGattCharacteristic.WRITE_TYPE_DEFAULT)
    }

    fun close() {
        bluetoothManager?.adapter?.bluetoothLeScanner?.stopScan(scanCallback)
        gatt?.close()
        gatt = null
    }

    private fun writeByte(characteristic: BluetoothGattCharacteristic?, value: Int) {
        val char = characteristic ?: return
        gatt?.writeCharacteristic(char, byteArrayOf(value.toByte()), BluetoothGattCharacteristic.WRITE_TYPE_DEFAULT)
    }

    private val scanCallback = object : ScanCallback() {
        override fun onScanResult(callbackType: Int, result: ScanResult) {
            val name = result.scanRecord?.deviceName ?: result.device.name ?: return
            if (!name.contains(BleUuids.DEVICE_NAME_FILTER)) return
            bluetoothManager?.adapter?.bluetoothLeScanner?.stopScan(this)
            _connectionState.value = ConnectionState.Connecting
            gatt = result.device.connectGatt(context, false, gattCallback, BluetoothDevice.TRANSPORT_LE)
        }
    }

    private val gattCallback = object : BluetoothGattCallback() {
        override fun onConnectionStateChange(g: BluetoothGatt, status: Int, newState: Int) {
            when (newState) {
                BluetoothProfile.STATE_CONNECTED -> {
                    _connectionState.value = ConnectionState.DiscoveringServices
                    g.discoverServices()
                }
                BluetoothProfile.STATE_DISCONNECTED -> {
                    _connectionState.value = ConnectionState.Disconnected("status=$status")
                }
            }
        }

        override fun onServicesDiscovered(g: BluetoothGatt, status: Int) {
            if (status != BluetoothGatt.GATT_SUCCESS) {
                _connectionState.value = ConnectionState.Disconnected("service discovery failed: status=$status")
                return
            }
            ledChar1 = g.getService(BleUuids.IO_SERVICE)?.getCharacteristic(BleUuids.LED1)
            ledChar2 = g.getService(BleUuids.IO_SERVICE)?.getCharacteristic(BleUuids.LED2)
            buzzerChar = g.getService(BleUuids.IO_SERVICE)?.getCharacteristic(BleUuids.BUZZER)
            notifyIndex = 0
            enableNextNotification(g)
        }

        override fun onDescriptorWrite(g: BluetoothGatt, descriptor: BluetoothGattDescriptor, status: Int) {
            notifyIndex++
            enableNextNotification(g)
        }

        override fun onCharacteristicChanged(
            g: BluetoothGatt,
            characteristic: BluetoothGattCharacteristic,
            value: ByteArray,
        ) {
            when (characteristic.uuid) {
                BleUuids.LPS22HB_CHAR -> _readings.update { it.copy(lps22hb = decodeLps22hb(value)) }
                BleUuids.HDC2010_CHAR -> _readings.update { it.copy(hdc2010 = decodeHdc2010(value)) }
                BleUuids.ACCEL_CHAR -> _readings.update { it.copy(accel = decodeAccel(value)) }
                BleUuids.GYRO_CHAR -> _readings.update { it.copy(gyro = decodeGyro(value)) }
                BleUuids.MAG_CHAR -> _readings.update { it.copy(mag = decodeMag(value)) }
            }
        }
    }

    // GATT操作は直列化する必要がある: 前のリクエストが完了する前に次を発行すると
    // Androidは黙って無視することがあるため、onDescriptorWriteのコールバックを
    // 合図に1つずつNotifyを有効化していく。
    private fun enableNextNotification(g: BluetoothGatt) {
        if (notifyIndex >= NOTIFY_CHARACTERISTICS.size) {
            _connectionState.value = ConnectionState.Connected
            return
        }
        val (serviceUuid, charUuid) = NOTIFY_CHARACTERISTICS[notifyIndex]
        val characteristic = g.getService(serviceUuid)?.getCharacteristic(charUuid)
        val descriptor = characteristic?.getDescriptor(BleUuids.CCCD)
        if (characteristic == null || descriptor == null) {
            notifyIndex++
            enableNextNotification(g)
            return
        }
        g.setCharacteristicNotification(characteristic, true)
        g.writeDescriptor(descriptor, BluetoothGattDescriptor.ENABLE_NOTIFICATION_VALUE)
    }
}
```

- [ ] **Step 3: ビルドを確認する**

```bash
cd /Users/osanai/Proj/multi_sensor/claude/client/android
./gradlew :app:compileDebugKotlin
```

Expected: `BUILD SUCCESSFUL`。

- [ ] **Step 4: コミットする**

```bash
cd /Users/osanai/Proj/multi_sensor/claude
git add client/android/app/src/main/java/com/osaboh/multisensor/BleUuids.kt
git add client/android/app/src/main/java/com/osaboh/multisensor/BleClient.kt
git commit -m "$(cat <<'EOF'
Androidクライアント: BLEスキャン・GATT接続・Notify購読を行うBleClientを追加

5センサーCharacteristicのNotify購読、LED1/LED2/Buzzerの書き込みをカプ
セル化。接続状態とセンサー値はStateFlowで公開しUI側から購読する。

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: UI（`MultiSensorScreen`）と`MainActivity`の本実装

**Files:**
- Create: `client/android/app/src/main/java/com/osaboh/multisensor/MultiSensorScreen.kt`
- Modify: `client/android/app/src/main/java/com/osaboh/multisensor/MainActivity.kt`（Task 1のプレースホルダーを置き換え）

**Interfaces:**
- Consumes: Task 3の`BleClient`（`connectionState`, `readings`, `setLed1`, `setLed2`, `triggerBuzzer`, `startScan`, `close`）と`ConnectionState`
- Produces: 完成したv1アプリ（このタスクが最後のコード変更。Task 5は実機検証のみ）

- [ ] **Step 1: `MultiSensorScreen.kt`を作成する**

`client/android/app/src/main/java/com/osaboh/multisensor/MultiSensorScreen.kt`:

```kotlin
package com.osaboh.multisensor

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.material3.Button
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp

@Composable
fun MultiSensorScreen(bleClient: BleClient, modifier: Modifier = Modifier) {
    val connectionState by bleClient.connectionState.collectAsState()
    val readings by bleClient.readings.collectAsState()
    var led1On by remember { mutableStateOf(false) }
    var led2On by remember { mutableStateOf(false) }

    Column(modifier = modifier.fillMaxSize().padding(16.dp)) {
        Text(text = "接続状態: ${connectionStateLabel(connectionState)}")
        Spacer(modifier = Modifier.height(16.dp))

        Text(text = "気圧/温度(LPS22HB): " + (readings.lps22hb?.let {
            "%.2f hPa, %.2f ℃".format(it.pressureHPa, it.temperatureC)
        } ?: "-"))
        Text(text = "温湿度(HDC2010): " + (readings.hdc2010?.let {
            "%.2f ℃, %.2f %%RH".format(it.temperatureC, it.humidityPct)
        } ?: "-"))
        Text(text = "加速度: " + (readings.accel?.let {
            "%.1f, %.1f, %.1f mg".format(it.xMg, it.yMg, it.zMg)
        } ?: "-"))
        Text(text = "ジャイロ: " + (readings.gyro?.let {
            "%.1f, %.1f, %.1f °/s".format(it.xDps, it.yDps, it.zDps)
        } ?: "-"))
        Text(text = "磁力: " + (readings.mag?.let {
            "%.1f, %.1f, %.1f µT".format(it.xUt, it.yUt, it.zUt)
        } ?: "-"))

        Spacer(modifier = Modifier.height(16.dp))
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(text = "LED1")
            Spacer(modifier = Modifier.width(8.dp))
            Switch(
                checked = led1On,
                onCheckedChange = {
                    led1On = it
                    bleClient.setLed1(it)
                },
            )
        }
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(text = "LED2")
            Spacer(modifier = Modifier.width(8.dp))
            Switch(
                checked = led2On,
                onCheckedChange = {
                    led2On = it
                    bleClient.setLed2(it)
                },
            )
        }
        Spacer(modifier = Modifier.height(8.dp))
        Button(onClick = { bleClient.triggerBuzzer(300) }) {
            Text(text = "Buzzer 300ms")
        }
    }
}

private fun connectionStateLabel(state: ConnectionState): String = when (state) {
    ConnectionState.Idle -> "待機中"
    ConnectionState.Scanning -> "スキャン中..."
    ConnectionState.Connecting -> "接続中..."
    ConnectionState.DiscoveringServices -> "サービス探索中..."
    ConnectionState.Connected -> "接続済み"
    is ConnectionState.Disconnected -> "切断: ${state.reason}"
}
```

- [ ] **Step 2: `MainActivity.kt`を本実装に置き換える**

`client/android/app/src/main/java/com/osaboh/multisensor/MainActivity.kt`（全体を置き換え）:

```kotlin
package com.osaboh.multisensor

import android.Manifest
import android.content.Context
import android.content.pm.PackageManager
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.core.content.ContextCompat
import com.osaboh.multisensor.ui.theme.MultiSensorTheme

private val BLE_PERMISSIONS = arrayOf(
    Manifest.permission.BLUETOOTH_SCAN,
    Manifest.permission.BLUETOOTH_CONNECT,
)

private fun hasBlePermissions(context: Context): Boolean =
    BLE_PERMISSIONS.all {
        ContextCompat.checkSelfPermission(context, it) == PackageManager.PERMISSION_GRANTED
    }

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            MultiSensorTheme {
                Scaffold(modifier = Modifier.fillMaxSize()) { innerPadding ->
                    MultiSensorApp(modifier = Modifier.padding(innerPadding))
                }
            }
        }
    }
}

@Composable
private fun MultiSensorApp(modifier: Modifier = Modifier) {
    val context = LocalContext.current
    var permissionsGranted by remember { mutableStateOf(hasBlePermissions(context)) }

    val permissionLauncher = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestMultiplePermissions()
    ) { result ->
        permissionsGranted = result.values.all { it }
    }

    LaunchedEffect(Unit) {
        if (!permissionsGranted) {
            permissionLauncher.launch(BLE_PERMISSIONS)
        }
    }

    if (!permissionsGranted) {
        Text(text = "Bluetooth権限が必要です", modifier = modifier)
        return
    }

    val bleClient = remember { BleClient(context) }
    DisposableEffect(Unit) {
        bleClient.startScan()
        onDispose { bleClient.close() }
    }
    MultiSensorScreen(bleClient = bleClient, modifier = modifier)
}
```

- [ ] **Step 3: ビルドを確認する**

```bash
cd /Users/osanai/Proj/multi_sensor/claude/client/android
./gradlew :app:assembleDebug
```

Expected: `BUILD SUCCESSFUL`。

- [ ] **Step 4: コミットする**

```bash
cd /Users/osanai/Proj/multi_sensor/claude
git add client/android/app/src/main/java/com/osaboh/multisensor/MultiSensorScreen.kt
git add client/android/app/src/main/java/com/osaboh/multisensor/MainActivity.kt
git commit -m "$(cat <<'EOF'
Androidクライアント: センサー表示・LED/Buzzer操作画面を実装

MainActivityで権限要求後にBleClientをスキャン開始し、
MultiSensorScreenでセンサー値表示とLED1/LED2トグル・Buzzerボタンを提供。
これでv1スコープのコード実装が完了。

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 5: 実機での動作確認とTODO更新

**Files:**
- Modify: `docs/todo.md`

**Interfaces:**
- Consumes: Task 1〜4で完成したアプリ一式
- Produces: なし（検証とドキュメント更新のみ）

このタスクは実際のAndroid端末（USBデバッグ有効）とファームウェア実機ボードの両方が必要。`adb`は`/Users/osanai/Library/Android/sdk/platform-tools/adb`にある（PATH未登録）。

- [ ] **Step 1: ファームウェア実機ボードを広告状態にする**

すでに`make flash`済みで広告中のはずだが、未確認なら:

```bash
cd /Users/osanai/Proj/multi_sensor/claude
make scan
```

Expected: `Go MultiSenser`が検出される。

- [ ] **Step 2: Android端末をUSB接続し、デバッグアプリをインストールする**

```bash
cd /Users/osanai/Proj/multi_sensor/claude/client/android
export PATH="$PATH:/Users/osanai/Library/Android/sdk/platform-tools"
adb devices
./gradlew :app:installDebug
```

Expected: `adb devices`に端末が1台表示される。`installDebug`が`BUILD SUCCESSFUL`で完了する。

- [ ] **Step 3: アプリを起動し、下記を目視確認する**

端末上で「MultiSensor」アプリを起動し、以下をすべて確認する:

1. Bluetooth権限ダイアログが表示され、許可すると自動的にスキャン→接続が始まる
2. 「接続状態」が `待機中` → `スキャン中...` → `接続中...` → `サービス探索中...` → `接続済み` と遷移する
3. 気圧/温度・温湿度・加速度・ジャイロ・磁力の5行すべてに、`-`ではなく実測値らしき数値が表示され、1秒ごとに値が更新される
4. LED1トグルをONにすると実機のLED1が点灯し、OFFにすると消灯する。LED2も同様
5. 「Buzzer 300ms」ボタンを押すと実機のブザーが約300ms鳴動する
6. アプリをバックグラウンドに回してから再度フォアグラウンドに戻しても異常終了しない（`DisposableEffect`のクローズ処理がクラッシュしないことの確認）

いずれかで期待通りに動かない場合は、`docs/superpowers/systematic-debugging`スキルの要領で原因を特定してから次に進む（本タスクをスキップしない）。

- [ ] **Step 4: `docs/todo.md`の「クライアント実装」項目を完了に更新する**

`docs/todo.md`の該当行:

```markdown
## クライアント実装

- [ ] `docs/ble-protocol-reference.md` を参照して、実際のクライアント（スマホアプリ or PCツール）を実装する。現状は検証用の使い捨てPythonスクリプト（bleak）のみで、恒久的なクライアントは未着手
```

を次のように置き換える:

```markdown
## クライアント実装

- [x] ~~`docs/ble-protocol-reference.md` を参照して、実際のクライアント（スマホアプリ or PCツール）を実装する~~（2026-08-17完了・v1実装済み）: `client/android/`にAndroid(Kotlin+Compose)クライアントを実装。センサー5種のNotify表示、LED1/LED2トグル、Buzzer(固定300ms)に対応。実機で動作確認済み。SW_TOP/SW_SIDE表示・NUS経由コマンド・自動再接続は未対応（設計: `docs/superpowers/specs/2026-08-17-android-client-design.md`）
```

- [ ] **Step 5: コミットする**

```bash
cd /Users/osanai/Proj/multi_sensor/claude
git add docs/todo.md
git commit -m "$(cat <<'EOF'
Androidクライアント(v1)の実機動作確認完了、TODOを更新

センサー表示・LED1/LED2・Buzzerの全機能を実機ボード+実Android端末で
確認済み。docs/todo.mdの「クライアント実装」項目を完了マークに更新。

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```
