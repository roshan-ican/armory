package com.armory.kiosk.face

import android.graphics.Bitmap
import android.graphics.Matrix
import androidx.camera.core.ImageAnalysis
import androidx.camera.core.ImageProxy
import com.google.android.gms.tasks.Tasks
import com.google.mlkit.vision.common.InputImage
import com.google.mlkit.vision.face.Face
import com.google.mlkit.vision.face.FaceDetection
import com.google.mlkit.vision.face.FaceDetectorOptions

class Observation(val bitmap: Bitmap, val faces: List<Face>) {
    val face: Face? get() = faces.singleOrNull()
}

class FaceAnalyzer(private val onObservation: (Observation) -> Unit) : ImageAnalysis.Analyzer {
    private val detector = FaceDetection.getClient(
        FaceDetectorOptions.Builder()
            .setPerformanceMode(FaceDetectorOptions.PERFORMANCE_MODE_FAST)
            .setLandmarkMode(FaceDetectorOptions.LANDMARK_MODE_ALL)
            .setClassificationMode(FaceDetectorOptions.CLASSIFICATION_MODE_ALL)
            .setMinFaceSize(0.2f)
            .build()
    )
    private var last = 0L

    override fun analyze(image: ImageProxy) {
        val now = System.currentTimeMillis()
        if (now - last < MIN_INTERVAL_MS) {
            image.close()
            return
        }
        last = now
        try {
            val upright = upright(image)
            val faces = Tasks.await(detector.process(InputImage.fromBitmap(upright, 0)))
            onObservation(Observation(upright, faces))
        } catch (_: Exception) {
        } finally {
            image.close()
        }
    }

    fun close() = detector.close()

    private fun upright(image: ImageProxy): Bitmap {
        val raw = image.toBitmap()
        val degrees = image.imageInfo.rotationDegrees
        if (degrees == 0) return raw
        val matrix = Matrix().apply { postRotate(degrees.toFloat()) }
        return Bitmap.createBitmap(raw, 0, 0, raw.width, raw.height, matrix, true)
    }

    companion object {
        const val MIN_INTERVAL_MS = 90L
    }
}
