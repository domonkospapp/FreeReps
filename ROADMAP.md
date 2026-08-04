# Roadmap

**This file contains open work only.** Every row carries the status token
`[open]`. Closed work is not struck through here — it is removed and lives in
[`DECISIONS.md`](DECISIONS.md) (decisions, with their reasoning) or
[`INCIDENTS.md`](INCIDENTS.md) (postmortems).

Columns: **Status** is always `[open]`. **Where** names the artifact the work
touches. **Trigger** carries the condition for items that are deliberately
deferred, and is empty for items that are simply pending. **Notes** carries the
reasoning.

Before closing an item, check its entry for residual work, dates or triggers —
each becomes its own `[open]` row before the entry is moved out.

## Dashboard

| Status | Item | Where | Trigger | Notes |
|---|---|---|---|---|
| `[open]` | Trend views | `server/web/src/` | | Rolling averages and period comparison per metric. The correlation explorer answers "do these two move together"; a trend view answers "where is this one going". |
| `[open]` | Saveable dashboard configurations | `server/web/src/`, `server/migrations/` | | Metric selection and time range are currently per-session. Persisting them needs a per-user table, like `metric_visibility` already has. |
| `[open]` | Responsive optimization | `server/web/src/` | | The dashboard is laid out for a desktop viewport. The iOS app covers phone use for sync, not for reading charts. |

## Training data

| Status | Item | Where | Trigger | Notes |
|---|---|---|---|---|
| `[open]` | Exercise catalog with muscle groups | `server/migrations/`, `server/internal/hevy/` | | Pull `GET /v1/exercise_templates` into its own table. It carries `primary_muscle_group`, `secondary_muscle_groups` and `equipment_category` for every exercise — the mapping that weekly volume per muscle group needs and that no table holds today. |
| `[open]` | Training volume aggregates | `server/internal/storage/training_volume.go` | Exercise catalog exists | Weekly sets per muscle group, e1RM per exercise, frequency per muscle. Plus tonnage written as a derived series into `health_metrics`, which is what makes training correlatable against HRV and sleep — `GetCorrelation` reads that table exclusively. |
| `[open]` | Rename the colliding MCP tools | `server/internal/mcp/tools.go` | | `get_training_summary` and `get_training_intensity` sit one character away from `hevy-mcp`'s `get-training-summary`. Give them names that state their data basis, and sharpen the descriptions. |
| `[open]` | Planning project | outside this repo | Volume aggregates exist | Separate Claude Code project with both MCP servers, permissions denying the Hevy analysis tools. See [`DECISIONS.md`](DECISIONS.md), 2026-08-04. |
| `[open]` | Backfill the Alpha history into Hevy | outside this repo | | Roughly the last 8 to 12 weeks via `POST /v1/workouts`, so the app shows previous-session values from day one. Set `is_private` on the created workouts. The ingest cutoff in `hevy_credentials.sync_from` keeps them from flowing back. |
| `[open]` | Source deduplication in the training aggregates | `server/internal/storage/training_summary.go`, `training_intensity.go` | Two sources cover one period | Both queries sum across `workout_sets` without filtering on `source`. The cutoff currently keeps the periods disjoint; if they ever overlap, tonnage and set counts double. Same failure shape as [`INCIDENTS.md`](INCIDENTS.md), 2026-03-26. |
| `[open]` | Detail view for sessions without a workout row | `server/web/src/pages/WorkoutDetailPage.tsx` | | Alpha and Hevy sessions render from route state because `GET /api/v1/workouts/{id}` cannot resolve a synthetic id. Reloading such a page loses the data. Hevy now supplies a stable `external_id` to resolve against. |

## Operations

| Status | Item | Where | Trigger | Notes |
|---|---|---|---|---|
| `[open]` | Tear down the App Store review test server | `https://freereps-test.meltforce.net/` | App Store approval received | Public-facing instance without Tailscale, deployed for review only. It carries demo data, not real health data, but it is the one FreeReps endpoint reachable outside the tailnet. |
