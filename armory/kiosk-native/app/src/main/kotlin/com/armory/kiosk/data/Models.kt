package com.armory.kiosk.data

data class Slot(val no: Int, val reading: Int, val takenBy: String, val available: Boolean)

data class Locker(
    val id: Long,
    val name: String,
    val location: String,
    val kind: String,
    val online: Boolean,
    val available: Int,
    val slots: List<Slot>,
)

data class Chosen(val no: Int, val status: String)

data class SlotReading(val no: Int, val reading: Int)

data class GunRequest(
    val id: Long,
    val status: String,
    val kind: String,
    val lockerName: String,
    val slotNo: Int,
    val chosen: List<Chosen>,
    val slots: List<SlotReading>,
    val wrong: List<Int>,
) {
    fun chosenNos(status: String? = null): List<Int> =
        chosen.filter { status == null || it.status == status }.map { it.no }
}

data class Person(val name: String)

class ApiException(val status: Int, message: String) : Exception(message)
