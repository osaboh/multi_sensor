package main

import (
	"encoding/binary"
	"machine"
	"time"

	"tinygo.org/x/bluetooth"
)

var (
	ioServiceUUID  = bluetooth.NewUUID([16]byte{0xa0, 0xb4, 0x01, 0x00, 0x92, 0x6d, 0x4d, 0x61, 0x98, 0xdf, 0x8c, 0x5c, 0x62, 0xee, 0x53, 0xb3})
	led1CharUUID   = bluetooth.NewUUID([16]byte{0xa0, 0xb4, 0x01, 0x01, 0x92, 0x6d, 0x4d, 0x61, 0x98, 0xdf, 0x8c, 0x5c, 0x62, 0xee, 0x53, 0xb3})
	led2CharUUID   = bluetooth.NewUUID([16]byte{0xa0, 0xb4, 0x01, 0x02, 0x92, 0x6d, 0x4d, 0x61, 0x98, 0xdf, 0x8c, 0x5c, 0x62, 0xee, 0x53, 0xb3})
	buzzerCharUUID = bluetooth.NewUUID([16]byte{0xa0, 0xb4, 0x01, 0x03, 0x92, 0x6d, 0x4d, 0x61, 0x98, 0xdf, 0x8c, 0x5c, 0x62, 0xee, 0x53, 0xb3})
	swTopCharUUID  = bluetooth.NewUUID([16]byte{0xa0, 0xb4, 0x01, 0x11, 0x92, 0x6d, 0x4d, 0x61, 0x98, 0xdf, 0x8c, 0x5c, 0x62, 0xee, 0x53, 0xb3})
	swSideCharUUID = bluetooth.NewUUID([16]byte{0xa0, 0xb4, 0x01, 0x12, 0x92, 0x6d, 0x4d, 0x61, 0x98, 0xdf, 0x8c, 0x5c, 0x62, 0xee, 0x53, 0xb3})
)

// Buzzer state, updated by the WriteEvent callback (which runs from
// interrupt context — see docs/debug-log-bmx055-wake.md) and consumed by the
// always-running buzzWorker goroutine below. WriteEvent must never spawn a
// goroutine itself (that allocates a new stack, which panics with "heap
// alloc in interrupt"); it only ever does plain word-sized writes here.
var (
	buzzerRequestedMs uint16
	buzzerRequestGen  uint32
)

func startIOService() {
	var swTopChar, swSideChar bluetooth.Characteristic

	must("add I/O service", adapter.AddService(&bluetooth.Service{
		UUID: ioServiceUUID,
		Characteristics: []bluetooth.CharacteristicConfig{
			{
				UUID:  led1CharUUID,
				Value: []byte{0x00},
				Flags: bluetooth.CharacteristicWritePermission | bluetooth.CharacteristicWriteWithoutResponsePermission,
				WriteEvent: func(client bluetooth.Connection, offset int, value []byte) {
					if offset != 0 || len(value) != 1 {
						return
					}
					setLED(led1, value[0] == 0x01)
				},
			},
			{
				UUID:  led2CharUUID,
				Value: []byte{0x00},
				Flags: bluetooth.CharacteristicWritePermission | bluetooth.CharacteristicWriteWithoutResponsePermission,
				WriteEvent: func(client bluetooth.Connection, offset int, value []byte) {
					if offset != 0 || len(value) != 1 {
						return
					}
					setLED(led2, value[0] == 0x01)
				},
			},
			{
				UUID:  buzzerCharUUID,
				Value: []byte{0x00, 0x00},
				Flags: bluetooth.CharacteristicWritePermission | bluetooth.CharacteristicWriteWithoutResponsePermission,
				WriteEvent: func(client bluetooth.Connection, offset int, value []byte) {
					if offset != 0 || len(value) != 2 {
						return
					}
					buzzerRequestedMs = binary.LittleEndian.Uint16(value)
					buzzerRequestGen++
					if buzzerRequestedMs == 0 {
						buzzer.Low()
					}
				},
			},
			{
				Handle: &swTopChar,
				UUID:   swTopCharUUID,
				Value:  []byte{switchByte(switchPressed(sw_top))},
				Flags:  bluetooth.CharacteristicReadPermission | bluetooth.CharacteristicNotifyPermission,
			},
			{
				Handle: &swSideChar,
				UUID:   swSideCharUUID,
				Value:  []byte{switchByte(switchPressed(sw_side))},
				Flags:  bluetooth.CharacteristicReadPermission | bluetooth.CharacteristicNotifyPermission,
			},
		},
	}))

	go pollSwitches(&swTopChar, &swSideChar)
	go buzzWorker()
}

// setLED drives an active-low LED (Low = on, High = off).
func setLED(pin machine.Pin, on bool) {
	if on {
		pin.Low()
	} else {
		pin.High()
	}
}

// switchPressed reads an active-low tactile switch (pressed = Low).
func switchPressed(pin machine.Pin) bool {
	return !pin.Get()
}

func switchByte(pressed bool) byte {
	if pressed {
		return 0x01
	}
	return 0x00
}

func pollSwitches(swTopChar, swSideChar *bluetooth.Characteristic) {
	lastTop := switchPressed(sw_top)
	lastSide := switchPressed(sw_side)
	for {
		time.Sleep(20 * time.Millisecond)
		if top := switchPressed(sw_top); top != lastTop {
			lastTop = top
			swTopChar.Write([]byte{switchByte(top)})
		}
		if side := switchPressed(sw_side); side != lastSide {
			lastSide = side
			swSideChar.Write([]byte{switchByte(side)})
		}
	}
}

// buzzWorker is the single, always-running goroutine that actually drives
// the buzzer. It polls buzzerRequestGen (same pattern as pollSwitches)
// instead of the WriteEvent callback spawning a fresh goroutine per write.
func buzzWorker() {
	var servedGen uint32
	for {
		time.Sleep(5 * time.Millisecond)
		gen := buzzerRequestGen
		if gen == servedGen {
			continue
		}
		servedGen = gen
		durationMs := buzzerRequestedMs
		if durationMs == 0 {
			continue
		}
		buzz(gen, durationMs)
	}
}

// buzz drives the passive piezo buzzer with a square wave (toggling the pin
// produces the tone; holding it at a steady level would be silent) for the
// requested duration, unless superseded by a newer write.
func buzz(gen uint32, durationMs uint16) {
	deadline := time.Now().Add(time.Duration(durationMs) * time.Millisecond)
	for time.Now().Before(deadline) {
		if gen != buzzerRequestGen {
			return
		}
		buzzer.Set(!buzzer.Get())
		time.Sleep(time.Millisecond)
	}
	if gen == buzzerRequestGen {
		buzzer.Low()
	}
}
