package com.armory.kiosk.ui

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxScope
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Rect
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.PathOperation
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.platform.LocalContext
import android.content.Context
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

object Palette {
    var dark by mutableStateOf(true)

    private fun pick(d: Long, l: Long) = Color(if (dark) d else l)

    val Bg get() = pick(0xFF070708, 0xFFF2F2F7)
    val Surface get() = pick(0xFF262629, 0xFFFFFFFF)
    val Fill get() = pick(0x1AFFFFFF, 0x0F000000)
    val Text get() = pick(0xFFF5F5F7, 0xFF1C1C1E)
    val Muted get() = pick(0xFFA1A1A8, 0xFF636366)
    val Dim get() = pick(0xFF68686F, 0xFF8E8E93)
    val Blue get() = pick(0xFF0A84FF, 0xFF007AFF)
    val BlueSoft get() = pick(0xFF8ABFFF, 0xFF0062CC)
    val Green get() = pick(0xFF30D158, 0xFF248A3D)
    val Red get() = pick(0xFFFF453A, 0xFFD70015)
    val Yellow get() = pick(0xFFFFD60A, 0xFFB25000)
    val Grey get() = pick(0xFF636366, 0xFF8E8E93)
    val Edge get() = pick(0x12FFFFFF, 0x14000000)
    val FieldLine get() = pick(0x22FFFFFF, 0x2E000000)
    val Ghost get() = pick(0x14FFFFFF, 0x0F000000)
    val Sub get() = pick(0xFFD4D4D8, 0xFF3A3A3C)
    val DangerText get() = pick(0xFFFF918A, 0xFFD70015)
    val Rack get() = pick(0xFF34373A, 0xFFE4E4E9)
    val RackDot get() = pick(0x24000000, 0x14000000)
    val Plate get() = pick(0xFFD6D6D2, 0xFFB4B4BA)
    val RackLabel get() = pick(0xFFE6E6E3, 0xFF3A3A3C)
    val Selected get() = pick(0x2B0A84FF, 0x1F007AFF)

    private const val PREFS = "armory_ui"

    fun load(context: Context) {
        dark = context.getSharedPreferences(PREFS, Context.MODE_PRIVATE).getBoolean("dark", true)
    }

    fun toggle(context: Context) {
        dark = !dark
        context.getSharedPreferences(PREFS, Context.MODE_PRIVATE).edit().putBoolean("dark", dark).apply()
    }
}

@Composable
fun ArmoryTheme(content: @Composable () -> Unit) {
    val scheme = if (Palette.dark) {
        darkColorScheme(
            primary = Palette.Blue,
            background = Palette.Bg,
            surface = Palette.Surface,
            onSurface = Palette.Text,
            onBackground = Palette.Text,
        )
    } else {
        lightColorScheme(
            primary = Palette.Blue,
            background = Palette.Bg,
            surface = Palette.Surface,
            onSurface = Palette.Text,
            onBackground = Palette.Text,
        )
    }
    MaterialTheme(colorScheme = scheme, content = content)
}

@Composable
fun ThemeToggle() {
    val context = LocalContext.current
    val tint = Palette.Text
    Box(
        Modifier.size(38.dp).clip(CircleShape).background(Palette.Fill).clickable { Palette.toggle(context) },
        contentAlignment = Alignment.Center,
    ) {
        Canvas(Modifier.size(20.dp)) {
            val c = center
            val r = size.minDimension / 2
            if (Palette.dark) {
                val moon = Path().apply {
                    addOval(Rect(c, r * 0.9f))
                }
                val bite = Path().apply {
                    addOval(Rect(Offset(c.x + r * 0.5f, c.y - r * 0.4f), r * 0.8f))
                }
                val shape = Path.combine(PathOperation.Difference, moon, bite)
                drawPath(shape, tint)
            } else {
                drawCircle(tint, r * 0.42f, c)
                for (i in 0 until 8) {
                    val a = i * Math.PI / 4
                    val dx = Math.cos(a).toFloat()
                    val dy = Math.sin(a).toFloat()
                    drawLine(
                        tint,
                        Offset(c.x + dx * r * 0.7f, c.y + dy * r * 0.7f),
                        Offset(c.x + dx * r, c.y + dy * r),
                        strokeWidth = 2.dp.toPx(),
                        cap = StrokeCap.Round,
                    )
                }
            }
        }
    }
}

@Composable
fun Copy(eyebrow: String, title: String, body: String, modifier: Modifier = Modifier) {
    Column(modifier, horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Text(eyebrow, color = Palette.BlueSoft, fontSize = 12.sp, fontWeight = FontWeight.Bold, letterSpacing = 1.4.sp)
        Text(title, color = Palette.Text, fontSize = 38.sp, fontWeight = FontWeight.Bold, textAlign = TextAlign.Center, lineHeight = 42.sp)
        if (body.isNotEmpty()) {
            Text(body, color = Palette.Muted, fontSize = 17.sp, textAlign = TextAlign.Center, lineHeight = 24.sp)
        }
    }
}

@Composable
fun PrimaryButton(text: String, enabled: Boolean = true, onClick: () -> Unit) {
    Button(
        onClick = onClick,
        enabled = enabled,
        shape = RoundedCornerShape(17.dp),
        colors = ButtonDefaults.buttonColors(
            containerColor = Palette.Blue,
            contentColor = Color.White,
            disabledContainerColor = Palette.Blue.copy(alpha = 0.55f),
            disabledContentColor = Color.White.copy(alpha = 0.85f),
        ),
        modifier = Modifier.heightIn(min = 54.dp),
    ) {
        Text(text, fontSize = 16.sp, fontWeight = FontWeight.SemiBold, modifier = Modifier.padding(horizontal = 18.dp))
    }
}

@Composable
fun QuietButton(text: String, danger: Boolean = false, onClick: () -> Unit) {
    TextButton(onClick = onClick) {
        Text(text, color = if (danger) Palette.DangerText else Palette.BlueSoft, fontSize = 15.sp, fontWeight = FontWeight.SemiBold)
    }
}

@Composable
fun SecondaryButton(text: String, onClick: () -> Unit) {
    Button(
        onClick = onClick,
        shape = RoundedCornerShape(15.dp),
        colors = ButtonDefaults.buttonColors(containerColor = Palette.Fill, contentColor = Palette.Text),
        modifier = Modifier.heightIn(min = 48.dp),
    ) {
        Text(text, fontWeight = FontWeight.SemiBold)
    }
}

@Composable
fun Symbol(glyph: String, tone: Color) {
    Box(
        Modifier.size(104.dp).clip(RoundedCornerShape(31.dp)).background(tone.copy(alpha = 0.13f)),
        contentAlignment = Alignment.Center,
    ) {
        Text(glyph, color = tone, fontSize = 50.sp)
    }
}

@Composable
fun Card(
    modifier: Modifier = Modifier,
    selected: Boolean = false,
    onClick: (() -> Unit)? = null,
    content: @Composable BoxScope.() -> Unit,
) {
    val shape = RoundedCornerShape(22.dp)
    val line = if (selected) BorderStroke(2.dp, Palette.Blue) else BorderStroke(1.dp, Palette.Edge)
    var m = modifier.clip(shape).background(if (selected) Palette.Selected else Palette.Surface).border(line, shape)
    if (onClick != null) m = m.clickable(onClick = onClick)
    Box(m, content = content)
}
