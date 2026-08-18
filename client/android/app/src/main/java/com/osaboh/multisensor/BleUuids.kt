package com.osaboh.multisensor

import java.util.UUID

// docs/ble-protocol-reference.md の UUID一覧に準拠。
object BleUuids {
    val CCCD: UUID = UUID.fromString("00002902-0000-1000-8000-00805f9b34fb")

    val IO_SERVICE: UUID = UUID.fromString("a0b40100-926d-4d61-98df-8c5c62ee53b3")
    val LED1: UUID = UUID.fromString("a0b40101-926d-4d61-98df-8c5c62ee53b3")
    val LED2: UUID = UUID.fromString("a0b40102-926d-4d61-98df-8c5c62ee53b3")
    val BUZZER: UUID = UUID.fromString("a0b40103-926d-4d61-98df-8c5c62ee53b3")
    val SW_TOP: UUID = UUID.fromString("a0b40111-926d-4d61-98df-8c5c62ee53b3")
    val SW_SIDE: UUID = UUID.fromString("a0b40112-926d-4d61-98df-8c5c62ee53b3")

    val LPS22HB_SERVICE: UUID = UUID.fromString("a0b40120-926d-4d61-98df-8c5c62ee53b3")
    val LPS22HB_CHAR: UUID = UUID.fromString("a0b40121-926d-4d61-98df-8c5c62ee53b3")

    val HDC2010_SERVICE: UUID = UUID.fromString("a0b40130-926d-4d61-98df-8c5c62ee53b3")
    val HDC2010_CHAR: UUID = UUID.fromString("a0b40131-926d-4d61-98df-8c5c62ee53b3")

    val BMX055_SERVICE: UUID = UUID.fromString("a0b40140-926d-4d61-98df-8c5c62ee53b3")
    val ACCEL_CHAR: UUID = UUID.fromString("a0b40141-926d-4d61-98df-8c5c62ee53b3")
    val GYRO_CHAR: UUID = UUID.fromString("a0b40142-926d-4d61-98df-8c5c62ee53b3")
    val MAG_CHAR: UUID = UUID.fromString("a0b40143-926d-4d61-98df-8c5c62ee53b3")
    val BMX055_INTERVAL: UUID = UUID.fromString("a0b40144-926d-4d61-98df-8c5c62ee53b3")

    const val DEVICE_NAME_FILTER = "MultiSenser"
}
