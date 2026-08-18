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

// ブザーの状態。WriteEventコールバック（割込みコンテキストで実行される
// — docs/debug-log-bmx055-wake.md参照）から更新され、下の常駐buzzWorker
// ゴルーチンが消費する。WriteEventはゴルーチンを起動してはならない
// （新しいスタックを確保することになり「heap alloc in interrupt」で
// パニックする）。ここでは単純なワード単位の書き込みのみ行う。
var (
	buzzerRequestedMs uint16
	buzzerRequestGen  uint32
)

// ブザーの鳴動方式。以前はソフトウェアループでGPIOをトグルするビット
// バンギング方式だったが（BLEスタック等とのゴルーチン競合による
// time.Sleepのジッターで音が「濁って」聞こえる問題があった）、現在は
// nRF52840のハードウェアPWMペリフェラルでジッターフリーな方形波を生成
// している。任意のGPIOピンをPWMチャンネルにルーティングできる
// （PSEL.OUTはピン固定ではなくソフトウェアで設定可能）ため、
// BUZZER_PINの変更は不要だった。
//
// 松下電工（現Panasonic）の「ホロホロ」チャイムに近づけるため、2周波数を
// 往復するワーブルをまず試したが「近くない」と判定された。実際の「ホロ
// ホロ」音を解析した結果に基づき、単一の固定周波数を鳴らして止める、を
// 繰り返すゲーティング（オン/オフ断続）方式に変更した（周波数変調では
// なく振幅変調が実際の音の特徴だった）。当初の解析値は約2304.5Hz・
// 20ms鳴動/43ms休止だったが、実機で聞いた結果「前より近いが、全体を
// 低音・低速に」とのフィードバックを受け、周波数を約1300Hzへ、
// オン/オフ時間をおよそ1.5倍に調整した。
// buzzerOnMs/buzzerOffMs/buzzerFreqHzは実機での試聴により確定した値
// （松下電工「ホロホロ」チャイムに近い音として1000Hz、オン30ms/オフ65ms
// のゲーティングパターンを採用）。
//
// 【教訓】この値を試聴で追い込む過程で、音程チューニング用に別
// Characteristicを新設したところSoftDeviceのGATT属性テーブル容量を
// 使い切り「panic: failed to add BMX055 service: no memory for
// operation」で起動時にパニックする不具合を起こした（advertisingは既に
// 開始済みだったため接続はできてしまい、一見Android側のGATTキャッシュ
// 問題に見えて誤診断した — RTTログでのpanic確認が有効だった）。以後、
// 一時的な調整用途であっても新規Characteristicの追加はGATT容量を
// 圧迫するため避け、既存Characteristicのペイロードを一時的に拡張する
// 方針にした（このコミットで確定値に戻し、ペイロードも2バイト固定に
// 戻した）。
const (
	buzzerOnMs   = 30
	buzzerOffMs  = 65
	buzzerFreqHz = 1000
)

const buzzerPeriodNS = 1_000_000_000 / buzzerFreqHz

var (
	buzzerPWM = machine.PWM0
	buzzerCh  uint8
)

func initBuzzerPWM() {
	must("buzzer pwm configure", buzzerPWM.Configure(machine.PWMConfig{Period: buzzerPeriodNS}))
	ch, err := buzzerPWM.Channel(buzzer)
	must("buzzer pwm channel", err)
	buzzerCh = ch
	buzzerPWM.Set(buzzerCh, 0)
}

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
				// 2バイト（uint16 LE 鳴動時間ms）。0=即時停止。
				UUID:  buzzerCharUUID,
				Value: []byte{0x00, 0x00},
				Flags: bluetooth.CharacteristicWritePermission | bluetooth.CharacteristicWriteWithoutResponsePermission,
				WriteEvent: func(client bluetooth.Connection, offset int, value []byte) {
					if offset != 0 || len(value) != 2 {
						return
					}
					buzzerRequestedMs = binary.LittleEndian.Uint16(value[0:2])
					buzzerRequestGen++
					if buzzerRequestedMs == 0 {
						buzzerPWM.Set(buzzerCh, 0)
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

	initBuzzerPWM()
	go pollSwitches(&swTopChar, &swSideChar)
	go buzzWorker()
}

// setLEDはactive-lowのLEDを駆動する（Low=点灯、High=消灯）。
func setLED(pin machine.Pin, on bool) {
	if on {
		pin.Low()
	} else {
		pin.High()
	}
}

// switchPressedはactive-lowのタクトスイッチを読み取る（押下時=Low）。
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

// buzzWorkerは実際にブザーを駆動する、唯一の常駐ゴルーチンである。
// WriteEventコールバックが書き込みのたびに新しいゴルーチンを起動する
// のではなく、buzzerRequestGenをポーリングする（pollSwitchesと同じ
// パターン）。
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

// buzzはパッシブ型圧電ブザーをハードウェアPWM経由で駆動する。固定周波数
// （buzzerPeriodNS）をbuzzerOnMs鳴らしてはbuzzerOffMs止める、を繰り返し
// 「ホロホロ」チャイムのオン/オフ断続パターンを再現する。指定時間鳴動
// させるが、新しい書き込みがあれば処理を打ち切る。
func buzz(gen uint32, durationMs uint16) {
	deadline := time.Now().Add(time.Duration(durationMs) * time.Millisecond)
	for time.Now().Before(deadline) {
		if gen != buzzerRequestGen {
			return
		}
		buzzerPWM.Set(buzzerCh, buzzerPWM.Top()/2)
		time.Sleep(buzzerOnMs * time.Millisecond)
		buzzerPWM.Set(buzzerCh, 0)
		time.Sleep(buzzerOffMs * time.Millisecond)
	}
	if gen == buzzerRequestGen {
		buzzerPWM.Set(buzzerCh, 0)
	}
}
