-- Second run of the rule in 000027, for the 46 sessions it could not reach.
--
-- 000027 removed the sessions that the two differently-timezoned imports had
-- stored twice, up to 2026-02-19. The 46 sessions imported after that date were
-- written by the container alone, in UTC, so they had no second copy to delete —
-- they were simply an hour or two late, and the script
-- server/scripts/2026-08-10-alpha-utc-session-times.sql was written to move them
-- onto the same Europe/Berlin reading as the rest of the table.
--
-- Before that script ran, the export was imported again through the fixed
-- importer. It read the file in Europe/Berlin, so for those 46 sessions it wrote
-- the correct instants as new rows — 1067 of them — beside the UTC-read rows
-- already stored, which is the duplication of INCIDENTS.md 2026-08-10 in its
-- narrow form. It also brought in one genuinely new session, 2026-08-08, and
-- re-confirmed the 117 sessions below the cutoff, which conflicted as they
-- should and were not written again.
--
-- That leaves the stale UTC-read copy of each of the 46 as the thing to delete,
-- which is exactly what the rule below does: keep the earliest of two sessions
-- whose name and full set signature are identical. The correction the script
-- would have applied has in effect already happened, by import rather than by
-- UPDATE, so the script is removed in the same change.
--
-- On the deployed database this deleted 1067 set rows in 46 sessions, of which
-- 876 working sets, spanning 2026-02-21 to 2026-08-06. Every group held exactly
-- two copies and the copies differed in no column outside the signature, both
-- checked before running. A database without the defect is unaffected.

DO $$
DECLARE
    doomed_sessions BIGINT;
    doomed_rows     BIGINT;
BEGIN
    CREATE TEMP TABLE alpha_dedupe_doomed_2 ON COMMIT DROP AS
    WITH sessions AS (
        SELECT user_id, source, session_name, session_date,
               md5(string_agg(
                     exercise_name || '|' || exercise_number || '|' || set_number || '|' ||
                     coalesce(weight_kg::text, '~') || '|' || reps || '|' ||
                     coalesce(rir::text, '~') || '|' || coalesce(effort_rir::text, '~') || '|' ||
                     is_warmup || '|' || is_bodyweight_plus,
                     E'\n' ORDER BY exercise_number, is_warmup DESC, set_number, exercise_name)) AS sig
        FROM workout_sets
        GROUP BY user_id, source, session_name, session_date
    ),
    ranked AS (
        SELECT user_id, source, session_name, session_date,
               first_value(session_date) OVER (
                   PARTITION BY user_id, source, session_name, sig
                   ORDER BY session_date) AS keeper
        FROM sessions
    )
    SELECT user_id, source, session_name, session_date
    FROM ranked
    WHERE session_date <> keeper
      AND session_date <= keeper + INTERVAL '24 hours';

    SELECT count(*) INTO doomed_sessions FROM alpha_dedupe_doomed_2;

    DELETE FROM workout_sets w
    USING alpha_dedupe_doomed_2 d
    WHERE w.user_id      = d.user_id
      AND w.source       = d.source
      AND w.session_name = d.session_name
      AND w.session_date = d.session_date;

    GET DIAGNOSTICS doomed_rows = ROW_COUNT;
    RAISE NOTICE 'dedupe_alpha_sessions_after_reimport: deleted % set rows in % duplicate sessions',
        doomed_rows, doomed_sessions;
END $$;
