package com.armory.kiosk.face

object Align {
    const val TEMPLATE_SIZE = 112.0

    val TEMPLATE = doubleArrayOf(
        38.2946, 51.6963,
        73.5318, 51.5014,
        41.5493, 92.3655,
        70.7299, 92.2041,
    )

    fun similarity(src: DoubleArray, dst: DoubleArray): DoubleArray {
        val n = src.size / 2
        var msx = 0.0
        var msy = 0.0
        var mdx = 0.0
        var mdy = 0.0
        for (i in 0 until n) {
            msx += src[2 * i]
            msy += src[2 * i + 1]
            mdx += dst[2 * i]
            mdy += dst[2 * i + 1]
        }
        msx /= n
        msy /= n
        mdx /= n
        mdy /= n

        var re = 0.0
        var im = 0.0
        var norm = 0.0
        for (i in 0 until n) {
            val sx = src[2 * i] - msx
            val sy = src[2 * i + 1] - msy
            val dx = dst[2 * i] - mdx
            val dy = dst[2 * i + 1] - mdy
            re += sx * dx + sy * dy
            im += sx * dy - sy * dx
            norm += sx * sx + sy * sy
        }
        val a = re / norm
        val b = im / norm
        val tx = mdx - (a * msx - b * msy)
        val ty = mdy - (b * msx + a * msy)
        return doubleArrayOf(a, b, tx, ty)
    }
}
