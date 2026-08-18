package com.osaboh.multisensor

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.gestures.detectDragGestures
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.PathEffect
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.nativeCanvas
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlin.math.max
import kotlin.math.roundToInt

// dataviz skillの検証済みカテゴリカルパレット(references/palette.md)先頭3スロット
// (blue/orange/aqua)。light/dark双方でall-pairs CVD検証済みのため、X/Y/Zの3系列に
// 固定順で割り当てる（順序を変えない）。
private val SeriesColors = listOf(
    Color(0xFF2A78D6), // X
    Color(0xFFEB6834), // Y
    Color(0xFF1BAF7A), // Z
)
private val AxisColor = Color(0x33808080) // 控えめなグリッド線/クロスヘア

// 加速度・ジャイロの直近履歴をスクロール式ラインチャートで表示する。X/Y/Zは固定の
// カテゴリカル配色、凡例は常に表示（テキストラベルは色に依存しない識別手段）。
// タッチ&ドラッグでクロスヘア+値ツールチップを出す（モバイル版のhover相当）。
@Composable
fun SensorLineChart(points: List<ChartPoint>, unit: String, modifier: Modifier = Modifier) {
    var touchIndex by remember { mutableStateOf<Int?>(null) }
    // pointerInputはUnitキーで固定し、ドラッグ中にNotifyでpointsが更新されても
    // ジェスチャー検出自体は再生成されないようにする（再生成されるとタッチ中の
    // 追跡が途切れて「タッチ表示が止まる」不具合になる）。最新のpointsは
    // rememberUpdatedStateで参照する。
    val latestPoints by rememberUpdatedState(points)

    Row(verticalAlignment = Alignment.CenterVertically) {
        LegendDot(SeriesColors[0])
        Text(text = "X")
        Spacer(modifier = Modifier.width(8.dp))
        LegendDot(SeriesColors[1])
        Text(text = "Y")
        Spacer(modifier = Modifier.width(8.dp))
        LegendDot(SeriesColors[2])
        Text(text = "Z")
    }
    Canvas(
        modifier = modifier
            .fillMaxWidth()
            .height(120.dp)
            .pointerInput(Unit) {
                detectDragGestures(
                    onDragEnd = { touchIndex = null },
                    onDragCancel = { touchIndex = null },
                ) { change, _ ->
                    change.consume()
                    val current = latestPoints
                    if (current.isNotEmpty()) {
                        val ratio = (change.position.x / size.width).coerceIn(0f, 1f)
                        touchIndex = (ratio * (current.size - 1)).roundToInt()
                    }
                }
            },
    ) {
        if (points.size < 2) return@Canvas

        val allValues = points.flatMap { listOf(it.x, it.y, it.z) }
        val rawMax = allValues.max()
        val rawMin = allValues.min()
        val span = max(rawMax - rawMin, 0.001f)
        val pad = span * 0.1f
        val maxV = rawMax + pad
        val minV = rawMin - pad

        fun xFor(i: Int) = size.width * i / (points.size - 1)
        fun yFor(v: Float) = size.height * (1f - (v - minV) / (maxV - minV))

        if (minV < 0f && maxV > 0f) {
            drawLine(
                color = AxisColor,
                start = Offset(0f, yFor(0f)),
                end = Offset(size.width, yFor(0f)),
                strokeWidth = 1.dp.toPx(),
                pathEffect = PathEffect.dashPathEffect(floatArrayOf(6f, 6f)),
            )
        }

        listOf(
            SeriesColors[0] to points.map { it.x },
            SeriesColors[1] to points.map { it.y },
            SeriesColors[2] to points.map { it.z },
        ).forEach { (color, series) ->
            for (i in 0 until series.size - 1) {
                drawLine(
                    color = color,
                    start = Offset(xFor(i), yFor(series[i])),
                    end = Offset(xFor(i + 1), yFor(series[i + 1])),
                    strokeWidth = 2.dp.toPx(),
                    cap = StrokeCap.Round,
                )
            }
        }

        touchIndex?.let { idx ->
            val i = idx.coerceIn(0, points.size - 1)
            val x = xFor(i)
            drawLine(
                color = AxisColor,
                start = Offset(x, 0f),
                end = Offset(x, size.height),
                strokeWidth = 1.dp.toPx(),
            )
            val p = points[i]
            val label = "X:%.1f Y:%.1f Z:%.1f %s".format(p.x, p.y, p.z, unit)
            val paint = android.graphics.Paint().apply {
                color = android.graphics.Color.WHITE
                textSize = 12.sp.toPx()
                isAntiAlias = true
            }
            // ラベルはタッチ位置中心に描くが、画面端で切れないよう左右にクランプする。
            val textWidth = paint.measureText(label)
            val maxTextX = max(0f, size.width - textWidth)
            val textX = (x - textWidth / 2f).coerceIn(0f, maxTextX)
            drawContext.canvas.nativeCanvas.drawText(label, textX, 16.dp.toPx(), paint)
        }
    }
}

@Composable
private fun LegendDot(color: Color) {
    Box(modifier = Modifier.size(8.dp).background(color, CircleShape))
    Spacer(modifier = Modifier.width(4.dp))
}
