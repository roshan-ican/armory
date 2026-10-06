# Armory: what we built

Armory tracks which guns are in which locker, lets people request a gun with their face, lets an admin approve, opens the locker, and tells everyone what is happening in real time. This page explains the pieces and how they fit together.

## 1. The big picture

| Part | What it is | Where it runs |
|---|---|---|
| **Server** | One Go program (`armory/backend`). Holds the database, the rules, the web pages and the hardware commands. | The PC |
| **Kiosk (requester app)** | Native Kotlin Android app (`kiosk-native`). Scans the face on the tablet, then picks a locker and sends a request. | The tablet |
| **Admin app** | Installable web app under `/admin/`. Lockers, requests, activity, people. | PC or any browser |
| **Sensor boards** | One board per locker with 3 slot sensors. Sends a UDP packet about 10 times a second and listens for the 500 ms output frame. | Each locker |

Everything shares one SQLite database. Changes are pushed to every open screen over a live stream (`/events`), so nothing needs a refresh.

## 2. How a gun is taken and returned

1. **Face check.** The tablet camera makes a face descriptor in the browser. The server compares it with enrolled people (threshold 0.5) and starts a short session.
2. **Choose a locker.** The picker shows each locker with its state and its guns, then a review step with an optional reason. There is no return time: the system works only from sensor events.
3. **Pending.** The admin gets a badge, a toast with a chime, and, if enabled, a system notification.
4. **Approve.** The server picks a free slot (a gun that is present and not already held) and sends the **unlock** command to that locker.
5. **Collected.** When the assigned slot empties, the request becomes *collected* and the board sounds the **right gun** buzzer.
6. **Wrong gun.** If a different slot empties while a request is approved, the server records a wrong-gun event and sounds the **wrong gun** buzzer.
7. **Returned.** When the sensor sees the gun back in its slot, the request becomes *returned* and the gun is available to the next person.

While a gun is held, the picker and the admin show **"Taken by <name>"** under that slot.

## 3. Sensor boards and slot states

Each board sends the raw bytes `24 s1 s2 s3 crc 23` to the server on **UDP port 47810**. A change must hold for 10 frames before it counts (debounce). A board counts as **online** if it was heard in the last 5 seconds.

| Value | Meaning | Colour in the app |
|---|---|---|
| 1 | Gun present | Green |
| 0 | Gun missing | Red |
| 2 | Sensor not there / not detecting | Yellow |
| none yet | No reading stored | Grey ("No data yet") |

If a board is offline, every slot shows yellow and the locker says **Not detecting**. A locker with no gun that can be requested shows **Guns not available** and can't be tapped.

## 4. Commands sent to the boards

Frame (6 bytes): `24 seq open buzzer crc 23`. The checksum is CRC-8/MAXIM over the three bytes between `24` and the checksum. The frame leaves from the server's own port 47810, which the boards require.

| Packet | Bytes | When |
|---|---|---|
| Keep alive | `24 00 00 00 00 23` | Every 500 ms when nothing else is sent |
| Open (locker or door) | `24 seq 01 00 crc 23` | An admin approves a request |
| Correct (short sound) | `24 seq 00 01 crc 23` | The assigned gun is taken |
| Wrong (long sound) | `24 seq 00 02 crc 23` | A wrong gun is taken |

`seq` counts new commands (1 to 255, then back to 1, never 0), separately for each board. Each command is sent 3 times, 500 ms apart, with the same `seq`, so one lost packet does not matter and the board can ignore the repeats. A wrong gun sounds at once and then again every 3 seconds, with a new `seq`, until the gun is put back. Every new command is logged as `outputs <ip>: open seq=…`. The code is in `internal/services/outputs.go`.

`go run ./cmd/buzz <correct|wrong|unlock> <ip>` sends one by hand for a few seconds. Stop the server first, because it owns the port.

## 5. The two apps

**Kiosk (tablet).** The native Kotlin app calls the server's API at `https://<PC-IP>:8443`. The picker is one screen with no scrolling: lockers side by side, each with a Detecting pill, an "N available" count, a row of gun icons with slot numbers, and a legend. It refreshes live when a sensor changes.

**Admin.** Sign in with a username and password at `/admin/login`. Pages: **Lockers** (live guns, add and edit lockers), **Requests** (approve or decline), **Activity** (event log), **People** (enrol and manage faces). It uses the same dark look as the Android app. Press **Enable alerts** once to get system notifications for new requests while the admin app is open in the background.

## 6. Running it

```
cd armory/backend
go run ./cmd/server
```

- HTTP on `:8080`, HTTPS on `:8443` (the camera needs HTTPS or localhost), sensors on UDP `:47810`.
- `go run ./cmd/certgen` makes the certificates. The tablet installs the CA from `/ca.crt` once.
- Templates are baked into the program, so restart the server after any change. The tablet app is separate: after a change in `armory/kiosk-native`, rebuild and reinstall the APK (see `DEPLOY.md`).
- `ARMORY_SENSOR_DEBUG=1` logs every raw sensor frame. The server logs each request step and each command it sends.
- Checks: `gofmt -l internal`, `go vet ./...`, `go test ./...`.

## 7. Code layout (backend)

`transport` (web handlers, sensor listener) → `services` (rules, hardware) → `database` (SQLite queries) → `models`. A small `live` hub publishes changes to browsers.

## 8. Known gaps and next steps

- **Board 10.252.176.51 (Locker 2) is not sending.** It has been offline; check its power and network.
- **Sensor checksum is not checked yet.** Frames are accepted without validating the CRC byte.
- **Slot count vs capacity.** Boards send 3 slots but a locker can have up to 5; the extra slots never get readings.
- **Frame needs firmware support.** Boards must read `24 seq open buzzer crc 23` (the old `$1010#`, `$2020#`, `$2010#` text commands are gone). The byte order of `open` and `buzzer` was read from a screenshot; confirm it with the firmware.
- **Stuck holds.** A gun approved but never collected stays "Taken by" until someone returns it; decide on a timeout or an admin Release button.
- **Alerts only while the admin app is open.** Alerts when it is fully closed would need Web Push or ntfy.
- **Not built yet:** an alert when a gun leaves with no request, expiry of old pending requests, rate limiting on face matching, and a liveness check.
