TARGET      = xiao-ble
SRC         = $(wildcard *.go)
HEX         = advertisement.hex
SOFTDEVICE  = s140_nrf52_7.3.0/s140_nrf52_7.3.0_softdevice.hex
SCAN_VENV   = tools/venv
SCAN_NAME   = MultiSenser

.PHONY: all build test flash flash-softdevice flash-all recover unstick scan clean

all: build

# Run the unit tests for the hardware-independent conversion package.
test:
	go test ./convert/...

# Build the application hex (SoftDevice memory region excluded).
build: $(HEX)

$(HEX): $(SRC) go.mod go.sum
	tinygo build -target=$(TARGET) -o=$(HEX) .

# Erase the whole chip and flash the SoftDevice (s140). Needed once per
# chip, or any time --recover / a full erase has wiped it.
flash-softdevice: recover
	nrfjprog --program $(SOFTDEVICE) --verify --force

# Flash the application hex over J-Link (SoftDevice must already be present).
flash: build
	nrfjprog --program $(HEX) --sectorerase --verify --reset --force

# Full fresh setup: erase chip, flash SoftDevice, then flash the app.
flash-all: flash-softdevice flash

recover:
	nrfjprog --recover --force

# If nrfjprog/J-Link hangs or times out (-102), a leftover
# jlinkarm_nrf_worker_osx process is usually holding the debugger open.
# Kill it and retry.
unstick:
	-pkill -9 jlinkarm_nrf_worker_osx

$(SCAN_VENV)/bin/python3: tools/requirements.txt
	python3 -m venv $(SCAN_VENV)
	$(SCAN_VENV)/bin/pip install -q -r tools/requirements.txt

# Scan for the board's BLE advertisement to confirm it's running.
scan: $(SCAN_VENV)/bin/python3
	$(SCAN_VENV)/bin/python3 tools/scan_advertisement.py $(SCAN_NAME)

clean:
	rm -f $(HEX)
