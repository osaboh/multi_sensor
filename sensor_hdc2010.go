package main

import (
	"time"

	"tinygo.org/x/bluetooth"

	"main/convert"
)

const (
	hdc2010Addr = 0x40

	hdc2010ConfigAMM   = 0x0E
	hdc2010MeasConfig  = 0x0F
	hdc2010TempLSB     = 0x00
	hdc2010HumidityLSB = 0x02
)

var (
	hdc2010ServiceUUID = bluetooth.NewUUID([16]byte{0xa0, 0xb4, 0x01, 0x30, 0x92, 0x6d, 0x4d, 0x61, 0x98, 0xdf, 0x8c, 0x5c, 0x62, 0xee, 0x53, 0xb3})
	hdc2010CharUUID    = bluetooth.NewUUID([16]byte{0xa0, 0xb4, 0x01, 0x31, 0x92, 0x6d, 0x4d, 0x61, 0x98, 0xdf, 0x8c, 0x5c, 0x62, 0xee, 0x53, 0xb3})
)

func startHDC2010Service() {
	var char bluetooth.Characteristic
	must("add HDC2010 service", adapter.AddService(&bluetooth.Service{
		UUID: hdc2010ServiceUUID,
		Characteristics: []bluetooth.CharacteristicConfig{
			{
				Handle: &char,
				UUID:   hdc2010CharUUID,
				Value:  make([]byte, 4),
				Flags:  bluetooth.CharacteristicReadPermission | bluetooth.CharacteristicNotifyPermission,
			},
		},
	}))

	go func() {
		for {
			if temp, humidity, err := readHDC2010(); err == nil {
				b := convert.EncodeHDC2010(temp, humidity)
				char.Write(b[:])
			}
			time.Sleep(time.Second)
		}
	}()
}

// readHDC2010 triggers a one-shot temperature+humidity measurement and reads
// the result, mirroring adv_env/device.ino's measureHumidity().
func readHDC2010() (temperatureC, humidityPct float64, err error) {
	if err = regWrite(hdc2010Addr, hdc2010ConfigAMM, 0x00); err != nil { // AMM off
		return
	}
	if err = regWrite(hdc2010Addr, hdc2010MeasConfig, 0x01); err != nil { // start measurement
		return
	}
	for {
		var status [1]byte
		if err = regRead(hdc2010Addr, hdc2010MeasConfig, status[:]); err != nil {
			return
		}
		if status[0]&0x01 == 0x00 { // measurement complete
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	var tbuf [2]byte
	if err = regRead(hdc2010Addr, hdc2010TempLSB, tbuf[:]); err != nil {
		return
	}
	var hbuf [2]byte
	if err = regRead(hdc2010Addr, hdc2010HumidityLSB, hbuf[:]); err != nil {
		return
	}

	temperatureC = convert.DecodeHDC2010Temperature(tbuf[0], tbuf[1])
	humidityPct = convert.DecodeHDC2010Humidity(hbuf[0], hbuf[1])
	return
}
