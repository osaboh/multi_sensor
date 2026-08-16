package convert

import (
	"encoding/binary"
	"math"
)

// DecodeMagXY13Bit converts a raw magnetometer X or Y axis register pair
// (LSB/MSB) into the signed 13-bit ADC count (register stores it left-shifted
// 3 bits).
func DecodeMagXY13Bit(lsb, msb byte) int16 {
	raw16 := int16(uint16(lsb) | uint16(msb)<<8)
	return raw16 >> 3
}

// DecodeMagZ15Bit converts the raw magnetometer Z axis register pair
// (LSB/MSB) into the signed 15-bit ADC count (register stores it left-shifted
// 1 bit).
func DecodeMagZ15Bit(lsb, msb byte) int16 {
	raw16 := int16(uint16(lsb) | uint16(msb)<<8)
	return raw16 >> 1
}

// DecodeMagRHall14Bit converts the raw RHALL register pair (LSB/MSB) into the
// unsigned 14-bit ADC count (register stores it left-shifted 2 bits).
func DecodeMagRHall14Bit(lsb, msb byte) uint16 {
	raw16 := uint16(lsb) | uint16(msb)<<8
	return raw16 >> 2
}

// MagTrim holds the BMM050-style factory calibration ("trim") values, read
// once from the sensor's trim extended registers at startup. Field types
// match the Bosch reference driver exactly.
type MagTrim struct {
	DigX1, DigY1, DigX2, DigY2 int8
	DigXY1                     uint8
	DigXY2                     int8
	DigZ1                      uint16
	DigZ2, DigZ3, DigZ4        int16
	DigXYZ1                    uint16
}

// hallRatio is the RHALL-normalized intermediate value shared by the X and Y
// compensation formulas (Bosch bmm050_compensate_X/Y_float).
func hallRatio(dataR uint16, trim MagTrim) float32 {
	if dataR == 0 {
		return 0
	}
	return float32(trim.DigXYZ1)*16384.0/float32(dataR) - 16384.0
}

// CompensateMagX applies the Bosch BMM050 compensation formula to a raw
// X-axis magnetometer reading, returning µT.
func CompensateMagX(mdataX int16, dataR uint16, trim MagTrim) float32 {
	ratio := hallRatio(dataR, trim)
	crossTerm := float32(trim.DigXY2)*(ratio*ratio/268435456.0) + ratio*float32(trim.DigXY1)/16384.0
	scaled := (crossTerm + 256.0) * (float32(trim.DigX2) + 160.0)
	return ((float32(mdataX)*scaled)/8192.0+float32(trim.DigX1)*8.0)/16.0
}

// CompensateMagY applies the Bosch BMM050 compensation formula to a raw
// Y-axis magnetometer reading, returning µT.
func CompensateMagY(mdataY int16, dataR uint16, trim MagTrim) float32 {
	ratio := hallRatio(dataR, trim)
	crossTerm := float32(trim.DigXY2)*(ratio*ratio/268435456.0) + ratio*float32(trim.DigXY1)/16384.0
	scaled := (crossTerm + 256.0) * (float32(trim.DigY2) + 160.0)
	return ((float32(mdataY)*scaled)/8192.0+float32(trim.DigY1)*8.0)/16.0
}

// CompensateMagZ applies the Bosch BMM050 compensation formula to a raw
// Z-axis magnetometer reading, returning µT.
func CompensateMagZ(mdataZ int16, dataR uint16, trim MagTrim) float32 {
	numerator := (float32(mdataZ)-float32(trim.DigZ4))*131072.0 -
		float32(trim.DigZ3)*(float32(dataR)-float32(trim.DigXYZ1))
	denominator := (float32(trim.DigZ2) + float32(trim.DigZ1)*float32(dataR)/32768.0) * 4.0
	return (numerator / denominator) / 16.0
}

// EncodeMag packs the already-compensated magnetometer values (µT) into the
// BLE characteristic byte layout (a0b40143): float32 LE x, y, z.
func EncodeMag(x, y, z float32) [12]byte {
	var b [12]byte
	binary.LittleEndian.PutUint32(b[0:4], math.Float32bits(x))
	binary.LittleEndian.PutUint32(b[4:8], math.Float32bits(y))
	binary.LittleEndian.PutUint32(b[8:12], math.Float32bits(z))
	return b
}
