package convert

import "testing"

func TestDecodeHDC2010Temperature(t *testing.T) {
	got := DecodeHDC2010Temperature(0x7C, 0x62)
	want := 23.48
	if diff := got - want; diff > 0.01 || diff < -0.01 {
		t.Errorf("DecodeHDC2010Temperature() = %v, want ~%v", got, want)
	}
}

func TestDecodeHDC2010Humidity(t *testing.T) {
	got := DecodeHDC2010Humidity(0x50, 0x8D)
	want := 55.20
	if diff := got - want; diff > 0.01 || diff < -0.01 {
		t.Errorf("DecodeHDC2010Humidity() = %v, want ~%v", got, want)
	}
}

func TestEncodeHDC2010(t *testing.T) {
	got := EncodeHDC2010(23.48, 55.20)
	want := [4]byte{0x2C, 0x09, 0x90, 0x15}
	if got != want {
		t.Errorf("EncodeHDC2010() = % X, want % X", got, want)
	}
}

func TestEncodeHDC2010RoundsToNearest(t *testing.T) {
	// 12.346 * 100 = 1234.6, must round to 1235, not truncate to 1234.
	got := EncodeHDC2010(12.346, 0)
	want := int16(1235)
	gotTemp := int16(uint16(got[0]) | uint16(got[1])<<8)
	if gotTemp != want {
		t.Errorf("EncodeHDC2010(12.346, 0) temp = %v, want %v", gotTemp, want)
	}
}
