package com.armory.kiosk.ui

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.armory.kiosk.BuildConfig
import com.armory.kiosk.data.Api
import com.armory.kiosk.data.ApiException
import com.armory.kiosk.data.GunRequest
import com.armory.kiosk.data.Locker
import com.armory.kiosk.data.Person
import com.armory.kiosk.face.Embedder
import com.armory.kiosk.face.EnrollScanner
import com.armory.kiosk.face.Observation
import com.armory.kiosk.face.ScanState
import com.armory.kiosk.face.VerifyScanner
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import okhttp3.sse.EventSource

enum class Step {
    FACE, WELCOME, CATALOG, GUNS, REVIEW, WAITING, OPEN, WRONG, COLLECTED, DECLINED, RETURNED,
    ENROLL_NAME, ENROLL_FACE, ENROLL_PENDING,
}

data class UiState(
    val step: Step = Step.FACE,
    val forward: Boolean = true,
    val person: Person? = null,
    val catalog: List<Locker> = emptyList(),
    val selectedId: Long? = null,
    val picked: List<Int> = emptyList(),
    val reason: String = "",
    val request: GunRequest? = null,
    val error: String = "",
    val notice: String = "",
    val busy: Boolean = false,
    val scan: ScanState = ScanState.Starting,
    val enrollName: String = "",
    val modelMissing: Boolean = false,
    val connected: Boolean = true,
) {
    val selected: Locker? get() = catalog.firstOrNull { it.id == selectedId }
}

class KioskViewModel(app: Application) : AndroidViewModel(app) {
    private val api = Api(BuildConfig.SERVER_URL)
    private val embedder: Embedder? = if (Embedder.hasModel(app)) Embedder(app) else null

    private val _state = MutableStateFlow(UiState(modelMissing = embedder == null))
    val state: StateFlow<UiState> = _state.asStateFlow()

    private var verify: VerifyScanner? = null
    private var enroll: EnrollScanner? = null
    private var follower: Job? = null
    private var stream: EventSource? = null
    private var streamJob: Job? = null
    private var lastChange = 0L

    init {
        beginFace()
        startStream()
    }

    private fun go(step: Step, forward: Boolean = true) =
        _state.update { it.copy(step = step, forward = forward, error = "", notice = "") }

    fun onObservation(obs: Observation) {
        when (_state.value.step) {
            Step.FACE -> verify?.handle(obs)
            Step.ENROLL_FACE -> enroll?.handle(obs)
            else -> Unit
        }
    }

    fun beginFace() {
        follower?.cancel()
        enroll = null
        viewModelScope.launch(Dispatchers.IO) { api.logout() }
        _state.update {
            UiState(modelMissing = embedder == null, connected = it.connected, forward = false)
        }
        val e = embedder ?: return
        verify = VerifyScanner(e) { embedding -> viewModelScope.launch { match(embedding) } }.also { scanner ->
            scanner.onState = { s -> _state.update { it.copy(scan = s) } }
            scanner.restart()
        }
    }

    private suspend fun match(embedding: FloatArray) {
        val person = try {
            withContext(Dispatchers.IO) { api.match(embedding) }
        } catch (e: ApiException) {
            _state.update { it.copy(notice = e.message.orEmpty()) }
            delay(1500)
            verify?.restart()
            return
        }
        if (person == null) {
            _state.update { it.copy(notice = "We don't recognise you yet.") }
            delay(1800)
            _state.update { it.copy(notice = "") }
            verify?.restart()
            return
        }
        _state.update { it.copy(person = person) }
        go(Step.WELCOME)
        try {
            val current = withContext(Dispatchers.IO) { api.currentRequest() }
            if (current != null) follow(current.id)
        } catch (e: ApiException) {
            _state.update { it.copy(error = e.message.orEmpty()) }
        }
    }

    fun loadCatalog() = viewModelScope.launch {
        _state.update { it.copy(busy = true) }
        try {
            val list = withContext(Dispatchers.IO) { api.catalog() }
            _state.update { it.copy(catalog = list) }
            go(Step.CATALOG)
        } catch (e: ApiException) {
            _state.update { it.copy(error = e.message.orEmpty()) }
        } finally {
            _state.update { it.copy(busy = false) }
        }
    }

    fun refreshNow() {
        if (browsing()) refreshCatalog()
    }

    private fun refreshCatalog() = viewModelScope.launch {
        val list = try {
            withContext(Dispatchers.IO) { api.catalog() }
        } catch (e: ApiException) {
            if (e.status == 401) beginFace() else _state.update { it.copy(connected = false) }
            return@launch
        }
        _state.update { s ->
            val fresh = list.firstOrNull { it.id == s.selectedId }
            val picked = if (fresh == null) s.picked else s.picked.filter { no -> fresh.slots.firstOrNull { it.no == no }?.available == true }
            s.copy(catalog = list, picked = picked, connected = true)
        }
    }

    private fun browsing() = _state.value.step in setOf(Step.CATALOG, Step.GUNS, Step.REVIEW)

    fun chooseLocker(locker: Locker) {
        _state.update { it.copy(selectedId = locker.id, picked = emptyList()) }
        go(Step.GUNS)
    }

    fun togglePick(no: Int) = _state.update {
        val next = if (no in it.picked) it.picked - no else (it.picked + no).sorted()
        it.copy(picked = next)
    }

    fun pickAll() = _state.update { s ->
        s.copy(picked = s.selected?.slots?.filter { it.available }?.map { it.no }.orEmpty())
    }

    fun back(to: Step) = go(to, forward = false)
    fun next(to: Step) = go(to)
    fun setReason(text: String) = _state.update { it.copy(reason = text.take(120)) }

    fun sendRequest() = viewModelScope.launch {
        val s = _state.value
        val locker = s.selected ?: return@launch
        _state.update { it.copy(busy = true, error = "") }
        try {
            val created = withContext(Dispatchers.IO) { api.createRequest(locker.id, s.picked, s.reason) }
            _state.update { it.copy(request = created) }
            follow(created.id)
        } catch (e: ApiException) {
            _state.update { it.copy(error = e.message.orEmpty()) }
        } finally {
            _state.update { it.copy(busy = false) }
        }
    }

    private fun follow(id: Long) {
        follower?.cancel()
        follower = viewModelScope.launch {
            while (isActive) {
                try {
                    val value = withContext(Dispatchers.IO) { api.request(id) }
                    if (apply(value)) return@launch
                } catch (e: ApiException) {
                    if (e.status == 401) {
                        beginFace()
                        return@launch
                    }
                }
                delay(1000)
            }
        }
    }

    private fun apply(value: GunRequest): Boolean {
        _state.update { it.copy(request = value) }
        val step = when (value.status) {
            "pending" -> Step.WAITING
            "approved" -> if (value.wrong.isNotEmpty()) Step.WRONG else Step.OPEN
            "collected" -> Step.COLLECTED
            "returned" -> Step.RETURNED
            "rejected" -> Step.DECLINED
            else -> null
        }
        if (step != null && step != _state.value.step) go(step)
        return value.status == "returned" || value.status == "rejected"
    }

    fun cancelRequest() = viewModelScope.launch {
        _state.value.request?.let { r -> withContext(Dispatchers.IO) { api.cancel(r.id) } }
        beginFace()
    }

    fun startEnrollment() {
        enroll = null
        _state.update { it.copy(enrollName = "") }
        go(Step.ENROLL_NAME)
    }

    fun setEnrollName(v: String) = _state.update { it.copy(enrollName = v) }

    fun prepareEnrollCamera() {
        go(Step.ENROLL_FACE)
        val e = embedder ?: return
        enroll = EnrollScanner(e) { shots -> viewModelScope.launch { submitEnrollment(shots) } }.also { scanner ->
            scanner.onState = { s -> _state.update { it.copy(scan = s) } }
            scanner.restart()
        }
    }

    private suspend fun submitEnrollment(shots: List<FloatArray>) {
        val s = _state.value
        _state.update { it.copy(busy = true) }
        try {
            withContext(Dispatchers.IO) { api.enroll(s.enrollName, shots) }
            go(Step.ENROLL_PENDING)
        } catch (e: ApiException) {
            _state.update { it.copy(error = e.message.orEmpty()) }
            delay(2500)
            enroll?.restart()
        } finally {
            _state.update { it.copy(busy = false) }
        }
    }

    private fun startStream() {
        streamJob = viewModelScope.launch {
            while (isActive) {
                val closed = kotlinx.coroutines.CompletableDeferred<Unit>()
                stream = api.watch(
                    onOpen = {
                        _state.update { it.copy(connected = true) }
                        refreshNow()
                    },
                    onChange = {
                        val now = System.currentTimeMillis()
                        if (now - lastChange > 150) {
                            lastChange = now
                            if (browsing()) refreshCatalog()
                        }
                        _state.update { it.copy(connected = true) }
                    },
                    onDrop = {
                        _state.update { it.copy(connected = false) }
                        closed.complete(Unit)
                    },
                )
                closed.await()
                stream?.cancel()
                delay(2000)
            }
        }
        viewModelScope.launch {
            while (isActive) {
                delay(1000)
                if (browsing()) refreshCatalog()
            }
        }
    }

    override fun onCleared() {
        stream?.cancel()
        streamJob?.cancel()
        embedder?.close()
    }

}
