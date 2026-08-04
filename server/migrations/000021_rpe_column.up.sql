-- Keep RPE and RIR as separate raw values.
--
-- Migration 000020 shipped a Hevy ingest that converted RPE to RIR while
-- writing, on the grounds that the Alpha history and every intensity query use
-- RIR. That normalises a source at write time, which DECISIONS.md 2026-03-25
-- rules out: sources keep their own rows and are resolved in the query path.
--
-- The scales are complements arithmetically, but not the same judgement. RPE
-- rates perceived exertion, RIR estimates repetitions left in the tank.

ALTER TABLE workout_sets ADD COLUMN IF NOT EXISTS rpe DOUBLE PRECISION;

-- One definition of "how hard was this set" for every query that bands or
-- weights effort, so the conversion lives in a single place instead of being
-- repeated in each WHERE clause.
--
-- Alpha Progression writes rir and uses -1 for an unrated set; Hevy writes rpe
-- and leaves rir NULL. NULL here means the set carries no rating at all, which
-- is what lets the intensity bands stop treating -1 as a magic number.
ALTER TABLE workout_sets ADD COLUMN IF NOT EXISTS effort_rir DOUBLE PRECISION
    GENERATED ALWAYS AS (COALESCE(NULLIF(rir, -1), 10 - rpe)) STORED;
