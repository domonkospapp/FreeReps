# Incidents

Postmortems for things that broke. One section per incident, newest first.

Add an entry after fixing something that was not obvious — the kind of failure
where the useful question six months later is "have I seen this before?". Skip
routine config changes, dependency bumps, and one-line typos.

Structure per entry: **symptoms** (what was visible), **root cause** (concrete:
component, version, why), **fix** (what changed), **lesson** (one line,
actionable).

The entries below were written on 2026-08-04 from the commit history. Their
symptoms are as recorded in the commit messages; where a commit did not say how
the fix was verified, this file does not claim it was.

---

## 2026-04-08 — Alpha Progression sets with "N+" notation were dropped or read as zero

**Symptoms.** Strength training sets imported from Alpha Progression CSV were
missing from the workout detail view, and some that did arrive carried an RIR of
0. The import reported success and no error; the set count was simply lower than
the CSV's.

**Root cause.** Alpha Progression writes `5+` for "at least 5" in the reps, RIR
and weight columns. Two code paths in `server/internal/ingest/alpha/parser.go`
handled it differently, and both silently:

- `parseEuropeanFloat` returned 0 for `5+`, so an RIR of "5 or more" became 0 —
  the value that means "to failure".
- The reps column was matched by a strict `\d+` regex, which rejected `5+`, and
  the rejection dropped the whole set line rather than the one field.

**Fix.** The trailing `+` is stripped before parsing, so the numeric value
survives in both paths (commit `bb929f1`). The commit adds cases to
`parser_test.go` covering the notation in each column.

**Lesson.** A parser that returns a zero value on a format it does not know
turns a format gap into wrong data; return an error or normalize explicitly.

---

## 2026-03-26 — Sleep durations of 22 hours and more after the Oura integration

**Symptoms.** Sleep sessions showed impossible durations — 22 hours and above —
for nights where both Oura and Health Auto Export had delivered data. Correct
Oura sessions had been present earlier and were overwritten.

**Root cause.** The sleep session backfill in `server/internal/storage/sleep.go`
read all rows in `sleep_stages` for a date regardless of source and summed their
durations. With two sources reporting the same night, every stage was counted
twice. The backfill then wrote the result over the session a direct source had
already produced.

**Fix.** Commit `731b001`. The backfill uses `ON CONFLICT DO NOTHING`, so it only
creates sessions for dates that have none, and never overwrites a direct source.
Oura sync writes `sleep_analysis` metrics itself, so the dashboard chart no
longer depends on the backfill running.

**Lesson.** A derivation that aggregates across sources needs a source filter at
the point it reads, not a priority applied afterwards.

---

## 2026-03-25 — Respiratory rate 60× too high, SpO2 shown as 0.94 for one source and 94.5 for the other

**Symptoms.** After the Oura integration, respiratory rate read around 900 per
minute. SpO2 rendered as `0.94` for Apple Health values and `94.5` for Oura
values in the same chart.

**Root cause.** Two unit mismatches in `server/internal/oura/mapper.go`:

- The Oura OpenAPI specification documents `average_breath` as breaths per
  second. It is breaths per minute. The ingest applied a ×60 conversion the data
  did not need.
- Apple Health stores SpO2 as a fraction (`0.94`), Oura as a percentage
  (`94.5`). Neither was normalized, so both landed in the same metric.

**Fix.** Commit `5554cd9` removes the ×60 conversion, normalizes Oura SpO2 to a
fraction on ingest, and adds a `display_multiplier` of ×100 for presentation.
Migration `000019_fix_spo2_multiplier` corrects the rows already stored.

**Lesson.** Vendor documentation is a claim about units, not evidence — compare
a known value against the vendor's own app before trusting a conversion factor,
and store one unit per metric across all sources.
