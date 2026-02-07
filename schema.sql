CREATE TABLE IF NOT EXISTS participants (
    chat_id    INTEGER NOT NULL,
    user_id    INTEGER NOT NULL,
    first_name TEXT NOT NULL,
    username   TEXT NOT NULL DEFAULT '',
    joined_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (chat_id, user_id)
);

CREATE TABLE IF NOT EXISTS results (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    chat_id     INTEGER NOT NULL,
    user_id     INTEGER NOT NULL,
    played_date TEXT NOT NULL,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_results_chat_date ON results (chat_id, played_date);

CREATE TABLE IF NOT EXISTS message_sets (
    id INTEGER PRIMARY KEY AUTOINCREMENT
);

CREATE TABLE IF NOT EXISTS set_messages (
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    set_id  INTEGER NOT NULL REFERENCES message_sets(id),
    position INTEGER NOT NULL,
    body    TEXT NOT NULL,
    UNIQUE(set_id, position)
);

CREATE TABLE IF NOT EXISTS translations (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
