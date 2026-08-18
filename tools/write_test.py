#!/usr/bin/env python3
"""I/O Service (LED/Buzzer) の書き込み確認スクリプト。

ボードに接続し、LED1/LED2の点灯とBuzzerの鳴動をBLE経由で操作する。

使い方:
  python3 write_test.py [デバイス名]

例:
  python3 write_test.py "Go MultiSenser"
"""

import asyncio
import struct
import sys

from bleak import BleakClient, BleakScanner

LED1 = "a0b40101-926d-4d61-98df-8c5c62ee53b3"
LED2 = "a0b40102-926d-4d61-98df-8c5c62ee53b3"
BUZZER = "a0b40103-926d-4d61-98df-8c5c62ee53b3"


async def main(device_name: str) -> None:
    print(f"scanning for {device_name!r}...")
    dev = await BleakScanner.find_device_by_name(device_name, timeout=10)
    if dev is None:
        print("device not found")
        sys.exit(1)

    print(f"connecting to {dev.address}")
    async with BleakClient(dev) as client:
        print("connected")

        print("LED1 ON")
        await client.write_gatt_char(LED1, bytes([0x01]), response=True)
        await asyncio.sleep(1)
        print("LED1 OFF")
        await client.write_gatt_char(LED1, bytes([0x00]), response=True)

        print("LED2 ON")
        await client.write_gatt_char(LED2, bytes([0x01]), response=True)
        await asyncio.sleep(1)
        print("LED2 OFF")
        await client.write_gatt_char(LED2, bytes([0x00]), response=True)

        print("Buzzer 800ms")
        await client.write_gatt_char(BUZZER, struct.pack("<H", 800), response=True)
        await asyncio.sleep(1)

        print("done, disconnecting")


if __name__ == "__main__":
    device_name = sys.argv[1] if len(sys.argv) > 1 else "Go MultiSenser"
    asyncio.run(main(device_name))
