-- Hevy integration: second source for strength training sets.
--
-- Until now workout_sets held Alpha Progression data exclusively and carried no
-- source column. Hevy delivers the same kind of data plus fields Alpha does not
-- have (stable workout id, routine reference, set type, exercise template id).

-- ------------------------------------------------------------
-- workout_sets: source tracking and Hevy-specific fields
-- ------------------------------------------------------------

ALTER TABLE workout_sets
    ADD COLUMN IF NOT EXISTS source               TEXT    NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS external_id          TEXT    NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS routine_id           TEXT    NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS exercise_template_id TEXT    NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS set_type             TEXT    NOT NULL DEFAULT 'normal',
    ADD COLUMN IF NOT EXISTS superset_id          INTEGER,
    ADD COLUMN IF NOT EXISTS exercise_notes       TEXT    NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS session_end          TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS distance_m           DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS duration_sec         DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS custom_metric        DOUBLE PRECISION;

-- Every existing row came from the Alpha Progression CSV import.
UPDATE workout_sets SET source = 'Alpha Progression' WHERE source = '';

-- The original constraint was declared inline in 000001_init.up.sql and carries
-- a name PostgreSQL generated and truncated to 63 characters. Matching on the
-- name or on the exact definition string would break as soon as either differs,
-- so drop every unique constraint on the table instead — there is exactly one,
-- and the primary key has contype 'p' and is not touched.
DO $$
DECLARE
    con RECORD;
BEGIN
    FOR con IN
        SELECT conname
        FROM pg_constraint
        WHERE conrelid = 'workout_sets'::regclass
          AND contype = 'u'
          AND conname <> 'workout_sets_source_natural_key'
    LOOP
        EXECUTE format('ALTER TABLE workout_sets DROP CONSTRAINT %I', con.conname);
    END LOOP;
END $$;

ALTER TABLE workout_sets
    ADD CONSTRAINT workout_sets_source_natural_key
    UNIQUE (user_id, source, session_date, exercise_number, set_number, is_warmup);

-- Drives DeleteWorkoutSetsByExternalID, which runs before every re-insert of an
-- updated Hevy workout.
CREATE INDEX IF NOT EXISTS idx_workout_sets_source_external
    ON workout_sets (source, external_id);

-- ------------------------------------------------------------
-- Hevy credentials and sync state
-- ------------------------------------------------------------

-- One row per user. The API key is a single account-wide token; Hevy offers no
-- scoping. sync_from bounds the ingest: workouts that started before this date
-- are discarded, so an Alpha history later imported into Hevy cannot flow back
-- and double-count against the rows already in workout_sets.
CREATE TABLE hevy_credentials (
    user_id    INTEGER     NOT NULL PRIMARY KEY REFERENCES users(id),
    api_key    TEXT        NOT NULL,
    sync_from  DATE        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- last_event_at is the `since` parameter of the next GET /v1/workouts/events
-- call. Unlike Oura this is a timestamp, because the endpoint filters on event
-- time rather than on calendar days.
CREATE TABLE hevy_sync_state (
    user_id       INTEGER     NOT NULL PRIMARY KEY REFERENCES users(id),
    last_event_at TIMESTAMPTZ NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ------------------------------------------------------------
-- Clean up orphaned Alpha workout rows
-- ------------------------------------------------------------

-- Commit 7060d9d made the Alpha import write rows into `workouts`; commit
-- 411f4c1 reverted that 3.5 hours later in favour of query-time synthesis, but
-- the rows written in between stayed. Their UUID collides with the synthetic id
-- QueryWorkoutsMerged derives for the same session, so the same session can be
-- served from two different places depending on which path produced it.
--
-- No active code path writes this source value: Apple Health leaves source
-- empty, Oura writes 'Oura', the demo seed writes 'demo'. Every row matched here
-- is therefore residue of the reverted feature.
--
-- The rows are copied before deletion rather than dropped outright. Their exact
-- number is not known ahead of time — the API merges real and synthetic entries,
-- so they cannot be counted from outside the database — and a backup table makes
-- the change both auditable and reversible.
CREATE TABLE IF NOT EXISTS workouts_alpha_orphans_backup AS
    SELECT * FROM workouts WHERE source = 'Alpha Progression';

DELETE FROM workouts WHERE source = 'Alpha Progression';
