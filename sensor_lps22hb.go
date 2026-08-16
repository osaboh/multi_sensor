package main

import (
	"time"

	"tinygo.org/x/bluetooth"

	"main/convert"
)

const (
	lps22hbAddr = 0x5C

	lps22hbCtrlReg2   = 0x11
	lps22hbStatus     = 0x27
	lps22hbPressOutXL = 0x28
	lps22hbTempOutL   = 0x2B
)

var (
	lps22hbServiceUUID = bluetooth.NewUUID([16]byte{0xa0, 0xb4, 0x01, 0x20, 0x92, 0x6d, 0x4d, 0x61, 0x98, 0xdf, 0x8c, 0x5c, 0x62, 0xee, 0x53, 0xb3})
	lps22hbCharUUID    = bluetooth.NewUUID([16]byte{0xa0, 0xb4, 0x01, 0x21, 0x92, 0x6d, 0x4d, 0x61, 0x98, 0xdf, 0x8c, 0x5c, 0x62, 0xee, 0x53, 0xb3})
)

func startLPS22HBService() {
	var char bluetooth.Characteristic
	must("add LPS22HB service", adapter.AddService(&bluetooth.Service{
		UUID: lps22hbServiceUUID,
		Characteristics: []bluetooth.CharacteristicConfig{
			{
				Handle: &char,
				UUID:   lps22hbCharUUID,
				Value:  make([]byte, 4),
				Flags:  bluetooth.CharacteristicReadPermission | bluetooth.CharacteristicNotifyPermission,
			},
		},
	}))

	go func() {
		for {
			if pressure, temp, err := readLPS22HB(); err == nil {
				b := convert.EncodeLPS22HB(pressure, temp)
				char.Write(b[:])
			}
			time.Sleep(time.Second)
		}
	}()
}

// readLPS22HB triggers a one-shot pressure+temperature measurement and reads
// the result, mirroring adv_env/device.ino's measurePressure().
func readLPS22HB() (pressureHPa, temperatureC float64, err error) {
	if err = regWrite(lps22hbAddr, lps22hbCtrlReg2, 0x11); err != nil { // one-shot enable
		return
	}
	for {
		var status [1]byte
		if err = regRead(lps22hbAddr, lps22hbStatus, status[:]); err != nil {
			return
		}
		if status[0]&0x03 == 0x03 { // T_DA and P_DA both ready
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	var pbuf [3]byte
	if err = regRead(lps22hbAddr, lps22hbPressOutXL, pbuf[:]); err != nil {
		return
	}
	var tbuf [2]byte
	if err = regRead(lps22hbAddr, lps22hbTempOutL, tbuf[:]); err != nil {
		return
	}

	pressureHPa = convert.DecodeLPS22HBPressure(pbuf[0], pbuf[1], pbuf[2])
	temperatureC = convert.DecodeLPS22HBTemperature(tbuf[0], tbuf[1])
	return
}
