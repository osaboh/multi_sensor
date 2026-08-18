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
				// コマンドプロトコル未定義（docs/ble-protocol-reference.md）。現状は書き込みを無視する。
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

// logLineは、ローカルのデバッグコンソールに加えて、NUS TXキャラクタ
// リスティック（サービス登録済みの場合）にも1行分のテキストを送信する。
func logLine(s string) {
	println(s)
	if nusReady {
		nusTXChar.Write([]byte(s))
	}
}
