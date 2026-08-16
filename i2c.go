package main

import "sync"

// i2cMu serializes access to the shared I2C bus: LPS22HB, HDC2010, and
// BMX055 are each polled from their own goroutine, and machine.I2C.Tx has no
// built-in mutual exclusion.
var i2cMu sync.Mutex

func regWrite(addr uint8, reg uint8, value uint8) error {
	i2cMu.Lock()
	defer i2cMu.Unlock()
	return i2c.Tx(uint16(addr), []byte{reg, value}, nil)
}

func regRead(addr uint8, reg uint8, buf []byte) error {
	i2cMu.Lock()
	defer i2cMu.Unlock()
	return i2c.Tx(uint16(addr), []byte{reg}, buf)
}
