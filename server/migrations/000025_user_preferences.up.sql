-- Per-user UI preferences, keyed by name so a new preference needs no
-- migration. The front page hero selection is the first entry; theme stays in
-- localStorage because it must apply before the first request completes.
CREATE TABLE user_preferences (
    user_id INTEGER NOT NULL,
    key     TEXT    NOT NULL,
    value   JSONB   NOT NULL,
    PRIMARY KEY (user_id, key)
);
