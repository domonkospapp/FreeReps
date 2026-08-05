-- Withings Public API integration: token storage, sync state, and the body
-- composition metrics the scale reports that no other source delivers.

-- Per-user Withings application credentials and OAuth2 tokens.
-- client_id/client_secret come from the user's app in the Withings Partner Hub.
-- access_token/refresh_token are populated after OAuth2 authorization, so a row
-- can exist with credentials only.
CREATE TABLE IF NOT EXISTS withings_tokens (
    user_id          INTEGER     NOT NULL PRIMARY KEY,
    client_id        TEXT        NOT NULL DEFAULT '',
    client_secret    TEXT        NOT NULL DEFAULT '',
    access_token     TEXT        NOT NULL DEFAULT '',
    refresh_token    TEXT        NOT NULL DEFAULT '',
    token_type       TEXT        NOT NULL DEFAULT 'Bearer',
    expires_at       TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01',
    withings_user_id TEXT        NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Incremental sync tracking. last_update holds the `updatetime` value from the
-- previous getmeas response and is passed back as `lastupdate`; it is a Unix
-- timestamp in seconds, not a date, because the API filters to the second.
CREATE TABLE IF NOT EXISTS withings_sync_state (
    user_id     INTEGER NOT NULL,
    data_type   TEXT    NOT NULL,
    last_update BIGINT  NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, data_type)
);

-- Metrics the Withings scale and blood pressure monitor report that have no
-- allowlist entry yet. They go into the existing body/cardiovascular categories
-- rather than a source-exclusive one: Apple Health can carry the same
-- measurements, so excluding them from source priority would be wrong.
INSERT INTO metric_allowlist (metric_name, category) VALUES
    ('fat_mass',                  'body'),
    ('muscle_mass',               'body'),
    ('bone_mass',                 'body'),
    ('body_water',                'body'),
    ('blood_pressure_heart_rate', 'cardiovascular')
ON CONFLICT (metric_name) DO NOTHING;

UPDATE metric_allowlist SET display_label = 'Fat Mass',    display_unit = 'kg'   WHERE metric_name = 'fat_mass';
UPDATE metric_allowlist SET display_label = 'Muscle Mass', display_unit = 'kg'   WHERE metric_name = 'muscle_mass';
UPDATE metric_allowlist SET display_label = 'Bone Mass',   display_unit = 'kg'   WHERE metric_name = 'bone_mass';
UPDATE metric_allowlist SET display_label = 'Body Water',  display_unit = 'kg'   WHERE metric_name = 'body_water';
UPDATE metric_allowlist SET display_label = 'BP Pulse',    display_unit = 'bpm'  WHERE metric_name = 'blood_pressure_heart_rate';
