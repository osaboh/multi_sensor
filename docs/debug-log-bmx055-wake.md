# デバッグログ: BMX055初期化でのクラッシュ調査 (2026-08-16)

## 症状

GATTプロファイル実装（`docs/ble-protocol-reference.md`）を全サービス実装後、`make flash`で書き込むとBLEアドバタイズが確認できなくなった（`make scan`で"Go MultiSenser"が見つからない）。

## 切り分け手順

1. **各Serviceを1つずつ無効化して`make scan`で確認**（advertising自体は`adv.Start()`後に登録される各サービスと無関係に飛び続けるはずだが、実際には後続処理のクラッシュが影響していた）
   - NUS / I/O / LPS22HB / HDC2010 のみ有効 → advertising OK
   - BMX055を有効化 → advertising NG
   - → **BMX055関連のコードが原因と特定**

2. **advertising継続は「クラッシュしていない証拠にはならない」ことに気づく**
   `adv.Start()`は`startBMX055Service()`より前に呼ばれるため、後続処理がpanicしても以前のadvertising設定が生きたまま飛び続ける可能性がある（実際には後述の通りpanicで完全に停止することが判明）。この前提が誤りだったため、advertisingの有無だけでは正確な切り分けができなかった。

3. **J-Linkでレジスタを読み、PCをシンボルに解決して実際にクラッシュしているか確認**
   ```
   nrfjprog --readregs          # PCを取得
   tinygo build -o=debug.elf .  # デバッグ情報付きELFをビルド
   arm-none-eabi-addr2line -e debug.elf -f -C <PC>
   ```
   → `runtime.panicOrGoexit` に一致。**panicしていることを確定。**

4. **コンソール出力が無い問題への対処: J-Link RTTで直接メモリを読む**

   このボードにはUSBシリアルが実装として繋がっていない（`xiao-ble`ターゲットは本来Seeed XIAO board向けで、このISP1807ボードとはUSB配線が異なる）。`println`の出力先が無い状態だった。

   - `tinygo build -serial=rtt` でRTT出力を有効化してビルド
   - `JLinkRTTLogger`はRTT制御ブロックの自動検出に**毎回失敗**した（原因不明、ツール側の問題の可能性）
   - 代替策: **ELFのシンボルテーブルからRTTバッファの実アドレスを直接特定し、`nrfjprog --memrd`で生読みする**
     ```
     arm-none-eabi-nm debug.elf | grep rttBufferUpData
     # => 20007800 b machine.rttBufferUpData
     nrfjprog --memrd 0x20007800 --n 512
     ```
     この方法はRTT制御ブロックの自動検出に依存せず確実に動作した。今後も同じ手法が使える。

5. **`must()`にNUS TX経由のログ出力を追加**（`logLine()`呼び出しを追加、恒久的な改善として残した）。ただしBLE接続がまだ確立できない段階のデバッグでは上記のmemrd手法の方が有効だった。

6. **`wakeBMX055()`の各レジスタ書き込み前にログを出すよう一時的に変更**し、RTTバッファを読んで進行状況を可視化 → **`mag pwrcntl1 on`（磁力センサーのPWR_CNTL1に`0x01`を書く処理）でI2C NACKにより停止**していることが判明。

## 根本原因

`wakeBMX055()`の磁力センサー起動シーケンスで、以下の2段階書き込みを行っていた（[kriswiner/BMX-055](https://github.com/kriswiner/BMX-055)のリファレンス実装を参考にした際の記述）:

```
PWR_CNTL1 (0x4B) = 0x82   // ソフトリセットトリガ (bit7|bit1)
delay(100ms)
PWR_CNTL1 (0x4B) = 0x01   // Power Control bit ON
```

1段目（`0x82`）は成功するが、2段目（`0x01`）が待機時間を100ms→リトライ5回×20ms(計200ms超)に延ばしても一貫してNACKした。リトライで解決しないことから、単なるタイミング不足ではなく、**このボード上のBMX055ではこの2段階シーケンス自体が不要/不適切**と判断した。

## 対処

ソフトリセットの段（`0x82`書き込み）を削除し、電源投入直後の状態から直接 `PWR_CNTL1 = 0x01` のみを書く1段階シーケンスに変更したところ、即座に解消した。`sensor_bmx055.go`の`wakeBMX055()`を参照。

**教訓**: WebFetchでの外部リファレンス実装の参照は、ツールによる要約を経由するため実際のソースコードの値と食い違うリスクがある。今回のように実機で検証できる場合は、疑わしい箇所は実機でのビット単位の検証（今回のようなI2C NACKの有無）を優先すべき。

## 副産物として整備したデバッグ手法（今後も再利用可能）

- **RTTバッファの直接memrd手法**（上記4番）。`tools/`配下にスクリプト化は未実施、必要になったら追加する
- **`i2cscan_debug.go`**: I2Cバス上の全アドレス(0x03-0x77)をスキャンしてACKするものを列挙する一時デバッグコード。今回は全センサーのアドレス(0x10, 0x18, 0x40, 0x5C, 0x68)がすべて正常応答していることの確認に使った。現在`main.go`から`scanI2C()`呼び出しをコメントアウト/削除するか検討中（一時debug用のため本来は削除予定）
- **PCからシンボル解決**: `arm-none-eabi-addr2line -e <elf> -f -C <PC>` でクラッシュ位置を関数名まで特定できる

## 未対応・要フォローアップ

- `i2cscan_debug.go` と `main.go` 内の `scanI2C()` 呼び出し、`sensor_bmx055.go`内の`wakeBMX055()`のステップ毎`logLine`呼び出しは調査用の暫定コード。整理して削除する必要がある
- `-serial=rtt`でビルドしたデバッグ用hexと、本来のMakefile経由でビルドされるhexが今は混在している。最終的に通常の`make build`（RTT無効）に戻して再検証が必要

---

## 追加で発見した第2の問題: `heap alloc in interrupt` panic（解決済み, 2026-08-17）

BMX055の`wakeBMX055()`修正後、advertisingは正常化し、macOSの`bleak`から実際にGATT接続を試みたところ、接続直後に**新しいpanicが発生し、advertisingごと完全に停止**した。

### 症状

`gatt_check.py`（`a0b40121`等のcharacteristicをRead）で接続を試みると、Python側は`TimeoutError`で失敗する。直後にRTTバッファ（`nrfjprog --memrd 0x20007800 --n 1024`）を読むと:

```
panic: runtime error at 0x0002d433: heap alloc in interrupt
```

`arm-none-eabi-addr2line -e debug.elf -f -C 0x0002d433` → `main.startBMX055Service$1`（`startBMX055Service()`内の`go func(){ for { ... } }()`匿名関数、1秒毎にセンサー値をcharacteristicへWriteしているgoroutine）。

このpanicの後は完全に停止し、`make scan`でも`NOT FOUND`（wakeBMX055のpanicと同様、advertisingごと止まる。「advertisingが継続していればpanicしていない証拠」という前提はこの調査全体を通じて誤りだったことが再確認された）。

### 現時点でわかっていること

- LPS22HB/HDC2010/スイッチのcharacteristic.Write()は同種の処理をしているが、**接続なし(advertisingのみ)の状態では一度もこのpanicが発生していない**
- BMX055を組み込んで初めて、かつ**実際にBLE接続が発生した直後**にのみ再現した
- → SoftDevice側でBLE接続イベント/notify処理が実際の割り込みハンドラ内でGoコードを呼び出しており、その経路でヒープ確保（スライスリテラル `[]byte{...}` や `b[:]` 変換など、TinyGoのエスケープ解析では容易にヒープに逃げる）が発生するとpanicする、というTinyGo/nRF SoftDeviceでの既知の類の問題と推測される
- BMX055特有のバグというより、**Notify購読中のcharacteristic.Write()全般に潜在する問題**の可能性が高い（LPS22HB/HDC2010/スイッチでも接続中にNotifyが飛べば同様に再現する可能性がある。未検証）

### 調査 (2026-08-17)

優先度順に4項目で調査した:

1. **既知issue検索(旧項目3)**: [tinygo-org/bluetooth#176](https://github.com/tinygo-org/bluetooth/issues/176) が完全に一致する既知issue。nRF51/52でnusserverサンプルを動かすと同じ`heap alloc in interrupt`が発生し、原因はSoftDeviceへ渡すCの構造体がGoのエスケープ解析でヒープに逃げること。**未解決のまま放置されている**issue。関連する[tinygo-org/tinygo#2871](https://github.com/tinygo-org/tinygo/issues/2871)「Allow heap allocations in interrupts」も未解決 — 割り込み中のヒープ確保はTinyGoのGC設計上、恒久的に禁止する方針
2. **ライブラリソース確認(旧項目2)**: `tinygo.org/x/bluetooth@v0.13.0`の`gatts_sd.go`を直接確認したところ、`sd_ble_gatts_hvx`/`sd_ble_gatts_value_set`へ渡す内部パラメータ構造体は**すでに`_noescape`というCラッパー関数でエスケープ対策済み**だった（issue #176の「ワークアラウンド」が本体に反映済み）。つまりライブラリ内部の構造体は原因ではない
3. **グローバルバッファ化(旧項目1、対応案として先に実施)**: 呼び出し側で`characteristic.Write()`に渡すバイト列をローカル変数のスライス化からパッケージレベルのグローバル配列に変更(`lps22hbNotifyBuf`等)。**→ これだけでは解消しなかった**。パニック位置が`main.startBMX055Service$1`から`main.main$1`に変わったのみ
4. **再現条件の切り分け(旧項目4)**: 3の修正後に切り分けた結果、新しいパニック位置`main.main$1`から**真犯人が判明** — `main()`内の`adapter.SetConnectHandler`のコールバック（接続イベントで割り込みから直接呼ばれる）にあった`println("device connected:", device.Address.String())`。`device.Address.String()`の文字列変換がヒープ確保していた。**characteristic.Write()もsensor系コードも実は無関係だった**

### 対処

`main.go`の接続ハンドラから`device.Address.String()`呼び出しを削除（固定文字列の`println("device connected")`に変更）。これで解決。

**検証結果**: `bleak`から接続し、LPS22HB/HDC2010/Accel/Gyro/Mag/SW_TOP/SW_SIDE全キャラクタリスティックにNotify購読、20秒間クラッシュなく安定して受信できることを確認。

### 追加調査: グローバルバッファ化(3番)は本当に必要だったか (2026-08-17)

`characteristic.Write()`は`main()`の接続ハンドラとは異なり、**私たちが自分のgoroutineから呼んでいるだけの普通の関数呼び出し**であり、割り込みハンドラそのものではない。`tinygo.org/x/bluetooth@v0.13.0`の`adapter_sd.go`を確認すると、割り込みコンテキストで実際に動くのは`interrupt.New(nrf.IRQ_SWI2, ...)`に登録された`handleEvent()`と、そこから呼ばれる`connectHandler`/`WriteEvent`コールバックだけ。周期的なセンサーNotify用goroutine内の`char.Write(b[:])`はこの経路に含まれないため、**グローバルバッファ化は不要だった**と結論。`sensor_lps22hb.go`/`sensor_hdc2010.go`/`sensor_bmx055.go`/`ble_io.go`のグローバルバッファをローカル変数に戻した（実機再検証済み、問題なし）。

### 第3の問題: Buzzerの`WriteEvent`でも同種のpanicを発見・修正 (2026-08-17)

`ble_io.go`の`WriteEvent`コールバック（LED1/LED2/Buzzer）は`SetConnectHandler`と同じ`handleEvent()`経由で**割り込みコンテキストから直接呼ばれる**。このうちBuzzerのハンドラ`handleBuzzerWrite()`は`go buzz(gen, durationMs)`で**新しいgoroutineを起動**していた。TinyGoでの`go`文はgoroutine用スタックをヒープから確保するため、これも`heap alloc in interrupt`の対象になると推測し、実機でBuzzer characteristicに書き込んで検証したところ、想定通り再現した（`main.startIOService$3` = Buzzerの`WriteEvent`クロージャでpanic）。

**修正**: `WriteEvent`側は単純な変数書き込み（`buzzerRequestedMs`/`buzzerRequestGen`の更新）のみ行うようにし、実際にブザーを鳴らす処理は起動時に一度だけ立ち上げる常駐goroutine（`buzzWorker`。スイッチ監視の`pollSwitches`と同じポーリング方式）に移した。割り込みコンテキストのコールバックから新規goroutineを起動しない、という原則を徹底。

**検証結果**: 修正後、Buzzer characteristicへの書き込み（response有り/無し両方）→接続維持→切断までクラッシュなく完了することを確認。

### 教訓（一般化）

`SetConnectHandler`・`WriteEvent`はいずれも`tinygo.org/x/bluetooth`のSoftDevice実装で**実際の割り込みハンドラから直接呼ばれる**。この中では:
- 文字列の書式化・連結（`.String()`, `fmt.Sprintf`等）
- `go`文によるgoroutine起動
- その他ヒープ確保を伴うあらゆる操作（スライスの`append`によるサイズ拡張、mapへの新規要素追加等）

を行ってはならない。単純な数値/構造体フィールドの読み書きに留め、重い処理は常駐goroutineに委譲してポーリングまたは（サイズ固定の）チャネルで受け渡す設計にする。

### 副産物として気づいた別の小さな問題（未調査）

接続中のNotify受信テストで、**LPS22HBだけ一度もNotifyが飛んでこなかった**（他の全センサーは正常に届いた）。`readLPS22HB()`がエラーを返し続けている可能性がある（現状はエラーを黙って握りつぶしているだけでログに出していない）。次回軽く調査する価値あり。
