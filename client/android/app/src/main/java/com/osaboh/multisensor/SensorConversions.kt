package com.osaboh.multisensor

data class Lps22hbReading(val pressureHPa: Double, val temperatureC: Double)
data class Hdc2010Reading(val temperatureC: Double, val humidityPct: Double)
data class AccelReading(val xMg: Double, val yMg: Double, val zMg: Double)
data class GyroReading(val xDps: Double, val yDps: Double, val zDps: Double)
data class MagReading(val xUt: Float, val yUt: Float, val zUt: Float)

private fun ByteArray.int16LE(offset: Int): Int {
    val raw = (this[offset].toInt() and 0xFF) or ((this[offset + 1].toInt() and 0xFF) shl 8)
    return raw.toShort().toInt()
}

private fun ByteArray.float32LE(offset: Int): Float {
    val bits = (this[offset].toInt() and 0xFF) or
        ((this[offset + 1].toInt() and 0xFF) shl 8) or
        ((this[offset + 2].toInt() and 0xFF) shl 16) or
        ((this[offset + 3].toInt() and 0xFF) shl 24)
    return Float.fromBits(bits)
}

// LPS22HBキャラクタリスティック（a0b40121）: int16 LE
// 1013.25hPaからの偏差×100、int16 LE 温度×100。
// docs/ble-protocol-reference.md 3節参照。
fun decodeLps22hb(bytes: ByteArray): Lps22hbReading {
    val pressureDevRaw = bytes.int16LE(0)
    val tempRaw = bytes.int16LE(2)
    return Lps22hbReading(
        pressureHPa = pressureDevRaw / 100.0 + 1013.25,
        temperatureC = tempRaw / 100.0,
    )
}

// HDC2010キャラクタリスティック（a0b40131）: int16 LE 温度×100、
// int16 LE 湿度×100。docs/ble-protocol-reference.md 4節参照。
fun decodeHdc2010(bytes: ByteArray): Hdc2010Reading {
    val tempRaw = bytes.int16LE(0)
    val humidityRaw = bytes.int16LE(2)
    return Hdc2010Reading(
        temperatureC = tempRaw / 100.0,
        humidityPct = humidityRaw / 100.0,
    )
}

// BMX055加速度センサー（a0b40141）: int16 LE 生カウント値x,y,z、
// ±2gレンジ、0.98 mg/LSB。docs/ble-protocol-reference.md 5.1節参照。
fun decodeAccel(bytes: ByteArray): AccelReading {
    val x = bytes.int16LE(0)
    val y = bytes.int16LE(2)
    val z = bytes.int16LE(4)
    return AccelReading(x * 0.98, y * 0.98, z * 0.98)
}

// BMX055ジャイロ（a0b40142）: int16 LE 生カウント値x,y,z、
// ±2000°/sレンジ、16.4 LSB/(°/s)。docs/ble-protocol-reference.md 5.2節参照。
fun decodeGyro(bytes: ByteArray): GyroReading {
    val x = bytes.int16LE(0)
    val y = bytes.int16LE(2)
    val z = bytes.int16LE(4)
    return GyroReading(x / 16.4, y / 16.4, z / 16.4)
}

// BMX055磁力センサー（a0b40143）: float32 LE x,y,z（µT単位）、
// ファームウェア側で較正済み。docs/ble-protocol-reference.md 5.3節参照。
fun decodeMag(bytes: ByteArray): MagReading {
    return MagReading(bytes.float32LE(0), bytes.float32LE(4), bytes.float32LE(8))
}
