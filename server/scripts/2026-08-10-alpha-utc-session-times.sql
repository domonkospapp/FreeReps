-- One-time correction: re-read the UTC-imported Alpha session times in
-- Europe/Berlin. Run once, against the deployed database, after migration
-- 000027 has removed the duplicate sessions.
--
--   ssh root@freereps-lxc \
--     'docker exec -i freereps-db-1 psql -U freereps -d freereps' \
--     < server/scripts/2026-08-10-alpha-utc-session-times.sql
--
-- Why this is a script and not a migration: the rule it applies is true of this
-- database's import history, not of the schema. INCIDENTS.md 2026-08-10
-- describes the two importing hosts — sessions imported before 2026-02-20 came
-- from a machine in Europe/Berlin and are already correct; everything after was
-- imported by the container, which has no /etc/localtime, so time.Local was UTC
-- and the export's wall clock was stored as though it were a UTC instant. A
-- migration would have to encode "the rows this one deployment imported after
-- this one date", which is not a property any other installation shares.
--
-- Leaving these rows alone was not an option: with the importer now reading
-- Europe/Berlin, re-importing the same export would insert a second copy of
-- each of them and reproduce the incident.
--
-- Expected on the deployed database: 46 sessions, 1067 set rows, between
-- 2026-02-21 and 2026-08-06, each moving 3600 s (CET) or 7200 s (CEST) earlier.
--
-- Apple Health is the reference, not the selection rule. Its workouts carry a
-- real zone. Measured on the rehearsal copy before this script ran: of the 46
-- candidates, 0 were within 30 minutes of that day's nearest
-- Traditional Strength Training start and 45 are afterwards; of the 117
-- sessions below the cutoff, all 114 that have such a workout already agree.
-- 2026-06-12 is the single session that agrees with neither reading — its
-- logged time is 09:00 against a workout starting 07:43 — and it is corrected
-- with the rest, because its provenance is the same. The final report names it.

\pset pager off
\set ON_ERROR_STOP on

BEGIN;

CREATE TEMP TABLE alpha_tz_fix ON COMMIT DROP AS
SELECT DISTINCT user_id, source, session_date,
       ((session_date AT TIME ZONE 'UTC') AT TIME ZONE 'Europe/Berlin') AS corrected
FROM workout_sets
WHERE source = 'Alpha Progression'
  AND session_date >= TIMESTAMPTZ '2026-02-20';

-- Agreement with the nearest Apple strength workout of the same day, before and
-- after. "Nearest" and not "earliest": 2025-04-03 holds an abandoned 10-minute
-- workout at 04:15 alongside the real one at 06:23, and matching on the earlier
-- of the two would call a correct session wrong.
CREATE TEMP VIEW alpha_tz_check AS
SELECT f.*, n.apple_start,
       round(extract(epoch FROM (f.session_date - n.apple_start)) / 60) AS delta_now_min,
       round(extract(epoch FROM (f.corrected - n.apple_start)) / 60)    AS delta_after_min
FROM alpha_tz_fix f
LEFT JOIN LATERAL (
    SELECT w.start_time AS apple_start
    FROM workouts w
    WHERE w.user_id = f.user_id
      AND w.name = 'Traditional Strength Training'
      AND (w.start_time AT TIME ZONE 'UTC')::date = (f.session_date AT TIME ZONE 'UTC')::date
    ORDER BY abs(extract(epoch FROM (w.start_time - f.corrected)))
    LIMIT 1
) n ON TRUE;

\echo '=== candidates ==='
SELECT count(*) AS sessions,
       min(session_date) AS first, max(session_date) AS last,
       string_agg(DISTINCT extract(epoch FROM (session_date - corrected))::text, ', ') AS shift_seconds
FROM alpha_tz_fix;

\echo '=== agreement with Apple Health: now vs after ==='
SELECT count(*) AS candidates,
       count(*) FILTER (WHERE apple_start IS NULL)                AS without_apple_workout,
       count(*) FILTER (WHERE abs(delta_now_min)   <= 30)         AS agree_now,
       count(*) FILTER (WHERE abs(delta_after_min) <= 30)         AS agree_after
FROM alpha_tz_check;

DO $$
DECLARE
    candidates BIGINT;
    checkable  BIGINT;
    agree_now  BIGINT;
    collisions BIGINT;
    moved_rows BIGINT;
BEGIN
    SELECT count(*),
           count(*) FILTER (WHERE apple_start IS NOT NULL),
           count(*) FILTER (WHERE apple_start IS NOT NULL AND abs(delta_now_min) <= 30)
      INTO candidates, checkable, agree_now
    FROM alpha_tz_check;

    IF candidates = 0 THEN
        RAISE EXCEPTION 'aborting: no candidate sessions — has migration 000027 been applied to this database?';
    END IF;

    -- Re-run guard. Before the correction none of the candidates agree with
    -- Apple Health; afterwards nearly all do, so a second run stops here rather
    -- than shifting the same sessions an hour further.
    IF checkable > 0 AND agree_now * 10 > checkable THEN
        RAISE EXCEPTION 'aborting: % of % candidates already agree with Apple Health — this correction looks applied',
            agree_now, checkable;
    END IF;

    SELECT count(*) INTO collisions
    FROM alpha_tz_fix f
    WHERE EXISTS (SELECT 1 FROM workout_sets w
                  WHERE w.user_id = f.user_id AND w.source = f.source
                    AND w.session_date = f.corrected);
    IF collisions > 0 THEN
        RAISE EXCEPTION 'aborting: % corrected timestamps collide with an existing session', collisions;
    END IF;

    UPDATE workout_sets w
    SET session_date = f.corrected
    FROM alpha_tz_fix f
    WHERE w.user_id = f.user_id AND w.source = f.source AND w.session_date = f.session_date;

    GET DIAGNOSTICS moved_rows = ROW_COUNT;
    RAISE NOTICE 'alpha-utc-session-times: moved % set rows in % sessions', moved_rows, candidates;
END $$;

\echo '=== sessions still more than 30 minutes from their Apple workout ==='
WITH s AS (
    SELECT DISTINCT user_id, session_date FROM workout_sets WHERE source = 'Alpha Progression')
SELECT s.session_date, n.apple_start,
       round(extract(epoch FROM (s.session_date - n.apple_start)) / 60) AS delta_min
FROM s
JOIN LATERAL (
    SELECT w.start_time AS apple_start
    FROM workouts w
    WHERE w.user_id = s.user_id
      AND w.name = 'Traditional Strength Training'
      AND (w.start_time AT TIME ZONE 'UTC')::date = (s.session_date AT TIME ZONE 'UTC')::date
    ORDER BY abs(extract(epoch FROM (w.start_time - s.session_date)))
    LIMIT 1
) n ON TRUE
WHERE abs(extract(epoch FROM (s.session_date - n.apple_start))) > 1800
ORDER BY 1;

COMMIT;
