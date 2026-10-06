package com.armory.kiosk.face

import kotlin.math.abs

enum class ChallengeKind(val prompt: String) {
    BLINK("Blink your eyes"),
    TURN("Turn your head to either side, then look back"),
    SMILE("Give a smile"),
}

data class FaceSignal(
    val yaw: Float,
    val pitch: Float,
    val leftEye: Float?,
    val rightEye: Float?,
    val smile: Float?,
    val widthRatio: Float,
) {
    val eyesOpen: Boolean
        get() = leftEye != null && rightEye != null && leftEye > OPEN && rightEye > OPEN

    val eyesClosed: Boolean
        get() = leftEye != null && rightEye != null && leftEye < CLOSED && rightEye < CLOSED

    val frontal: Boolean
        get() = abs(yaw) < FRONTAL_YAW && abs(pitch) < FRONTAL_PITCH

    val neutral: Boolean
        get() = smile != null && smile < NEUTRAL

    val usable: Boolean
        get() = widthRatio >= MIN_WIDTH && frontal && eyesOpen

    companion object {
        const val OPEN = 0.6f
        const val CLOSED = 0.25f
        const val NEUTRAL = 0.35f
        const val SMILING = 0.75f
        const val FRONTAL_YAW = 12f
        const val FRONTAL_PITCH = 15f
        const val TURNED_YAW = 22f
        const val MIN_WIDTH = 0.28f
    }
}

enum class Progress { PENDING, PASSED, EXPIRED }

class Challenge(
    val kind: ChallengeKind,
    private val timeoutMs: Long = 8000,
    private val clock: () -> Long = System::currentTimeMillis,
) {
    private val started = clock()
    private var sawOpen = false
    private var sawClosed = false
    private var sawTurn = false
    private var sawNeutral = false

    fun update(s: FaceSignal): Progress {
        if (clock() - started > timeoutMs) return Progress.EXPIRED
        val done = when (kind) {
            ChallengeKind.BLINK -> {
                if (s.eyesOpen && !sawClosed) sawOpen = true
                if (sawOpen && s.eyesClosed) sawClosed = true
                sawClosed && s.eyesOpen
            }
            ChallengeKind.TURN -> {
                if (abs(s.yaw) > FaceSignal.TURNED_YAW) sawTurn = true
                sawTurn && s.frontal
            }
            ChallengeKind.SMILE -> {
                if (s.neutral) sawNeutral = true
                sawNeutral && (s.smile ?: 0f) > FaceSignal.SMILING
            }
        }
        return if (done) Progress.PASSED else Progress.PENDING
    }

    companion object {
        fun random(): Challenge = Challenge(ChallengeKind.entries.random())
    }
}
