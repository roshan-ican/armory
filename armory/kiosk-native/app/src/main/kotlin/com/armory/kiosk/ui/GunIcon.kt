package com.armory.kiosk.ui

import android.content.Context
import android.graphics.Bitmap
import android.graphics.Canvas
import android.graphics.Matrix
import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.height
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.produceState
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.BlendMode
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.ColorFilter
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import com.caverock.androidsvg.SVG
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.util.concurrent.ConcurrentHashMap

private object GunArt {
    private val cache = ConcurrentHashMap<String, Bitmap>()

    fun load(context: Context, kind: String): Bitmap? {
        val key = if (kind == "pistol") "pistol" else "rifle"
        cache[key]?.let { return it }
        return runCatching {
            val svg = context.assets.open("guns/$key.svg").use { SVG.getFromInputStream(it) }
            val box = svg.documentViewBox
            val width = if (key == "rifle") 1000 else 760
            val height = (width * box.height() / box.width()).toInt()
            val flat = Bitmap.createBitmap(width, height, Bitmap.Config.ARGB_8888)
            svg.setDocumentWidth(width.toFloat())
            svg.setDocumentHeight(height.toFloat())
            svg.renderToCanvas(Canvas(flat))
            val turn = Matrix().apply { postRotate(if (key == "rifle") -90f else 90f) }
            Bitmap.createBitmap(flat, 0, 0, width, height, turn, true).also { cache[key] = it }
        }.getOrNull()
    }
}

fun preloadGunArt(context: Context) {
    val app = context.applicationContext
    Thread { listOf("rifle", "pistol").forEach { GunArt.load(app, it) } }.start()
}

@Composable
fun GunIcon(kind: String, tone: Color, heightDp: Int) {
    val context = LocalContext.current
    val bitmap by produceState<Bitmap?>(null, kind) {
        value = withContext(Dispatchers.Default) { GunArt.load(context, kind) }
    }
    val art = bitmap
    if (art == null) {
        Box(Modifier.height(heightDp.dp))
        return
    }
    Image(
        bitmap = art.asImageBitmap(),
        contentDescription = null,
        colorFilter = ColorFilter.tint(tone, BlendMode.SrcIn),
        contentScale = ContentScale.Fit,
        modifier = Modifier.height(heightDp.dp).aspectRatio(art.width.toFloat() / art.height),
    )
}
