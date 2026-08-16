package convert

import "math"

// DecodeLPS22HBPressure converts the raw 24-bit PRESS_OUT (XL/L/H) registers into hPa.
func DecodeLPS22HBPressure(xl, l, h byte) float64 {
	raw := int32(xl) | int32(l)<<8 | int32(h)<<16
	return float64(raw) / 4096.0
}

// DecodeLPS22HBTemperature converts the raw 16-bit TEMP_OUT (L/H) registers into °C.
func DecodeLPS22HBTemperature(l, h byte) float64 {
	raw := int16(uint16(l) | uint16(h)<<8)
	return float64(raw) / 100.0
}

// EncodeLPS22HB packs pressure(hPa) and temperature(°C) into the BLE characteristic
// byte layout (a0b40121): int16 LE pressure-deviation-from-1013.25hPa ×100, int16 LE temp×100.
func EncodeLPS22HB(pressureHPa, temperatureC float64) [4]byte {
	pressureDev := uint16(int16(math.Round((pressureHPa - 1013.25) * 100)))
	temp := uint16(int16(math.Round(temperatureC * 100)))
	return [4]byte{
		byte(pressureDev), byte(pressureDev >> 8),
		byte(temp), byte(temp >> 8),
	}
}
