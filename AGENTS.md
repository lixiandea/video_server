# Repository Guidelines

## Project Structure & Module Organization
- `cmd/` — service entrypoints: `api-server`, `scheduler`, `transcode-worker`.
- `internal/` — application code: `handlers/` (Gin HTTP), `services/` (business logic), `models/` (GORM), `middleware/`, `config/`, `utils/`.
- `pkg/` — reusable libraries: `auth`, `database`, `redis`, `storage`, `queue`, `metrics`, `tracing`, `logging`, `errors`, `validation`.
- `Makefile` / `scripts/` — single entry point (`make help`) plus infra & app start/stop helpers.
- `docs/` — roadmap (`project-analysis-and-roadmap.md`) and work logs (`worklog/`).
- `api/` — request/response contracts; `frontend-new/` — the single frontend (Vue 3 + Vite), served by Nginx in Docker.
- Frontend tests live in `frontend-new/src/tests/{unit,e2e}`. Shell suites: `test-api.sh`, `test-monitoring.sh`. Config in `config.yaml`, `docker/`, `deploy/`, docs in `docs/`.

## Build, Test, and Development Commands
- `make help` — list every supported task; prefer it over ad-hoc commands.
- `./build.sh` (or `make build`) — builds `api-server`, `scheduler`, `transcode-worker` into `bin/`.
- `go build ./...` — compile all Go packages; run `go vet ./...` and `go test ./...` before committing.
- `go run ./cmd/api-server` — run the API locally (needs MySQL/Redis).
- `make up` / `make down` — start / stop the compose stack (the one-shot `migrate` service runs first).
- `make migrate-up` / `make migrate-status` / `make migrate-down` — database schema changes and rollback.
- `make test-api` (`./test-api.sh`) — curl-based integration tests against a running server.
- `cd frontend-new && npm install && npm run dev` — dev server on `:3000` (`make frontend-dev`).
- `npm run build`, `npm run lint`, `npm test` (unit + e2e), `npm run test:unit`, `npm run test:e2e`.

## Docker Deployment
- Copy `.env.example` to `.env` first; `docker-compose.yml` uses `${VAR:?}` so missing secrets abort the run. `docker compose config -q` validates the file without starting anything.
- `make up` (`docker compose up -d`) starts all 8 services (mysql, redis, api-server, scheduler, transcode-worker, frontend, prometheus, grafana). Frontend: `:8000` (`FRONTEND_PORT`), API: `:8080`, Prometheus: `:9090`, Grafana: `:3301` (`GRAFANA_PORT`).
- Override `FRONTEND_PORT` / `GRAFANA_PORT` when the host reserves those ports (common on Windows/WSL2, e.g. the 2952-3051 range).
- Schema changes go through versioned migrations: add `migrations/<NNNN>_<name>.{up,down}.sql` (up/down must be paired) and run `make migrate-up`. The one-shot `migrate` compose service runs `up` before api-server / scheduler / transcode-worker start. Services no longer auto-migrate: `DB_AUTO_MIGRATE=true` is a local/CI-only escape hatch and is ignored in release mode. `docker/mysql/init.sql` only creates the database.
- Alpine-based images use a domestic apk mirror and `frontend-new` uses `registry.npmmirror.com`; keep both in mind when builds fail on network access.
- `.dockerignore` keeps `node_modules/`, `dist/`, `.env` and docs out of the build context; verify new build inputs are not accidentally excluded.

## Coding Style & Naming Conventions
- Go: `gofmt`/`goimports` formatting; packages lowercase single words; exported names PascalCase; files `snake_case.go` (e.g. `video_handler.go`). Wrap errors with `fmt.Errorf("...: %w", err)`.
- Keep HTTP parsing in `handlers/`, logic in `services/`; read config via Viper with env overrides (`DB_HOST`, `REDIS_ADDR`).
- Vue: 2-space indent; components PascalCase (`VideoDetail.vue`), stores/composables camelCase; import through the `@` alias. Lint with `npm run lint`.

## Testing Guidelines
- Frontend unit tests: Vitest, `src/tests/unit/*.test.js`; e2e: Playwright, `src/tests/e2e/*.spec.js`. Coverage: `npm run test:unit:coverage`.
- Backend unit tests live next to the code (`internal/middleware/ratelimit_test.go`, `pkg/metrics/metrics_endpoint_test.go`); run `go test ./...`. CI (`.github/workflows/test.yml`) builds services and runs the API scripts against MySQL/Redis.
- E2E expects the API on `:8080` and the frontend dev server on `:3000`.

## Commit & Pull Request Guidelines
- Use Conventional Commits prefixes seen in history: `feat:`, `fix:`, `docs:`, `chore:`, `refactor:`, `perf:`, `test:`. Chinese subjects are acceptable (e.g. `fix: 修复转码队列`).
- PRs target `main`/`develop`, must pass CI, and should describe the change, link issues, list test commands run, and include screenshots for UI changes.

## Security & Configuration Tips
- Do not commit real credentials; rotate leaked passwords and JWT keys, and inject secrets at deploy time. `config.yaml`/`.env` hold development secrets, so deploy via environment variables. Never commit `bin/`, `node_modules/`, or `storage/` artifacts.
