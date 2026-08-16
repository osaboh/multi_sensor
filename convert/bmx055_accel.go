package convert

// DecodeAccel12Bit converts a raw accelerometer axis register pair (LSB/MSB) into
// the signed 12-bit ADC count. The register stores the 12-bit value left-shifted 4
// bits (the low nibble is reserved/new-data-flag), so recovering it is an arithmetic
// right shift on the reassembled int16.
func DecodeAccel12Bit(lsb, msb byte) int16 {
	raw16 := int16(uint16(lsb) | uint16(msb)<<8)
	return raw16 >> 4
}

// EncodeAccel packs the raw (uncalibrated) accelerometer ADC counts into the BLE
// characteristic byte layout (a0b40141): int16 LE x, y, z. ±2g range, 0.98 mg/LSB;
// physical-unit conversion is left to the client.
func EncodeAccel(x, y, z int16) [6]byte {
	ux, uy, uz := uint16(x), uint16(y), uint16(z)
	return [6]byte{
		byte(ux), byte(ux >> 8),
		byte(uy), byte(uy >> 8),
		byte(uz), byte(uz >> 8),
	}
}
