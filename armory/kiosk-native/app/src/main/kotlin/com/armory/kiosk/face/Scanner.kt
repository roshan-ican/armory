package com.armory.kiosk.face

sealed interface ScanState {
    data object Starting : ScanState
    data object NoFace : ScanState
    data object ManyFaces : ScanState
    data class Adjust(val hint: String) : ScanState
    data class Prompt(val kind: ChallengeKind, val name: String = "") : ScanState
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
    private val onIdentify: (FloatArray) -> Unit,
    private val onEmbedding: (FloatArray) -> Unit,
) : Scanner(embedder) {
    private enum class Stage { COLLECT, IDENTIFYING, CHALLENGE, CONFIRM, DONE }

    @Volatile
    private var stage = Stage.COLLECT
    private var challenge: Challenge? = null
    private var identity: FloatArray? = null
    private var who = ""
    private var passedAt = 0L
    private val samples = mutableListOf<FloatArray>()
    private var lastSampleAt = 0L

    fun restart() {
        stage = Stage.COLLECT
        challenge = null
        identity = null
        who = ""
        passedAt = 0
        samples.clear()
        lastSampleAt = 0
        publish(ScanState.NoFace)
    }

    fun identified(name: String) {
        who = name
        challenge = null
        stage = Stage.CHALLENGE
    }

    override fun handle(obs: Observation) {
        if (stage == Stage.IDENTIFYING || stage == Stage.DONE) return
        if (obs.faces.isEmpty()) {
            restart()
            return
        }
        val face = obs.face
        if (face == null) {
            restart()
            publish(ScanState.ManyFaces)
            return
        }
        val signal = Crop.signalOf(face, obs.bitmap.width)
        when (stage) {
            Stage.COLLECT -> collect(obs, signal)
            Stage.CHALLENGE -> runChallenge(signal)
            Stage.CONFIRM -> confirm(obs, signal)
            else -> Unit
        }
    }

    private fun collect(obs: Observation, signal: FaceSignal) {
        val problem = hint(signal) ?: if (signal.smile != null && !signal.neutral) "Relax your face." else null
        if (problem != null) {
            samples.clear()
            publish(ScanState.Adjust(problem))
            return
        }
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
        if (samples.size >= IDENT_SAMPLES) {
            val avg = average(samples)
            identity = avg
            stage = Stage.IDENTIFYING
            onIdentify(avg)
        }
    }

    private fun runChallenge(signal: FaceSignal) {
        var active = challenge
        if (active == null) {
            active = Challenge.any()
            challenge = active
        }
        publish(ScanState.Prompt(active.kind, who))
        when (active.update(signal)) {
            Progress.PASSED -> {
                passedAt = System.currentTimeMillis()
                stage = Stage.CONFIRM
                publish(ScanState.Checking)
            }
            Progress.EXPIRED -> restart()
            Progress.PENDING -> Unit
        }
    }

    private fun confirm(obs: Observation, signal: FaceSignal) {
        if (signal.widthRatio >= FaceSignal.MIN_WIDTH && signal.frontal) {
            val known = identity ?: return restart()
            if (gap(known, embed(obs)) > SAME_PERSON_MAX) {
                restart()
                return
            }
            stage = Stage.DONE
            onEmbedding(known)
        } else if (System.currentTimeMillis() - passedAt > CONFIRM_MS) {
            restart()
        }
    }

    companion object {
        const val CONFIRM_MS = 2000L
        const val IDENT_SAMPLES = 2
        const val SAMPLE_GAP_MS = 100L
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
