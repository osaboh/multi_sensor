package com.osaboh.multisensor

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.Slider
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
import kotlin.math.roundToInt
import kotlinx.coroutines.delay

@Composable
fun MultiSensorScreen(bleClient: BleClient, modifier: Modifier = Modifier) {
    val connectionState by bleClient.connectionState.collectAsState()
    val readings by bleClient.readings.collectAsState()
    val writeStatus by bleClient.writeStatus.collectAsState()
    val accelHistory by bleClient.accelHistory.collectAsState()
    val gyroHistory by bleClient.gyroHistory.collectAsState()
    var led1On by remember { mutableStateOf(false) }
    var led2On by remember { mutableStateOf(false) }
    var buzzerOn by remember { mutableStateOf(false) }
    var intervalMs by remember { mutableStateOf(1000f) }
    val isConnected = connectionState == ConnectionState.Connected

    LaunchedEffect(connectionState) {
        if (!isConnected) {
            led1On = false
            led2On = false
            buzzerOn = false
        }
    }

    // ブザーはファームウェア側で300ms後に自動停止する(Notifyでの状態通知はない)ため、
    // スイッチの見た目も実機の鳴動時間に合わせて自動でOFFへ戻す。
    LaunchedEffect(buzzerOn) {
        if (buzzerOn) {
            delay(300)
            buzzerOn = false
        }
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            Text(text = "接続状態: ${connectionStateLabel(connectionState)}")
            if (connectionState is ConnectionState.Idle || connectionState is ConnectionState.Disconnected) {
                Button(onClick = { bleClient.startScan() }) {
                    Text(text = "再スキャン")
                }
            } else {
                Button(onClick = { bleClient.disconnect() }) {
                    Text(text = "切断")
                }
            }
        }
        Spacer(modifier = Modifier.height(16.dp))

        val pressureText = readings.lps22hb?.let { "%.2f hPa".format(it.pressureHPa) } ?: "-"
        val humidityText = readings.hdc2010?.let { "%.2f %%RH".format(it.humidityPct) } ?: "-"
        val lps22hbTemp = readings.lps22hb?.temperatureC
        val hdc2010Temp = readings.hdc2010?.temperatureC
        val tempText = if (lps22hbTemp != null && hdc2010Temp != null) {
            "%.2f ℃".format((lps22hbTemp + hdc2010Temp) / 2)
        } else {
            (lps22hbTemp ?: hdc2010Temp)?.let { "%.2f ℃".format(it) } ?: "-"
        }
        Text(text = "温度: $tempText　湿度: $humidityText")
        Text(text = "気圧: $pressureText")
        Text(text = "加速度: " + (readings.accel?.let {
            "%.1f, %.1f, %.1f mg".format(it.xMg, it.yMg, it.zMg)
        } ?: "-"))
        SensorLineChart(points = accelHistory, unit = "mg", modifier = Modifier.padding(top = 4.dp, bottom = 12.dp))
        Text(text = "ジャイロ: " + (readings.gyro?.let {
            "%.1f, %.1f, %.1f °/s".format(it.xDps, it.yDps, it.zDps)
        } ?: "-"))
        SensorLineChart(points = gyroHistory, unit = "°/s", modifier = Modifier.padding(top = 4.dp, bottom = 12.dp))
        Text(text = "更新間隔: ${intervalMs.roundToInt()}ms（速いほど電池消費増）")
        Slider(
            value = intervalMs,
            onValueChange = { intervalMs = it },
            onValueChangeFinished = { bleClient.setBmx055IntervalMs(intervalMs.roundToInt()) },
            valueRange = 150f..5000f,
            enabled = isConnected,
        )
        Spacer(modifier = Modifier.height(8.dp))
        Text(text = "磁力: " + (readings.mag?.let {
            "%.1f, %.1f, %.1f µT".format(it.xUt, it.yUt, it.zUt)
        } ?: "-"))
        Text(text = "SW_TOP: " + (readings.swTop?.let { if (it) "ON" else "OFF" } ?: "-"))
        Text(text = "SW_SIDE: " + (readings.swSide?.let { if (it) "ON" else "OFF" } ?: "-"))

        Spacer(modifier = Modifier.height(16.dp))
        Row(modifier = Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Text(text = "LED 緑")
            Spacer(modifier = Modifier.width(8.dp))
            Switch(
                checked = led1On,
                onCheckedChange = {
                    led1On = it
                    bleClient.setLed1(it)
                },
                enabled = isConnected,
            )
            Spacer(modifier = Modifier.width(24.dp))
            Text(text = "LED 赤")
            Spacer(modifier = Modifier.width(8.dp))
            Switch(
                checked = led2On,
                onCheckedChange = {
                    led2On = it
                    bleClient.setLed2(it)
                },
                enabled = isConnected,
            )
            Spacer(modifier = Modifier.weight(1f))
            Text(text = "ブザー")
            Spacer(modifier = Modifier.width(8.dp))
            Switch(
                checked = buzzerOn,
                onCheckedChange = { on ->
                    buzzerOn = on
                    bleClient.triggerBuzzer(if (on) 300 else 0)
                },
                enabled = isConnected,
            )
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
