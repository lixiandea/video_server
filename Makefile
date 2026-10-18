# 视频服务统一命令入口。
# 更细粒度的运维脚本见 scripts/ 与根目录的 build.sh、test-api.sh 等。
SHELL := /bin/bash
COMPOSE ?= docker compose

.DEFAULT_GOAL := help

.PHONY: help up down ps logs build check images test test-unit test-e2e test-api lint fmt frontend-dev migrate-up migrate-status migrate-down monitoring-up monitoring-check security-check clean

help: ## 显示所有可用目标
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

up: ## 启动完整服务栈（首次请先 cp .env.example .env 并填写密钥）
	$(COMPOSE) up -d --remove-orphans

down: ## 停止并移除服务栈（保留数据卷）
	$(COMPOSE) down

ps: ## 查看服务状态
	$(COMPOSE) ps

logs: ## 跟踪服务日志
	$(COMPOSE) logs -f --tail=100

images: ## 重建全部服务镜像
	$(COMPOSE) build

build: ## 本地编译 Go 二进制到 bin/
	./build.sh

check: ## 提交前检查（pre-commit 钩子）：后端 build + vet
	go build ./... && go vet ./...

security-check: ## 依赖漏洞巡检：govulncheck + npm 生产依赖（均阻断）
	go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
	cd frontend-new && npm audit --omit=dev --audit-level=high

test: ## 后端 build/vet/test + 前端单元测试
	go build ./... && go vet ./... && go test ./...
	cd frontend-new && npx vitest run

test-unit: ## 仅前端单元测试
	cd frontend-new && npx vitest run

test-e2e: ## 前端 e2e（需 :8080 API 与 :3000 前端开发服务器）
	cd frontend-new && npm run test:e2e

test-api: ## 对运行中的服务执行 API 集成测试
	./test-api.sh

lint: ## 前端 ESLint
	cd frontend-new && npm run lint

fmt: ## 用 gofmt 格式化全部 Go 文件
	gofmt -w $$(git ls-files '*.go')
	@echo "gofmt: $$(gofmt -l $$(git ls-files '*.go') | wc -l) 个文件仍需格式化"

frontend-dev: ## 启动前端开发服务器（:3000）
	cd frontend-new && npm install && npm run dev

migrate-up: ## 应用数据库迁移（一次性容器）
	$(COMPOSE) run --rm migrate up

migrate-status: ## 查看迁移版本状态
	$(COMPOSE) run --rm migrate status

migrate-down: ## 回滚最近 1 个迁移版本（破坏性操作）
	$(COMPOSE) run --rm migrate down --steps 1 --yes

monitoring-up: ## 单独启动 Prometheus + Grafana
	./start-monitoring.sh

monitoring-check: ## 检查监控组件状态
	./check-monitoring.sh

clean: ## 清理本地进程与构建产物
	./cleanup.sh
