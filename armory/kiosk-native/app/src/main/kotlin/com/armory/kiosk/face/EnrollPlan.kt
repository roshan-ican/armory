package com.armory.kiosk.face

import kotlin.math.abs
import kotlin.math.sign

enum class Pose(val prompt: String) {
    STRAIGHT("Look straight at the camera."),
    TURN_FIRST("Turn your head a little to one side."),
    TURN_SECOND("Now turn a little to the other side."),
    CLOSER("Move a little closer."),
    SMILE("Give a small smile."),
    FINAL("Relax and look straight again."),
}

class EnrollPlan(private val steps: List<Pose> = DEFAULT) {
    private var index = 0
    private var turnSign = 0f
    private var baseWidth = 0f

    val total: Int get() = steps.size
    val done: Int get() = index
    val finished: Boolean get() = index >= steps.size
    val current: Pose get() = steps[index]

    fun fits(s: FaceSignal): Boolean {
        if (finished || !s.eyesOpen || s.widthRatio < FaceSignal.MIN_WIDTH) return false
        return when (current) {
            Pose.STRAIGHT, Pose.FINAL -> s.frontal
            Pose.TURN_FIRST -> abs(s.yaw) in TURN_MIN..TURN_MAX
            Pose.TURN_SECOND -> abs(s.yaw) in TURN_MIN..TURN_MAX && sign(s.yaw) != turnSign
            Pose.CLOSER -> s.frontal && s.widthRatio >= (baseWidth * CLOSER_RATIO).coerceAtMost(CLOSER_CAP)
            Pose.SMILE -> s.frontal && (s.smile ?: 1f) >= SMILING
        }
    }

    fun commit(s: FaceSignal) {
        if (finished) return
        when (current) {
            Pose.STRAIGHT -> baseWidth = s.widthRatio
            Pose.TURN_FIRST -> turnSign = sign(s.yaw)
            else -> Unit
        }
        index++
    }

    companion object {
        val DEFAULT = listOf(Pose.STRAIGHT, Pose.TURN_FIRST, Pose.TURN_SECOND, Pose.CLOSER, Pose.SMILE, Pose.FINAL)
        const val TURN_MIN = 8f
        const val TURN_MAX = 22f
        const val CLOSER_RATIO = 1.2f
        const val CLOSER_CAP = 0.85f
        const val SMILING = 0.6f
    }
}
