# Androidクライアントアプリ 設計書

**ステータス: 承認済み・実装計画待ち（2026-08-17）**

## 背景・目的

`docs/ble-protocol-reference.md`で定義されたBLE GATTプロトコルを実装したISP1807マルチセンサーボード（ファームウェアは実装・実機検証済み）に対する、初めての恒久的なクライアントを実装する。現状は検証用の使い捨てPythonスクリプト（`tools/gatt_verify.py`, `tools/write_test.py`, bleak製）のみで、恒久的なクライアントは未着手だった（`docs/todo.md`「クライアント実装」項目）。

「まずは簡単なAndroidアプリクライアント」というユーザー要望に基づき、スコープを絞った最初のバージョンを実装する。

## 参照した既存資産

`/Users/osanai/Proj/multi_sensor/claude/client/sample/test01`にKotlin+Jetpack ComposeのAndroidサンプルプロジェクトが存在する（`app`モジュール＋`wear`モジュール、独自のgitリポジトリで管理されメインの`claude/`リポジトリには未追跡）。

- `app`モジュール: BLEスキャン→GATT接続→Read/Write疎通テストの実装（生の`android.bluetooth`/`android.bluetooth.le` API、コールバック方式）。minSdk 33 / targetSdk 37 / compileSdk 37、Kotlin 2.2.10、Compose BOM 2026.02.01、AGP 9.3.1、Gradle 9.5.0。UWB機能テストのコードも含む。
- `wear`モジュール: Pixel Watch向けアプリ。**今回無視する**（ユーザー指示）。

sampleのBLEスキャン/GATT接続パターン（権限要求、`BluetoothLeScanner`、`BluetoothGattCallback`の使い方）を実装の参考にするが、UWB関連コード・wearモジュールはコピーしない。sampleが接続する`BLE_TEST_SERVICE_UUID`(`6c1e0001-...`)は本プロジェクトの実マルチセンサーボードとは無関係の疎通テスト専用UUIDであり、本アプリでは使用しない。

## スコープ（v1: 「簡単な」クライアント）

含む:
- BLEスキャン→自動接続（デバイス名に"MultiSenser"を含む広告を検出したら接続）
- 5センサーCharacteristic（LPS22HB/HDC2010/BMX055 Accel/Gyro/Mag）のNotify購読とリアルタイム表示（物理値に変換して表示）
- LED1/LED2のON/OFF制御（トグル）
- Buzzerの鳴動（固定時間300ms、ボタン1つ）

含まない（将来の拡張候補、今回は対象外）:
- SW_TOP/SW_SIDEスイッチ状態の表示
- NUS経由のログ表示・コマンド送信（ファームウェア側のRXプロトコル自体が`docs/todo.md`上も未設計）
- 複数デバイスの一覧・選択UI（単一デバイス自動接続のみ）
- 切断時の自動再接続
- Buzzer鳴動時間のユーザー入力（固定300msのみ）
- 通信ログの永続化・エクスポート

## アーキテクチャ

新規Gradleプロジェクトを`client/android/`に作成し、メインの`claude/`リポジトリでバージョン管理する（sampleとは別プロジェクト。同じ`claude/`リポジトリのgit履歴で一元管理できる）。

- **パッケージ名**: `com.osaboh.multisensor`、アプリ名「MultiSensor」
- **ビルド設定**: sampleに合わせる — minSdk 33 / targetSdk 37 / compileSdk 37、Kotlin 2.2.10系、Jetpack Compose（Compose BOM）、AGP/Gradleもsampleと同等バージョンを踏襲
- **モジュール構成**: 単一`app`モジュールのみ（wear/UWBは不要なので追加しない）

### コンポーネント構成

1. **`MainActivity`**: エントリーポイント。権限要求とComposeコンテンツのセットアップのみ担当
2. **`BleClient`**（新規、独立クラス）: BLEスキャン・GATT接続・Notify購読・Characteristic Write を担当。接続状態とパースされたセンサー値を`StateFlow`で公開し、Composable側はこれを`collectAsState()`で購読するだけにする。sampleのようにコールバックをComposable内に直書きせず、UIとBLE通信ロジックを分離する
3. **`SensorConversions`**（新規、純粋関数群）: `ble-protocol-reference.md`のraw→物理値変換式をKotlinに移植。入力はCharacteristicの生バイト列、出力は物理値（Double/Float）。ファームウェア側`convert/`パッケージのGoコードと1:1対応させる
4. **`MultiSensorScreen`**（Composable）: `BleClient`の`StateFlow`を購読し、接続ステータス・センサー値・LED/Buzzerコントロールを表示する単一画面

### データフロー

1. 起動時、`BLUETOOTH_SCAN`/`BLUETOOTH_CONNECT`権限を要求（sampleと同じ`ActivityResultContracts.RequestMultiplePermissions`パターン。UWB権限は不要なので要求しない）
2. 権限許可後、`BleClient`が`BluetoothLeScanner.startScan()`を開始。スキャン結果の`deviceName`（または`scanRecord.deviceName`）に"MultiSenser"を含むデバイスを検出したら即座にスキャン停止し`connectGatt()`
3. `onServicesDiscovered`で5つのセンサーCharacteristicのCCCDに`0x0001`を書き込みNotifyを有効化
4. `onCharacteristicChanged`で受信した生バイト列を`SensorConversions`の対応関数でデコードし、`StateFlow`を更新
5. UIはLED1/LED2トグルのオン/オフ変更、およびBuzzerボタン押下でそれぞれ`BleClient`の書き込みメソッドを呼び出す（`writeCharacteristic`、1バイトLED値 or 2バイトuint16 LE 300ms固定値）

### エラーハンドリング

sampleと同じ方針で、シンプルなステータステキスト表示に留める（自動リトライ・自動再接続は行わない — v1スコープ外）:

- 権限拒否 → 「Bluetooth権限が許可されていません」等のテキスト表示
- Bluetooth OFF → 「Bluetoothが無効です」表示
- スキャンタイムアウト（デバイス未検出）→ 「デバイスが見つかりません」表示、再スキャンボタンで再試行可能にする
- GATT接続失敗・切断 → 接続状態テキストを更新するのみ。切断後の自動再接続はしない（再接続したい場合はユーザーが再スキャンボタンを押す)
- Characteristic Write失敗 → `onCharacteristicWrite`のstatusを見てステータステキストにエラー表示

## センサー値変換式（`SensorConversions`、`ble-protocol-reference.md` 3〜5節に準拠）

| Characteristic | 変換 |
|---|---|
| LPS22HB Pressure/Temperature | pressure_hPa = raw_pressure_dev(int16 LE) ÷ 100 + 1013.25、temperature_C = raw_temperature(int16 LE) ÷ 100 |
| HDC2010 Temperature/Humidity | temperature_C = raw_temperature(int16 LE) ÷ 100、humidity_pct = raw_humidity(int16 LE) ÷ 100 |
| BMX055 Accelerometer | accel_x/y/z_mg = raw(int16 LE) × 0.98 |
| BMX055 Gyroscope | gyro_x/y/z_dps = raw(int16 LE) ÷ 16.4 |
| BMX055 Magnetometer | mag_x/y/z_uT = raw(float32 LE)をそのまま使用（変換不要、ファームウェア側で較正済み） |

## テスト方針

- `SensorConversions`の変換関数群はJVMユニットテスト（`app/src/test`）で検証する。ファームウェア側`convert/`パッケージの対応するGoテストケース（`convert/*_test.go`）と同じ入出力ペアを使い、変換式の実装がプロトコル定義と食い違わないことを担保する
- BLEスキャン・GATT通信部分（`BleClient`）は実機依存のため自動テスト対象外。実装完了後、実機ボード＋実Android端末で以下を手動確認する: スキャン→自動接続、5センサーのNotify値が画面に表示され定期更新されること、LED1/LED2トグルで実機LEDが点灯/消灯すること、Buzzerボタンで実機ブザーが鳴動すること

## 関連ドキュメント

- `docs/ble-protocol-reference.md` — 本アプリが実装するBLEプロトコルの正本
- `docs/todo.md` — 「クライアント実装」項目（本実装で着手）
