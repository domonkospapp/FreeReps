# FreeReps

**FREE Records, Evaluation & Processing Server**

- **Repo**: `FreeReps` (monorepo: `server/` + `app/`)
- **Server binary**: `freereps` (in `server/`)
- **iOS app**: `FreeReps.xcodeproj` (in `app/`)

## Project Vision

FreeReps is a self-hosted server that receives Apple Health data, stores it persistently, visualizes it through a web dashboard with freely configurable relations/correlations, and exposes it as an MCP server for LLMs.

**Core idea**: Collect the data, store it cleanly, visualize it flexibly — and delegate intelligent analysis to Claude (via MCP). No built-in AI coach, no proprietary score algorithms. Instead: maximum transparency and flexibility in data exploration.

**Visual inspiration**: [Athlytic](https://www.athlyticapp.com/) — but as a self-hosted web app, without scores, with free correlation views and an MCP interface instead.

## Data Sources

The iOS app **[Health Auto Export](https://healthyapps.dev)** serves as the bridge between Apple Health and FreeReps.
The iOS app **[Alpha Progression](https://alphaprogression.com/de/)** serves as source for detailed strength training workout data (example data available in input/).

Wire formats, endpoints and payload shapes are specified in `server/specs/`
(`hae-export-format.md`, `hae-rest-api.md`, `alpha-progression.md`,
`database-schema.md`). Architecture, tech stack and dashboard features are
described in `README.md`.

### Reference Documentation

- [REST API Automation](https://help.healthyapps.dev/en/health-auto-export/automations/rest-api/)
- [Server Connection (TCP/MCP)](https://help.healthyapps.dev/en/health-auto-export/automations/server-connection/)
- [Export Formats](https://help.healthyapps.dev/en/health-auto-export/export-format/)
- [GitHub Server (Grafana reference)](https://github.com/HealthyApps/health-auto-export-server)

## Design Principles

- **Privacy first**: All data stays local. No cloud uploads, no telemetry.
- **Self-hosted**: Runs on your own server/homelab (Docker-compatible).
- **Data over scores**: Raw data + visualization + LLM instead of proprietary algorithms.
- **Flexible over opinionated**: Correlation explorer instead of hard-wired dashboards.
- **Single binary** (if possible): `freereps` with embedded web UI.
- **File-based configuration**: YAML/TOML for server, DB, auth, personal parameters (age, HR zones).
- **Idempotent ingest**: Duplicate data → no duplicates stored.

## Architecture Decisions

| Decision | Choice | Rationale |
|---|---|---|
| **Backend** | Go 1.25+ | Single binary, fast development, good concurrency |
| **Frontend** | React 19 + Vite + Tailwind CSS 4 | Large ecosystem, TypeScript, rich chart libraries |
| **Charts** | uPlot (time-series) + Recharts (bar/scatter) | uPlot for performance on large datasets, Recharts for declarative composability |
| **Database** | PostgreSQL + TimescaleDB | Time-series optimized, hypertables, rolling aggregates |
| **MCP Transport** | stdio + SSE | stdio for local Claude Code, SSE for remote/Tailscale access |
| **Deployment** | Docker Compose | PostgreSQL + app in one stack, multi-stage build |
| **Auth** | Tailscale tsnet | Zero-config TLS + identity, no passwords |

## Non-Goals (v1)

- Computed composite scores (Recovery, Exertion, etc.) — Claude can do this on demand via MCP
- Native iOS/watchOS app
- Direct Apple HealthKit integration
- Multi-user support
- Workout planning or automated coaching
- Third-party app integration (Strava, etc.)
- Push notifications

## Rules

Read `server/specs/` before implementing. Specs are the source of truth.
Search before writing. Don't assume something is missing — ripgrep the codebase first.
One thing at a time. Implement, test, lint, commit. Then move on.
Lint before committing. Run `cd server && go vet ./...` and `golangci-lint run ./...` (or `make lint`) before every commit. Fix all issues first.
No placeholders. Full implementations only. No TODO stubs.
Tests are mandatory. Every new function gets a test. Test doc comments must explain WHY the test exists.
Update fix_plan.md after completing a task or discovering a bug.
Update this file when you learn something about building/running the project.
Commit after each unit of work with a descriptive message.

Build and run instructions live next to the code they apply to:
`server/CLAUDE.md` and `app/CLAUDE.md`.

### Repository & CI/CD

**Forgejo is the source of truth**: `git.coydog-fence.ts.net/meltforce.net/freereps`
(`origin`). `github.com/meltforce/FreeReps` is a push mirror — never push there
directly. The mirror is `git push --mirror`: it force-pushes *and* prunes refs
that don't exist on Forgejo, so anything that must survive on GitHub has to
exist on Forgejo first (a GitHub-only branch is deleted at the next sync).

| Where | What runs | Triggered by |
|---|---|---|
| `.forgejo/workflows/ci.yml` | Go build/vet/test/lint, frontend tsc + build, then build → `git.coydog-fence.ts.net/meltforce.net/freereps:edge` → redeploy on `freereps-lxc` | push/PR on `main` |
| `.github/workflows/ios.yml` | Xcode build (no macOS runner exists on Forgejo) | mirror push |
| `.github/workflows/release.yml` | Docker Hub image + GitHub Release with `freereps-upload` binaries | tag push, carried over by the mirror |

Deploy runs through the shared reusable workflow
`meltforce.net/ci-workflows/.forgejo/workflows/build-push-deploy.yml@v4` with
`sync_compose: false` — **the deployed compose belongs to the homelab repo**
(`docker/stacks/freereps/compose.yaml`, plus the catalog entry in
`configuration/docker-stacks/stacks/freereps.yml`, which renders `.env` and
`config.yaml`). Change the image ref, ports or volumes there, not here.
`server/docker-compose.yml` is for local development only.

Runner labels: `docker` (normal jobs), `docker-buildx` (image builds), `host`
(runs on the runner LXC itself — needed for Tailscale SSH into deploy targets).
The Forgejo org `meltforce.net` already provides `REGISTRY_USER`,
`REGISTRY_TOKEN`, `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN`; no repo secrets.

## Development Roadmap

Phases 1–4 (data foundation, visualization, Tailscale + user management, MCP
integration) are implemented.

### Phase 5 — Polish (not started)
- Trend views
- Saveable dashboard configurations
- Responsive optimization

## References

- [Athlytic App](https://www.athlyticapp.com/) — Visual inspiration
- [Health Auto Export – Docs](https://help.healthyapps.dev/en/)
- [Health Auto Export – REST API](https://help.healthyapps.dev/en/health-auto-export/automations/rest-api/)
- [Health Auto Export – Server Connection](https://help.healthyapps.dev/en/health-auto-export/automations/server-connection/)
- [Health Auto Export – Export Formats](https://help.healthyapps.dev/en/health-auto-export/export-format/)
- [Health Auto Export – GitHub Server](https://github.com/HealthyApps/health-auto-export-server)
- [MCP Specification](https://modelcontextprotocol.io/)
