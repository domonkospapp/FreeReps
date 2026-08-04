# Server Development

- Build: `cd server && make build` (or `make build` from root)
- Test: `cd server && go test ./...`
- Frontend stub for Go build: `mkdir -p server/web/dist && touch server/web/dist/.gitkeep`
- Frontend build: `cd server/web && npm ci && npm run build`
