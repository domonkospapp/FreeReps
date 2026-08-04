-- Exercise catalog, the prerequisite for volume per muscle group.
--
-- No table in this schema carries a muscle. Hevy's GET /v1/exercise_templates
-- supplies one primary and any number of secondary muscle groups per exercise,
-- from a fixed 20-value vocabulary, plus the equipment category.

CREATE TABLE exercise_templates (
    id                      TEXT        PRIMARY KEY,   -- Hevy exercise_template_id
    title                   TEXT        NOT NULL,
    exercise_type           TEXT        NOT NULL DEFAULT '',
    primary_muscle_group    TEXT        NOT NULL DEFAULT '',
    secondary_muscle_groups TEXT[]      NOT NULL DEFAULT '{}',
    equipment_category      TEXT        NOT NULL DEFAULT '',
    is_custom               BOOLEAN     NOT NULL DEFAULT FALSE,
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Maps a free-text exercise name onto the catalog, for sources that carry no
-- template id. Only Alpha Progression needs this; Hevy rows reference the
-- catalog directly through workout_sets.exercise_template_id.
--
-- A NULL exercise_template_id records a name that was looked at and had no
-- counterpart, which is a different statement from a name nobody has examined
-- yet. Volume queries report both as unmapped rather than dropping them.
CREATE TABLE exercise_name_map (
    source               TEXT NOT NULL,
    exercise_name        TEXT NOT NULL,
    exercise_template_id TEXT REFERENCES exercise_templates(id),
    note                 TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (source, exercise_name)
);

-- Every training query filters on user_id as well as the date range; the only
-- existing index covers session_date alone.
CREATE INDEX idx_workout_sets_user_date ON workout_sets (user_id, session_date DESC);
