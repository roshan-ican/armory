package com.armory.kiosk.face

sealed interface ScanState {
    data object Starting : ScanState
    data object NoFace : ScanState
    data object ManyFaces : ScanState
    data class Adjust(val hint: String) : ScanState
    data class Prompt(val kind: ChallengeKind) : ScanState
    data object Settle : ScanState
    data object Checking : ScanState
    data class Guide(val prompt: String, val done: Int, val total: Int) : ScanState
    data class Captured(val done: Int, val total: Int) : ScanState
}

abstract class Scanner(private val embedder: Embedder) {
    @Volatile
    var state: ScanState = ScanState.Starting
        protected set
    var onState: (ScanState) -> Unit = {}

    abstract fun handle(obs: Observation)

    protected fun publish(next: ScanState) {
        if (next != state) {
            state = next
            onState(next)
        }
    }

    protected fun embed(obs: Observation): FloatArray {
        val face = obs.face!!
        return embedder.embed(Crop.aligned(obs.bitmap, face, embedder.inputSize))
    }

    protected fun hint(signal: FaceSignal): String? = when {
        signal.widthRatio < FaceSignal.MIN_WIDTH -> "Move closer to the camera."
        !signal.frontal -> "Look straight at the camera."
        !signal.eyesOpen -> "Keep your eyes open."
        else -> null
    }
}

class VerifyScanner(
    embedder: Embedder,
    private val onEmbedding: (FloatArray) -> Unit,
) : Scanner(embedder) {
    private var challenge: Challenge? = null
    private var passedAt = 0L
    private val samples = mutableListOf<FloatArray>()
    private var lastSampleAt = 0L

    @Volatile
    private var waiting = false

    fun restart() {
        challenge = null
        passedAt = 0
        samples.clear()
        lastSampleAt = 0
        waiting = false
        publish(ScanState.NoFace)
    }

    override fun handle(obs: Observation) {
        if (waiting) return
        if (obs.faces.isEmpty()) {
            challenge = null
            passedAt = 0
            publish(ScanState.NoFace)
            return
        }
        val face = obs.face
        if (face == null) {
            challenge = null
            passedAt = 0
            publish(ScanState.ManyFaces)
            return
        }
        val signal = Crop.signalOf(face, obs.bitmap.width)

        if (passedAt != 0L) {
            if (signal.usable && (signal.smile == null || signal.neutral)) {
                val now = System.currentTimeMillis()
                if (now - lastSampleAt < SAMPLE_GAP_MS) return
                lastSampleAt = now
                publish(ScanState.Checking)
                val vec = embed(obs)
                if (samples.isNotEmpty() && gap(samples.first(), vec) > SAME_PERSON_MAX) {
                    restart()
                    return
                }
                samples += vec
                if (samples.size >= SAMPLES) {
                    waiting = true
                    onEmbedding(average(samples))
                }
            } else if (System.currentTimeMillis() - passedAt > SETTLE_MS) {
                restart()
            } else {
                publish(ScanState.Settle)
            }
            return
        }

        if (challenge == null) {
            val problem = hint(signal)
            if (problem != null) {
                publish(ScanState.Adjust(problem))
                return
            }
            challenge = Challenge.random()
        }
        val active = challenge!!
        publish(ScanState.Prompt(active.kind))
        when (active.update(signal)) {
            Progress.PASSED -> {
                passedAt = System.currentTimeMillis()
                publish(ScanState.Settle)
            }
            Progress.EXPIRED -> restart()
            Progress.PENDING -> Unit
        }
    }

    companion object {
        const val SETTLE_MS = 3000L
        const val SAMPLES = 3
        const val SAMPLE_GAP_MS = 120L
        const val SAME_PERSON_MAX = 0.8f

        fun average(vectors: List<FloatArray>): FloatArray {
            val sum = FloatArray(vectors.first().size)
            for (v in vectors) for (i in sum.indices) sum[i] += v[i]
            return Embedder.normalise(FloatArray(sum.size) { sum[it] / vectors.size })
        }

        fun gap(a: FloatArray, b: FloatArray): Float {
            var total = 0f
            for (i in a.indices) {
                val d = a[i] - b[i]
                total += d * d
            }
            return kotlin.math.sqrt(total)
        }
    }
}

class EnrollScanner(
    embedder: Embedder,
    private val onDone: (List<FloatArray>) -> Unit,
) : Scanner(embedder) {
    private val taken = mutableListOf<FloatArray>()
    private var plan = EnrollPlan()
    private var streak = 0
    private var lastAt = 0L
    private var finished = false

    fun restart() {
        taken.clear()
        plan = EnrollPlan()
        streak = 0
        lastAt = 0
        finished = false
        publish(ScanState.NoFace)
    }

    override fun handle(obs: Observation) {
        if (finished) return
        if (obs.faces.isEmpty()) {
            streak = 0
            publish(ScanState.NoFace)
            return
        }
        val face = obs.face
        if (face == null) {
            streak = 0
            publish(ScanState.ManyFaces)
            return
        }
        val signal = Crop.signalOf(face, obs.bitmap.width)
        if (!plan.fits(signal)) {
            streak = 0
            val problem = when {
                signal.widthRatio < FaceSignal.MIN_WIDTH -> "Move closer to the camera."
                !signal.eyesOpen -> "Keep your eyes open."
                else -> null
            }
            publish(if (problem != null) ScanState.Adjust(problem) else guide())
            return
        }
        publish(guide())
        streak++
        val now = System.currentTimeMillis()
        if (streak < HOLD_FRAMES || now - lastAt < GAP_MS) return
        lastAt = now
        streak = 0
        taken += embed(obs)
        plan.commit(signal)
        if (plan.finished) {
            finished = true
            publish(ScanState.Captured(taken.size, plan.total))
            onDone(taken.toList())
        } else {
            publish(guide())
        }
    }

    private fun guide() = ScanState.Guide(plan.current.prompt, plan.done, plan.total)

    companion object {
        const val GAP_MS = 900L
        const val HOLD_FRAMES = 2
    }
}
