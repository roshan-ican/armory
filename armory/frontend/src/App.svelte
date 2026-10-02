<script>
  import { onMount, tick } from "svelte";

  let step = new URLSearchParams(location.search).has("enroll") ? "enroll-name" : "face";
  let direction = 1;
  let user = null;
  let video;
  let faceMessage = "Preparing the camera…";
  let stopWatching;
  let pollTimer;
  let catalog = [];
  let selected = null;
  let picked = [];
  let reason = "";
  let request = null;
  let error = "";
  let busy = false;
  let enroll = { name: "", serviceNo: "" };

  const titles = {
    face: "Face verification", welcome: "Welcome", catalog: "Choose a locker", guns: "Choose guns",
    review: "Review", waiting: "Approval",
    open: "Locker access", correct: "Collected", wrong: "Wrong slot",
    declined: "Declined", returned: "Complete", "enroll-name": "Your details",
    "enroll-face": "Face capture", "enroll-pending": "Request sent"
  };

  function go(next, forward = true) {
    direction = forward ? 1 : -1;
    error = "";
    step = next;
  }

  async function api(method, url, body) {
    const res = await fetch(url, {
      method,
      headers: body ? { "Content-Type": "application/json" } : {},
      body: body ? JSON.stringify(body) : undefined,
      credentials: "same-origin"
    });
    let data = {};
    try { data = await res.json(); } catch {}
    if (!res.ok) throw Object.assign(new Error(data.error || (typeof data === "string" ? data : "Something went wrong.")), { status: res.status });
    return data;
  }

  function stopCamera() {
    if (stopWatching) stopWatching();
    stopWatching = null;
    if (video && window.armoryFace) window.armoryFace.stop(video);
  }

  async function beginFace() {
    clearInterval(pollTimer);
    stopCamera();
    await fetch("/api/logout", { method: "POST", credentials: "same-origin" }).catch(() => {});
    user = null; selected = null; picked = []; request = null; reason = "";
    go("face", false);
    await tick();
    faceMessage = "Preparing the camera…";
    try {
      await window.armoryFace.load();
      await window.armoryFace.start(video);
      faceMessage = "Hold still and look straight ahead.";
      stopWatching = window.armoryFace.watch(video, {
        onLooking: () => faceMessage = "Looking for your face…",
        onNoMatch: () => faceMessage = "We don't recognise you yet.",
        onError: () => faceMessage = "Keep your face inside the frame.",
        onMatch: async (matched) => {
          stopCamera();
          user = matched;
          go("welcome");
          try {
            const current = await api("GET", "/api/requests/current");
            follow(current.id);
          } catch (e) {
            if (e.status !== 404) error = e.message;
          }
        }
      });
    } catch (e) { faceMessage = e.message || "The camera is unavailable."; }
  }

  async function loadCatalog() {
    busy = true;
    try { catalog = await api("GET", "/api/catalog"); go("catalog"); }
    catch (e) { error = e.message; }
    finally { busy = false; }
  }

  async function refreshCatalog() {
    try { catalog = await api("GET", "/api/catalog"); } catch { return; }
    if (!selected) return;
    const fresh = catalog.find((c) => c.id === selected.id);
    if (!fresh) return;
    selected = fresh;
    picked = picked.filter((no) => fresh.slots.find((s) => s.no === no)?.available);
  }

  const browsing = () => step === "catalog" || step === "guns" || step === "review";

  function chooseLocker(item) {
    selected = item;
    picked = [];
    go("guns");
  }

  function togglePick(no) {
    picked = picked.includes(no) ? picked.filter((n) => n !== no) : [...picked, no].sort((a, b) => a - b);
  }

  function slotLabel(item, slot) {
    if (slot.available) return "Available";
    if (slot.taken_by) return `Taken by ${slot.taken_by}`;
    if (!item.online || slot.reading === 2) return "Sensor not there";
    if (slot.reading === 0) return "Missing";
    return "No data yet";
  }

  const listNos = (nos) => nos.length < 2 ? `${nos[0] ?? ""}` : `${nos.slice(0, -1).join(", ")} and ${nos[nos.length - 1]}`;
  const chosenNos = (r, status) => (r?.chosen || []).filter((c) => !status || c.status === status).map((c) => c.no);

  function watchChanges() {
    if (!window.EventSource) return () => {};
    let timer;
    const events = new EventSource("/events");
    events.onmessage = () => {
      clearTimeout(timer);
      timer = setTimeout(() => { if (browsing()) refreshCatalog(); }, 150);
    };
    return () => { clearTimeout(timer); events.close(); };
  }

  function refreshCatalogIfVisible() {
    if (!document.hidden && browsing()) refreshCatalog();
  }

  async function sendRequest() {
    busy = true; error = "";
    try {
      request = await api("POST", "/api/requests", { locker_id: selected.id, slots: picked, reason });
      follow(request.id);
    } catch (e) { error = e.message; }
    finally { busy = false; }
  }

  function applyRequest(value) {
    request = value;
    if (value.status === "pending") go("waiting");
    else if (value.status === "approved") go(value.wrong?.length ? "wrong" : "open");
    else if (value.status === "collected") go("correct");
    else if (value.status === "returned") { clearInterval(pollTimer); go("returned"); }
    else if (value.status === "rejected") { clearInterval(pollTimer); go("declined"); }
  }

  function follow(id) {
    clearInterval(pollTimer);
    const refresh = async () => {
      try { applyRequest(await api("GET", `/api/requests/${id}`)); } catch {}
    };
    refresh();
    pollTimer = setInterval(refresh, 1000);
  }

  async function cancelRequest() {
    if (request) await api("POST", `/api/requests/${request.id}/cancel`).catch(() => {});
    beginFace();
  }

  async function startEnrollment() {
    stopCamera();
    go("enroll-name");
  }

  async function prepareEnrollmentCamera() {
    go("enroll-face");
    await tick();
    faceMessage = "Preparing the camera…";
    try {
      await window.armoryFace.load();
      await window.armoryFace.start(video);
      faceMessage = "Center your face and hold still.";
    } catch (e) { faceMessage = e.message || "The camera is unavailable."; }
  }

  async function submitEnrollment() {
    busy = true; error = "";
    try {
      const descriptors = [];
      for (let i = 0; i < 3; i++) {
        faceMessage = `Capturing ${i + 1} of 3…`;
        const descriptor = await window.armoryFace.descriptor(video);
        if (descriptor) descriptors.push(Array.from(descriptor));
        await window.armoryFace.sleep(180);
      }
      if (descriptors.length < 2) throw new Error("We couldn't see your face clearly. Try again in better light.");
      await window.armoryFace.post("/enroll", { name: enroll.name, service_no: enroll.serviceNo, descriptors });
      stopCamera();
      go("enroll-pending");
    } catch (e) { error = e.message || "Could not send the enrollment request."; }
    finally { busy = false; }
  }

  onMount(() => {
    if (step === "face") beginFace();
    const stopEvents = watchChanges();
    const catalogTimer = setInterval(refreshCatalogIfVisible, 1000);
    window.addEventListener("pageshow", refreshCatalogIfVisible);
    document.addEventListener("visibilitychange", refreshCatalogIfVisible);
    return () => {
      stopCamera();
      clearInterval(pollTimer);
      clearInterval(catalogTimer);
      window.removeEventListener("pageshow", refreshCatalogIfVisible);
      document.removeEventListener("visibilitychange", refreshCatalogIfVisible);
      stopEvents();
    };
  });
</script>

<svelte:head><title>Armory</title></svelte:head>

<div class="shell" data-direction={direction}>
  <header class="topbar">
    <div class="brand"><span class="brand-mark">A</span><span>Armory</span></div>
    <span class="context">{titles[step]}</span>
    {#if user}<span class="identity">{user.name}</span>{:else}<span></span>{/if}
  </header>

  <main>
    {#key step}
      <section class:catalog-screen={step === "catalog"} class="screen">
        {#if step === "face"}
          <div class="camera-wrap"><video bind:this={video} playsinline muted></video><div class="face-oval"></div><div class="scan-line"></div></div>
          <div class="copy"><p class="eyebrow">IDENTITY CHECK</p><h1>Look at the camera</h1><p>{faceMessage}</p></div>
          <button class="secondary" on:click={startEnrollment}>Enroll yourself</button>

        {:else if step === "welcome"}
          <div class="symbol success">✓</div>
          <div class="copy"><p class="eyebrow">VERIFIED</p><h1>Good {new Date().getHours() < 12 ? "morning" : new Date().getHours() < 18 ? "afternoon" : "evening"}, {user?.name?.split(" ")[0]}</h1><p>Your identity has been confirmed.</p></div>
          <button class="primary" disabled={busy} on:click={loadCatalog}>Continue <span>→</span></button>

        {:else if step === "catalog"}
          <div class="copy"><p class="eyebrow">STEP 1 OF 3</p><h1>Choose a locker</h1><p>Select the inventory you want to request from.</p></div>
          <div class="cards">
            {#each catalog as item}
              <button class="choice-card locker-choice" disabled={item.available === 0} on:click={() => chooseLocker(item)}>
                <span class="locker-head">
                  <span class="choice-copy"><strong>{item.name}</strong><small>{item.location || item.kind}</small></span>
                  <span class:online={item.online} class="board-state"><i></i>{item.online ? "Detecting" : "Not detecting"}</span>
                  <span class:available={item.available > 0} class="count">{item.available ? `${item.available} available` : "Guns not available"}</span>
                  <span class="chevron">›</span>
                </span>
                <span class="gun-row gun-row-{item.kind}">
                  {#each item.slots || [] as slot}
                    <span class="gun-cell">
                      <span class="gun {!item.online || slot.reading === 2 ? "gun-detecting" : slot.reading === 1 ? "gun-in" : slot.reading === 0 ? "gun-out" : "gun-fault"}"></span>
                      <b>{slot.no}</b>
                      {#if slot.taken_by}<em><small>Taken by</small>{slot.taken_by}</em>{/if}
                    </span>
                  {/each}
                </span>
                <span class="gun-legend">
                  {#if item.online}
                    <span><i class="gun-in"></i>Present</span><span><i class="gun-out"></i>Missing</span><span><i class="gun-detecting"></i>Sensor not there</span>{#if item.slots?.some((s) => s.reading < 0)}<span><i class="gun-fault"></i>No data yet</span>{/if}
                  {:else}
                    <span><i class="gun-detecting"></i>No sensor data is being received</span>
                  {/if}
                </span>
              </button>
            {/each}
          </div>
          <button class="quiet" on:click={beginFace}>Cancel</button>

        {:else if step === "guns"}
          <button class="back" on:click={() => go("catalog", false)}>‹ Back</button>
          <div class="copy"><p class="eyebrow">STEP 2 OF 3</p><h1>Choose your {selected?.kind}s</h1><p>Tap each gun you need from {selected?.name}. You can pick more than one.</p></div>
          <div class="gun-pick-row" class:pistol={selected?.kind === "pistol"}>
            {#each selected?.slots || [] as slot}
              <button class="gun-pick" class:picked={picked.includes(slot.no)} aria-pressed={picked.includes(slot.no)} disabled={!slot.available} on:click={() => togglePick(slot.no)}>
                <span class="tick">✓</span>
                <span class="gun {!selected.online || slot.reading === 2 ? "gun-detecting" : slot.reading === 1 ? "gun-in" : slot.reading === 0 ? "gun-out" : "gun-fault"}"></span>
                <b>Gun {slot.no}</b>
                <small>{slotLabel(selected, slot)}</small>
              </button>
            {/each}
          </div>
          {#if (selected?.slots || []).filter((s) => s.available).length > 1}
            <button class="quiet" on:click={() => picked = selected.slots.filter((s) => s.available).map((s) => s.no)}>Select all available</button>
          {/if}
          <button class="primary" disabled={!picked.length} on:click={() => go("review")}>{picked.length > 1 ? `Continue with ${picked.length} guns` : picked.length ? "Continue with 1 gun" : "Pick at least one gun"} <span>→</span></button>

        {:else if step === "review"}
          <button class="back" on:click={() => go("guns", false)}>‹ Back</button>
          <div class="copy"><p class="eyebrow">STEP 3 OF 3</p><h1>Review your request</h1><p>Nothing opens until an admin approves it.</p></div>
          <div class="summary">
            <div><span>Locker</span><strong>{selected?.name}</strong></div><div><span>Type</span><strong class="capitalize">{selected?.kind}</strong></div><div><span>{picked.length > 1 ? "Guns" : "Gun"}</span><strong>{picked.join(", ")}</strong></div>
          </div>
          <label class="field"><span>Reason <i>Optional</i></span><input bind:value={reason} maxlength="120" placeholder="e.g. Range practice"></label>
          <button class="primary" disabled={busy} on:click={sendRequest}>{busy ? "Sending…" : "Send request"}</button>

        {:else if step === "waiting"}
          <div class="progress-ring"><span></span></div>
          <div class="copy"><p class="eyebrow">REQUEST SENT</p><h1>Waiting for approval</h1><p>An admin has been notified. Keep this screen open.</p></div>
          <div class="request-pill"><span>{selected?.name || request?.locker_name}</span><strong>{chosenNos(request).length ? `${chosenNos(request).length > 1 ? "Guns" : "Gun"} ${chosenNos(request).join(", ")}` : selected?.kind || request?.kind}</strong></div>
          <button class="quiet danger" on:click={cancelRequest}>Cancel request</button>

        {:else if step === "open" || step === "wrong"}
          <div class:danger-symbol={step === "wrong"} class="symbol">{step === "wrong" ? "!" : "↗"}</div>
          <div class="copy"><p class="eyebrow">{step === "wrong" ? "WRONG SLOT" : "APPROVED"}</p><h1>{step === "wrong" ? "Put it back" : `${request?.locker_name} is opening`}</h1><p>{step === "wrong" ? `Return gun ${request?.wrong?.join(", ")}, then use the highlighted ${chosenNos(request, "chosen").length > 1 ? "slots" : "slot"}.` : chosenNos(request, "collected").length ? `Now take gun ${listNos(chosenNos(request, "chosen"))}.` : `Take ${chosenNos(request).length > 1 ? `guns ${listNos(chosenNos(request))}` : `the ${request?.kind} from slot ${request?.slot_no}`}.`}</p></div>
          <div class="slots">{#each request?.slots || [] as slot}<div class:target={chosenNos(request, "chosen").includes(slot.no)} class:done={chosenNos(request, "collected").includes(slot.no)} class:wrong={request?.wrong?.includes(slot.no)}><span>{slot.no}</span></div>{/each}</div>

        {:else if step === "correct"}
          <div class="symbol success">✓</div><div class="copy"><p class="eyebrow">COLLECTED</p><h1>{chosenNos(request).length > 1 ? "All guns collected" : "Correct item"}</h1><p>{chosenNos(request).length > 1 ? `Return them to ${request?.locker_name}, slots ${listNos(chosenNos(request))}, when you are finished.` : `Return it to ${request?.locker_name}, slot ${request?.slot_no}, when you are finished.`}</p></div>
          <button class="primary" on:click={beginFace}>Done</button>

        {:else if step === "declined"}
          <div class="symbol danger-symbol">×</div><div class="copy"><p class="eyebrow">NOT APPROVED</p><h1>Request declined</h1><p>Ask an administrator if you need help.</p></div>
          <button class="primary" on:click={beginFace}>Start over</button>

        {:else if step === "returned"}
          <div class="symbol success">✓</div><div class="copy"><p class="eyebrow">COMPLETE</p><h1>Returned safely</h1><p>The slot is locked again. Thank you.</p></div>
          <button class="primary" on:click={beginFace}>Done</button>

        {:else if step === "enroll-name"}
          <button class="back" on:click={beginFace}>‹ Cancel</button>
          <div class="copy"><p class="eyebrow">ENROLLMENT · STEP 1 OF 2</p><h1>Tell us who you are</h1><p>An admin will verify these details before activating your face.</p></div>
          <div class="form-stack"><label class="field"><span>Full name</span><input bind:value={enroll.name} autocomplete="name" placeholder="Your full name"></label><label class="field"><span>Service number</span><input bind:value={enroll.serviceNo} autocomplete="off" placeholder="Your service number"></label></div>
          <button class="primary" disabled={!enroll.name.trim() || !enroll.serviceNo.trim()} on:click={prepareEnrollmentCamera}>Continue <span>→</span></button>

        {:else if step === "enroll-face"}
          <button class="back" on:click={() => { stopCamera(); go("enroll-name", false); }}>‹ Back</button>
          <div class="camera-wrap compact"><video bind:this={video} playsinline muted></video><div class="face-oval"></div></div>
          <div class="copy"><p class="eyebrow">ENROLLMENT · STEP 2 OF 2</p><h1>Capture your face</h1><p>{faceMessage}</p></div>
          <button class="primary" disabled={busy} on:click={submitEnrollment}>{busy ? "Capturing…" : "Send enrollment request"}</button>

        {:else if step === "enroll-pending"}
          <div class="symbol success">✓</div><div class="copy"><p class="eyebrow">SUBMITTED</p><h1>Request sent</h1><p>An admin must approve your enrollment before your face can be used.</p></div>
          <button class="primary" on:click={beginFace}>Return to face scan</button>
        {/if}
        {#if error}<p class="error" role="alert">{error}</p>{/if}
      </section>
    {/key}
  </main>

  <footer><span>Secure local system</span><span class="step-dot"></span><span>{titles[step]}</span></footer>
</div>
