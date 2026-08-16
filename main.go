package main

/*
   - GPIO: LED OK
   - GPIO: SW OK
   - GPIO: buzzer 鳴動を ゴルーチンで実装する。キャンセルも。タイマーだと周波数にブレがある気がする。
   - I2C: BMX055 9軸センサー（3軸加速度、3軸磁力、3軸ジャイロ）
   - I2C: HDC2010 温湿度センサー
   - I2C: LPS22HB 気圧センサー
   - USB: からログが出力されてるかも ?
   - BLE: キャラクタリスティクスを設計する。
   - スマホアプリ: ???
   - 電力制御: スリープモード ?
*/

import (
	"context"
	"machine"
	"time"

	"tinygo.org/x/bluetooth"
)

const (
	LED1_PIN    = machine.P0_04 // GPIO P0.04
	LED2_PIN    = machine.P0_07 // GPIO P0.07
	SW_TOP_PIN  = machine.P0_31 // GPIO P0.31
	SW_SIDE_PIN = machine.P0_12 // GPIO P0.12

	BUZZER_PIN = machine.P0_23 // GPIO P0.23

	SCL_PIN = machine.P0_26 // GPIO P0.26
	SDA_PIN = machine.P0_24 // GPIO P0.24
)

var adapter = bluetooth.DefaultAdapter

var led1 = machine.Pin(LED1_PIN)
var led2 = machine.Pin(LED2_PIN)

var sw_top = machine.Pin(SW_TOP_PIN)
var sw_side = machine.Pin(SW_SIDE_PIN)
var buzzer = machine.Pin(BUZZER_PIN)

var i2c = machine.I2C0

func init() {

	// 各ピンを出力モードに設定します。
	led1.Configure(machine.PinConfig{Mode: machine.PinOutput})
	led2.Configure(machine.PinConfig{Mode: machine.PinOutput})
	sw_top.Configure(machine.PinConfig{Mode: machine.PinInput})
	sw_side.Configure(machine.PinConfig{Mode: machine.PinInput})
	buzzer.Configure(machine.PinConfig{Mode: machine.PinOutput})

	//  cancel := make(chan struct{})
	//	go buzz(cancel)

	if err := i2c.Configure(machine.I2CConfig{SCL: SCL_PIN, SDA: SDA_PIN}); err != nil {
		println("could not configure I2C:", err)
		panic(err)
	}
}

func main() {
	must("enable BLE stack", adapter.Enable())

	ctx, cancel := context.WithCancel(context.Background())
	adapter.SetConnectHandler(func(device bluetooth.Device, connected bool) {
		if connected {
			println("device connected:", device.Address.String())
			return
		}

		println("device disconnected:", device.Address.String())
		cancel()
	})

	adv := adapter.DefaultAdvertisement()
	must("config adv", adv.Configure(bluetooth.AdvertisementOptions{
		LocalName: "Go MultiSenser",
		ManufacturerData: []bluetooth.ManufacturerDataElement{
			{CompanyID: 0xffff, Data: []byte{0x01, 0x02}},
		},
	}))
	must("start adv", adv.Start())
	defer adv.Stop()

	println("advertising...")
	address, _ := adapter.Address()

	for {
		select {
		case <-time.After(500 * time.Millisecond):
			println("Go Bluetooth /", address.MAC.String())

			if !sw_top.Get() {
				led1.Set(!led1.Get())
				//				close(cancel)
			}

			if !sw_side.Get() {
				led2.Set(!led2.Get())
			}

		case <-ctx.Done():
			return
		}
	}
}

func buzz(cancel chan struct{}) {

	for {
		select {
		case <-cancel:
			// 中断通知が来た
			return
		default:
			time.Sleep(time.Millisecond * 1)
			buzzer.Set(!buzzer.Get())
		}

	}
}

func must(action string, err error) {
	if err != nil {
		panic("failed to " + action + ": " + err.Error())
	}
}
