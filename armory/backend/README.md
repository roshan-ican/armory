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

## Sensor packet

One UDP datagram per locker, raw bytes:

```
24  s0  s1  ...  sN  crc  23
$   slot states       sum  #
```

- Slot state: `00` gun taken, `01` gun present, `02` fault
- `crc` = CRC-8/MAXIM (poly 0x8C reflected, init 0) over the 3 slot bytes
- Examples from the board: `24 02 02 00 de 23`, `24 02 00 01 11 23`
