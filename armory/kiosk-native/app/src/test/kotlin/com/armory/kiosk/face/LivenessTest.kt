package com.armory.kiosk.face

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class LivenessTest {
    private fun signal(
        yaw: Float = 0f,
        left: Float? = 0.9f,
        right: Float? = 0.9f,
        smile: Float? = 0.1f,
        width: Float = 0.4f,
    ) = FaceSignal(yaw, 0f, left, right, smile, width)

    @Test
    fun blinkNeedsCloseThenOpen() {
        val c = Challenge(ChallengeKind.BLINK)
        assertEquals(Progress.PENDING, c.update(signal()))
        assertEquals(Progress.PENDING, c.update(signal(left = 0.1f, right = 0.1f)))
        assertEquals(Progress.PASSED, c.update(signal()))
    }

    @Test
    fun staticOpenEyesNeverPassBlink() {
        val c = Challenge(ChallengeKind.BLINK)
        repeat(50) { assertEquals(Progress.PENDING, c.update(signal())) }
    }

    @Test
    fun startingWithClosedEyesDoesNotPassBlink() {
        val c = Challenge(ChallengeKind.BLINK)
        assertEquals(Progress.PENDING, c.update(signal(left = 0.1f, right = 0.1f)))
        assertEquals(Progress.PENDING, c.update(signal()))
    }

    @Test
    fun turnNeedsTurnThenFrontal() {
        val c = Challenge(ChallengeKind.TURN)
        assertEquals(Progress.PENDING, c.update(signal(yaw = 5f)))
        assertEquals(Progress.PENDING, c.update(signal(yaw = 30f)))
        assertEquals(Progress.PASSED, c.update(signal(yaw = 4f)))
    }

    @Test
    fun turnWorksInEitherDirection() {
        val c = Challenge(ChallengeKind.TURN)
        c.update(signal(yaw = -30f))
        assertEquals(Progress.PASSED, c.update(signal(yaw = 0f)))
    }

    @Test
    fun smileNeedsNeutralFirst() {
        val photo = Challenge(ChallengeKind.SMILE)
        assertEquals(Progress.PENDING, photo.update(signal(smile = 0.9f)))

        val live = Challenge(ChallengeKind.SMILE)
        live.update(signal(smile = 0.1f))
        assertEquals(Progress.PASSED, live.update(signal(smile = 0.9f)))
    }

    @Test
    fun expiresAfterTimeout() {
        var now = 1_000L
        val c = Challenge(ChallengeKind.BLINK, timeoutMs = 8000, clock = { now })
        assertEquals(Progress.PENDING, c.update(signal()))
        now += 8_001
        assertEquals(Progress.EXPIRED, c.update(signal()))
    }

    @Test
    fun usableNeedsSizeFrontalAndOpenEyes() {
        assertTrue(signal().usable)
        assertFalse(signal(width = 0.1f).usable)
        assertFalse(signal(yaw = 20f).usable)
        assertFalse(signal(left = 0.1f, right = 0.1f).usable)
        assertFalse(signal(left = null, right = null).usable)
    }
}
