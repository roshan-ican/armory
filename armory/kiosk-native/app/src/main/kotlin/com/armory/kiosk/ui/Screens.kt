package com.armory.kiosk.ui

import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.armory.kiosk.R
import com.armory.kiosk.data.GunRequest
import com.armory.kiosk.data.Locker
import com.armory.kiosk.data.Slot
import com.armory.kiosk.face.ScanState
import java.util.Calendar

private val titles = mapOf(
    Step.FACE to "Face verification", Step.WELCOME to "Welcome", Step.CATALOG to "Choose a locker",
    Step.GUNS to "Choose guns", Step.REVIEW to "Review", Step.WAITING to "Approval",
    Step.OPEN to "Locker access", Step.WRONG to "Wrong slot", Step.COLLECTED to "Collected",
    Step.DECLINED to "Declined", Step.RETURNED to "Complete", Step.ENROLL_NAME to "Your details",
    Step.ENROLL_FACE to "Face capture", Step.ENROLL_PENDING to "Request sent",
)

@Composable
fun KioskApp(model: KioskViewModel, cameraGranted: Boolean) {
    val state by model.state.collectAsStateWithLifecycle()
    ArmoryTheme {
        Column(Modifier.fillMaxSize().background(Palette.Bg)) {
            TopBar(titles[state.step].orEmpty(), state.person?.name, state.connected)
            AnimatedContent(
                targetState = state.step,
                modifier = Modifier.weight(1f).fillMaxWidth(),
                transitionSpec = {
                    val dir = if (state.forward) 1 else -1
                    (slideInHorizontally { it / 8 * dir } + fadeIn()) togetherWith (slideOutHorizontally { -it / 8 * dir } + fadeOut())
                },
                label = "step",
            ) { step ->
                Box(Modifier.fillMaxSize().padding(horizontal = 24.dp, vertical = 12.dp), contentAlignment = Alignment.Center) {
                    when (step) {
                        Step.FACE -> FaceStep(model, state, cameraGranted)
                        Step.WELCOME -> WelcomeStep(model, state)
                        Step.CATALOG -> CatalogStep(model, state)
                        Step.GUNS -> GunsStep(model, state)
                        Step.REVIEW -> ReviewStep(model, state)
                        Step.WAITING -> WaitingStep(model, state)
                        Step.OPEN, Step.WRONG -> OpenStep(state, step == Step.WRONG)
                        Step.COLLECTED -> CollectedStep(model, state)
                        Step.DECLINED -> Simple("×", Palette.Red, "NOT APPROVED", "Request declined", "Ask an administrator if you need help.", "Start over", model::beginFace)
                        Step.RETURNED -> Simple("✓", Palette.Green, "COMPLETE", "Returned safely", "The slot is locked again. Thank you.", "Done", model::beginFace)
                        Step.ENROLL_NAME -> EnrollNameStep(model, state)
                        Step.ENROLL_FACE -> EnrollFaceStep(model, state, cameraGranted)
                        Step.ENROLL_PENDING -> Simple("✓", Palette.Green, "SUBMITTED", "Request sent", "An admin must approve your enrollment before your face can be used.", "Return to face scan", model::beginFace)
                    }
                }
            }
            if (state.step == Step.GUNS) GunsBar(model, state) else Footer(titles[state.step].orEmpty())
        }
    }
}

@Composable
private fun TopBar(context: String, name: String?, connected: Boolean) {
    Row(
        Modifier.fillMaxWidth().height(60.dp).padding(horizontal = 32.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Row(Modifier.weight(1f), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            Image(painterResource(R.drawable.logo_mark), contentDescription = "Armory", modifier = Modifier.height(30.dp))
            Text("Armory", color = Palette.Text, fontWeight = FontWeight.SemiBold, fontSize = 17.sp)
            if (!connected) Text("Offline", color = Palette.Yellow, fontSize = 12.sp)
        }
        Text(context, color = Palette.Muted, fontSize = 13.sp)
        Box(Modifier.weight(1f), contentAlignment = Alignment.CenterEnd) {
            if (name != null) Text(name, color = Color(0xFFD4D4D8), fontSize = 14.sp)
        }
    }
}

@Composable
private fun Footer(step: String) {
    Row(Modifier.fillMaxWidth().height(34.dp), horizontalArrangement = Arrangement.Center, verticalAlignment = Alignment.CenterVertically) {
        Text("Secure local system", color = Palette.Dim, fontSize = 12.sp)
        Box(Modifier.padding(horizontal = 10.dp).size(4.dp).clip(CircleShape).background(Palette.Dim))
        Text(step, color = Palette.Dim, fontSize = 12.sp)
    }
}

@Composable
private fun Screen(content: @Composable () -> Unit) {
    BoxWithConstraints(Modifier.fillMaxSize(), contentAlignment = Alignment.TopCenter) {
        Column(
            Modifier.widthIn(max = 720.dp).fillMaxWidth().heightIn(min = maxHeight).verticalScroll(rememberScrollState()).padding(vertical = 12.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(20.dp, Alignment.CenterVertically),
        ) { content() }
    }
}

@Composable
private fun ErrorLine(text: String) {
    if (text.isNotEmpty()) {
        Text(
            text, color = Color(0xFFFF918A), textAlign = TextAlign.Center,
            modifier = Modifier.clip(RoundedCornerShape(14.dp)).background(Color(0x1FFF453A)).padding(horizontal = 16.dp, vertical = 12.dp),
        )
    }
}

@Composable
private fun CameraBox(model: KioskViewModel, granted: Boolean, heightDp: Int, missing: String = "Allow camera access to continue.") {
    Box(
        Modifier.height(heightDp.dp).aspectRatio(3f / 4f).clip(RoundedCornerShape(34.dp)).background(Color(0xFF1C1C1E)),
        contentAlignment = Alignment.Center,
    ) {
        if (granted) {
            FaceCamera(Modifier.fillMaxSize(), model::onObservation)
            Box(
                Modifier.fillMaxSize().padding(horizontal = 40.dp, vertical = 36.dp)
                    .border(2.dp, Color(0xE0FFFFFF), RoundedCornerShape(50)),
            )
        } else {
            Text(missing, color = Palette.Muted, textAlign = TextAlign.Center, modifier = Modifier.padding(24.dp))
        }
    }
}

private fun scanMessage(scan: ScanState, notice: String, modelMissing: Boolean): String {
    if (modelMissing) return "The face model is missing from this build."
    if (notice.isNotEmpty()) return notice
    return when (scan) {
        ScanState.Starting -> "Preparing the camera…"
        ScanState.NoFace -> "Look at the camera."
        ScanState.ManyFaces -> "Only one person at a time, please."
        is ScanState.Adjust -> scan.hint
        is ScanState.Prompt -> scan.kind.prompt
        ScanState.Settle -> "Now look straight ahead."
        ScanState.Checking -> "Checking…"
        is ScanState.Guide -> "${scan.prompt} (${scan.done + 1} of ${scan.total})"
        is ScanState.Captured -> "Captured ${scan.done} of ${scan.total}"
    }
}

@Composable
private fun FaceStep(model: KioskViewModel, state: UiState, granted: Boolean) {
    Row(Modifier.fillMaxSize(), horizontalArrangement = Arrangement.spacedBy(48.dp, Alignment.CenterHorizontally), verticalAlignment = Alignment.CenterVertically) {
        CameraBox(model, granted && !state.modelMissing, 420, if (granted) "Camera paused: no face model installed." else "Allow camera access to continue.")
        Column(Modifier.width(420.dp), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(20.dp)) {
            Copy("IDENTITY CHECK", "Look at the camera", "")
            val prompt = state.scan is ScanState.Prompt
            Text(
                scanMessage(state.scan, state.notice, state.modelMissing),
                color = if (prompt) Palette.Text else Palette.Muted,
                fontSize = if (prompt) 24.sp else 18.sp,
                fontWeight = if (prompt) FontWeight.SemiBold else FontWeight.Normal,
                textAlign = TextAlign.Center,
            )
            SecondaryButton("Enroll yourself", model::startEnrollment)
        }
    }
}

@Composable
private fun WelcomeStep(model: KioskViewModel, state: UiState) {
    val hour = Calendar.getInstance().get(Calendar.HOUR_OF_DAY)
    val local = if (hour < 12) "morning" else if (hour < 18) "afternoon" else "evening"
    val greeting = state.person?.greeting.orEmpty().ifEmpty { "Good $local" }
    Screen {
        Symbol("✓", Palette.Green)
        Copy("VERIFIED", "$greeting, ${state.person?.name?.substringBefore(' ').orEmpty()}", "Your identity has been confirmed.")
        PrimaryButton("Continue  →", enabled = !state.busy, onClick = { model.loadCatalog() })
        ErrorLine(state.error)
    }
}

fun toneOf(locker: Locker, slot: Slot): Color = when {
    !locker.online || slot.reading == 2 -> Palette.Yellow
    slot.reading == 1 -> Palette.Green
    slot.reading == 0 -> Palette.Red
    else -> Palette.Grey
}

@Composable
private fun CatalogStep(model: KioskViewModel, state: UiState) {
    Column(Modifier.fillMaxSize(), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(10.dp, Alignment.CenterVertically)) {
        Copy("STEP 1 OF 3", "Choose a locker", "")
        LazyRow(
            Modifier.weight(1f, fill = false),
            horizontalArrangement = Arrangement.spacedBy(14.dp, Alignment.CenterHorizontally),
            contentPadding = PaddingValues(horizontal = 4.dp),
        ) {
            items(state.catalog, key = { it.id }) { locker ->
                Card(Modifier.width(400.dp), onClick = if (locker.available > 0) ({ model.chooseLocker(locker) }) else null) {
                    Column(Modifier.padding(18.dp), verticalArrangement = Arrangement.spacedBy(14.dp)) {
                        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                            Column(Modifier.weight(1f)) {
                                Text(locker.name, color = Color.White, fontWeight = FontWeight.SemiBold, fontSize = 17.sp)
                                Text(locker.location.ifEmpty { locker.kind }, color = Palette.Muted, fontSize = 13.sp)
                            }
                            StatePill(locker.online)
                            Text(
                                if (locker.available > 0) "${locker.available} available" else "Guns not available",
                                color = if (locker.available > 0) Palette.Green else Palette.Dim, fontSize = 13.sp,
                            )
                        }
                        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(10.dp, Alignment.CenterHorizontally), verticalAlignment = Alignment.Top) {
                            locker.slots.forEach { slot ->
                                Column(Modifier.width(64.dp), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(6.dp)) {
                                    GunIcon(locker.kind, toneOf(locker, slot), if (locker.kind == "pistol") 76 else 140)
                                    Text("${slot.no}", color = Palette.Muted, fontSize = 11.sp)
                                    if (slot.takenBy.isNotEmpty()) {
                                        Text(slot.takenBy, color = Color(0xFFFF918A), fontSize = 11.sp, maxLines = 1, textAlign = TextAlign.Center)
                                    }
                                }
                            }
                        }
                        Legend(locker)
                    }
                }
            }
        }
        QuietButton("Cancel", onClick = model::beginFace)
        ErrorLine(state.error)
    }
}

@Composable
private fun StatePill(online: Boolean) {
    val tone = if (online) Palette.Green else Palette.Yellow
    Row(
        Modifier.clip(RoundedCornerShape(50)).background(tone.copy(alpha = 0.12f)).padding(horizontal = 10.dp, vertical = 4.dp),
        verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        Box(Modifier.size(6.dp).clip(CircleShape).background(tone))
        Text(if (online) "Detecting" else "Not detecting", color = tone, fontSize = 11.sp, fontWeight = FontWeight.SemiBold)
    }
}

@Composable
private fun Legend(locker: Locker) {
    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(14.dp, Alignment.CenterHorizontally)) {
        if (locker.online) {
            LegendItem(Palette.Green, "Present")
            LegendItem(Palette.Red, "Missing")
            LegendItem(Palette.Yellow, "Sensor not there")
            if (locker.slots.any { it.reading < 0 }) LegendItem(Palette.Grey, "No data yet")
        } else {
            LegendItem(Palette.Yellow, "No sensor data is being received")
        }
    }
}

@Composable
private fun LegendItem(color: Color, text: String) {
    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
        Box(Modifier.size(8.dp).clip(CircleShape).background(color))
        Text(text, color = Palette.Muted, fontSize = 12.sp)
    }
}

@Composable
private fun GunsStep(model: KioskViewModel, state: UiState) {
    val locker = state.selected
    Screen {
        Row(Modifier.fillMaxWidth()) { QuietButton("‹ Back") { model.back(Step.CATALOG) } }
        Copy("STEP 2 OF 3", "Choose your ${locker?.kind.orEmpty()}s", "Tap each gun you need from ${locker?.name.orEmpty()}. You can pick more than one.")
        if (locker != null) {
            GunRack(
                locker, Modifier.width(680.dp), picked = state.picked, onToggle = model::togglePick,
                iconHeight = if (locker.kind == "pistol") 84 else 130, panelHeight = 300,
            )
        }
    }
}

@Composable
private fun GunsBar(model: KioskViewModel, state: UiState) {
    val n = state.picked.size
    val availableCount = state.selected?.slots?.count { it.available } ?: 0
    Row(
        Modifier.fillMaxWidth().background(Palette.Bg).padding(horizontal = 32.dp, vertical = 14.dp),
        horizontalArrangement = Arrangement.spacedBy(16.dp, Alignment.CenterHorizontally),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (availableCount > 1) QuietButton("Select all available", onClick = model::pickAll)
        PrimaryButton(
            when {
                n > 1 -> "Continue with $n guns  →"
                n == 1 -> "Continue with 1 gun  →"
                else -> "Select a gun to continue"
            },
            enabled = n > 0,
            onClick = { model.next(Step.REVIEW) },
        )
    }
}

@Composable
private fun SummaryRow(label: String, value: String) {
    Row(Modifier.fillMaxWidth().padding(vertical = 18.dp), horizontalArrangement = Arrangement.SpaceBetween) {
        Text(label, color = Palette.Muted)
        Text(value, color = Color.White, fontWeight = FontWeight.SemiBold)
    }
}

@Composable
private fun ReviewStep(model: KioskViewModel, state: UiState) {
    val locker = state.selected
    Screen {
        Row(Modifier.fillMaxWidth()) { QuietButton("‹ Back") { model.back(Step.GUNS) } }
        Copy("STEP 3 OF 3", "Review your request", "Nothing opens until an admin approves it.")
        Card(Modifier.width(520.dp)) {
            Column(Modifier.padding(horizontal = 22.dp)) {
                SummaryRow("Locker", locker?.name.orEmpty())
                SummaryRow("Type", locker?.kind.orEmpty().replaceFirstChar { it.uppercase() })
                SummaryRow(if (state.picked.size > 1) "Guns" else "Gun", state.picked.joinToString(", "))
            }
        }
        OutlinedTextField(
            value = state.reason,
            onValueChange = model::setReason,
            label = { Text("Reason (optional)") },
            placeholder = { Text("e.g. Range practice") },
            singleLine = true,
            keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
            colors = fieldColors(),
            shape = RoundedCornerShape(16.dp),
            modifier = Modifier.width(520.dp),
        )
        PrimaryButton(if (state.busy) "Sending…" else "Send request", enabled = !state.busy, onClick = { model.sendRequest() })
        ErrorLine(state.error)
    }
}

@Composable
private fun fieldColors() = OutlinedTextFieldDefaults.colors(
    focusedTextColor = Color.White,
    unfocusedTextColor = Color.White,
    focusedBorderColor = Palette.Blue,
    unfocusedBorderColor = Color(0x22FFFFFF),
    focusedLabelColor = Palette.BlueSoft,
    unfocusedLabelColor = Palette.Muted,
    cursorColor = Palette.Blue,
    focusedContainerColor = Palette.Fill,
    unfocusedContainerColor = Palette.Fill,
)

private fun listNos(nos: List<Int>): String = when {
    nos.size < 2 -> nos.firstOrNull()?.toString().orEmpty()
    else -> nos.dropLast(1).joinToString(", ") + " and " + nos.last()
}

@Composable
private fun WaitingStep(model: KioskViewModel, state: UiState) {
    val r = state.request
    val nos = r?.chosenNos().orEmpty()
    Screen {
        Box(Modifier.size(104.dp).clip(CircleShape).background(Color(0x1A0A84FF)), contentAlignment = Alignment.Center) {
            CircularProgressIndicator(color = Palette.Blue, strokeWidth = 4.dp, modifier = Modifier.size(58.dp))
        }
        Copy("REQUEST SENT", "Waiting for approval", "An admin has been notified. Keep this screen open.")
        Row(
            Modifier.width(320.dp).clip(RoundedCornerShape(16.dp)).background(Palette.Surface).padding(16.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            Text(state.selected?.name ?: r?.lockerName.orEmpty(), color = Color.White, fontWeight = FontWeight.SemiBold)
            Text(if (nos.isNotEmpty()) "${if (nos.size > 1) "Guns" else "Gun"} ${nos.joinToString(", ")}" else r?.kind.orEmpty(), color = Palette.Muted)
        }
        QuietButton("Cancel request", danger = true) { model.cancelRequest() }
    }
}

@Composable
private fun OpenStep(state: UiState, wrong: Boolean) {
    val r = state.request
    val remaining = r?.chosenNos("chosen").orEmpty()
    val body = when {
        wrong -> "Return gun ${r?.wrong?.joinToString(", ").orEmpty()}, then use the highlighted ${if (remaining.size > 1) "slots" else "slot"}."
        r?.chosenNos("collected")?.isNotEmpty() == true -> "Now take gun ${listNos(remaining)}."
        else -> {
            val all = r?.chosenNos().orEmpty()
            if (all.size > 1) "Take guns ${listNos(all)}." else "Take the ${r?.kind.orEmpty()} from slot ${r?.slotNo ?: 0}."
        }
    }
    Screen {
        Symbol(if (wrong) "!" else "↗", if (wrong) Palette.Red else Palette.Blue)
        Copy(if (wrong) "WRONG SLOT" else "APPROVED", if (wrong) "Put it back" else "${r?.lockerName.orEmpty()} is opening", body)
        Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            r?.slots?.forEach { slot ->
                val target = slot.no in remaining
                val done = slot.no in r.chosenNos("collected")
                val bad = slot.no in r.wrong
                val line = when {
                    bad -> Palette.Red
                    done -> Palette.Green
                    target -> Palette.Blue
                    else -> Color(0x14FFFFFF)
                }
                Box(
                    Modifier.size(64.dp, 92.dp).clip(RoundedCornerShape(15.dp)).background(line.copy(alpha = if (line == Color(0x14FFFFFF)) 1f else 0.18f))
                        .border(2.dp, line, RoundedCornerShape(15.dp)),
                    contentAlignment = Alignment.BottomCenter,
                ) { Text("${slot.no}", color = Color.White, fontWeight = FontWeight.Bold, modifier = Modifier.padding(10.dp)) }
            }
        }
    }
}

@Composable
private fun CollectedStep(model: KioskViewModel, state: UiState) {
    val r: GunRequest? = state.request
    val nos = r?.chosenNos().orEmpty()
    val back = r?.chosenNos("returned").orEmpty()
    val locker = r?.lockerName.orEmpty()
    if (nos.size > 1 && back.isNotEmpty()) {
        val left = nos - back.toSet()
        Simple(
            "↩", Palette.Yellow, "RETURNING", "${back.size} of ${nos.size} guns returned",
            "Still to return to $locker: slot${if (left.size > 1) "s" else ""} ${listNos(left)}.", "Done", model::beginFace,
        )
        return
    }
    val body = if (nos.size > 1) "Return them to $locker, slots ${listNos(nos)}, when you are finished."
    else "Return it to $locker, slot ${r?.slotNo ?: 0}, when you are finished."
    Simple("✓", Palette.Green, "COLLECTED", if (nos.size > 1) "All guns collected" else "Correct item", body, "Done", model::beginFace)
}

@Composable
private fun Simple(glyph: String, tone: Color, eyebrow: String, title: String, body: String, action: String, onClick: () -> Unit) {
    Screen {
        Symbol(glyph, tone)
        Copy(eyebrow, title, body)
        PrimaryButton(action, onClick = onClick)
    }
}

@Composable
private fun EnrollNameStep(model: KioskViewModel, state: UiState) {
    Screen {
        Row(Modifier.fillMaxWidth()) { QuietButton("‹ Cancel", onClick = model::beginFace) }
        Copy("ENROLLMENT · STEP 1 OF 2", "Tell us who you are", "An admin will verify these details before activating your face.")
        OutlinedTextField(
            value = state.enrollName, onValueChange = model::setEnrollName, label = { Text("Full name") },
            singleLine = true, colors = fieldColors(), shape = RoundedCornerShape(16.dp), modifier = Modifier.width(520.dp),
            keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
        )
        PrimaryButton("Continue  →", enabled = state.enrollName.isNotBlank(), onClick = model::prepareEnrollCamera)
    }
}

@Composable
private fun EnrollFaceStep(model: KioskViewModel, state: UiState, granted: Boolean) {
    Row(Modifier.fillMaxSize(), horizontalArrangement = Arrangement.spacedBy(48.dp, Alignment.CenterHorizontally), verticalAlignment = Alignment.CenterVertically) {
        CameraBox(model, granted && !state.modelMissing, 380, if (granted) "Camera paused: no face model installed." else "Allow camera access to continue.")
        Column(Modifier.width(420.dp), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(18.dp)) {
            Row(Modifier.fillMaxWidth()) { QuietButton("‹ Back") { model.back(Step.ENROLL_NAME) } }
            Copy("ENROLLMENT · STEP 2 OF 2", "Capture your face", "")
            Text(
                if (state.busy) "Sending…" else scanMessage(state.scan, "", state.modelMissing),
                color = Palette.Muted, fontSize = 18.sp, textAlign = TextAlign.Center,
            )
            ErrorLine(state.error)
            Spacer(Modifier.height(4.dp))
        }
    }
}
