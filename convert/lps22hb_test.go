package convert

import "testing"

func TestDecodeLPS22HBPressure(t *testing.T) {
	got := DecodeLPS22HBPressure(0x00, 0xE0, 0x3E)
	want := 1006.00
	if got != want {
		t.Errorf("DecodeLPS22HBPressure() = %v, want %v", got, want)
	}
}

func TestDecodeLPS22HBTemperature(t *testing.T) {
	got := DecodeLPS22HBTemperature(0xCA, 0x08)
	want := 22.50
	if got != want {
		t.Errorf("DecodeLPS22HBTemperature() = %v, want %v", got, want)
	}
}

func TestEncodeLPS22HB(t *testing.T) {
	got := EncodeLPS22HB(1006.00, 22.50)
	want := [4]byte{0x2B, 0xFD, 0xCA, 0x08}
	if got != want {
		t.Errorf("EncodeLPS22HB() = % X, want % X", got, want)
	}
}
