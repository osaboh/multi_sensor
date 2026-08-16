package convert

import "math"

// DecodeHDC2010Temperature converts the raw 16-bit TEMP (LSB/MSB) register into °C.
func DecodeHDC2010Temperature(lsb, msb byte) float64 {
	raw := uint16(lsb) | uint16(msb)<<8
	return float64(raw)*165.0/65536.0 - 40.0
}

// DecodeHDC2010Humidity converts the raw 16-bit HUMIDITY (LSB/MSB) register into %RH.
func DecodeHDC2010Humidity(lsb, msb byte) float64 {
	raw := uint16(lsb) | uint16(msb)<<8
	return float64(raw) * 100.0 / 65536.0
}

// EncodeHDC2010 packs temperature(°C) and humidity(%RH) into the BLE characteristic
// byte layout (a0b40131): int16 LE temp×100, int16 LE humidity×100.
func EncodeHDC2010(temperatureC, humidityPct float64) [4]byte {
	temp := uint16(int16(math.Round(temperatureC * 100)))
	humidity := uint16(int16(math.Round(humidityPct * 100)))
	return [4]byte{
		byte(temp), byte(temp >> 8),
		byte(humidity), byte(humidity >> 8),
	}
}
