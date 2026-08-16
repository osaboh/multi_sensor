package main

import "tinygo.org/x/bluetooth"

var (
	nusTXChar bluetooth.Characteristic
	nusReady  bool
)

func startNUSService() {
	var rxChar bluetooth.Characteristic
	must("add NUS service", adapter.AddService(&bluetooth.Service{
		UUID: bluetooth.ServiceUUIDNordicUART,
		Characteristics: []bluetooth.CharacteristicConfig{
			{
				Handle: &rxChar,
				UUID:   bluetooth.CharacteristicUUIDUARTRX,
				Flags:  bluetooth.CharacteristicWritePermission | bluetooth.CharacteristicWriteWithoutResponsePermission,
				// Command protocol not yet defined (docs/ble-protocol-reference.md); writes are ignored for now.
			},
			{
				Handle: &nusTXChar,
				UUID:   bluetooth.CharacteristicUUIDUARTTX,
				Flags:  bluetooth.CharacteristicReadPermission | bluetooth.CharacteristicNotifyPermission,
			},
		},
	}))
	nusReady = true
}

// logLine sends a line of text to the NUS TX characteristic (once the
// service is registered), in addition to the local debug console.
func logLine(s string) {
	println(s)
	if nusReady {
		nusTXChar.Write([]byte(s))
	}
}
