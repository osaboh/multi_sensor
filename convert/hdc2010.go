package convert

import "math"

// DecodeHDC2010Temperatureは、16ビットTEMP（LSB/MSB）レジスタを℃に変換する。
func DecodeHDC2010Temperature(lsb, msb byte) float64 {
	raw := uint16(lsb) | uint16(msb)<<8
	return float64(raw)*165.0/65536.0 - 40.0
}

// DecodeHDC2010Humidityは、16ビットHUMIDITY（LSB/MSB）レジスタを%RHに変換する。
func DecodeHDC2010Humidity(lsb, msb byte) float64 {
	raw := uint16(lsb) | uint16(msb)<<8
	return float64(raw) * 100.0 / 65536.0
}

// EncodeHDC2010は、温度(℃)と湿度(%RH)をBLEキャラクタリスティックの
// バイトレイアウト（a0b40131）に詰める: int16 LE temp×100, int16 LE humidity×100。
func EncodeHDC2010(temperatureC, humidityPct float64) [4]byte {
	temp := uint16(int16(math.Round(temperatureC * 100)))
	humidity := uint16(int16(math.Round(humidityPct * 100)))
	return [4]byte{
		byte(temp), byte(temp >> 8),
		byte(humidity), byte(humidity >> 8),
	}
}
