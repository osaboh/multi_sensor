package convert

// DecodeGyro16Bit converts a raw gyroscope axis register pair (LSB/MSB) into the
// signed 16-bit ADC count. Unlike the accelerometer, no bit-shift is needed.
func DecodeGyro16Bit(lsb, msb byte) int16 {
	return int16(uint16(lsb) | uint16(msb)<<8)
}

// EncodeGyro packs the raw (uncalibrated) gyroscope ADC counts into the BLE
// characteristic byte layout (a0b40142): int16 LE x, y, z. ±2000°/s range,
// 16.4 LSB/(°/s); physical-unit conversion is left to the client.
func EncodeGyro(x, y, z int16) [6]byte {
	ux, uy, uz := uint16(x), uint16(y), uint16(z)
	return [6]byte{
		byte(ux), byte(ux >> 8),
		byte(uy), byte(uy >> 8),
		byte(uz), byte(uz >> 8),
	}
}
