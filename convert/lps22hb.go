package convert

import "math"

// DecodeLPS22HBPressureは、24ビットPRESS_OUT（XL/L/H）レジスタをhPaに変換する。
func DecodeLPS22HBPressure(xl, l, h byte) float64 {
	raw := int32(xl) | int32(l)<<8 | int32(h)<<16
	return float64(raw) / 4096.0
}

// DecodeLPS22HBTemperatureは、16ビットTEMP_OUT（L/H）レジスタを℃に変換する。
func DecodeLPS22HBTemperature(l, h byte) float64 {
	raw := int16(uint16(l) | uint16(h)<<8)
	return float64(raw) / 100.0
}

// EncodeLPS22HBは、気圧(hPa)と温度(℃)をBLEキャラクタリスティックの
// バイトレイアウト（a0b40121）に詰める: int16 LE 1013.25hPaからの偏差×100,
// int16 LE temp×100。
func EncodeLPS22HB(pressureHPa, temperatureC float64) [4]byte {
	pressureDev := uint16(int16(math.Round((pressureHPa - 1013.25) * 100)))
	temp := uint16(int16(math.Round(temperatureC * 100)))
	return [4]byte{
		byte(pressureDev), byte(pressureDev >> 8),
		byte(temp), byte(temp >> 8),
	}
}
