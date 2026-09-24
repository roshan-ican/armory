CREATE TABLE IF NOT EXISTS categories (
    id         INTEGER PRIMARY KEY,
    name       TEXT    NOT NULL UNIQUE,
    active     INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1))
) STRICT;

CREATE TABLE IF NOT EXISTS lockers (
    id         INTEGER PRIMARY KEY,
    name       TEXT    NOT NULL UNIQUE,
    location   TEXT,
    ip_address TEXT
) STRICT;

CREATE TABLE IF NOT EXISTS users (
    id         INTEGER PRIMARY KEY,
    name       TEXT    NOT NULL,
    service_no TEXT    NOT NULL UNIQUE,
    role       TEXT    NOT NULL CHECK (role IN ('admin', 'requester')),
    active     INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_at TEXT    NOT NULL,
    updated_at TEXT    NOT NULL
) STRICT;

CREATE TABLE IF NOT EXISTS slots (
    id         INTEGER PRIMARY KEY,
    locker_id  INTEGER NOT NULL REFERENCES lockers (id),
    slot_no    INTEGER NOT NULL CHECK (slot_no BETWEEN 1 AND 5),
    sensor_id  INTEGER NOT NULL UNIQUE,
    active     INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_at TEXT    NOT NULL,
    updated_at TEXT    NOT NULL,
    UNIQUE (locker_id, slot_no)
) STRICT;

CREATE TABLE IF NOT EXISTS guns (
    id          INTEGER PRIMARY KEY,
    slot_id     INTEGER UNIQUE REFERENCES slots (id),
    category_id INTEGER NOT NULL REFERENCES categories (id),
    serial      TEXT    NOT NULL UNIQUE,
    model       TEXT,
    notes       TEXT,
    status      TEXT    NOT NULL DEFAULT 'in' CHECK (status IN ('in', 'out', 'retired')),
    created_at  TEXT    NOT NULL,
    updated_at  TEXT    NOT NULL
) STRICT;

CREATE TABLE IF NOT EXISTS face_enrollments (
    id            INTEGER PRIMARY KEY,
    user_id       INTEGER NOT NULL REFERENCES users (id),
    face_ref      TEXT    NOT NULL,
    model_version TEXT    NOT NULL,
    enrolled_by   INTEGER REFERENCES users (id),
    created_at    TEXT    NOT NULL,
    revoked_at    TEXT
) STRICT;

CREATE TABLE IF NOT EXISTS requests (
    id                 INTEGER PRIMARY KEY,
    requester_id       INTEGER NOT NULL REFERENCES users (id),
    category_id        INTEGER REFERENCES categories (id),
    gun_id             INTEGER REFERENCES guns (id),
    status             TEXT    NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'collected', 'returned', 'expired', 'cancelled')),
    reason             TEXT,
    admin_note         TEXT,
    decided_by         INTEGER REFERENCES users (id),
    valid_until        TEXT,
    expected_return_at TEXT,
    created_at         TEXT    NOT NULL,
    decided_at         TEXT,
    collected_at       TEXT,
    returned_at        TEXT
) STRICT;

CREATE TABLE IF NOT EXISTS events (
    id          INTEGER PRIMARY KEY,
    occurred_at TEXT    NOT NULL,
    type        TEXT    NOT NULL,
    user_id     INTEGER REFERENCES users (id),
    gun_id      INTEGER REFERENCES guns (id),
    slot_id     INTEGER REFERENCES slots (id),
    request_id  INTEGER REFERENCES requests (id),
    details     TEXT
) STRICT;
