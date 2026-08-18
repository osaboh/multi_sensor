package convert

import (
	"encoding/binary"
	"math"
)

// DecodeMagXY13Bitは、磁力センサーのX軸またはY軸のレジスタペア
// （LSB/MSB）を符号付き13ビットADCカウント値に変換する
// （レジスタは3ビット左シフトして格納している）。
func DecodeMagXY13Bit(lsb, msb byte) int16 {
	raw16 := int16(uint16(lsb) | uint16(msb)<<8)
	return raw16 >> 3
}

// DecodeMagZ15Bitは、磁力センサーのZ軸レジスタペア（LSB/MSB）を符号付き
// 15ビットADCカウント値に変換する（レジスタは1ビット左シフトして格納）。
func DecodeMagZ15Bit(lsb, msb byte) int16 {
	raw16 := int16(uint16(lsb) | uint16(msb)<<8)
	return raw16 >> 1
}

// DecodeMagRHall14Bitは、RHALLレジスタペア（LSB/MSB）を符号なし14ビット
// ADCカウント値に変換する（レジスタは2ビット左シフトして格納）。
func DecodeMagRHall14Bit(lsb, msb byte) uint16 {
	raw16 := uint16(lsb) | uint16(msb)<<8
	return raw16 >> 2
}

// MagTrimは、起動時にセンサーのトリム拡張レジスタから一度だけ読み取る
// BMM050形式の工場較正（「トリム」）値を保持する。フィールドの型は
// Boschのリファレンスドライバに正確に合わせてある。
type MagTrim struct {
	DigX1, DigY1, DigX2, DigY2 int8
	DigXY1                     uint8
	DigXY2                     int8
	DigZ1                      uint16
	DigZ2, DigZ3, DigZ4        int16
	DigXYZ1                    uint16
}

// hallRatioは、X軸・Y軸の補正式（Bosch bmm050_compensate_X/Y_float）で
// 共有される、RHALLで正規化した中間値。
func hallRatio(dataR uint16, trim MagTrim) float32 {
	if dataR == 0 {
		return 0
	}
	return float32(trim.DigXYZ1)*16384.0/float32(dataR) - 16384.0
}

// CompensateMagXは、磁力センサーX軸の生の測定値にBosch BMM050補正式を
// 適用し、µT単位の値を返す。
func CompensateMagX(mdataX int16, dataR uint16, trim MagTrim) float32 {
	ratio := hallRatio(dataR, trim)
	crossTerm := float32(trim.DigXY2)*(ratio*ratio/268435456.0) + ratio*float32(trim.DigXY1)/16384.0
	scaled := (crossTerm + 256.0) * (float32(trim.DigX2) + 160.0)
	return ((float32(mdataX)*scaled)/8192.0+float32(trim.DigX1)*8.0)/16.0
}

// CompensateMagYは、磁力センサーY軸の生の測定値にBosch BMM050補正式を
// 適用し、µT単位の値を返す。
func CompensateMagY(mdataY int16, dataR uint16, trim MagTrim) float32 {
	ratio := hallRatio(dataR, trim)
	crossTerm := float32(trim.DigXY2)*(ratio*ratio/268435456.0) + ratio*float32(trim.DigXY1)/16384.0
	scaled := (crossTerm + 256.0) * (float32(trim.DigY2) + 160.0)
	return ((float32(mdataY)*scaled)/8192.0+float32(trim.DigY1)*8.0)/16.0
}

// CompensateMagZは、磁力センサーZ軸の生の測定値にBosch BMM050補正式を
// 適用し、µT単位の値を返す。
func CompensateMagZ(mdataZ int16, dataR uint16, trim MagTrim) float32 {
	numerator := (float32(mdataZ)-float32(trim.DigZ4))*131072.0 -
		float32(trim.DigZ3)*(float32(dataR)-float32(trim.DigXYZ1))
	denominator := (float32(trim.DigZ2) + float32(trim.DigZ1)*float32(dataR)/32768.0) * 4.0
	return (numerator / denominator) / 16.0
}

// EncodeMagは、補正済みの磁力センサー値（µT）をBLEキャラクタリスティックの
// バイトレイアウト（a0b40143）に詰める: float32 LE x, y, z。
func EncodeMag(x, y, z float32) [12]byte {
	var b [12]byte
	binary.LittleEndian.PutUint32(b[0:4], math.Float32bits(x))
	binary.LittleEndian.PutUint32(b[4:8], math.Float32bits(y))
	binary.LittleEndian.PutUint32(b[8:12], math.Float32bits(z))
	return b
}
