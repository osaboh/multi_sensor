#!/usr/bin/env python3
"""GATT接続確認スクリプト。

ボードに接続し、docs/ble-protocol-reference.md で定義された全Notify対応
characteristicを購読して、実際にNotifyが届くか確認する。

使い方:
  python3 gatt_verify.py [デバイス名] [購読時間(秒)]

例:
  python3 gatt_verify.py "Go MultiSenser" 20
"""

import asyncio
import sys

from bleak import BleakClient, BleakScanner

CHARS = {
    "LPS22HB (pressure, temp)": "a0b40121-926d-4d61-98df-8c5c62ee53b3",
    "HDC2010 (temp, humidity)": "a0b40131-926d-4d61-98df-8c5c62ee53b3",
    "BMX055 Accel": "a0b40141-926d-4d61-98df-8c5c62ee53b3",
    "BMX055 Gyro": "a0b40142-926d-4d61-98df-8c5c62ee53b3",
    "BMX055 Mag": "a0b40143-926d-4d61-98df-8c5c62ee53b3",
    "SW_TOP": "a0b40111-926d-4d61-98df-8c5c62ee53b3",
    "SW_SIDE": "a0b40112-926d-4d61-98df-8c5c62ee53b3",
}


async def main(device_name: str, duration: float) -> None:
    print(f"scanning for {device_name!r}...")
    dev = await BleakScanner.find_device_by_name(device_name, timeout=10)
    if dev is None:
        print("device not found")
        sys.exit(1)

    print(f"connecting to {dev.address}")
    async with BleakClient(dev) as client:
        print("connected")
        counts: dict[str, int] = {}

        def make_callback(name: str):
            def callback(_, data: bytearray) -> None:
                counts[name] = counts.get(name, 0) + 1
                print(f"NOTIFY {name}: {data.hex(' ')}")
            return callback

        for name, uuid in CHARS.items():
            try:
                await client.start_notify(uuid, make_callback(name))
                print(f"subscribed {name}")
            except Exception as e:
                print(f"subscribe {name} FAILED: {e}")

        print(f"listening for {duration:.0f}s...")
        await asyncio.sleep(duration)
        print(f"total notifications received: {counts}")


if __name__ == "__main__":
    device_name = sys.argv[1] if len(sys.argv) > 1 else "Go MultiSenser"
    duration = float(sys.argv[2]) if len(sys.argv) > 2 else 20.0
    asyncio.run(main(device_name, duration))
