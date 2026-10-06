package com.armory.kiosk

import android.Manifest
import android.content.pm.PackageManager
import android.os.Bundle
import android.view.WindowManager
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.result.contract.ActivityResultContracts
import androidx.activity.viewModels
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.core.content.ContextCompat
import androidx.core.view.WindowCompat
import androidx.core.view.WindowInsetsCompat
import androidx.core.view.WindowInsetsControllerCompat
import com.armory.kiosk.ui.KioskApp
import com.armory.kiosk.ui.KioskViewModel
import com.armory.kiosk.ui.preloadGunArt

class MainActivity : ComponentActivity() {
    private val model: KioskViewModel by viewModels()

    private var granted by mutableStateOf(false)

    private val permission = registerForActivityResult(ActivityResultContracts.RequestPermission()) { granted = it }

    override fun onResume() {
        super.onResume()
        granted = ContextCompat.checkSelfPermission(this, Manifest.permission.CAMERA) == PackageManager.PERMISSION_GRANTED
        model.refreshNow()
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        window.addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON)
        WindowCompat.setDecorFitsSystemWindows(window, false)
        WindowInsetsControllerCompat(window, window.decorView).apply {
            systemBarsBehavior = WindowInsetsControllerCompat.BEHAVIOR_SHOW_TRANSIENT_BARS_BY_SWIPE
            hide(WindowInsetsCompat.Type.systemBars())
        }
        preloadGunArt(applicationContext)
        granted = ContextCompat.checkSelfPermission(this, Manifest.permission.CAMERA) == PackageManager.PERMISSION_GRANTED
        if (!granted) permission.launch(Manifest.permission.CAMERA)
        setContent { KioskApp(model, granted) }
    }
}
