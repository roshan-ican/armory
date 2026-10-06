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
    val Bg = Color(0xFF070708)
    val Surface = Color(0xFF262629)
    val Fill = Color(0x1AFFFFFF)
    val Text = Color(0xFFF5F5F7)
    val Muted = Color(0xFFA1A1A8)
    val Dim = Color(0xFF68686F)
    val Blue = Color(0xFF0A84FF)
    val BlueSoft = Color(0xFF8ABFFF)
    val Green = Color(0xFF30D158)
    val Red = Color(0xFFFF453A)
    val Yellow = Color(0xFFFFD60A)
    val Grey = Color(0xFF636366)
}

@Composable
fun ArmoryTheme(content: @Composable () -> Unit) {
    MaterialTheme(
        colorScheme = darkColorScheme(
            primary = Palette.Blue,
            background = Palette.Bg,
            surface = Palette.Surface,
            onSurface = Palette.Text,
            onBackground = Palette.Text,
        ),
        content = content,
    )
}

@Composable
fun Copy(eyebrow: String, title: String, body: String, modifier: Modifier = Modifier) {
    Column(modifier, horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Text(eyebrow, color = Palette.BlueSoft, fontSize = 12.sp, fontWeight = FontWeight.Bold, letterSpacing = 1.4.sp)
        Text(title, color = Color.White, fontSize = 38.sp, fontWeight = FontWeight.Bold, textAlign = TextAlign.Center, lineHeight = 42.sp)
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
        Text(text, color = if (danger) Color(0xFFFF6961) else Palette.BlueSoft, fontSize = 15.sp, fontWeight = FontWeight.SemiBold)
    }
}

@Composable
fun SecondaryButton(text: String, onClick: () -> Unit) {
    Button(
        onClick = onClick,
        shape = RoundedCornerShape(15.dp),
        colors = ButtonDefaults.buttonColors(containerColor = Palette.Fill, contentColor = Color.White),
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
    val line = if (selected) BorderStroke(2.dp, Palette.Blue) else BorderStroke(1.dp, Color(0x12FFFFFF))
    var m = modifier.clip(shape).background(if (selected) Color(0x2B0A84FF) else Palette.Surface).border(line, shape)
    if (onClick != null) m = m.clickable(onClick = onClick)
    Box(m, content = content)
}
