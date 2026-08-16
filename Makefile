TARGET      = xiao-ble
SRC         = $(wildcard *.go)
HEX         = advertisement.hex
SOFTDEVICE  = s140_nrf52_7.3.0/s140_nrf52_7.3.0_softdevice.hex
SCAN_VENV   = tools/venv
SCAN_NAME   = MultiSenser

.PHONY: all help build test flash flash-softdevice flash-all recover unstick scan verify write-test clean

all: build

# List all available targets with their one-line descriptions (parsed from
# the "## ..." comment on each target line below).
help: ## ターゲット一覧を表示
	@awk 'BEGIN {FS = ":.*## "}; /^[a-zA-Z0-9_.-]+:.*## / {printf "  %-16s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Run the unit tests for the hardware-independent conversion package.
test: ## convert/パッケージの単体テストを実行
	go test ./convert/...

# Build the application hex (SoftDevice memory region excluded).
build: $(HEX) ## アプリケーションのhexをビルド

$(HEX): $(SRC) go.mod go.sum
	tinygo build -target=$(TARGET) -o=$(HEX) .

# Erase the whole chip and flash the SoftDevice (s140). Needed once per
# chip, or any time --recover / a full erase has wiped it.
flash-softdevice: recover ## チップを全消去してSoftDeviceを書き込み
	nrfjprog --program $(SOFTDEVICE) --verify --force

# Flash the application hex over J-Link (SoftDevice must already be present).
flash: build ## ビルドしてJ-Link経由でアプリを書き込み
	nrfjprog --program $(HEX) --sectorerase --verify --reset --force

# Full fresh setup: erase chip, flash SoftDevice, then flash the app.
flash-all: flash-softdevice flash ## フルセットアップ（SoftDevice+アプリ）

recover: ## チップを全消去（フルリカバー）
	nrfjprog --recover --force

# If nrfjprog/J-Link hangs or times out (-102), a leftover
# jlinkarm_nrf_worker_osx process is usually holding the debugger open.
# Kill it and retry.
unstick: ## 詰まったjlinkarm_nrf_worker_osxプロセスを強制終了
	-pkill -9 jlinkarm_nrf_worker_osx

$(SCAN_VENV)/bin/python3: tools/requirements.txt
	python3 -m venv $(SCAN_VENV)
	$(SCAN_VENV)/bin/pip install -q -r tools/requirements.txt

# Scan for the board's BLE advertisement to confirm it's running.
scan: $(SCAN_VENV)/bin/python3 ## ボードのBLEアドバタイズをスキャン
	$(SCAN_VENV)/bin/python3 tools/scan_advertisement.py $(SCAN_NAME)

# Connect and subscribe to every sensor characteristic's Notify.
verify: $(SCAN_VENV)/bin/python3 ## 接続して全センサーのNotifyを確認
	$(SCAN_VENV)/bin/python3 tools/gatt_verify.py

# Connect and exercise LED1/LED2/Buzzer writes.
write-test: $(SCAN_VENV)/bin/python3 ## 接続してLED1/LED2/Buzzerの書き込みを試す
	$(SCAN_VENV)/bin/python3 tools/write_test.py

clean: ## ビルド成果物を削除
	rm -f $(HEX)
