package main

// DEBUG: scanI2C probes every 7-bit I2C address and logs which ones ACK.
// Temporary, to be removed once BMX055 addressing is confirmed.
func scanI2C() {
	logLine("i2c scan start")
	for addr := uint8(0x03); addr < 0x78; addr++ {
		if err := i2c.Tx(uint16(addr), []byte{0x00}, nil); err == nil {
			logLine("i2c found: 0x" + hex8(addr))
		}
	}
	logLine("i2c scan done")
}

func hex8(v uint8) string {
	const digits = "0123456789ABCDEF"
	return string([]byte{digits[v>>4], digits[v&0x0F]})
}
