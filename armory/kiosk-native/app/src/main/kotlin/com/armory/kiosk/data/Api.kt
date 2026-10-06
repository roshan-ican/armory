package com.armory.kiosk.data

import okhttp3.Cookie
import okhttp3.CookieJar
import okhttp3.HttpUrl
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.Response
import okhttp3.sse.EventSource
import okhttp3.sse.EventSourceListener
import okhttp3.sse.EventSources
import org.json.JSONArray
import org.json.JSONObject
import java.io.IOException
import java.util.concurrent.TimeUnit

class Api(private val baseUrl: String) {
    private val jar = object : CookieJar {
        private var cookies = emptyList<Cookie>()

        @Synchronized
        override fun saveFromResponse(url: HttpUrl, cookies: List<Cookie>) {
            val names = cookies.map { it.name }.toSet()
            this.cookies = this.cookies.filter { it.name !in names } + cookies.filter { it.expiresAt > System.currentTimeMillis() }
        }

        @Synchronized
        override fun loadForRequest(url: HttpUrl): List<Cookie> = cookies.filter { it.matches(url) }
    }

    private val client = OkHttpClient.Builder()
        .cookieJar(jar)
        .connectTimeout(6, TimeUnit.SECONDS)
        .readTimeout(10, TimeUnit.SECONDS)
        .build()

    private val streamClient = client.newBuilder()
        .readTimeout(0, TimeUnit.MILLISECONDS)
        .build()

    private val json = "application/json".toMediaType()

    private fun call(method: String, path: String, body: JSONObject? = null): String {
        val builder = Request.Builder().url(baseUrl + path)
        if (method == "POST") builder.post((body?.toString() ?: "{}").toRequestBody(json)) else builder.get()
        try {
            client.newCall(builder.build()).execute().use { res -> return read(res) }
        } catch (e: IOException) {
            throw ApiException(0, "Cannot reach the server.")
        }
    }

    private fun read(res: Response): String {
        val text = res.body?.string().orEmpty()
        if (res.isSuccessful) return text
        val message = runCatching { JSONObject(text).getString("error") }.getOrNull()
            ?: text.trim().ifEmpty { "Something went wrong." }
        throw ApiException(res.code, message)
    }

    fun match(embedding: FloatArray): Person? {
        val out = JSONObject(call("POST", "/face/match", JSONObject().put("descriptor", embedding.toJson())))
        if (!out.optBoolean("matched")) return null
        return Person(out.getString("name"))
    }

    fun logout() {
        runCatching { call("POST", "/api/logout") }
    }

    fun catalog(): List<Locker> {
        val arr = JSONArray(call("GET", "/api/catalog"))
        return List(arr.length()) { lockerOf(arr.getJSONObject(it)) }
    }

    fun createRequest(lockerId: Long, slots: List<Int>, reason: String): GunRequest {
        val body = JSONObject()
            .put("locker_id", lockerId)
            .put("slots", JSONArray(slots))
            .put("reason", reason)
        return requestOf(JSONObject(call("POST", "/api/requests", body)))
    }

    fun currentRequest(): GunRequest? = try {
        requestOf(JSONObject(call("GET", "/api/requests/current")))
    } catch (e: ApiException) {
        if (e.status == 404) null else throw e
    }

    fun request(id: Long): GunRequest = requestOf(JSONObject(call("GET", "/api/requests/$id")))

    fun cancel(id: Long) {
        runCatching { call("POST", "/api/requests/$id/cancel") }
    }

    fun enroll(name: String, embeddings: List<FloatArray>) {
        val body = JSONObject()
            .put("name", name)
            .put("descriptors", JSONArray().also { arr -> embeddings.forEach { arr.put(it.toJson()) } })
        call("POST", "/enroll", body)
    }

    fun watch(onOpen: () -> Unit, onChange: () -> Unit, onDrop: () -> Unit): EventSource {
        val req = Request.Builder().url("$baseUrl/events").header("Accept", "text/event-stream").build()
        val listener = object : EventSourceListener() {
            override fun onOpen(eventSource: EventSource, response: Response) = onOpen()
            override fun onEvent(eventSource: EventSource, id: String?, type: String?, data: String) = onChange()
            override fun onFailure(eventSource: EventSource, t: Throwable?, response: Response?) = onDrop()
            override fun onClosed(eventSource: EventSource) = onDrop()
        }
        return EventSources.createFactory(streamClient).newEventSource(req, listener)
    }

    private fun FloatArray.toJson() = JSONArray().also { arr -> forEach { arr.put(it.toDouble()) } }

    private fun lockerOf(o: JSONObject): Locker {
        val slots = o.optJSONArray("slots") ?: JSONArray()
        return Locker(
            id = o.getLong("id"),
            name = o.getString("name"),
            location = o.optString("location"),
            kind = o.optString("kind"),
            online = o.optBoolean("online"),
            available = o.optInt("available"),
            slots = List(slots.length()) {
                val s = slots.getJSONObject(it)
                Slot(s.getInt("no"), s.optInt("reading", -1), s.optString("taken_by"), s.optBoolean("available"))
            },
        )
    }

    private fun requestOf(o: JSONObject): GunRequest {
        val chosen = o.optJSONArray("chosen") ?: JSONArray()
        val slots = o.optJSONArray("slots") ?: JSONArray()
        val wrong = o.optJSONArray("wrong") ?: JSONArray()
        return GunRequest(
            id = o.getLong("id"),
            status = o.getString("status"),
            kind = o.optString("kind"),
            lockerName = o.optString("locker_name"),
            slotNo = o.optInt("slot_no"),
            chosen = List(chosen.length()) { chosen.getJSONObject(it).let { c -> Chosen(c.getInt("no"), c.optString("status")) } },
            slots = List(slots.length()) { slots.getJSONObject(it).let { s -> SlotReading(s.getInt("no"), s.optInt("reading", -1)) } },
            wrong = List(wrong.length()) { wrong.getInt(it) },
        )
    }
}
