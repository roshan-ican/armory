package com.armory.kiosk.ui

import android.content.Context
import android.speech.tts.TextToSpeech
import java.util.Locale

private const val VOICE = "en-us-x-iom-local"

class Speaker(context: Context) {
    @Volatile
    private var ready = false
    private var tts: TextToSpeech? = null

    init {
        tts = TextToSpeech(context.applicationContext) { status ->
            val engine = tts
            if (status == TextToSpeech.SUCCESS && engine != null) {
                val result = engine.setLanguage(Locale.getDefault())
                engine.voices?.firstOrNull { it.name == VOICE && !it.isNetworkConnectionRequired }?.let { engine.voice = it }
                ready = result != TextToSpeech.LANG_MISSING_DATA && result != TextToSpeech.LANG_NOT_SUPPORTED
            }
        }
    }

    fun say(text: String) {
        if (ready) tts?.speak(text, TextToSpeech.QUEUE_FLUSH, null, "armory")
    }

    fun shutdown() {
        tts?.shutdown()
        tts = null
    }
}
