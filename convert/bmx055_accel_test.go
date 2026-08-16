package convert

import "testing"

func TestDecodeAccel12Bit(t *testing.T) {
	cases := []struct {
		name     string
		lsb, msb byte
		want     int16
	}{
		{"X=51", 0x30, 0x03, 51},
		{"Y=-20", 0xC0, 0xFE, -20},
		{"Z=1020", 0xC0, 0x3F, 1020},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := DecodeAccel12Bit(c.lsb, c.msb)
			if got != c.want {
				t.Errorf("DecodeAccel12Bit(0x%02X, 0x%02X) = %v, want %v", c.lsb, c.msb, got, c.want)
			}
		})
	}
}

func TestEncodeAccel(t *testing.T) {
	got := EncodeAccel(51, -20, 1020)
	want := [6]byte{0x33, 0x00, 0xEC, 0xFF, 0xFC, 0x03}
	if got != want {
		t.Errorf("EncodeAccel() = % X, want % X", got, want)
	}
}
