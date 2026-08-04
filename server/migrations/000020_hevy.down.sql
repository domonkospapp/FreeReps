-- Restore the `workouts` rows the up migration moved aside, then drop the backup.
INSERT INTO workouts SELECT * FROM workouts_alpha_orphans_backup
    ON CONFLICT DO NOTHING;
DROP TABLE IF EXISTS workouts_alpha_orphans_backup;

DROP TABLE IF EXISTS hevy_sync_state;
DROP TABLE IF EXISTS hevy_credentials;

DROP INDEX IF EXISTS idx_workout_sets_source_external;

ALTER TABLE workout_sets DROP CONSTRAINT IF EXISTS workout_sets_source_natural_key;

-- Restoring the original constraint requires the rows to be unique without the
-- source column. Delete everything that is not Alpha Progression first.
DELETE FROM workout_sets WHERE source <> 'Alpha Progression';

-- The name differs from the one PostgreSQL generated for the inline constraint
-- in 000001_init.up.sql; the column set is what matters.
ALTER TABLE workout_sets
    ADD CONSTRAINT workout_sets_natural_key
    UNIQUE (user_id, session_date, exercise_number, set_number, is_warmup);

ALTER TABLE workout_sets
    DROP COLUMN IF EXISTS source,
    DROP COLUMN IF EXISTS external_id,
    DROP COLUMN IF EXISTS routine_id,
    DROP COLUMN IF EXISTS exercise_template_id,
    DROP COLUMN IF EXISTS set_type,
    DROP COLUMN IF EXISTS superset_id,
    DROP COLUMN IF EXISTS exercise_notes,
    DROP COLUMN IF EXISTS session_end,
    DROP COLUMN IF EXISTS distance_m,
    DROP COLUMN IF EXISTS duration_sec,
    DROP COLUMN IF EXISTS custom_metric;
