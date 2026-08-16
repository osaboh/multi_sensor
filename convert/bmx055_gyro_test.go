package convert

import "testing"

func TestDecodeGyro16Bit(t *testing.T) {
	cases := []struct {
		name     string
		lsb, msb byte
		want     int16
	}{
		{"X=164", 0xA4, 0x00, 164},
		{"Y=-82", 0xAE, 0xFF, -82},
		{"Z=0", 0x00, 0x00, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := DecodeGyro16Bit(c.lsb, c.msb)
			if got != c.want {
				t.Errorf("DecodeGyro16Bit(0x%02X, 0x%02X) = %v, want %v", c.lsb, c.msb, got, c.want)
			}
		})
	}
}

func TestEncodeGyro(t *testing.T) {
	got := EncodeGyro(164, -82, 0)
	want := [6]byte{0xA4, 0x00, 0xAE, 0xFF, 0x00, 0x00}
	if got != want {
		t.Errorf("EncodeGyro() = % X, want % X", got, want)
	}
}
