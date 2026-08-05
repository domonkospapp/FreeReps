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
| `[open]` | Tonnage ignores body weight | `server/internal/storage/training_metrics.go`, `training_summary.go`, `training_volume.go` | | `SUM(weight_kg * reps)` counts external load only. 827 working sets are body-weight-plus, where Alpha recorded just the added weight: at 81.2 kg and 8285 reps that is 672,742 kg uncounted, about 31 % of total tonnage. A further 292 sets carry no weight at all — hanging leg raises, chin-ups — and contribute zero. Tonnage is therefore a relative measure at a stable exercise mix, not an absolute workload. A fix needs a decision on which body weight to use: the measurement nearest the session, or a rolling average. |
| `[open]` | Planning project | outside this repo | | Separate Claude Code project with both MCP servers, permissions denying the Hevy analysis tools. See [`DECISIONS.md`](DECISIONS.md), 2026-08-04. |
| `[open]` | Backfill the Alpha history into Hevy | outside this repo | | Roughly the last 8 to 12 weeks via `POST /v1/workouts`, so the app shows previous-session values from day one. Set `is_private` on the created workouts. The ingest cutoff in `hevy_credentials.sync_from` keeps them from flowing back. |
| `[open]` | Source deduplication in the training aggregates | `server/internal/storage/training_summary.go`, `training_intensity.go` | Two sources cover one period | Both queries sum across `workout_sets` without filtering on `source`. The cutoff currently keeps the periods disjoint; if they ever overlap, tonnage and set counts double. Same failure shape as [`INCIDENTS.md`](INCIDENTS.md), 2026-03-26. |
| `[open]` | Detail view for sessions without a workout row | `server/web/src/pages/WorkoutDetailPage.tsx` | | Alpha and Hevy sessions render from route state because `GET /api/v1/workouts/{id}` cannot resolve a synthetic id. Reloading such a page loses the data. Hevy now supplies a stable `external_id` to resolve against. |

## Operations

| Status | Item | Where | Trigger | Notes |
|---|---|---|---|---|
| `[open]` | Tear down the App Store review test server | `https://freereps-test.meltforce.net/` | App Store approval received | Public-facing instance without Tailscale, deployed for review only. It carries demo data, not real health data, but it is the one FreeReps endpoint reachable outside the tailnet. |
