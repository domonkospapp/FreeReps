---
name: verify
description: Run the checks that have to pass before a FreeReps commit lands — Go build, vet, tests and golangci-lint, the frontend type check and build, and the document contract check. Triggers — "verify", "prüf das durch", "vor dem commit", "läuft das durch", "check before committing", "run the checks", "does CI pass". Not for running the server itself — that is `server/CLAUDE.md`.
---

# Verify

Run from the repo root. The order below is cheapest-first: a failure in step 1
makes the rest irrelevant.

## 1. Go

```bash
mkdir -p server/web/dist && touch server/web/dist/.gitkeep   # only if dist is absent
cd server && go build ./... && go vet ./... && go test ./...
```

The stub matters: `server/web.go` embeds `web/dist` via `go:embed`, and without
the directory the build fails with an embed error that names the directive
rather than the missing directory.

`go test ./...` skips the integration tests. Those need a running TimescaleDB and
run with `go test -tags integration ./...` — run them when the change touches
`server/internal/storage/` or a migration, because the unit tests do not execute
a single SQL statement against a real server.

## 2. golangci-lint

```bash
cd server && golangci-lint run ./...
```

CI pins v2.10.1 (`.forgejo/workflows/ci.yml`). A local version that differs can
pass where CI fails; on a disagreement, the pinned version decides.

## 3. Frontend

```bash
cd server/web && npm ci && npx tsc --noEmit && npm run build
```

`npm run build` runs `tsc -b` itself, so the separate `tsc --noEmit` is only
worth running on its own when you want the type errors without waiting for Vite.

## 4. Documents

```bash
tools/check-docs.sh --all
```

Checks the movement rule and status tokens in `ROADMAP.md`, ISO dates in the
tracked documents, and sweeps the repo for German text. It detects; it does not
prevent. On a German hit that is a verbatim quote of upstream output, add that
specific line to `tools/check-docs.allow` with a reason — do not weaken the
pattern in the script.

## What CI repeats, and what it does not

`.forgejo/workflows/ci.yml` runs steps 1 through 4 on every push and PR against
`main`. It does **not** run the integration tests, and it does not build the iOS
app — that happens on GitHub via the mirror, in `.github/workflows/ios.yml`.

A green CI run on `main` continues into build and deploy to `freereps-lxc`. A
failure after the build stage means `:edge` may be in a broken state; the ntfy
alert in `notify-deploy-failure` carries the run URL.

## Tests

A new test carries a doc comment saying **why the test exists** — which failure
it would catch. *Why:* a test named `TestParseSet` documents nothing; the
regression tests in `server/internal/ingest/alpha/parser_test.go` exist because
of a specific silent data loss, and that is the part worth keeping.
