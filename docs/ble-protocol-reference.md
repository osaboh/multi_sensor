# BLEプロトコル仕様書 — MultiSensor Board

**ステータス: 実装済み・実機検証済み（2026-08-17時点）。** 本書に記載の全Service/Characteristicはファームウェアに実装済みで、BLE接続してのRead/Notify/Write動作を実機（`tools/gatt_verify.py`, `tools/write_test.py`）で確認している。クライアント（スマホアプリ、PCツール等）を**ソースコードを読まずに本書だけを見て実装できること**を目的とする。ファームウェアを変更してこの仕様と差異が出た場合は本書も追従して更新すること。

## 概要

| 項目 | 値 |
|---|---|
| ボード | switch-science ISP1807搭載BLEマルチセンサーボード |
| チップ | Nordic nRF52840 (Insight SiP ISP1807モジュール) |
| BLEスタック | SoftDevice s140 7.3.0 |
| オンボードセンサー/デバイス | BMX055(9軸), HDC2010(温湿度), LPS22HB(気圧/温度), LED×2, タクトスイッチ×2, 圧電スピーカー |
| GAPデバイス名 | `Go MultiSenser`（暫定。変更される可能性あり） |
| セキュリティ | ペアリング/ボンディング未実装。オープンアクセス（今後のセキュリティ要件次第で変更の可能性あり） |
| バイトオーダー | 特記なき限り**リトルエンディアン** |

## Service一覧

| Service | UUID | 用途 |
|---|---|---|
| Nordic UART Service (NUS) | `6E400001-B5A3-F393-E0A9-E50E24DCCA9E`（標準UUID） | ログ出力（現状）。将来コマンド入力用途を追加予定（プロトコル未定） |
| MultiSensor I/O Service | `a0b40100-926d-4d61-98df-8c5c62ee53b3` | LED/Buzzer制御、スイッチ状態 |
| LPS22HB Service | `a0b40120-926d-4d61-98df-8c5c62ee53b3` | 気圧・温度 |
| HDC2010 Service | `a0b40130-926d-4d61-98df-8c5c62ee53b3` | 温度・湿度 |
| BMX055 Motion Service | `a0b40140-926d-4d61-98df-8c5c62ee53b3` | 加速度・ジャイロ・磁力 |

> `a0b4xxxx-926d-4d61-98df-8c5c62ee53b3` は本プロジェクト独自のUUIDファミリー（ランダム生成のv4 UUIDをベースに使用）。Nordicの標準NUS UUID（予約領域）とは無関係で、混在させていない。広告(Advertising)パケットで使われている `f96d0000-1139-4e07-8ccf-d28be904fc0f`（`adv_env`系ボードのサービスデータUUID）とも無関係の別の仕組み・別のUUID体系。

Notifyを受信するには、対象characteristicのCCCD (`0x2902`, Client Characteristic Configuration Descriptor) に `0x0001`（Notify有効）を書き込む必要がある（標準BLE作法）。

---

## 1. Nordic UART Service (NUS)

Service UUID: `6E400001-B5A3-F393-E0A9-E50E24DCCA9E`

| Characteristic | UUID | Properties | 内容 |
|---|---|---|---|
| RX | `6E400002-B5A3-F393-E0A9-E50E24DCCA9E` | Write / Write Without Response | ホスト→デバイス。**現状未定義**（将来のコマンドプロトコルは別途仕様化） |
| TX | `6E400003-B5A3-F393-E0A9-E50E24DCCA9E` | Notify | デバイス→ホスト。ログ1行ごとにNotify。プレーンテキスト(UTF-8)、行末の改行コードは付与しない |

**注意**: メッセージ長はネゴシエートされたATT MTUに制限される（デフォルトATT_MTU=23、実データ最大20byte）。長いログ行の分割/切り詰め挙動は未定義（実装時に確定させる）。

---

## 2. MultiSensor I/O Service

Service UUID: `a0b40100-926d-4d61-98df-8c5c62ee53b3`

| Characteristic | UUID | Properties | サイズ | 内容 |
|---|---|---|---|---|
| LED1 | `a0b40101-...` | Write | 1 byte | `0x00`=OFF, `0x01`=ON（他の値は未定義） |
| LED2 | `a0b40102-...` | Write | 1 byte | 同上 |
| Buzzer | `a0b40103-...` | Write | 2 byte | uint16 LE、鳴動時間[ms]。`0`=即時停止。書き込み後、指定時間で自動停止。音は固定（約1000Hz、オン30ms/オフ65msのゲーティング、松下電工「ホロホロ」チャイムに近づけたもの） |
| SW_TOP | `a0b40111-...` | Read, Notify | 1 byte | `0x00`=解放, `0x01`=押下。**状態変化時のみ**Notify（エッジトリガ、ポーリングではない） |
| SW_SIDE | `a0b40112-...` | Read, Notify | 1 byte | 同上 |

### 例

- LED1点灯: `a0b40101` に `01` を書き込み
- Buzzerを800ms鳴らす: `a0b40103` に uint16 LE で `0x0320`(=800) → バイト列 `20 03` を書き込み
- SW_TOPが押されるとNotifyで `01` が飛ぶ。離すと `00` が飛ぶ

---

## 3. LPS22HB Service（気圧・温度）

Service UUID: `a0b40120-926d-4d61-98df-8c5c62ee53b3`

| Characteristic | UUID | Properties | サイズ | Notifyタイミング |
|---|---|---|---|---|
| Pressure/Temperature | `a0b40121-...` | Read, Notify | 4 byte | 周期的（1秒間隔） |

**フォーマット** (offset順、LE):

| offset | 型 | フィールド | 単位/変換式 |
|---|---|---|---|
| 0 | int16 | pressure_dev | (実気圧hPa − 1013.25) × 100。単位0.01hPa |
| 2 | int16 | temperature | 実温度℃ × 100。単位0.01℃ |

### 例: 気圧1006.00hPa, 温度22.50℃

- pressure_dev = (1006.00 − 1013.25) × 100 = **−725** → LE `2B FD`
- temperature = 22.50 × 100 = **2250** → LE `CA 08`
- 送信バイト列: `2B FD CA 08`

---

## 4. HDC2010 Service（温度・湿度）

Service UUID: `a0b40130-926d-4d61-98df-8c5c62ee53b3`

| Characteristic | UUID | Properties | サイズ | Notifyタイミング |
|---|---|---|---|---|
| Temperature/Humidity | `a0b40131-...` | Read, Notify | 4 byte | 周期的（1秒間隔） |

**フォーマット** (offset順、LE):

| offset | 型 | フィールド | 単位/変換式 |
|---|---|---|---|
| 0 | int16 | temperature | 実温度℃ × 100。単位0.01℃ |
| 2 | int16 | humidity | 実湿度%RH × 100。単位0.01% |

### 例: 温度23.48℃, 湿度55.20%RH

- temperature = 23.48 × 100 = **2348** → LE `2C 09`
- humidity = 55.20 × 100 = **5520** → LE `90 15`
- 送信バイト列: `2C 09 90 15`

---

## 5. BMX055 Motion Service（加速度・ジャイロ・磁力）

Service UUID: `a0b40140-926d-4d61-98df-8c5c62ee53b3`

| Characteristic | UUID | Properties | サイズ | Notifyタイミング |
|---|---|---|---|---|
| Accelerometer | `a0b40141-...` | Read, Notify | 6 byte | 周期的（デフォルト1秒間隔、Intervalで変更可） |
| Gyroscope | `a0b40142-...` | Read, Notify | 6 byte | 周期的（同上） |
| Magnetometer | `a0b40143-...` | Read, Notify | 12 byte | 周期的（同上） |
| Interval | `a0b40144-...` | Write, Write Without Response | 2 byte | — |

加速度・ジャイロは**未較正の生カウント値**（クライアント側で物理値に変換）、磁力は**ファームウェア側で工場トリム値による較正済みのµT値**を返す。この非対称は意図的（詳細: 磁力の較正はBoschの専用アルゴリズムが必須で単純な倍率変換が不可能なため）。

### 5.0 Interval (`a0b40144`)

加速度・ジャイロ・磁力は同一ループで駆動されており、Notify間隔（＝ループのSleep時間）を共通で変更できる。

| offset | 型 | フィールド | 単位/範囲 |
|---|---|---|---|
| 0 | uint16 (LE) | interval_ms | ミリ秒。150〜5000の範囲外は自動的にクランプされる |

**実測周期の注意**: 実際のNotify周期は、書き込んだinterval_msに加えてジャイロduty-cyclingの起床時間（90ms）とI2C読み取り時間が上乗せされる（実測で+90〜105ms程度）。例えばinterval_ms=200を書き込んでも実際の周期は約285〜300msになる。クライアント側は各Notifyの受信タイムスタンプを見て周期を判断すること（`interval_ms`をそのまま周期として仮定しない）。

### 例: 更新間隔を200msに変更

- `a0b40144`に uint16 LE で`0x00C8`(=200) → バイト列 `C8 00` を書き込み

### 5.1 Accelerometer (`a0b40141`)

±2g設定、感度 **0.98 mg/LSB**。

| offset | 型 | フィールド | 変換式（クライアント側で実施） |
|---|---|---|---|
| 0 | int16 | accel_x | 生カウント × 0.98 = mg |
| 2 | int16 | accel_y | 同上 |
| 4 | int16 | accel_z | 同上 |

**例**: X=+50mg, Y=−20mg, Z=+1000mg(≒1G)
- raw = 物理値[mg]÷0.98mg: X=51, Y=−20, Z=1020
- バイト列: `33 00 EC FF FC 03`（51, −20, 1020のint16 LE）

### 5.2 Gyroscope (`a0b40142`)

±2000°/s設定、感度 **16.4 LSB/(°/s)**。

| offset | 型 | フィールド | 変換式（クライアント側で実施） |
|---|---|---|---|
| 0 | int16 | gyro_x | 生カウント ÷ 16.4 = °/s |
| 2 | int16 | gyro_y | 同上 |
| 4 | int16 | gyro_z | 同上 |

**例**: X=+10°/s, Y=−5°/s, Z=0°/s
- raw = 物理値[°/s]×16.4: X=164, Y=−82, Z=0
- バイト列: `A4 00 AE FF 00 00`

### 5.3 Magnetometer (`a0b40143`)

**ファームウェアがBosch互換のトリム補正計算を実施済み。クライアント側での追加変換は不要**、そのままµT単位の値として使用する。

| offset | 型 | フィールド | 単位 |
|---|---|---|---|
| 0 | float32 | mag_x | µT（そのまま使用） |
| 4 | float32 | mag_y | µT |
| 8 | float32 | mag_z | µT |

範囲の目安: X/Y ±1300µT, Z ±2500µT（センサー分解能 約0.3µT）。float32はIEEE754単精度、リトルエンディアン。

---

## 実装メモ（クライアント開発者向け）

- 全てのNotify対応characteristicはRead属性も持つため、接続直後にNotifyを待たずポーリングで初期値を取得できる
- SW_TOP/SW_SIDEは**エッジトリガ**（状態変化時のみ）、センサー系（LPS22HB/HDC2010/BMX055）は**周期トリガ**（1秒毎、値の変化有無に関わらず送信）— Notify頻度の想定が異なる点に注意
- 加速度・ジャイロが生カウント値である一方、磁力だけ物理値(µT)である非対称に注意（3節参照）
- 現状ペアリング/ボンディングは実装されていないため、BLE接続時に特別な認証手続きは不要

## 関連ドキュメント

- `docs/sensor-characteristic-examples.md` — 本書のセンサー変換式のより詳細な導出・データシート根拠
