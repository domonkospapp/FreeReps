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

## Operations

| Status | Item | Where | Trigger | Notes |
|---|---|---|---|---|
| `[open]` | Tear down the App Store review test server | `https://freereps-test.meltforce.net/` | App Store approval received | Public-facing instance without Tailscale, deployed for review only. It carries demo data, not real health data, but it is the one FreeReps endpoint reachable outside the tailnet. |
