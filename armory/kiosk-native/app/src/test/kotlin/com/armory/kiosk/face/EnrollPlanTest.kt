package com.armory.kiosk.face

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class EnrollPlanTest {
    private fun signal(
        yaw: Float = 0f,
        smile: Float? = 0.1f,
        width: Float = 0.4f,
        eyes: Float = 0.9f,
    ) = FaceSignal(yaw, 0f, eyes, eyes, smile, width)

    @Test
    fun startsWithAStraightShot() {
        val plan = EnrollPlan()
        assertEquals(Pose.STRAIGHT, plan.current)
        assertFalse(plan.fits(signal(yaw = 15f)))
        assertTrue(plan.fits(signal()))
    }

    @Test
    fun rejectsClosedEyesAndSmallFaces() {
        val plan = EnrollPlan()
        assertFalse(plan.fits(signal(eyes = 0.1f)))
        assertFalse(plan.fits(signal(width = 0.1f)))
    }

    @Test
    fun secondTurnMustBeTheOtherSide() {
        val plan = EnrollPlan(listOf(Pose.TURN_FIRST, Pose.TURN_SECOND))
        assertFalse(plan.fits(signal(yaw = 2f)))
        assertTrue(plan.fits(signal(yaw = 15f)))
        plan.commit(signal(yaw = 15f))
        assertFalse(plan.fits(signal(yaw = 15f)))
        assertFalse(plan.fits(signal(yaw = -3f)))
        assertTrue(plan.fits(signal(yaw = -15f)))
    }

    @Test
    fun turnsBeyondTheLimitAreRejected() {
        val plan = EnrollPlan(listOf(Pose.TURN_FIRST))
        assertFalse(plan.fits(signal(yaw = 40f)))
    }

    @Test
    fun closerShotNeedsALargerFaceThanTheFirst() {
        val plan = EnrollPlan(listOf(Pose.STRAIGHT, Pose.CLOSER))
        plan.commit(signal(width = 0.4f))
        assertFalse(plan.fits(signal(width = 0.42f)))
        assertTrue(plan.fits(signal(width = 0.5f)))
    }

    @Test
    fun smileShotNeedsASmile() {
        val plan = EnrollPlan(listOf(Pose.SMILE))
        assertFalse(plan.fits(signal(smile = 0.2f)))
        assertTrue(plan.fits(signal(smile = 0.9f)))
    }

    @Test
    fun finishesAfterEveryStep() {
        val plan = EnrollPlan(listOf(Pose.STRAIGHT, Pose.FINAL))
        assertEquals(2, plan.total)
        plan.commit(signal())
        assertEquals(1, plan.done)
        assertFalse(plan.finished)
        plan.commit(signal())
        assertTrue(plan.finished)
        assertFalse(plan.fits(signal()))
    }
}
