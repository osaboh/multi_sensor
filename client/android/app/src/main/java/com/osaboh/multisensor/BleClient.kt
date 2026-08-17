package com.osaboh.multisensor

import android.annotation.SuppressLint
import android.bluetooth.BluetoothDevice
import android.bluetooth.BluetoothGatt
import android.bluetooth.BluetoothGattCallback
import android.bluetooth.BluetoothGattCharacteristic
import android.bluetooth.BluetoothGattDescriptor
import android.bluetooth.BluetoothManager
import android.bluetooth.BluetoothProfile
import android.bluetooth.le.ScanCallback
import android.bluetooth.le.ScanResult
import android.content.Context
import android.util.Log
import java.util.UUID
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update

sealed interface ConnectionState {
    data object Idle : ConnectionState
    data object Scanning : ConnectionState
    data object Connecting : ConnectionState
    data object DiscoveringServices : ConnectionState
    data object Connected : ConnectionState
    data class Disconnected(val reason: String) : ConnectionState
}

data class SensorReadings(
    val lps22hb: Lps22hbReading? = null,
    val hdc2010: Hdc2010Reading? = null,
    val accel: AccelReading? = null,
    val gyro: GyroReading? = null,
    val mag: MagReading? = null,
)

private val NOTIFY_CHARACTERISTICS: List<Pair<UUID, UUID>> = listOf(
    BleUuids.LPS22HB_SERVICE to BleUuids.LPS22HB_CHAR,
    BleUuids.HDC2010_SERVICE to BleUuids.HDC2010_CHAR,
    BleUuids.BMX055_SERVICE to BleUuids.ACCEL_CHAR,
    BleUuids.BMX055_SERVICE to BleUuids.GYRO_CHAR,
    BleUuids.BMX055_SERVICE to BleUuids.MAG_CHAR,
)

// BLEスキャン・GATT接続・Notify購読・Write を担当する。UIはconnectionState/
// readingsのStateFlowを購読するだけでよく、BluetoothGattCallbackの詳細を
// 知る必要はない。呼び出し側はBLUETOOTH_SCAN/BLUETOOTH_CONNECT権限を事前に
// 取得しておくこと（権限確認自体はこのクラスの責務ではない）。
@SuppressLint("MissingPermission")
class BleClient(private val context: Context) {

    private val _connectionState = MutableStateFlow<ConnectionState>(ConnectionState.Idle)
    val connectionState: StateFlow<ConnectionState> = _connectionState.asStateFlow()

    private val _readings = MutableStateFlow(SensorReadings())
    val readings: StateFlow<SensorReadings> = _readings.asStateFlow()

    private val bluetoothManager: BluetoothManager? =
        context.getSystemService(Context.BLUETOOTH_SERVICE) as? BluetoothManager

    private var gatt: BluetoothGatt? = null
    private var ledChar1: BluetoothGattCharacteristic? = null
    private var ledChar2: BluetoothGattCharacteristic? = null
    private var buzzerChar: BluetoothGattCharacteristic? = null
    private var notifyIndex = 0

    fun startScan() {
        val adapter = bluetoothManager?.adapter
        val scanner = adapter?.bluetoothLeScanner
        if (adapter == null || !adapter.isEnabled || scanner == null) {
            _connectionState.value = ConnectionState.Disconnected("Bluetoothが無効です")
            return
        }
        _connectionState.value = ConnectionState.Scanning
        scanner.startScan(scanCallback)
    }

    fun setLed1(on: Boolean) = writeByte(ledChar1, if (on) 0x01 else 0x00)
    fun setLed2(on: Boolean) = writeByte(ledChar2, if (on) 0x01 else 0x00)

    fun triggerBuzzer(durationMs: Int) {
        val characteristic = buzzerChar ?: return
        val value = byteArrayOf((durationMs and 0xFF).toByte(), ((durationMs shr 8) and 0xFF).toByte())
        gatt?.writeCharacteristic(characteristic, value, BluetoothGattCharacteristic.WRITE_TYPE_DEFAULT)
    }

    fun close() {
        bluetoothManager?.adapter?.bluetoothLeScanner?.stopScan(scanCallback)
        gatt?.close()
        gatt = null
    }

    private fun writeByte(characteristic: BluetoothGattCharacteristic?, value: Int) {
        val char = characteristic ?: return
        gatt?.writeCharacteristic(char, byteArrayOf(value.toByte()), BluetoothGattCharacteristic.WRITE_TYPE_DEFAULT)
    }

    private val scanCallback = object : ScanCallback() {
        override fun onScanResult(callbackType: Int, result: ScanResult) {
            if (_connectionState.value != ConnectionState.Scanning) return
            val name = result.scanRecord?.deviceName ?: result.device.name ?: return
            if (!name.contains(BleUuids.DEVICE_NAME_FILTER)) return
            bluetoothManager?.adapter?.bluetoothLeScanner?.stopScan(this)
            _connectionState.value = ConnectionState.Connecting
            gatt = result.device.connectGatt(context, false, gattCallback, BluetoothDevice.TRANSPORT_LE)
        }
    }

    private val gattCallback = object : BluetoothGattCallback() {
        override fun onConnectionStateChange(g: BluetoothGatt, status: Int, newState: Int) {
            when (newState) {
                BluetoothProfile.STATE_CONNECTED -> {
                    _connectionState.value = ConnectionState.DiscoveringServices
                    g.discoverServices()
                }
                BluetoothProfile.STATE_DISCONNECTED -> {
                    g.close()
                    gatt = null
                    _connectionState.value = ConnectionState.Disconnected("status=$status")
                }
            }
        }

        override fun onServicesDiscovered(g: BluetoothGatt, status: Int) {
            if (status != BluetoothGatt.GATT_SUCCESS) {
                _connectionState.value = ConnectionState.Disconnected("service discovery failed: status=$status")
                return
            }
            ledChar1 = g.getService(BleUuids.IO_SERVICE)?.getCharacteristic(BleUuids.LED1)
            ledChar2 = g.getService(BleUuids.IO_SERVICE)?.getCharacteristic(BleUuids.LED2)
            buzzerChar = g.getService(BleUuids.IO_SERVICE)?.getCharacteristic(BleUuids.BUZZER)
            notifyIndex = 0
            enableNextNotification(g)
        }

        override fun onDescriptorWrite(g: BluetoothGatt, descriptor: BluetoothGattDescriptor, status: Int) {
            if (status != BluetoothGatt.GATT_SUCCESS) {
                Log.w("BleClient", "CCCD write failed for ${NOTIFY_CHARACTERISTICS.getOrNull(notifyIndex)}: status=$status")
            }
            notifyIndex++
            enableNextNotification(g)
        }

        override fun onCharacteristicChanged(
            g: BluetoothGatt,
            characteristic: BluetoothGattCharacteristic,
            value: ByteArray,
        ) {
            when (characteristic.uuid) {
                BleUuids.LPS22HB_CHAR -> _readings.update { it.copy(lps22hb = decodeLps22hb(value)) }
                BleUuids.HDC2010_CHAR -> _readings.update { it.copy(hdc2010 = decodeHdc2010(value)) }
                BleUuids.ACCEL_CHAR -> _readings.update { it.copy(accel = decodeAccel(value)) }
                BleUuids.GYRO_CHAR -> _readings.update { it.copy(gyro = decodeGyro(value)) }
                BleUuids.MAG_CHAR -> _readings.update { it.copy(mag = decodeMag(value)) }
            }
        }
    }

    // GATT操作は直列化する必要がある: 前のリクエストが完了する前に次を発行すると
    // Androidは黙って無視することがあるため、onDescriptorWriteのコールバックを
    // 合図に1つずつNotifyを有効化していく。
    private fun enableNextNotification(g: BluetoothGatt) {
        if (notifyIndex >= NOTIFY_CHARACTERISTICS.size) {
            _connectionState.value = ConnectionState.Connected
            return
        }
        val (serviceUuid, charUuid) = NOTIFY_CHARACTERISTICS[notifyIndex]
        val characteristic = g.getService(serviceUuid)?.getCharacteristic(charUuid)
        val descriptor = characteristic?.getDescriptor(BleUuids.CCCD)
        if (characteristic == null || descriptor == null) {
            notifyIndex++
            enableNextNotification(g)
            return
        }
        g.setCharacteristicNotification(characteristic, true)
        g.writeDescriptor(descriptor, BluetoothGattDescriptor.ENABLE_NOTIFICATION_VALUE)
    }
}
