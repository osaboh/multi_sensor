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
	bmx055GyrLPM1Suspend = 0x80 // LPM1 bit7（suspend）、BMG160データシートのレジスタ0x11参照

	// Suspendモードから測定値が安定するまでの起床時間。BMG160データシートの
	// twusm仕様（30ms typical、8ページ）は実際には不十分だった: duty-cycling
	// 有効時、静止状態でもジャイロY軸だけが-34〜+666°/sと大きく暴れ、X/Zは
	// 暴れなかった。これはY軸の共振ループがSuspendからの復帰後、データシートの
	// typical値が示唆するより長く安定化に時間を要することを示している。実機で
	// 二分探索した結果: 30/50msは不安定、75msは境界（時々外れ値あり）、90msと
	// 100msはいずれも完全に安定（Y軸ノイズが約1-5°/sでX/Zと同等）。90msを
	// 安定性が確認できた最小値として採用した。
	bmx055GyrWakeSettle = 90 * time.Millisecond

	bmx055MagPwrCntl1 = 0x4B
	bmx055MagPwrCntl2 = 0x4C
	bmx055MagDX       = 0x42
	bmx055MagRHall    = 0x48

	// 磁力センサーのトリム（"dig_*"）レジスタアドレス、BMM050互換。
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

	// BMX055ループ（加速度+ジャイロ+磁力が1つのゴルーチン/sleepを共有）の
	// Notify間隔の範囲。下限はジャイロのduty-cycle起床時間90ms+3センサー分の
	// I2C読み取りオーバーヘッドに余裕を持たせた値。上限は妥当な範囲で任意に設定。
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

// bmx055IntervalMsは加速度+ジャイロ+磁力ループのsleep間隔（ミリ秒）。
// Intervalキャラクタリスティックの WriteEvent（割込みコンテキスト:
// atomicなストアのみ、allocationなし）から書き込まれ、サービスの
// ゴルーチンが毎サイクル読み取る。
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
				Value: []byte{0xE8, 0x03}, // 1000ms LE、上のデフォルト値と一致
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

// wakeBMX055は、adv_env/device.inoのsuspendPeripheralDevices()相当の処理で
// 入ったdeep-suspendモードから加速度センサーとジャイロを復帰させ、
// ±2g / ±2000°/sレンジを設定し（docs/ble-protocol-reference.md記載の
// 感度と一致）、磁力センサーを起動する。レジスタ値は
// https://github.com/kriswiner/BMX-055 を参照した。
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

	// ジャイロはduty-cycle制御されている（readGyroDutyCycled参照）:
	// サンプル読み取りの瞬間だけ起こし、それ以外はSuspendモードのままにする
	// （BMG160データシートによればNormalモードの5mAに対しSuspendは25µA）。
	// 設定レジスタはSuspend中も保持されるため、ここで一度設定すればよい。
	if err := regWriteRetry(bmx055GyrAddr, bmx055GyrLPM1, bmx055GyrLPM1Suspend, 5, 20*time.Millisecond); err != nil {
		return errors.New("gyr suspend: " + err.Error())
	}
	return nil
}

// regWriteRetryはレジスタ書き込みを、試行間に待機を挟みながら数回リトライ
// する。磁力センサーのソフトリセット書き込み（PWR_CNTL1=0x82）の直後に
// 必要: リセット完了までの間チップが短時間I2Cをネガティブアクノリッジ
// (NACK)するため、固定の単発待機だけでは短すぎて不安定だった。
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

// readGyroDutyCycledはジャイロをSuspendモードから起こし、
// （実測で確認した）起床時間だけ待ってから読み取り、再びSuspendへ戻す。
// これにより、このボード上で圧倒的に電流消費が大きいジャイロ
// （Normalモードで5mA）は、サンプリングに必要な一瞬だけ電源が入る。
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
