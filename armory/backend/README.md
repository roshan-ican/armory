# Armory backend

## Run

From `armory/backend`:

```powershell
go run ./cmd/server
```

| Service         | Default  | Env var              |
| --------------- | -------- | -------------------- |
| Web UI (HTTP)   | `:8080`  | `ARMORY_ADDR`        |
| Sensor listener | `:47810` | `ARMORY_SENSOR_ADDR` |
| Database        | `armory.db` | `ARMORY_DB`       |

## Firewall (Windows)

Sensor boards send UDP packets to this machine on port `47810`. Windows blocks incoming traffic by default, so allow it once per server machine from an **Administrator** PowerShell.

Add:

```powershell
netsh advfirewall firewall add rule name="Armory sensor UDP" dir=in action=allow protocol=UDP localport=47810
```

Check:

```powershell
netsh advfirewall firewall show rule name="Armory sensor UDP"
```

Remove:

```powershell
netsh advfirewall firewall delete rule name="Armory sensor UDP"
```

If `ARMORY_SENSOR_ADDR` is changed, use the same port in these commands.
