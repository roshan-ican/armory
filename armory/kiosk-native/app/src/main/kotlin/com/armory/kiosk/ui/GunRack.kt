package com.armory.kiosk.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.armory.kiosk.data.Locker
import com.armory.kiosk.data.Slot

private val PanelColor = Color(0xFF34373A)
private val PlateColor = Color(0xFFD6D6D2)
private const val COLUMNS = 4

@Composable
fun GunRack(
    locker: Locker,
    modifier: Modifier = Modifier,
    picked: List<Int> = emptyList(),
    onToggle: ((Int) -> Unit)? = null,
    iconHeight: Int = 54,
    panelHeight: Int? = null,
) {
    val dot = Color(0x24000000)
    var panel = modifier
        .clip(RoundedCornerShape(14.dp))
        .background(PanelColor)
        .drawBehind {
            val step = 14.dp.toPx()
            var y = step / 2
            while (y < size.height) {
                var x = step / 2
                while (x < size.width) {
                    drawCircle(dot, radius = 2.2.dp.toPx(), center = Offset(x, y))
                    x += step
                }
                y += step
            }
        }
    if (panelHeight != null) panel = panel.height(panelHeight.dp)
    Box(panel, contentAlignment = Alignment.Center) {
        Column(Modifier.padding(horizontal = 14.dp, vertical = 12.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
            locker.slots.chunked(COLUMNS).forEach { row ->
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    row.forEach { slot ->
                        RackCell(
                            locker = locker,
                            slot = slot,
                            picked = slot.no in picked,
                            iconHeight = iconHeight,
                            onClick = if (onToggle != null && slot.available) ({ onToggle(slot.no) }) else null,
                        )
                    }
                }
            }
        }
    }
}

private const val CELL_WIDTH = 64

@Composable
private fun RackCell(locker: Locker, slot: Slot, picked: Boolean, iconHeight: Int, onClick: (() -> Unit)?) {
    val shape = RoundedCornerShape(10.dp)
    var m = Modifier.width(CELL_WIDTH.dp).clip(shape)
    m = if (picked) m.background(Color(0x330A84FF)).border(2.dp, Palette.Blue, shape) else m
    if (onClick != null) m = m.clickable(onClick = onClick)
    val dim = !slot.available && onClick == null && picked.not() && locker.online && slot.reading == 0
    Column(m.padding(vertical = 6.dp), horizontalAlignment = Alignment.CenterHorizontally) {
        Box(Modifier.height(iconHeight.dp), contentAlignment = Alignment.BottomCenter) {
            GunIcon(locker.kind, toneOf(locker, slot), iconHeight)
        }
        Box(
            Modifier.padding(top = 2.dp).width(40.dp).height(8.dp).clip(RoundedCornerShape(3.dp))
                .background(PlateColor).alpha(if (dim) 0.55f else 1f),
        )
        Text(
            "${slot.no}",
            color = Color(0xFFE6E6E3),
            fontSize = 11.sp,
            fontWeight = FontWeight.SemiBold,
            modifier = Modifier.padding(top = 3.dp),
        )
        if (slot.takenBy.isNotEmpty()) {
            Text(
                slot.takenBy, color = Color(0xFFFF918A), fontSize = 10.sp, maxLines = 1,
                textAlign = TextAlign.Center, modifier = Modifier.fillMaxWidth(),
            )
        }
    }
}
