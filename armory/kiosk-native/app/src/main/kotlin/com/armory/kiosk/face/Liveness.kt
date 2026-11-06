package com.armory.kiosk.face

import kotlin.math.abs

enum class ChallengeKind(val prompt: String) {
    BLINK("Blink your eyes"),
    TURN("Turn your head to either side, then look back"),
    SMILE("Give a smile"),
    ANY("Blink, smile or turn your head"),
    TURN_LEFT("Turn your head to your left, then look back"),
    TURN_RIGHT("Turn your head to your right, then look back"),
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
        const val SMILING = 0.6f
        const val FRONTAL_YAW = 12f
        const val FRONTAL_PITCH = 15f
        const val TURNED_YAW = 18f
        const val MIN_WIDTH = 0.28f
        const val LEFT_YAW_SIGN = 1f
    }
}

enum class Progress { PENDING, PASSED, EXPIRED }

class Challenge(
    val kind: ChallengeKind,
    private val timeoutMs: Long = 6000,
    private val clock: () -> Long = System::currentTimeMillis,
) {
    private val started = clock()
    private var sawOpen = false
    private var sawClosed = false
    private var sawTurn = false
    private var sawNeutral = false

    private fun blinkDone(s: FaceSignal): Boolean {
        if (s.eyesOpen && !sawClosed) sawOpen = true
        if (sawOpen && s.eyesClosed) sawClosed = true
        return sawClosed && s.eyesOpen
    }

    private fun turnDone(s: FaceSignal): Boolean {
        if (abs(s.yaw) > FaceSignal.TURNED_YAW) sawTurn = true
        return sawTurn && s.frontal
    }

    private fun turnDoneToward(s: FaceSignal, sign: Float): Boolean {
        if (s.yaw * sign > FaceSignal.TURNED_YAW) sawTurn = true
        return sawTurn && s.frontal
    }

    private fun smileDone(s: FaceSignal): Boolean {
        if (s.neutral) sawNeutral = true
        return sawNeutral && (s.smile ?: 0f) > FaceSignal.SMILING
    }

    fun update(s: FaceSignal): Progress {
        if (clock() - started > timeoutMs) return Progress.EXPIRED
        val done = when (kind) {
            ChallengeKind.BLINK -> blinkDone(s)
            ChallengeKind.TURN -> turnDone(s)
            ChallengeKind.SMILE -> smileDone(s)
            ChallengeKind.ANY -> listOf(blinkDone(s), turnDone(s), smileDone(s)).any { it }
            ChallengeKind.TURN_LEFT -> turnDoneToward(s, FaceSignal.LEFT_YAW_SIGN)
            ChallengeKind.TURN_RIGHT -> turnDoneToward(s, -FaceSignal.LEFT_YAW_SIGN)
        }
        return if (done) Progress.PASSED else Progress.PENDING
    }

    companion object {
        fun random(): Challenge = Challenge(ChallengeKind.entries.filter { it != ChallengeKind.ANY }.random())

        fun any(): Challenge = Challenge(ChallengeKind.ANY)

        private val SEQUENCE_POOL = listOf(ChallengeKind.BLINK, ChallengeKind.SMILE, ChallengeKind.TURN_LEFT, ChallengeKind.TURN_RIGHT)

        fun sequence(count: Int = 2, random: kotlin.random.Random = kotlin.random.Random.Default): List<ChallengeKind> =
            SEQUENCE_POOL.shuffled(random).take(count)
    }
}
