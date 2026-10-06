package com.armory.kiosk.face

import org.junit.Assert.assertEquals
import org.junit.Test
import kotlin.math.cos
import kotlin.math.sin

class AlignTest {
    private fun apply(m: DoubleArray, p: DoubleArray): DoubleArray {
        val out = DoubleArray(p.size)
        for (i in 0 until p.size / 2) {
            out[2 * i] = m[0] * p[2 * i] - m[1] * p[2 * i + 1] + m[2]
            out[2 * i + 1] = m[1] * p[2 * i] + m[0] * p[2 * i + 1] + m[3]
        }
        return out
    }

    @Test
    fun recoversAKnownSimilarityTransform() {
        val scale = 1.7
        val angle = Math.toRadians(14.0)
        val moved = DoubleArray(Align.TEMPLATE.size)
        for (i in 0 until moved.size / 2) {
            val x = Align.TEMPLATE[2 * i]
            val y = Align.TEMPLATE[2 * i + 1]
            moved[2 * i] = scale * (cos(angle) * x - sin(angle) * y) + 130
            moved[2 * i + 1] = scale * (sin(angle) * x + cos(angle) * y) + 40
        }
        val back = apply(Align.similarity(moved, Align.TEMPLATE), moved)
        for (i in back.indices) assertEquals(Align.TEMPLATE[i], back[i], 1e-6)
    }

    @Test
    fun identityWhenAlreadyAligned() {
        val m = Align.similarity(Align.TEMPLATE, Align.TEMPLATE)
        assertEquals(1.0, m[0], 1e-9)
        assertEquals(0.0, m[1], 1e-9)
        assertEquals(0.0, m[2], 1e-9)
        assertEquals(0.0, m[3], 1e-9)
    }
}
