#!/usr/bin/env python3
"""BLE アドバタイズ確認用スクリプト。

指定した名前を含むデバイスのアドバタイズを探す。見つからない場合は、
その時間内にスキャンできた全デバイス(name/manufacturer_data/service_data)
を表示するので、目的のデバイスが別のシグネチャで出ていないか確認できる。

使い方:
  python3 scan_advertisement.py [名前でのフィルタ文字列] [スキャン秒数]

例:
  python3 scan_advertisement.py "MultiSenser" 8
"""

import asyncio
import sys

from bleak import BleakScanner


async def main(name_filter: str, duration: float) -> None:
    seen: dict[str, tuple[str, dict, dict]] = {}
    found = False

    async def callback(bd, ad):
        nonlocal found
        name = ad.local_name or bd.name or ""
        seen[bd.address] = (name, dict(ad.manufacturer_data), dict(ad.service_data))
        if name_filter and name_filter in name:
            manu_hex = {hex(k): v.hex() for k, v in ad.manufacturer_data.items()}
            print(f"FOUND: {bd.address} name={name!r} manufacturer_data={manu_hex}")
            found = True

    scanner = BleakScanner(callback)
    await scanner.start()
    await asyncio.sleep(duration)
    await scanner.stop()

    if not found:
        print(f"'{name_filter}' を含むデバイスは見つかりませんでした。")
        print(f"\nスキャン中に見つかった全デバイス ({len(seen)}件):")
        for addr, (name, manu, svc) in seen.items():
            manu_hex = {hex(k): v.hex() for k, v in manu.items()}
            print(f"  {addr}  name={name!r} manufacturer_data={manu_hex} service_data={svc}")
        sys.exit(1)


if __name__ == "__main__":
    name_filter = sys.argv[1] if len(sys.argv) > 1 else "MultiSenser"
    duration = float(sys.argv[2]) if len(sys.argv) > 2 else 8.0
    asyncio.run(main(name_filter, duration))
