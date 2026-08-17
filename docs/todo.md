# TODO

現状の実装で残っている作業をまとめる。優先度は目安。

## バグ

- [x] ~~切断すると二度と再接続できなくなる~~（2026-08-17発見・同日修正済み）: `main.go`が`SetConnectHandler`の切断イベントで`cancel()`を呼び`main()`をreturnさせ`defer adv.Stop()`を実行していたのが原因。実は`tinygo.org/x/bluetooth`のライブラリ自体が切断時に`sd_ble_gap_adv_start`でadvertisingを自動再開する処理を既に持っており（`adapter_nrf528xx-peripheral.go`/`-full.go`）、アプリ側が余計にそれを止めていただけだった。`cancel()`呼び出しと`context`の使用を削除し、`main()`の末尾を`select{}`で永久ブロックする形に変更。接続→切断を2回連続で行い再接続できることを実機確認済み

## 整理・クリーンアップ

- [x] ~~`i2cscan_debug.go` と `main.go` 内の `scanI2C()` 呼び出しを削除する~~（2026-08-17完了）: `i2cscan_debug.go`を削除し、`main.go`から呼び出しも削除。ビルド・実機でのadvertising動作を確認済み
- [ ] `docs/ble-protocol-reference.md` の先頭ステータスを更新する（現在「Draft。ファームウェア未実装」のままだが、実際にはGATTプロファイルは実装・実機検証済み）

## 設計・仕様が必要

- [ ] NUSのRXキャラクタリスティック（コマンド入力）のプロトコル設計。現状はログ出力(TX)のみ定義済みで、RX側は未定義のまま
- [ ] セキュリティ（ペアリング/ボンディング）の要否を検討する。現状はオープンアクセス
- [ ] 電源管理・スリープモードの検討。ボードはコイン電池(CR2025/CR2032)駆動を想定しているが、現状スリープ制御は一切実装していない（元のadv_env.inoは起動時に未使用センサーをsuspendしていたが、今回は全センサー常時稼働）

## クライアント実装

- [ ] `docs/ble-protocol-reference.md` を参照して、実際のクライアント（スマホアプリ or PCツール）を実装する。現状は検証用の使い捨てPythonスクリプト（bleak）のみで、恒久的なクライアントは未着手

## 検証環境の整備

- [x] 検証に使ったPythonスクリプトを `tools/` 配下に正式なツールとして整備する。`tools/gatt_verify.py`（`make verify`）と `tools/write_test.py`（`make write-test`）を追加、実機で動作確認済み
