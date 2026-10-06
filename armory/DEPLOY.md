# Run, update and deploy Armory

The server is one Go program. It serves the admin app and the API for the tablet app, listens for the sensor boards, and keeps one SQLite database. Install it on **one** PC only. Every other device on the network just opens a link.

## Addresses

| What | Where |
|---|---|
| Admin | `https://192.168.4.200:8443/admin/login` |
| Tablet app API | `https://192.168.4.200:8443` (used by the Android app, not opened in a browser) |
| CA certificate | `https://192.168.4.200:8443/ca.crt` |
| Sensor boards | send UDP to this PC on port `47810` |
| Plain HTTP (redirects to HTTPS) | `http://<pc>:8080` |

`192.168.4.200` is added to the server PC by the installer, next to the PC's own fixed address (`192.168.4.201`).

## Run for development

Stops when you close the terminal.

```powershell
cd armory\backend
go run ./cmd/server
```

Log every raw sensor packet:

```powershell
$env:ARMORY_SENSOR_DEBUG=1; go run ./cmd/server
```

Run tests:

```powershell
cd armory\backend
go test ./...
```

## Deploy (always on)

Run once on the server PC, from an **Administrator** PowerShell. Stop any `go run` first, because two servers cannot share the ports.

```powershell
cd armory\deploy
.\Install-Armory.ps1
```

It does this:

1. Builds `C:\Armory\armory-server.exe`.
2. Copies `certs` and `armory.db` to `C:\Armory` (only if they are not there yet).
3. Adds `192.168.4.200` to the PC's static `192.168.4.x` adapter.
4. Sets machine-wide `ARMORY_DB` and `ARMORY_CERTS` so every run uses the same files.
5. Opens the firewall for TCP 8080, 8443 and UDP 47810.
6. Registers a startup task, "Armory Server", that starts at boot and restarts after a crash.

Check it:

```powershell
Get-ScheduledTask 'Armory Server'
```

## Update after code changes

Save your changes, then run the same installer again from an Administrator PowerShell.

```powershell
cd armory\deploy
.\Install-Armory.ps1
```

- It stops the server, rebuilds the exe and starts it again at the same links.
- `C:\Armory\armory.db` is never overwritten, so data is kept.
- Everyone is logged out, because sessions live in memory.
- The tablet app is separate. Rebuild it only when you change `kiosk-native` (see Tablet app).
- A new table or column must be handled by the server's own startup code.

## Set up a device

**Admin PC**

1. Open `https://192.168.4.200:8443/ca.crt` to download the CA.
2. Double-click it, choose Install Certificate, Local Machine, then Trusted Root Certification Authorities.
3. Close and reopen the browser, then open the admin link.
4. In Edge or Chrome, install the site as an app, and allow notifications.
5. To start it with Windows, put a shortcut to the app in `shell:startup`.

**Tablet:** install the CA once from `/ca.crt`, then install the Kotlin app (see Tablet app).

**Sensor board:** it must send to this PC on UDP 47810. The locker's IP in the admin app must match the board's real IP (currently `192.168.4.50`), otherwise its packets are dropped.

## Tablet app

The tablet runs a native Kotlin app (`armory/kiosk-native`). It does the face scan on the tablet with the bundled `mobilefacenet.tflite` model, then talks to the server over HTTPS (`/face/match`, `/enroll`, `/api/...`, `/events`). Nothing is served to it as a web page, so a server update never changes the tablet screens. The server address is built into the app.

| You changed | Do this |
|---|---|
| Backend or admin templates | Re-run `Install-Armory.ps1`. |
| Anything in `kiosk-native` (screens, face code, server address, permissions) | Rebuild and reinstall the APK (below). |

You need JDK 17 and the Android SDK. Android Studio installs both, and writes the SDK path to `kiosk-native\local.properties`.

Build the APK:

```powershell
cd armory\kiosk-native
.\gradlew.bat assembleDebug
```

The APK is `armory\kiosk-native\app\build\outputs\apk\debug\app-debug.apk`. Debug builds install as `com.armory.kiosk.native`.

Run the unit tests (face alignment, liveness, enrollment plan, match math):

```powershell
.\gradlew.bat testDebugUnitTest
```

Install it on the tablet with USB debugging on:

```powershell
adb devices
adb install -r armory\kiosk-native\app\build\outputs\apk\debug\app-debug.apk
```

`-r` keeps the app's data. The tablet must trust the server CA once (install `/ca.crt` in Android settings as a CA certificate). The app trusts user-installed certificates, and it needs the camera permission for face login.

If the server address ever changes, edit the `SERVER_URL` line in `kiosk-native\app\build.gradle.kts`, then rebuild and reinstall the APK. An old APK keeps calling the old address.

If `adb devices` lists no device, check the USB cable, that USB debugging is on, and accept the "Allow USB debugging" prompt on the tablet.

### Enrolling a face

1. A person enrolls in the tablet app. This sends a request to the server.
2. An admin approves it in the admin app (Users page).
3. After approval the person can sign in with their face.

Enrollments made with the old web kiosk (face-api, 128 numbers) do not work with this app (192 numbers). Those people must enroll again in the tablet app.

## Files and settings

| Item | Location |
|---|---|
| Server, database, certs, backups | `C:\Armory` |
| Daily backups | `C:\Armory\backups` |
| Sensor address | `ARMORY_SENSOR_ADDR` (default `:47810`) |
| HTTP and HTTPS ports | `ARMORY_ADDR`, `ARMORY_TLS_ADDR` (defaults `:8080`, `:8443`) |
| Database and cert folder | `ARMORY_DB`, `ARMORY_CERTS` |
| Debug sensor log | `ARMORY_SENSOR_DEBUG=1` |

## Troubleshooting

| Problem | Check |
|---|---|
| Browser says "site can't be reached" | Is `192.168.4.200` on the server PC? Run `Get-NetIPAddress -AddressFamily IPv4` and re-run the installer if not. |
| Certificate warning | Install the CA on that PC (see above). |
| Sensors do not show | The locker IP in the admin app must equal the board's IP. Run with `ARMORY_SENSOR_DEBUG=1` to see packets. |
| Board never connects | The firewall rule for UDP 47810 must exist, and the board must be on the same network. |
| Installer says to run as Administrator | Open PowerShell with "Run as administrator". |
| Port already in use | Stop any `go run` or old server, then run the installer again. |
| Tablet app shows a blank page or an error | Is the server up and `192.168.4.200` reachable from the tablet? Is the CA installed on the tablet? Does `SERVER_URL` in the APK match the server address? |
| Tablet says the face model is missing | `mobilefacenet.tflite` must be in `kiosk-native\app\src\main\assets`. Rebuild and reinstall the APK. |
| Face never matches after the move to the Kotlin app | The person was enrolled with the old web kiosk. Enroll again in the tablet app and have an admin approve it. |
| `adb devices` shows nothing | Check the USB cable, USB debugging, and the prompt on the tablet. |
