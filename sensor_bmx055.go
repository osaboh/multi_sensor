package main

import (
	"encoding/binary"
	"errors"
	"sync/atomic"
	"time"

	"tinygo.org/x/bluetooth"

	"main/convert"
)

const (
	bmx055AccAddr = 0x18
	bmx055GyrAddr = 0x68
	bmx055MagAddr = 0x10

	bmx055AccPMURange = 0x0F
	bmx055AccPMUBW    = 0x10
	bmx055AccPMULPW   = 0x11
	bmx055AccDHBW     = 0x13
	bmx055AccDX       = 0x02

	bmx055GyrRange = 0x0F
	bmx055GyrBW    = 0x10
	bmx055GyrLPM1  = 0x11
	bmx055GyrDX    = 0x02

	bmx055GyrLPM1Normal  = 0x00
	bmx055GyrLPM1Suspend = 0x80 // LPM1 bit7 (suspend), see BMG160 datasheet register 0x11

	// Wake-up time from Suspend mode to a settled measurement. The BMG160
	// datasheet's twusm spec (30ms typical, page 8) proved insufficient in
	// practice: with duty-cycling enabled, the gyro's Y axis specifically
	// showed wild swings (-34 to +666 deg/s while stationary) that X/Z did
	// not, indicating Y's resonator loop needed longer to settle after
	// waking from Suspend than the datasheet's typical value suggests.
	// Bisected empirically on real hardware: 30/50ms unstable, 75ms
	// borderline (occasional outliers), 90ms and 100ms both fully stable
	// (Y axis noise ~1-5 deg/s, matching X/Z). 90ms adopted as the minimum
	// confirmed-stable value.
	bmx055GyrWakeSettle = 90 * time.Millisecond

	bmx055MagPwrCntl1 = 0x4B
	bmx055MagPwrCntl2 = 0x4C
	bmx055MagDX       = 0x42
	bmx055MagRHall    = 0x48

	// Magnetometer trim ("dig_*") register addresses, BMM050-compatible.
	bmm050DigX1    = 0x5D
	bmm050DigY1    = 0x5E
	bmm050DigZ4LSB = 0x62
	bmm050DigX2    = 0x64
	bmm050DigY2    = 0x65
	bmm050DigZ2LSB = 0x68
	bmm050DigZ1LSB = 0x6A
	bmm050DigXYZ1LSB = 0x6C
	bmm050DigZ3LSB = 0x6E
	bmm050DigXY2   = 0x70
	bmm050DigXY1   = 0x71

	// Notify interval bounds for the BMX055 loop (accel+gyro+mag share one
	// goroutine/sleep). Floor keeps headroom above the gyro's 90ms
	// duty-cycle wake time plus I2C read overhead for all three sensors;
	// ceiling is an arbitrary sane upper bound.
	bmx055IntervalMinMs = 150
	bmx055IntervalMaxMs = 5000
)

var (
	bmx055ServiceUUID    = bluetooth.NewUUID([16]byte{0xa0, 0xb4, 0x01, 0x40, 0x92, 0x6d, 0x4d, 0x61, 0x98, 0xdf, 0x8c, 0x5c, 0x62, 0xee, 0x53, 0xb3})
	bmx055AccelCharUUID  = bluetooth.NewUUID([16]byte{0xa0, 0xb4, 0x01, 0x41, 0x92, 0x6d, 0x4d, 0x61, 0x98, 0xdf, 0x8c, 0x5c, 0x62, 0xee, 0x53, 0xb3})
	bmx055GyroCharUUID   = bluetooth.NewUUID([16]byte{0xa0, 0xb4, 0x01, 0x42, 0x92, 0x6d, 0x4d, 0x61, 0x98, 0xdf, 0x8c, 0x5c, 0x62, 0xee, 0x53, 0xb3})
	bmx055MagCharUUID    = bluetooth.NewUUID([16]byte{0xa0, 0xb4, 0x01, 0x43, 0x92, 0x6d, 0x4d, 0x61, 0x98, 0xdf, 0x8c, 0x5c, 0x62, 0xee, 0x53, 0xb3})
	bmx055IntervalCharUUID = bluetooth.NewUUID([16]byte{0xa0, 0xb4, 0x01, 0x44, 0x92, 0x6d, 0x4d, 0x61, 0x98, 0xdf, 0x8c, 0x5c, 0x62, 0xee, 0x53, 0xb3})
)

// bmx055IntervalMs is the accel+gyro+mag loop's sleep interval, in
// milliseconds. Written from the Interval characteristic's WriteEvent
// (interrupt context: atomic store only, no allocation), read by the
// service goroutine every cycle.
var bmx055IntervalMs uint32 = 1000

func startBMX055Service() {
	must("bmx055 wake", wakeBMX055())
	trim, err := readMagTrim()
	must("bmx055 read trim", err)

	var accelChar, gyroChar, magChar bluetooth.Characteristic
	must("add BMX055 service", adapter.AddService(&bluetooth.Service{
		UUID: bmx055ServiceUUID,
		Characteristics: []bluetooth.CharacteristicConfig{
			{
				Handle: &accelChar,
				UUID:   bmx055AccelCharUUID,
				Value:  make([]byte, 6),
				Flags:  bluetooth.CharacteristicReadPermission | bluetooth.CharacteristicNotifyPermission,
			},
			{
				Handle: &gyroChar,
				UUID:   bmx055GyroCharUUID,
				Value:  make([]byte, 6),
				Flags:  bluetooth.CharacteristicReadPermission | bluetooth.CharacteristicNotifyPermission,
			},
			{
				Handle: &magChar,
				UUID:   bmx055MagCharUUID,
				Value:  make([]byte, 12),
				Flags:  bluetooth.CharacteristicReadPermission | bluetooth.CharacteristicNotifyPermission,
			},
			{
				UUID:  bmx055IntervalCharUUID,
				Value: []byte{0xE8, 0x03}, // 1000ms LE, matches the default above
				Flags: bluetooth.CharacteristicWritePermission | bluetooth.CharacteristicWriteWithoutResponsePermission,
				WriteEvent: func(client bluetooth.Connection, offset int, value []byte) {
					if offset != 0 || len(value) != 2 {
						return
					}
					ms := binary.LittleEndian.Uint16(value)
					if ms < bmx055IntervalMinMs {
						ms = bmx055IntervalMinMs
					} else if ms > bmx055IntervalMaxMs {
						ms = bmx055IntervalMaxMs
					}
					atomic.StoreUint32(&bmx055IntervalMs, uint32(ms))
				},
			},
		},
	}))

	go func() {
		for {
			if x, y, z, err := readAccel(); err == nil {
				b := convert.EncodeAccel(x, y, z)
				accelChar.Write(b[:])
			}
			if x, y, z, err := readGyroDutyCycled(); err == nil {
				b := convert.EncodeGyro(x, y, z)
				gyroChar.Write(b[:])
			}
			if x, y, z, err := readMag(trim); err == nil {
				b := convert.EncodeMag(x, y, z)
				magChar.Write(b[:])
			}
			time.Sleep(time.Duration(atomic.LoadUint32(&bmx055IntervalMs)) * time.Millisecond)
		}
	}()
}

// wakeBMX055 brings the accelerometer and gyroscope out of the deep-suspend
// mode entered by adv_env/device.ino's suspendPeripheralDevices() equivalent,
// configures ±2g / ±2000°/s range (matching the sensitivities documented in
// docs/ble-protocol-reference.md), and powers up the magnetometer.
// Register values sourced from https://github.com/kriswiner/BMX-055.
func wakeBMX055() error {
	steps := []struct {
		name string
		addr uint8
		reg  uint8
		val  uint8
		wait time.Duration
	}{
		{"acc range", bmx055AccAddr, bmx055AccPMURange, 0x03, 0},
		{"acc bw", bmx055AccAddr, bmx055AccPMUBW, 0x09, 0},
		{"acc lpw", bmx055AccAddr, bmx055AccPMULPW, 0x00, 0},
		{"acc hbw", bmx055AccAddr, bmx055AccDHBW, 0x00, 0},
		{"gyr range", bmx055GyrAddr, bmx055GyrRange, 0x00, 0},
		{"gyr bw", bmx055GyrAddr, bmx055GyrBW, 0x04, 0},
		{"gyr lpm1", bmx055GyrAddr, bmx055GyrLPM1, 0x00, 0},
		{"mag pwrcntl1 on", bmx055MagAddr, bmx055MagPwrCntl1, 0x01, 5 * time.Millisecond},
		{"mag pwrcntl2 odr", bmx055MagAddr, bmx055MagPwrCntl2, 0x00, 0},
	}
	for _, s := range steps {
		logLine("DEBUG wakeBMX055: " + s.name)
		if err := regWriteRetry(s.addr, s.reg, s.val, 5, 20*time.Millisecond); err != nil {
			return errors.New(s.name + ": " + err.Error())
		}
		if s.wait > 0 {
			time.Sleep(s.wait)
		}
	}

	// The gyroscope is duty-cycled (see readGyroDutyCycled): woken only for
	// the moment it takes to read a sample, otherwise left in Suspend mode
	// (25µA vs. 5mA in Normal mode, per the BMG160 datasheet). Configuration
	// registers survive Suspend, so this only needs to happen once here.
	if err := regWriteRetry(bmx055GyrAddr, bmx055GyrLPM1, bmx055GyrLPM1Suspend, 5, 20*time.Millisecond); err != nil {
		return errors.New("gyr suspend: " + err.Error())
	}
	return nil
}

// regWriteRetry retries a register write a few times with a delay between
// attempts. Needed right after the magnetometer's soft-reset write
// (PWR_CNTL1=0x82): the chip briefly NACKs I2C while the reset completes,
// and a single fixed delay proved too short/unreliable in practice.
func regWriteRetry(addr, reg, value uint8, attempts int, delay time.Duration) error {
	var err error
	for i := 0; i < attempts; i++ {
		if err = regWrite(addr, reg, value); err == nil {
			return nil
		}
		time.Sleep(delay)
	}
	return err
}

func readAccel() (x, y, z int16, err error) {
	var buf [6]byte
	if err = regRead(bmx055AccAddr, bmx055AccDX, buf[:]); err != nil {
		return
	}
	x = convert.DecodeAccel12Bit(buf[0], buf[1])
	y = convert.DecodeAccel12Bit(buf[2], buf[3])
	z = convert.DecodeAccel12Bit(buf[4], buf[5])
	return
}

// readGyroDutyCycled wakes the gyroscope from Suspend mode, waits out its
// datasheet-specified wake-up time, takes a reading, and returns it to
// Suspend — so the gyro (5mA in Normal mode, by far the dominant current
// draw on this board) is only powered up for the fraction of a second it
// takes to sample it.
func readGyroDutyCycled() (x, y, z int16, err error) {
	if err = regWriteRetry(bmx055GyrAddr, bmx055GyrLPM1, bmx055GyrLPM1Normal, 5, 20*time.Millisecond); err != nil {
		return
	}
	time.Sleep(bmx055GyrWakeSettle)

	x, y, z, err = readGyro()

	if e := regWriteRetry(bmx055GyrAddr, bmx055GyrLPM1, bmx055GyrLPM1Suspend, 5, 20*time.Millisecond); e != nil && err == nil {
		err = e
	}
	return
}

func readGyro() (x, y, z int16, err error) {
	var buf [6]byte
	if err = regRead(bmx055GyrAddr, bmx055GyrDX, buf[:]); err != nil {
		return
	}
	x = convert.DecodeGyro16Bit(buf[0], buf[1])
	y = convert.DecodeGyro16Bit(buf[2], buf[3])
	z = convert.DecodeGyro16Bit(buf[4], buf[5])
	return
}

func readMag(trim convert.MagTrim) (x, y, z float32, err error) {
	var buf [6]byte
	if err = regRead(bmx055MagAddr, bmx055MagDX, buf[:]); err != nil {
		return
	}
	var rbuf [2]byte
	if err = regRead(bmx055MagAddr, bmx055MagRHall, rbuf[:]); err != nil {
		return
	}

	rawX := convert.DecodeMagXY13Bit(buf[0], buf[1])
	rawY := convert.DecodeMagXY13Bit(buf[2], buf[3])
	rawZ := convert.DecodeMagZ15Bit(buf[4], buf[5])
	rawR := convert.DecodeMagRHall14Bit(rbuf[0], rbuf[1])

	x = convert.CompensateMagX(rawX, rawR, trim)
	y = convert.CompensateMagY(rawY, rawR, trim)
	z = convert.CompensateMagZ(rawZ, rawR, trim)
	return
}

func readMagTrim() (trim convert.MagTrim, err error) {
	var b1 [1]byte
	if err = regRead(bmx055MagAddr, bmm050DigX1, b1[:]); err != nil {
		return
	}
	trim.DigX1 = int8(b1[0])
	if err = regRead(bmx055MagAddr, bmm050DigY1, b1[:]); err != nil {
		return
	}
	trim.DigY1 = int8(b1[0])
	if err = regRead(bmx055MagAddr, bmm050DigX2, b1[:]); err != nil {
		return
	}
	trim.DigX2 = int8(b1[0])
	if err = regRead(bmx055MagAddr, bmm050DigY2, b1[:]); err != nil {
		return
	}
	trim.DigY2 = int8(b1[0])
	if err = regRead(bmx055MagAddr, bmm050DigXY2, b1[:]); err != nil {
		return
	}
	trim.DigXY2 = int8(b1[0])
	if err = regRead(bmx055MagAddr, bmm050DigXY1, b1[:]); err != nil {
		return
	}
	trim.DigXY1 = b1[0]

	var b2 [2]byte
	if err = regRead(bmx055MagAddr, bmm050DigZ4LSB, b2[:]); err != nil {
		return
	}
	trim.DigZ4 = int16(uint16(b2[0]) | uint16(b2[1])<<8)
	if err = regRead(bmx055MagAddr, bmm050DigZ2LSB, b2[:]); err != nil {
		return
	}
	trim.DigZ2 = int16(uint16(b2[0]) | uint16(b2[1])<<8)
	if err = regRead(bmx055MagAddr, bmm050DigZ1LSB, b2[:]); err != nil {
		return
	}
	trim.DigZ1 = uint16(b2[0]) | uint16(b2[1])<<8
	if err = regRead(bmx055MagAddr, bmm050DigXYZ1LSB, b2[:]); err != nil {
		return
	}
	trim.DigXYZ1 = uint16(b2[0]) | uint16(b2[1])<<8
	if err = regRead(bmx055MagAddr, bmm050DigZ3LSB, b2[:]); err != nil {
		return
	}
	trim.DigZ3 = int16(uint16(b2[0]) | uint16(b2[1])<<8)
	return
}
