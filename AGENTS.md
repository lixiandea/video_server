# Repository Guidelines

## Project Structure & Module Organization
- `cmd/` — service entrypoints: `api-server`, `scheduler`, `worker`, `transcode-worker`.
- `internal/` — application code: `handlers/` (Gin HTTP), `services/` (business logic), `models/` (GORM), `middleware/`, `config/`, `utils/`.
- `pkg/` — reusable libraries: `auth`, `database`, `redis`, `storage`, `queue`, `metrics`, `tracing`, `logging`, `errors`, `validation`.
- `api/` — request/response contracts; `frontend/` (legacy static), `frontend-vue/` (Vue 3 SSR), `frontend-new/` (primary Vue 3 + Vite app).
- Frontend tests live in `frontend-new/src/tests/{unit,e2e}`. Shell suites: `test-api.sh`, `test-monitoring.sh`. Config in `config.yaml`, `docker/`, `deploy/`, docs in `docs/`.

## Build, Test, and Development Commands
- `./build.sh` — builds `api-server`, `scheduler`, `worker` into `bin/`.
- `go build ./...` — compile all Go packages; `go vet ./...` before committing.
- `go run ./cmd/api-server` — run the API locally (needs MySQL/Redis).
- `docker compose up -d` or `./start-docker.sh` — start the full stack.
- `./test-api.sh` — curl-based integration tests against a running server.
- `cd frontend-new && npm install && npm run dev` — dev server on `:3000`.
- `npm run build`, `npm run lint`, `npm test` (unit + e2e), `npm run test:unit`, `npm run test:e2e`.

## Docker Deployment
- Copy `.env.example` to `.env` first; `docker-compose.yml` uses `${VAR:?}` so missing secrets abort the run. `docker compose config -q` validates the file without starting anything.
- `docker compose up -d --build` starts all 9 services (mysql, redis, api-server, scheduler, worker, transcode-worker, frontend, prometheus, grafana). Frontend: `:3000`, API: `:8080`, Grafana: `:3001`.
- Override `FRONTEND_PORT` / `GRAFANA_PORT` when the host reserves those ports (common on Windows/WSL2, e.g. the 2952-3051 range).
- `docker/mysql/init.sql` only seeds the database and the `users` table; every other table is created by GORM `AutoMigrate` at startup, serialized across services with a MySQL `GET_LOCK` named lock. Keep DDL out of `init.sql` unless you also change `internal/models`.
- Alpine-based images use a domestic apk mirror and `frontend-new` uses `registry.npmmirror.com`; keep both in mind when builds fail on network access.
- `.dockerignore` keeps `node_modules/`, `dist/`, `.env` and docs out of the build context; verify new build inputs are not accidentally excluded.

## Coding Style & Naming Conventions
- Go: `gofmt`/`goimports` formatting; packages lowercase single words; exported names PascalCase; files `snake_case.go` (e.g. `video_handler.go`). Wrap errors with `fmt.Errorf("...: %w", err)`.
- Keep HTTP parsing in `handlers/`, logic in `services/`; read config via Viper with env overrides (`DB_HOST`, `REDIS_ADDR`).
- Vue: 2-space indent; components PascalCase (`VideoDetail.vue`), stores/composables camelCase; import through the `@` alias. Lint with `npm run lint`.

## Testing Guidelines
- Frontend unit tests: Vitest, `src/tests/unit/*.test.js`; e2e: Playwright, `src/tests/e2e/*.spec.js`. Coverage: `npm run test:unit:coverage`.
- Backend has no Go unit tests yet; add `*_test.go` files and run `go test ./...`. CI (`.github/workflows/test.yml`) builds services and runs the API scripts against MySQL/Redis.
- E2E expects the API on `:8080` and the frontend dev server on `:3000`.

## Commit & Pull Request Guidelines
- Use Conventional Commits prefixes seen in history: `feat:`, `fix:`, `docs:`, `chore:`, `refactor:`, `perf:`, `test:`. Chinese subjects are acceptable (e.g. `fix: 修复转码队列`).
- PRs target `main`/`develop`, must pass CI, and should describe the change, link issues, list test commands run, and include screenshots for UI changes.

## Security & Configuration Tips
- Do not commit real credentials; rotate leaked passwords and JWT keys, and inject secrets at deploy time. `config.yaml`/`.env` hold development secrets, so deploy via environment variables. Never commit `bin/`, `node_modules/`, or `storage/` artifacts.
