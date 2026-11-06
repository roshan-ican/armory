# Face recognition: what we built and what to remember

## How it works, in four lines

1. The tablet finds a face (ML Kit) and turns it into a list of numbers called an embedding (MobileFaceNet, 192 numbers).
2. The server compares that list with every enrolled list. The distance between two lists says how different two faces are.
3. Small distance means same person. A threshold decides how small is small enough (0.8 for the native model, 0.5 for face-api).
4. ML Kit only finds faces and measures pose, eyes and smile. It cannot tell who someone is.

## What we improved

| Change | Why it helps | Where |
|---|---|---|
| Two-look enrollment (glasses on, then off) | A face with glasses and without are far apart in number space. Storing both means either one matches. | `face/EnrollPlan.kt`, `face/Scanner.kt` |
| Second-best margin | The best match must beat the next different user by 0.1 (0.05 for face-api). Stops look-alikes sneaking in. | `services/face.go` |
| Calibration tool | Shows real same-person and different-person distances so thresholds come from data, not guesses. | `cmd/calibrate` |

Real numbers from the live database (native model, 11 people):
- same person: 0.145 to 0.943, median 0.518
- different people: 0.873 to 1.670, median 1.320
- the closest two different people are 0.873 apart, so 0.8 has a thin margin. Keep it and re-measure as users are added.

## Rules of thumb

- Measure before changing a threshold. Raising it to fix false rejects lets look-alikes in.
- Nearest-vector matching: more varied vectors per person means more coverage. Variety beats quantity (poses, lighting, glasses).
- Pairwise stats overstate false rejects. Verification compares one near-frontal face to the nearest stored vector.
- Embeddings from different models cannot be compared. Changing the model means everyone re-enrolls.
- Enrollment duplicate check ignores the margin on purpose: a new face that resembles an existing user must still be blocked.

## Ideas not built yet

- Passive anti-spoofing (a photo or phone screen must not pass). This matters most for a gun safe.
- Stronger model (ArcFace or SFace class). Needs a TFLite file and re-enrollment.
- Several frames per attempt, with voting.
- Quality gate: blur, darkness, size.
- Adaptive templates: add confident logins as new vectors, with caps and anti-poisoning rules.
- A "try again with or without glasses" hint on near misses.

## Run the calibration

Copy the live database first, never point it at the original:

```bash
cp /c/Armory/armory.db /tmp/copy.db
cd armory/backend
go run ./cmd/calibrate -db /tmp/copy.db
```

## Other things learned this session

- Maintenance requests are blocked while someone else's job is active (`ErrMaintenanceBusy`). Their own repeat request still gets "already has a job".
- A locker will not unlock if its board is powered off. The server sends the same UDP packet to a locker as to the door (`24 seq 01 00 crc 23` to port 47810), so the board must be on. To unlock with sensors off, keep the board powered and switch only the sensor power.
- Check a board's state before blaming code: last-seen time in the database, and whether `192.168.4.x` answers.
- Redeploying the server needs an Administrator PowerShell: `armory\deploy\Install-Armory.ps1`. Then compare the exe's modified time and the process ID to confirm it really restarted.
- If an APK crashes with `ClassNotFoundException` for `MainActivity`, the build is stale. Run `gradlew clean assembleDebug`, then reinstall.
