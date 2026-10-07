package com.armory.kiosk.data

import java.net.ServerSocket
import java.util.concurrent.CopyOnWriteArrayList
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

class ApiWatchTest {
    private lateinit var socket: ServerSocket
    private val requested = CopyOnWriteArrayList<String>()

    @Before
    fun start() {
        socket = ServerSocket(0, 1, java.net.InetAddress.getByName("127.0.0.1"))
        Thread {
            runCatching {
                socket.accept().use { conn ->
                    val reader = conn.getInputStream().bufferedReader()
                    requested.add(reader.readLine().split(" ")[1])
                    while (reader.readLine().orEmpty().isNotEmpty()) {
                    }
                    val out = conn.getOutputStream()
                    out.write("HTTP/1.1 200 OK\r\nContent-Type: text/event-stream\r\nConnection: close\r\n\r\n".toByteArray())
                    out.write(": connected\n\n".toByteArray())
                    out.write("data: changed\n\n".toByteArray())
                    out.write("event: say\ndata: Roshan, you picked the gun from slot 2.\n\n".toByteArray())
                    out.write("data: changed\n\n".toByteArray())
                    out.flush()
                    Thread.sleep(1500)
                }
            }
        }.apply { isDaemon = true }.start()
    }

    @After
    fun stop() = socket.close()

    @Test
    fun sayEventsGoToSpeechAndOthersToChange() {
        val said = CopyOnWriteArrayList<String>()
        val changes = CopyOnWriteArrayList<Int>()
        val done = CountDownLatch(1)
        val api = Api("http://127.0.0.1:${socket.localPort}")

        val stream = api.watch(
            onOpen = {},
            onChange = { changes.add(1); if (changes.size == 2) done.countDown() },
            onSay = { said.add(it) },
            onDrop = {},
        )

        assertTrue("stream events did not arrive", done.await(5, TimeUnit.SECONDS))
        stream.cancel()
        assertEquals(listOf("Roshan, you picked the gun from slot 2."), said.toList())
        assertEquals(2, changes.size)
        assertEquals(listOf("/events?speech=1"), requested.toList())
    }
}
