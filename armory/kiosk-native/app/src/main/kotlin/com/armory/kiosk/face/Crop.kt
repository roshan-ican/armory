package com.armory.kiosk.face

import android.graphics.Bitmap
import android.graphics.Canvas
import android.graphics.Matrix
import android.graphics.Paint
import com.google.mlkit.vision.face.Face
import com.google.mlkit.vision.face.FaceLandmark
import kotlin.math.max

object Crop {
    private const val MARGIN = 1.15f
    private val paint = Paint(Paint.FILTER_BITMAP_FLAG)

    fun aligned(src: Bitmap, face: Face, size: Int): Bitmap {
        val points = landmarkPoints(face)
        val matrix = if (points != null) landmarkMatrix(points, size) else boxMatrix(face, size)
        val out = Bitmap.createBitmap(size, size, Bitmap.Config.ARGB_8888)
        Canvas(out).drawBitmap(src, matrix, paint)
        return out
    }

    private fun landmarkPoints(face: Face): DoubleArray? {
        val eyes = listOf(FaceLandmark.LEFT_EYE, FaceLandmark.RIGHT_EYE).map { face.getLandmark(it)?.position }
        val mouth = listOf(FaceLandmark.MOUTH_LEFT, FaceLandmark.MOUTH_RIGHT).map { face.getLandmark(it)?.position }
        if (eyes.any { it == null } || mouth.any { it == null }) return null
        val (eyeA, eyeB) = eyes.filterNotNull().sortedBy { it.x }
        val (mouthA, mouthB) = mouth.filterNotNull().sortedBy { it.x }
        return doubleArrayOf(
            eyeA.x.toDouble(), eyeA.y.toDouble(), eyeB.x.toDouble(), eyeB.y.toDouble(),
            mouthA.x.toDouble(), mouthA.y.toDouble(), mouthB.x.toDouble(), mouthB.y.toDouble(),
        )
    }

    private fun landmarkMatrix(points: DoubleArray, size: Int): Matrix {
        val k = size / Align.TEMPLATE_SIZE
        val dst = DoubleArray(Align.TEMPLATE.size) { Align.TEMPLATE[it] * k }
        val (a, b, tx, ty) = Align.similarity(points, dst)
        return Matrix().apply {
            setValues(floatArrayOf(a.toFloat(), (-b).toFloat(), tx.toFloat(), b.toFloat(), a.toFloat(), ty.toFloat(), 0f, 0f, 1f))
        }
    }

    private fun boxMatrix(face: Face, size: Int): Matrix {
        val box = face.boundingBox
        val scale = size / (max(box.width(), box.height()) * MARGIN)
        return Matrix().apply {
            postTranslate(-box.exactCenterX(), -box.exactCenterY())
            postScale(scale, scale)
            postTranslate(size / 2f, size / 2f)
        }
    }

    fun signalOf(face: Face, imageWidth: Int): FaceSignal = FaceSignal(
        yaw = face.headEulerAngleY,
        pitch = face.headEulerAngleX,
        leftEye = face.leftEyeOpenProbability,
        rightEye = face.rightEyeOpenProbability,
        smile = face.smilingProbability,
        widthRatio = face.boundingBox.width().toFloat() / imageWidth,
    )
}
