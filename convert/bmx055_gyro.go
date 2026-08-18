package convert

// DecodeGyro16Bitは、ジャイロの軸レジスタペア（LSB/MSB）を符号付き16ビット
// ADCカウント値に変換する。加速度センサーと異なり、ビットシフトは不要。
func DecodeGyro16Bit(lsb, msb byte) int16 {
	return int16(uint16(lsb) | uint16(msb)<<8)
}

// EncodeGyroは、ジャイロの生（未較正）ADCカウント値をBLE
// キャラクタリスティックのバイトレイアウト（a0b40142）に詰める:
// int16 LE x, y, z。±2000°/sレンジ、16.4 LSB/(°/s)。物理値への変換は
// クライアント側の責務とする。
func EncodeGyro(x, y, z int16) [6]byte {
	ux, uy, uz := uint16(x), uint16(y), uint16(z)
	return [6]byte{
		byte(ux), byte(ux >> 8),
		byte(uy), byte(uy >> 8),
		byte(uz), byte(uz >> 8),
	}
}
