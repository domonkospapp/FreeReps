-- Remove sessions stored more than once by the timezone-dependent Alpha import.
--
-- Until the fix that accompanies this migration, the Alpha CSV's bare wall clock
-- was read in time.Local. The resulting instant is part of
-- workout_sets_source_natural_key, so importing one export from a host in
-- Europe/Berlin and again from the container — which has no /etc/localtime and
-- runs in UTC — inserted every session twice, an hour or two apart.
-- INCIDENTS.md, 2026-08-10.
--
-- The rule matches on content, never on the offset. Two sessions are the same
-- session only when user, source and name agree and the full ordered list of
-- their sets is identical, warm-ups included. An offset-based rule would delete
-- two genuinely different sessions that happen to sit an hour apart on one day.
--
-- The 24-hour bound keeps the rule from merging a workout that was genuinely
-- repeated, set for set, on a later day: a zone shift moves a session by at most
-- the difference between two UTC offsets, never by a day.
--
-- On the deployed database this deleted 117 sessions and 2936 set rows, of which
-- 2367 working sets, spanning 2025-03-18 to 2026-02-19. Every group held exactly
-- two copies, and the copies differed in no column outside the signature. A
-- database without the defect is unaffected: with no matching pair, nothing is
-- deleted.

DO $$
DECLARE
    doomed_sessions BIGINT;
    doomed_rows     BIGINT;
BEGIN
    CREATE TEMP TABLE alpha_dedupe_doomed ON COMMIT DROP AS
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

    SELECT count(*) INTO doomed_sessions FROM alpha_dedupe_doomed;

    DELETE FROM workout_sets w
    USING alpha_dedupe_doomed d
    WHERE w.user_id      = d.user_id
      AND w.source       = d.source
      AND w.session_name = d.session_name
      AND w.session_date = d.session_date;

    GET DIAGNOSTICS doomed_rows = ROW_COUNT;
    RAISE NOTICE 'dedupe_alpha_sessions: deleted % set rows in % duplicate sessions',
        doomed_rows, doomed_sessions;
END $$;
