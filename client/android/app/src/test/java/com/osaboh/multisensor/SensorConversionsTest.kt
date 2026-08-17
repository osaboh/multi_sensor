package com.osaboh.multisensor

import org.junit.Assert.assertEquals
import org.junit.Test

class SensorConversionsTest {

    @Test
    fun decodeLps22hb_matchesProtocolExample() {
        // docs/ble-protocol-reference.md 3節の例: 1006.00hPa, 22.50℃
        val bytes = byteArrayOf(0x2B, 0xFD.toByte(), 0xCA.toByte(), 0x08)
        val reading = decodeLps22hb(bytes)
        assertEquals(1006.00, reading.pressureHPa, 0.001)
        assertEquals(22.50, reading.temperatureC, 0.001)
    }

    @Test
    fun decodeHdc2010_matchesProtocolExample() {
        // docs/ble-protocol-reference.md 4節の例: 23.48℃, 55.20%RH
        val bytes = byteArrayOf(0x2C, 0x09, 0x90.toByte(), 0x15)
        val reading = decodeHdc2010(bytes)
        assertEquals(23.48, reading.temperatureC, 0.001)
        assertEquals(55.20, reading.humidityPct, 0.001)
    }

    @Test
    fun decodeAccel_matchesProtocolExample() {
        // docs/ble-protocol-reference.md 5.1節の例: raw X=51, Y=-20, Z=1020
        val bytes = byteArrayOf(0x33, 0x00, 0xEC.toByte(), 0xFF.toByte(), 0xFC.toByte(), 0x03)
        val reading = decodeAccel(bytes)
        assertEquals(51 * 0.98, reading.xMg, 0.001)
        assertEquals(-20 * 0.98, reading.yMg, 0.001)
        assertEquals(1020 * 0.98, reading.zMg, 0.001)
    }

    @Test
    fun decodeGyro_matchesProtocolExample() {
        // docs/ble-protocol-reference.md 5.2節の例: X=+10°/s, Y=-5°/s, Z=0°/s
        val bytes = byteArrayOf(0xA4.toByte(), 0x00, 0xAE.toByte(), 0xFF.toByte(), 0x00, 0x00)
        val reading = decodeGyro(bytes)
        assertEquals(10.0, reading.xDps, 0.001)
        assertEquals(-5.0, reading.yDps, 0.001)
        assertEquals(0.0, reading.zDps, 0.001)
    }

    @Test
    fun decodeMag_parsesLittleEndianFloat32() {
        fun floatBytesLE(v: Float): ByteArray {
            val bits = v.toRawBits()
            return byteArrayOf(
                (bits and 0xFF).toByte(),
                ((bits shr 8) and 0xFF).toByte(),
                ((bits shr 16) and 0xFF).toByte(),
                ((bits shr 24) and 0xFF).toByte(),
            )
        }
        val bytes = floatBytesLE(12.5f) + floatBytesLE(-3.25f) + floatBytesLE(0.0f)
        val reading = decodeMag(bytes)
        assertEquals(12.5f, reading.xUt, 0.0f)
        assertEquals(-3.25f, reading.yUt, 0.0f)
        assertEquals(0.0f, reading.zUt, 0.0f)
    }
}
