package main

func regWrite(addr uint8, reg uint8, value uint8) error {
	return i2c.Tx(uint16(addr), []byte{reg, value}, nil)
}

func regRead(addr uint8, reg uint8, buf []byte) error {
	return i2c.Tx(uint16(addr), []byte{reg}, buf)
}
