# Armory

Gun storage system: face-checked requests, admin approval, sensor-verified lockers, live updates everywhere.

## Parts

| Part | Tech | Folder |
|---|---|---|
| Server (DB, rules, web, hardware) | Go + SQLite | `backend` |
| Kiosk (requester tablet) | Kotlin Android | `kiosk-native` |
| Admin app | Installable web app (PWA, HTMX) at `/admin/` | `backend/web` |
| Face matching | Python helper | `faceserver` |
| Locker boards | ESP32, one per locker | hardware |

More detail: [OVERVIEW.md](OVERVIEW.md), [DEPLOY.md](DEPLOY.md), [backend/README.md](backend/README.md).

## How do lockers send data to the server?

**Plain UDP, pushed by the board. No polling, no WebSocket, no MQTT.**

- **Board to server:** each board sends a 6-byte raw frame `24 s1 s2 s3 crc 23` about 10 times a second to UDP port `47810`. Slot values: `1` gun present, `0` missing, `2` sensor fault. The checksum is CRC-8/MAXIM.
- **Server to board:** the server replies on the same port with `24 seq open buzzer crc 23` for unlock and buzzer commands. It sends a keep-alive every 500 ms. Each command is sent 3 times with the same `seq`, so a lost packet doesn't matter.
- **Server side:** a change must hold for 10 frames (debounce). A board counts as offline after 5 seconds of silence.
- **Server to screens:** the server pushes changes to the kiosk and admin over **Server-Sent Events** (`/events`). The kiosk uses OkHttp SSE and the admin uses `EventSource`. Nothing is refreshed by polling.

Code: `backend/internal/transport/sensor` (incoming) and `backend/internal/services/outputs.go` (outgoing).

## What is the sensor hardware?

An **ESP32** in each locker, with 3 slot sensors. The same board switches the lock power and drives the buzzer. There is a separate ESP32 for the door, set by `ARMORY_DOOR_IP`. Boards connect over Wi-Fi to the PC's LAN.

## Is MQTT worth switching to?

Probably not now.

- UDP on a single LAN with a fixed protocol already gives low latency, repeat-send reliability and offline detection.
- MQTT would add a broker as a new service to run (and fail over, see the 2-PC setup) and a firmware rewrite on every board.
- It would pay off if boards move off the LAN, if you want retained state or last-will offline detection from a broker, or if other systems (Home Assistant, dashboards) need to subscribe to locker events.

## Run

```powershell
cd backend
go run ./cmd/server
```

| Setting | Default | Env var |
|---|---|---|
| HTTP | `:8080` | `ARMORY_ADDR` |
| HTTPS | `:8443` | `ARMORY_TLS_ADDR` |
| Sensor UDP | `:47810` | `ARMORY_SENSOR_ADDR` |
| Database | `armory.db` | `ARMORY_DB` |
| Door board IP | `192.168.4.49` | `ARMORY_DOOR_IP` |

Allow UDP `47810` through the Windows firewall (see [backend/README.md](backend/README.md)). Admin login: `https://<PC-IP>:8443/admin/login`.

## Test

```powershell
cd backend
go test ./...
```
