package main

import "sync"

// i2cMuは共有I2Cバスへのアクセスを直列化する: LPS22HB、HDC2010、BMX055は
// それぞれ独自のゴルーチンからポーリングされており、machine.I2C.Txには
// 排他制御が組み込まれていないため。
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
