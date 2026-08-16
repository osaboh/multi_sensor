package convert

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestDecodeMagXY13Bit(t *testing.T) {
	got := DecodeMagXY13Bit(0x20, 0x03)
	want := int16(100)
	if got != want {
		t.Errorf("DecodeMagXY13Bit() = %v, want %v", got, want)
	}
}

func TestDecodeMagZ15Bit(t *testing.T) {
	got := DecodeMagZ15Bit(0x90, 0x01)
	want := int16(200)
	if got != want {
		t.Errorf("DecodeMagZ15Bit() = %v, want %v", got, want)
	}
}

func TestDecodeMagRHall14Bit(t *testing.T) {
	got := DecodeMagRHall14Bit(0xD0, 0x07)
	want := uint16(500)
	if got != want {
		t.Errorf("DecodeMagRHall14Bit() = %v, want %v", got, want)
	}
}

// Trim values chosen to zero out the cross-terms of the Bosch compensation
// formula so the expected result can be verified by hand:
//   X = mdata_x * 5 / 16, Y = mdata_y * 5 / 16, Z = mdata_z - dig_z4
var testTrim = MagTrim{
	DigX1: 0, DigY1: 0, DigX2: 0, DigY2: 0,
	DigXY1: 0, DigXY2: 0,
	DigZ1: 0, DigZ2: 2048, DigZ3: 0, DigZ4: 100,
	DigXYZ1: 6400,
}

func TestCompensateMagX(t *testing.T) {
	got := CompensateMagX(16, 32768, testTrim)
	want := float32(5.0)
	if got != want {
		t.Errorf("CompensateMagX() = %v, want %v", got, want)
	}
}

func TestCompensateMagY(t *testing.T) {
	got := CompensateMagY(32, 32768, testTrim)
	want := float32(10.0)
	if got != want {
		t.Errorf("CompensateMagY() = %v, want %v", got, want)
	}
}

func TestCompensateMagZ(t *testing.T) {
	got := CompensateMagZ(150, 32768, testTrim)
	want := float32(50.0)
	if got != want {
		t.Errorf("CompensateMagZ() = %v, want %v", got, want)
	}
}

func TestEncodeMag(t *testing.T) {
	x, y, z := float32(5.0), float32(10.0), float32(50.0)
	got := EncodeMag(x, y, z)

	var want [12]byte
	binary.LittleEndian.PutUint32(want[0:4], math.Float32bits(x))
	binary.LittleEndian.PutUint32(want[4:8], math.Float32bits(y))
	binary.LittleEndian.PutUint32(want[8:12], math.Float32bits(z))

	if got != want {
		t.Errorf("EncodeMag() = % X, want % X", got, want)
	}
}
