package convert

// DecodeAccel12Bitは、加速度センサーの軸レジスタペア（LSB/MSB）を符号付き
// 12ビットADCカウント値に変換する。レジスタは12ビット値を4ビット左シフト
// して格納している（下位ニブルは予約/new-dataフラグ）ため、復元には
// 再構成したint16に対する算術右シフトを行う。
func DecodeAccel12Bit(lsb, msb byte) int16 {
	raw16 := int16(uint16(lsb) | uint16(msb)<<8)
	return raw16 >> 4
}

// EncodeAccelは、加速度センサーの生（未較正）ADCカウント値をBLE
// キャラクタリスティックのバイトレイアウト（a0b40141）に詰める:
// int16 LE x, y, z。±2gレンジ、0.98 mg/LSB。物理値への変換は
// クライアント側の責務とする。
func EncodeAccel(x, y, z int16) [6]byte {
	ux, uy, uz := uint16(x), uint16(y), uint16(z)
	return [6]byte{
		byte(ux), byte(ux >> 8),
		byte(uy), byte(uy >> 8),
		byte(uz), byte(uz >> 8),
	}
}
