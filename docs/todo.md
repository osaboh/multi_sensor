# TODO

現状の実装で残っている作業をまとめる。優先度は目安。

## バグ

- [x] ~~切断すると二度と再接続できなくなる~~（2026-08-17発見・同日修正済み）: `main.go`が`SetConnectHandler`の切断イベントで`cancel()`を呼び`main()`をreturnさせ`defer adv.Stop()`を実行していたのが原因。実は`tinygo.org/x/bluetooth`のライブラリ自体が切断時に`sd_ble_gap_adv_start`でadvertisingを自動再開する処理を既に持っており（`adapter_nrf528xx-peripheral.go`/`-full.go`）、アプリ側が余計にそれを止めていただけだった。`cancel()`呼び出しと`context`の使用を削除し、`main()`の末尾を`select{}`で永久ブロックする形に変更。接続→切断を2回連続で行い再接続できることを実機確認済み

## 整理・クリーンアップ

- [x] ~~`i2cscan_debug.go` と `main.go` 内の `scanI2C()` 呼び出しを削除する~~（2026-08-17完了）: `i2cscan_debug.go`を削除し、`main.go`から呼び出しも削除。ビルド・実機でのadvertising動作を確認済み
- [x] ~~`docs/ble-protocol-reference.md` の先頭ステータスを更新する~~（2026-08-17完了）: 「Draft・未実装」から「実装済み・実機検証済み」に更新

## 設計・仕様が必要

- [ ] NUSのRXキャラクタリスティック（コマンド入力）のプロトコル設計。現状はログ出力(TX)のみ定義済みで、RX側は未定義のまま
- [ ] セキュリティ（ペアリング/ボンディング）の要否を検討する。現状はオープンアクセス
- [x] ~~電源管理・スリープモードの検討~~（2026-08-17完了・実装済み、2026-08-18 wake settle time修正）: ジャイロ(BMG160)が常時Normalモード(5mA)で支配的な電流消費要因だったため、`sensor_bmx055.go`に`readGyroDutyCycled()`を追加し、読み取り直前だけLPM1レジスタでNormalに起こし、読み取り後は即Suspend(25µA)へ戻す方式に変更。当初データシート記載のwake-up time 30msを採用したが、静止状態でもジャイロY軸が-34〜+666°/sに暴走するバグが判明し調査の結果、30msではY軸の共振ループが安定化しきらないことが根本原因と判明。実機で二分探索し90msで完全に安定することを確認、恒久値として採用（commit `523076a`）。加速度・磁気センサーは常時稼働のまま（今回は対象外）。CR2032想定でジャイロ平均電流は約473µA（当初見積もりの174µAから増加、常時5mAの約10倍省電力は維持）、全体の電池寿命は約6.5〜8.5日程度に改善見込み。実機でNotify正常確認済み（`make verify`）

## クライアント実装

- [x] ~~`docs/ble-protocol-reference.md` を参照して、実際のクライアント（スマホアプリ or PCツール）を実装する~~（2026-08-18完了・v1実装済み）: `client/android/`にAndroid(Kotlin+Compose)クライアントを実装。センサー5種のNotify表示、SW_TOP/SW_SIDEのON/OFF表示、LED1/LED2トグル、Buzzer(固定300ms)に対応。実機で動作確認済み（気圧/温度・温湿度・加速度・ジャイロ・磁力の値表示更新、スイッチON/OFF反映、LED1/LED2点灯、Buzzer鳴動、バックグラウンド/フォアグラウンド復帰後もクラッシュなし）。NUS経由コマンド・自動再接続は未対応（設計: `docs/superpowers/specs/2026-08-17-android-client-design.md`、実装計画: `docs/superpowers/plans/2026-08-17-android-client.md`）

## 検証環境の整備

- [x] 検証に使ったPythonスクリプトを `tools/` 配下に正式なツールとして整備する。`tools/gatt_verify.py`（`make verify`）と `tools/write_test.py`（`make write-test`）を追加、実機で動作確認済み
