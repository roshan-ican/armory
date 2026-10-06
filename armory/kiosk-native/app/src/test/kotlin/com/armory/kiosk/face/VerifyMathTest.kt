package com.armory.kiosk.face

import org.junit.Assert.assertEquals
import org.junit.Test

class VerifyMathTest {
    @Test
    fun averageIsUnitLength() {
        val avg = VerifyScanner.average(listOf(floatArrayOf(1f, 0f), floatArrayOf(0f, 1f)))
        assertEquals(1f, avg.map { it * it }.sum(), 1e-5f)
        assertEquals(avg[0], avg[1], 1e-5f)
    }

    @Test
    fun averageOfIdenticalVectorsIsThatVector() {
        val v = floatArrayOf(0.6f, 0.8f)
        val avg = VerifyScanner.average(listOf(v, v, v))
        assertEquals(0.6f, avg[0], 1e-5f)
        assertEquals(0.8f, avg[1], 1e-5f)
    }

    @Test
    fun gapIsEuclidean() {
        assertEquals(5f, VerifyScanner.gap(floatArrayOf(0f, 0f), floatArrayOf(3f, 4f)), 1e-5f)
    }
}
