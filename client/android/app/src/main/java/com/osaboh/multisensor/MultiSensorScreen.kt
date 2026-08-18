package com.osaboh.multisensor

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.material3.Button
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp

@Composable
fun MultiSensorScreen(bleClient: BleClient, modifier: Modifier = Modifier) {
    val connectionState by bleClient.connectionState.collectAsState()
    val readings by bleClient.readings.collectAsState()
    val writeStatus by bleClient.writeStatus.collectAsState()
    var led1On by remember { mutableStateOf(false) }
    var led2On by remember { mutableStateOf(false) }
    val isConnected = connectionState == ConnectionState.Connected

    LaunchedEffect(connectionState) {
        if (!isConnected) {
            led1On = false
            led2On = false
        }
    }

    Column(modifier = modifier.fillMaxSize().padding(16.dp)) {
        Text(text = "接続状態: ${connectionStateLabel(connectionState)}")
        if (connectionState is ConnectionState.Idle || connectionState is ConnectionState.Disconnected) {
            Spacer(modifier = Modifier.height(8.dp))
            Button(onClick = { bleClient.startScan() }) {
                Text(text = "再スキャン")
            }
        }
        Spacer(modifier = Modifier.height(16.dp))

        Text(text = "気圧/温度(LPS22HB): " + (readings.lps22hb?.let {
            "%.2f hPa, %.2f ℃".format(it.pressureHPa, it.temperatureC)
        } ?: "-"))
        Text(text = "温湿度(HDC2010): " + (readings.hdc2010?.let {
            "%.2f ℃, %.2f %%RH".format(it.temperatureC, it.humidityPct)
        } ?: "-"))
        Text(text = "加速度: " + (readings.accel?.let {
            "%.1f, %.1f, %.1f mg".format(it.xMg, it.yMg, it.zMg)
        } ?: "-"))
        Text(text = "ジャイロ: " + (readings.gyro?.let {
            "%.1f, %.1f, %.1f °/s".format(it.xDps, it.yDps, it.zDps)
        } ?: "-"))
        Text(text = "磁力: " + (readings.mag?.let {
            "%.1f, %.1f, %.1f µT".format(it.xUt, it.yUt, it.zUt)
        } ?: "-"))
        Text(text = "SW_TOP: " + (readings.swTop?.let { if (it) "ON" else "OFF" } ?: "-"))
        Text(text = "SW_SIDE: " + (readings.swSide?.let { if (it) "ON" else "OFF" } ?: "-"))

        Spacer(modifier = Modifier.height(16.dp))
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(text = "LED1")
            Spacer(modifier = Modifier.width(8.dp))
            Switch(
                checked = led1On,
                onCheckedChange = {
                    led1On = it
                    bleClient.setLed1(it)
                },
                enabled = isConnected,
            )
        }
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(text = "LED2")
            Spacer(modifier = Modifier.width(8.dp))
            Switch(
                checked = led2On,
                onCheckedChange = {
                    led2On = it
                    bleClient.setLed2(it)
                },
                enabled = isConnected,
            )
        }
        Spacer(modifier = Modifier.height(8.dp))
        Button(onClick = { bleClient.triggerBuzzer(300) }, enabled = isConnected) {
            Text(text = "Buzzer 300ms")
        }
        if (writeStatus != null) {
            Spacer(modifier = Modifier.height(8.dp))
            Text(text = writeStatus!!)
        }
    }
}

private fun connectionStateLabel(state: ConnectionState): String = when (state) {
    ConnectionState.Idle -> "待機中"
    ConnectionState.Scanning -> "スキャン中..."
    ConnectionState.Connecting -> "接続中..."
    ConnectionState.DiscoveringServices -> "サービス探索中..."
    ConnectionState.Connected -> "接続済み"
    is ConnectionState.Disconnected -> "切断: ${state.reason}"
}
