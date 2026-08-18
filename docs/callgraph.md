# コールグラフ（main.go起点）

`main`パッケージのGo関数を対象としたコールグラフ。`tinygo.org/x/bluetooth`や`machine`など外部ライブラリの呼び出しは葉として明示している。

```
main()
├── adapter.Enable()                                    [tinygo.org/x/bluetooth]
├── adapter.SetConnectHandler(func)                      ※登録のみ。切断/接続イベントで非同期に呼ばれる（割込みコンテキスト）
│     └── (connected時) println(...)                     ※heap alloc in interrupt回避のためlogLine()は使わない
├── adapter.DefaultAdvertisement() / adv.Configure() / adv.Start()
├── startNUSService()                                    [ble_nus.go]
│     └── adapter.AddService(...)                        NUS RX/TXキャラクタリスティック登録（RXは未実装、無視）
├── startIOService()                                      [ble_io.go]
│     ├── adapter.AddService(...)                        LED1/LED2/Buzzer(WriteEvent)、SW_TOP/SW_SIDE登録
│     │     ├── (LED1/LED2 WriteEvent, 割込みコンテキスト)
│     │     │     └── setLED(pin, on)
│     │     └── (Buzzer WriteEvent, 割込みコンテキスト)
│     │           └── buzzerRequestedMs/buzzerRequestGen を更新するのみ（goroutineは起こさない）
│     ├── go pollSwitches(swTopChar, swSideChar)
│     │     ├── switchPressed(pin) ×2
│     │     └── (変化時) swTopChar.Write / swSideChar.Write        [tinygo.org/x/bluetooth]
│     └── go buzzWorker()
│           └── buzz(gen, durationMs)
│                 └── buzzer.Set/Get/Low                  [machine]
├── startLPS22HBService()                                 [sensor_lps22hb.go]
│     ├── adapter.AddService(...)
│     └── go func(){ ループ: 1秒毎 }
│           ├── readLPS22HB()
│           │     ├── regWrite(addr, reg, val)            [i2c.go]
│           │     └── regRead(addr, reg, buf)             [i2c.go]
│           ├── convert.EncodeLPS22HB(pressure, temp)      [convert/lps22hb.go]
│           └── char.Write(b)                              [tinygo.org/x/bluetooth]
├── startHDC2010Service()                                 [sensor_hdc2010.go]
│     ├── adapter.AddService(...)
│     └── go func(){ ループ: 1秒毎 }
│           ├── readHDC2010()
│           │     ├── regWrite(...)                        [i2c.go]
│           │     └── regRead(...)                         [i2c.go]
│           ├── convert.EncodeHDC2010(temp, humidity)       [convert/hdc2010.go]
│           └── char.Write(b)
├── startBMX055Service()                                  [sensor_bmx055.go]
│     ├── wakeBMX055()
│     │     └── regWriteRetry(addr, reg, val, attempts, delay)
│     │           └── regWrite(...)                        [i2c.go]
│     ├── readMagTrim()
│     │     └── regRead(...) ×多数                          [i2c.go]
│     ├── adapter.AddService(...)
│     └── go func(){ ループ: 1秒毎 }
│           ├── readAccel()
│           │     ├── regRead(...)                          [i2c.go]
│           │     └── convert.DecodeAccel12Bit(lsb,msb) ×3   [convert/bmx055_accel.go]
│           ├── convert.EncodeAccel(x,y,z) → char.Write(b)
│           ├── readGyroDutyCycled()
│           │     ├── regWriteRetry(...)  (LPM1=Normal, wake)
│           │     ├── readGyro()
│           │     │     ├── regRead(...)
│           │     │     └── convert.DecodeGyro16Bit(lsb,msb) ×3   [convert/bmx055_gyro.go]
│           │     └── regWriteRetry(...)  (LPM1=Suspend)
│           ├── convert.EncodeGyro(x,y,z) → char.Write(b)
│           ├── readMag(trim)
│           │     ├── regRead(...) ×2
│           │     ├── convert.DecodeMagXY13Bit / DecodeMagZ15Bit / DecodeMagRHall14Bit   [convert/bmx055_mag.go]
│           │     └── convert.CompensateMagX/Y/Z(raw, rawR, trim)
│           └── convert.EncodeMag(x,y,z) → char.Write(b)
├── logLine("advertising...")                              [ble_nus.go]
│     └── nusTXChar.Write(...)  （NUSサービス登録済みなら）
└── select{}   ※永久ブロック

--- 補助/共通 ---
must(action, err)                                          [main.go]
  ├── logLine(msg)   （エラー時）
  └── panic(msg)     （エラー時）

regWrite(addr, reg, value) / regRead(addr, reg, buf)        [i2c.go]
  └── i2cMu.Lock/Unlock → i2c.Tx(...)                        [machine] （BMX055/LPS22HB/HDC2010の3ゴルーチンが共有）
```

## 特徴

- `startXxxService()`群はそれぞれ独立ゴルーチンでセンサーを1秒周期ポーリングし、`i2c.go`の`regWrite`/`regRead`（ミューテックス付き）に全て収束する。
- BLEのWriteEventコールバック（LED/Buzzer）は割込みコンテキストのためheap allocを避け、`buzzWorker`/`pollSwitches`という常駐ゴルーチンとのポーリング連携で処理される。
- ジャイロ（`readGyroDutyCycled`）のみSuspend/Wake制御を挟む（電源管理）。
