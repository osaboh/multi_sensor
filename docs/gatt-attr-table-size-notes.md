# 調査メモ: GATT属性テーブルサイズの拡張案 (2026-08-19)

## 背景

ブザー音程チューニング用に一時的なCharacteristicを追加した際、`panic: failed to
add BMX055 service: no memory for operation`で起動時にパニックする不具合が発生した
（詳細は`docs/debug-log-bmx055-wake.md`ではなく、ble_io.go内のコメント
および今回のセッションの対応を参照。要点はSoftDeviceのGATT属性テーブル容量を
使い切ったこと）。

この際に恒久対応として「新規Characteristicを追加せず、既存Characteristicの
ペイロードを拡張する」方針を採ったが、より根本的な対応として「そもそも属性
テーブルのサイズ自体を拡張できないか」を調査した。本メモはその調査結果と、
実現するための具体的な変更案をまとめたもの。**未実装の案**であり、実装する
かどうかは別途判断する。

## 現状の制約

### SoftDevice側の既定値

`tinygo.org/x/bluetooth@v0.13.0/s140_nrf52_7.3.0/s140_nrf52_7.3.0_API/include/ble_gatts.h`:

```c
#define BLE_GATTS_ATTR_TAB_SIZE_MIN         (248)  /* 最小サイズ */
#define BLE_GATTS_ATTR_TAB_SIZE_DEFAULT     (1408) /* 既定サイズ（現状これが使われている） */
```

### TinyGoのbluetoothライブラリがこの値を明示的に設定していない

nRF52系（s132/s140、このプロジェクトが使っているのはs140v7）の初期化コードは
`tinygo.org/x/bluetooth@v0.13.0/adapter_nrf528xx.go`の`enable()`:

```go
func (a *Adapter) enable() error {
	errCode := C.sd_softdevice_enable(clockConfig, C.nrf_fault_handler_t(C.assertHandler))
	if errCode != 0 {
		return Error(errCode)
	}
	appRAMBase := C.uint32_t(uintptr(unsafe.Pointer(&appRAMBase)))
	errCode = C.sd_ble_enable(&appRAMBase)
	return makeError(errCode)
}
```

`sd_ble_cfg_set()`を一度も呼ばずに`sd_ble_enable()`を呼んでいる。SoftDevice
のヘッダのドキュメントコメント（`ble.h`）には

> Any part of the BLE stack that is NOT configured with `sd_ble_cfg_set` will
> have default configuration.

とあり、`sd_ble_cfg_set(BLE_GATTS_CFG_ATTR_TAB_SIZE, ...)`を呼ばない限り
`BLE_GATTS_ATTR_TAB_SIZE_DEFAULT`(1408バイト)が使われる。つまり現状これは
**SoftDevice自体の絶対的なハード制限ではなく、TinyGoのバインディングが
拡張用APIを呼んでいないために生じている天井**である。

参考: 古い世代のSoftDevice（S110、nRF51向け）を使う
`tinygo.org/x/bluetooth@v0.13.0/adapter_nrf51.go`では、当時のAPI仕様上
`sd_ble_enable()`に渡す`ble_enable_params_t`構造体の中に直接`attr_tab_size`
フィールドがあり、そこに明示的に`BLE_GATTS_ATTR_TAB_SIZE_DEFAULT`を設定して
いる。つまりTinyGo側もこの値の存在自体は認識しており、単にnRF52系の新しい
`sd_ble_cfg_set`方式に対応するコードが書かれていないだけと考えられる。

## 拡張案: `sd_ble_cfg_set(BLE_GATTS_CFG_ATTR_TAB_SIZE, ...)`を追加する

### API仕様（`ble.h` / `ble_gatts.h`より）

```c
uint32_t sd_ble_cfg_set(uint32_t cfg_id, ble_cfg_t const * p_cfg, uint32_t app_ram_base);

typedef struct {
  uint32_t attr_tab_size; /* 4の倍数。最小 BLE_GATTS_ATTR_TAB_SIZE_MIN(248) */
} ble_gatts_cfg_attr_tab_size_t;
```

`sd_ble_enable()`より**前**に呼ぶ必要がある（`sd_ble_cfg_set`はSoftDevice
有効化後・BLE部有効化前のみ呼び出し可能、とヘッダに明記）。

イメージ（`adapter_nrf528xx.go`の`enable()`内、`sd_ble_enable`呼び出しの直前）:

```go
var cfg C.ble_cfg_t
// union の gatts_cfg.attr_tab_size フィールドに書き込む
gattsCfg := (*C.ble_gatts_cfg_attr_tab_size_t)(unsafe.Pointer(&cfg))
gattsCfg.attr_tab_size = 1408 * 2 // 例: 既定の2倍(2816バイト、4の倍数)
appRAMBase := C.uint32_t(uintptr(unsafe.Pointer(&appRAMBase)))
errCode := C.sd_ble_cfg_set(C.BLE_GATTS_CFG_ATTR_TAB_SIZE, &cfg, appRAMBase)
if errCode != 0 {
	return Error(errCode)
}
errCode = C.sd_ble_enable(&appRAMBase)
```

（`ble_cfg_t`はunion型なので、Cgoでは対応するGoの構造体にキャストするか
`unsafe.Pointer`経由でフィールドを書き込む形になる。正確な実装は
tinygo-org/bluetoothのCgo型定義を要確認。）

### 見落としてはいけない副作用: SoftDeviceが使うRAM領域も増える

`ble.h`のドキュメントコメント:

> At runtime the IC's RAM is split into 2 regions: The SoftDevice RAM region
> is located between `0x20000000` and `APP_RAM_BASE-1` and the application's
> RAM region is located between `APP_RAM_BASE` and the start of the call
> stack.

属性テーブルを大きくするとSoftDeviceが使うRAM領域（`APP_RAM_BASE`未満の
部分）が増える。`APP_RAM_BASE`より上がアプリ（このファームウェア）用の
RAMなので、**`APP_RAM_BASE`を今より大きい値に引き上げてSoftDevice用の
RAMを確保しないと、`sd_ble_enable`が`NRF_ERROR_NO_MEM`で失敗する**。

このプロジェクトのビルドで使われているリンカスクリプト
（`/opt/homebrew/Cellar/tinygo/0.41.1/targets/nrf52840-s140v7.ld`、
`tinygo env TINYGOROOT`配下、xiao-bleターゲットが
`nrf52840-s140v7-uf2`を継承）:

```ld
MEMORY
{
    FLASH_TEXT (rw) : ORIGIN = 0x00000000 + 0x00027000, LENGTH = 1M - 0x00027000
    RAM (xrw)       : ORIGIN = 0x20000000 + 0x000039c0, LENGTH = 256K - 0x000039c0
}
__app_ram_base = ORIGIN(RAM);
```

`RAM`の開始オフセット`0x000039c0`（=14784バイト）が、既定の属性テーブル
サイズ(1408バイト)を含むSoftDeviceの既定RAM使用量に対応する固定値。属性
テーブルを拡張するなら、この`0x000039c0`も増分ぶん（少なくとも
`新attr_tab_size - 1408`バイト以上）引き上げる必要がある。

## 実現するために必要な変更（まとめ）

1. **`tinygo.org/x/bluetooth`のフォーク/パッチ**: `adapter_nrf528xx.go`の
   `enable()`に`sd_ble_cfg_set(BLE_GATTS_CFG_ATTR_TAB_SIZE, ...)`呼び出しを
   追加。`go.mod`に`replace tinygo.org/x/bluetooth => <ローカルフォークのパス>`
   を追加してビルドに反映させる（upstreamへのPRも選択肢だが、時間がかかる）。
2. **カスタムリンカスクリプト**: `nrf52840-s140v7.ld`の`RAM`開始オフセットを
   増やしたコピーを作り、プロジェクト内に配置。TinyGoのターゲットJSONは
   `"linkerscript"`キーでリンカスクリプトのパスを指定できる仕組みがある
   （`nrf52840.json`で確認済み）ので、`xiao-ble`を継承しつつ
   `linkerscript`だけ上書きしたカスタムターゲットJSON
   （例: `targets/xiao-ble-bigattr.json`）を用意し、
   `tinygo build -target=<カスタムtarget>`でビルドする形になる。
3. **正確なオフセット値の実験**: 増やすべきRAMオフセットの正確な値は
   ドキュメント上の計算式が明示されていないため、実機での試行錯誤
   （小さすぎれば`sd_ble_enable`が`NRF_ERROR_NO_MEM`で失敗するので
   RTTログで検知可能）で確定させる必要がある。

## 評価: 今回のプロジェクトでやるべきか

- **メリット**: Characteristic数の将来的な拡張余地ができる（ブザー音程
  チューニングのような一時的な検証用途でも別Characteristicを気軽に足せる）。
- **コスト**: 依存ライブラリのフォーク維持、TinyGoターゲットのカスタム
  リンカスクリプト管理、正確なRAMオフセットの実験、という3つの追加の
  保守負担が常時発生する。現在のCharacteristic数（IO Service 5個、
  LPS22HB 1個、HDC2010 1個、BMX055 4個、NUS 2個）は既定の1408バイトの
  範囲に収まっており、直近で新しいセンサー/機能を追加する具体的な予定も
  今のところない。
- **結論（現時点)**: 費用対効果が見合わないため**保留**。既存
  Characteristicのペイロード拡張で足りる範囲は今後もその方針で対応する。
  Characteristicの新設がどうしても必要になった時点で、本メモを起点に
  再検討する。

## 関連ファイル

- `ble_io.go` — 現在のBuzzer Characteristic実装（このコミットで確定値に
  簡素化済み、GATT容量を圧迫させないための教訓コメントあり）
- `docs/ble-protocol-reference.md` — 現行のGATTプロファイル仕様
- `go.mod` — `tinygo.org/x/bluetooth v0.13.0`への依存
