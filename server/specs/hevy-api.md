# Hevy REST API — Wire Format Spec

Derived from the OpenAPI 3.0 specification served at
<https://api.hevyapp.com/docs/> (title "Hevy API Docs", version 0.0.1), read on
2026-08-04. Only the parts FreeReps consumes are documented here.

## Overview

- Base URL: `https://api.hevyapp.com`
- Auth: header `api-key: <key>`, created at <https://hevy.com/settings?developer>
- API access requires an active **Hevy Pro** subscription; without one every
  request is rejected.
- The key is account-wide and cannot be scoped. It grants read and write access
  to every endpoint, including creating workouts and replacing routines.
- Rate limits are not documented in the specification.

## Endpoints FreeReps uses

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/v1/workouts` | Full list; carries the initial backfill |
| `GET` | `/v1/workouts/events` | Delta feed; every sync after the first |
| `GET` | `/v1/user/info` | Validates an API key before it is stored |

**The event feed does not serve the backfill.** Its own summary says it exists so
a client can keep an existing local cache up to date "without having to fetch the
entire list of workouts". Called with a fresh API key and no `since` bound it
returns zero events even when the account holds workouts — the first deployment
imported nothing while a logged test workout sat in the account, and the import
log reported success because no error had occurred. The first sync therefore
walks `/v1/workouts`, and only later runs use the feed.

| `GET` | `/v1/exercise_templates` | Exercise catalog; the only source of muscle groups |

Not consumed yet: `/v1/routines`, `/v1/exercise_history/{id}`,
`/v1/body_measurements`.

## `GET /v1/exercise_templates`

Query parameters: `page`, `pageSize` (up to **100**, ten times what the workout
endpoints allow). Returns `{ page, page_count, exercise_templates[] }`.

```json
{
  "id": "79D0BB3A",
  "title": "Bench Press (Barbell)",
  "type": "weight_reps",
  "primary_muscle_group": "chest",
  "secondary_muscle_groups": ["triceps", "shoulders"],
  "equipment_category": "",
  "is_custom": false
}
```

`primary_muscle_group` and `secondary_muscle_groups` draw on a fixed vocabulary
of 20 values (`chest`, `lats`, `upper_back`, `quadriceps`, `glutes`, `traps`, …).
This is the only place muscle data comes from; FreeReps derives none of it.

**`equipment_category` arrives empty.** The specification declares it as an enum
and the schema documents nine values, but all 451 entries returned on 2026-08-04
carried an empty string. The equipment is instead encoded in the title —
`(Barbell)`, `(Dumbbell)`, `(Smith Machine)`. Do not rely on the field without
checking it again.

## Pagination

Both `/v1/workouts` and `/v1/workouts/events` cap `pageSize` at **10**; page
numbering starts at 1. The response carries `page` and `page_count`; iteration
stops once `page_count <= page`. `/v1/exercise_templates` allows up to 100.

## `GET /v1/workouts`

Query parameters: `page`, `pageSize`. Returns `{ page, page_count, workouts[] }`
with the same workout objects the event feed carries, newest first.

## `GET /v1/workouts/events`

Query parameters: `page`, `pageSize`, `since` (ISO 8601). Events are ordered
newest first and cover changes made after the key started observing — see the
note above on why this is not a backfill path.

```json
{
  "page": 1,
  "page_count": 3,
  "events": [
    { "type": "updated", "workout": { "...": "see below" } },
    { "type": "deleted", "id": "efe6801c-…", "deleted_at": "2026-08-04T11:00:00Z" }
  ]
}
```

The two event shapes are a `oneOf` in the specification. `updated` carries a
complete workout object; `deleted` carries only the id and a timestamp.

**An `updated` event fires for corrections as well as for new workouts.** A set
edited in the app days later reappears with the same workout id. The ingest
therefore deletes every row for that `external_id` before re-inserting, because
an upsert alone would neither apply a changed weight nor remove a deleted set.

## Workout object

```json
{
  "id": "b459cba5-cd6d-463c-abd6-54f8eafcadcb",
  "title": "Morning Workout",
  "routine_id": "b459cba5-…",
  "description": "Pushed myself today",
  "start_time": "2026-08-04T09:00:00Z",
  "end_time": "2026-08-04T10:15:00Z",
  "updated_at": "2026-08-04T10:15:00Z",
  "created_at": "2026-08-04T10:15:00Z",
  "exercises": [
    {
      "index": 0,
      "title": "Bench Press (Barbell)",
      "notes": "form felt good",
      "exercise_template_id": "05293BCA",
      "supersets_id": null,
      "sets": [
        {
          "index": 0,
          "type": "normal",
          "weight_kg": 100,
          "reps": 10,
          "distance_meters": null,
          "duration_seconds": null,
          "rpe": 9.5,
          "custom_metric": null
        }
      ]
    }
  ]
}
```

`routine_id` names the routine the session was started from, which is what makes
a comparison of prescribed against performed possible.

`index` is zero-based on both exercises and sets. FreeReps stores
`exercise_number` and `set_number` one-based, matching Alpha Progression.

## Set types

`type` is one of `normal`, `warmup`, `failure`, `dropset`. FreeReps keeps the
raw value in `set_type` and additionally sets `is_warmup` for `warmup`, because
every existing aggregate query filters on that column.

## RPE and RIR

Hevy records **RPE** (Rating of Perceived Exertion); the Alpha Progression
history records **RIR** (Reps in Reserve). Both are stored on their own scale in
their own column, and the database derives `effort_rir` from whichever is
present — see `database-schema.md`.

The two are arithmetic complements (`rir = 10 - rpe`) but not the same judgement:
RPE rates how hard a set felt, RIR estimates what was left in the tank.
Converting at ingest would normalize a source at write time, which
`DECISIONS.md` 2026-03-25 rules out.

When writing, the API restricts RPE to the enum `6, 7, 7.5, 8, 8.5, 9, 9.5, 10`,
which covers RIR 0 through 4 on a half-point grid. Reading returns a plain
number. `rpe` is `null` for an unrated set, and that null is preserved — a
converted 0 would claim the set was taken to failure.

## Null fields

Every value field on a set is nullable. A plank has `duration_seconds` but no
`reps`; a bodyweight set has `reps` but no `weight_kg`. `custom_metric` carries
floors or steps for stair machine exercises.

## What the workout object does not contain

- **`rest_seconds`** — rest is a prescription and lives on the routine, not on
  the completed workout.
- **Equipment** — an attribute of the exercise template, fetched separately.
- **Target reps or rep ranges** — likewise part of the routine.

## `GET /v1/user/info`

```json
{ "id": "9c465af3-…", "name": "John Doe", "url": "https://hevy.com/user/john" }
```

Used solely to verify a key before storing it, so a wrong key fails in the form
rather than silently hours later in a sync log.
