package main

import (
	"machine"

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

	// LEDはactive-low（Low=点灯）。Configure直後のGPIO出力はLowがデフォルトの
	// ため、明示的にHigh(消灯)へ倒さないと起動直後に両LEDが点灯してしまう。
	setLED(led1, false)
	setLED(led2, false)

	if err := i2c.Configure(machine.I2CConfig{SCL: SCL_PIN, SDA: SDA_PIN}); err != nil {
		println("could not configure I2C:", err)
		panic(err)
	}
}

func main() {
	must("enable BLE stack", adapter.Enable())

	adapter.SetConnectHandler(func(device bluetooth.Device, connected bool) {
		// This callback runs from interrupt context (SoftDevice BLE event
		// dispatch): no heap allocation allowed here, so no device.Address
		// formatting / string concatenation / logLine — see
		// docs/debug-log-bmx055-wake.md ("heap alloc in interrupt").
		//
		// No action needed on disconnect: the library's own event handler
		// (adapter_nrf528xx-*.go) already calls sd_ble_gap_adv_start to
		// resume advertising before this callback even runs. Previously this
		// called cancel() to unblock main() and return, which ran
		// `defer adv.Stop()` right after the library had just restarted
		// it — advertising never came back after the first disconnect.
		if connected {
			println("device connected")
			return
		}
		println("device disconnected")
	})

	adv := adapter.DefaultAdvertisement()
	must("config adv", adv.Configure(bluetooth.AdvertisementOptions{
		LocalName: "Go MultiSenser",
		ManufacturerData: []bluetooth.ManufacturerDataElement{
			{CompanyID: 0xffff, Data: []byte{0x01, 0x02}},
		},
	}))
	must("start adv", adv.Start())

	startNUSService()
	startIOService()
	startLPS22HBService()
	startHDC2010Service()
	startBMX055Service()

	logLine("advertising...")
	select {}
}

func must(action string, err error) {
	if err != nil {
		msg := "failed to " + action + ": " + err.Error()
		logLine(msg)
		panic(msg)
	}
}
