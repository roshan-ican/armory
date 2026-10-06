package com.armory.kiosk.face

import android.content.Context
import android.graphics.Bitmap
import org.tensorflow.lite.DataType
import org.tensorflow.lite.Interpreter
import java.io.FileInputStream
import java.nio.ByteBuffer
import java.nio.ByteOrder
import java.nio.MappedByteBuffer
import java.nio.channels.FileChannel
import kotlin.math.sqrt

class Embedder(context: Context, asset: String = MODEL_ASSET) {
    private val interpreter: Interpreter
    private val width: Int
    private val height: Int
    private val outputSize: Int
    private val input: ByteBuffer
    private val pixels: IntArray

    init {
        interpreter = Interpreter(load(context, asset), Interpreter.Options().setNumThreads(4))
        val tensor = interpreter.getInputTensor(0)
        check(tensor.dataType() == DataType.FLOAT32) { "The face model must take float input." }
        height = tensor.shape()[1]
        width = tensor.shape()[2]
        outputSize = interpreter.getOutputTensor(0).shape().last()
        input = ByteBuffer.allocateDirect(4 * width * height * 3).order(ByteOrder.nativeOrder())
        pixels = IntArray(width * height)
    }

    val dimensions: Int get() = outputSize
    val inputSize: Int get() = width

    @Synchronized
    fun embed(face: Bitmap): FloatArray {
        val sized = if (face.width == width && face.height == height) face else Bitmap.createScaledBitmap(face, width, height, true)
        sized.getPixels(pixels, 0, width, 0, 0, width, height)
        input.rewind()
        for (p in pixels) {
            input.putFloat((((p shr 16) and 0xFF) - 127.5f) / 128f)
            input.putFloat((((p shr 8) and 0xFF) - 127.5f) / 128f)
            input.putFloat(((p and 0xFF) - 127.5f) / 128f)
        }
        input.rewind()
        val out = Array(1) { FloatArray(outputSize) }
        interpreter.run(input, out)
        return normalise(out[0])
    }

    fun close() = interpreter.close()

    private fun load(context: Context, asset: String): MappedByteBuffer {
        context.assets.openFd(asset).use { fd ->
            FileInputStream(fd.fileDescriptor).use { stream ->
                return stream.channel.map(FileChannel.MapMode.READ_ONLY, fd.startOffset, fd.declaredLength)
            }
        }
    }

    companion object {
        const val MODEL_ASSET = "mobilefacenet.tflite"

        fun hasModel(context: Context, asset: String = MODEL_ASSET): Boolean =
            runCatching { context.assets.open(asset).close() }.isSuccess

        fun normalise(v: FloatArray): FloatArray {
            var sum = 0f
            for (x in v) sum += x * x
            val norm = sqrt(sum)
            if (norm == 0f) return v
            return FloatArray(v.size) { v[it] / norm }
        }
    }
}
