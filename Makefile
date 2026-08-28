SHELL := /bin/bash
FRONTEND_DIR := frontend
BACKEND_DIR := backend
BIN := bin/ledger
WEBDIST := $(BACKEND_DIR)/cmd/server/webdist

.PHONY: help bootstrap frontend backend webdist build run dev test clean

help:
	@echo "本地电气台账 — 常用命令"
	@echo "  make bootstrap   安装 Go 工具链（仅首次/新机器需要）"
	@echo "  make build       构建前端并产出单文件后端 bin/ledger"
	@echo "  make run         构建并启动，访问 http://127.0.0.1:8080"
	@echo "  make dev         开发模式（见下方提示，需两个终端）"
	@echo "  make test        后端 go test + 前端 vitest"
	@echo "  make clean       清理构建产物与运行数据(data/)"

bootstrap:
	@bash scripts/bootstrap-go.sh

frontend:
	cd $(FRONTEND_DIR) && pnpm install && pnpm build

# 将前端产物复制进后端 go:embed 目录
webdist:
	rm -rf $(WEBDIST) && mkdir -p $(WEBDIST)
	cp -r $(FRONTEND_DIR)/dist/. $(WEBDIST)/

backend: webdist
	cd $(BACKEND_DIR) && go build -o ../$(BIN) ./cmd/server

build: frontend backend
	@echo "✅ 构建完成: $(BIN)"

run: build
	cd $(BACKEND_DIR) && ../$(BIN)

dev: webdist
	@echo "==== 开发模式 ===="
	@echo "  终端 1: cd $(BACKEND_DIR) && go run ./cmd/server   # API :8080"
	@echo "  终端 2: cd $(FRONTEND_DIR) && pnpm dev             # 页面 http://localhost:5173（已代理 /api）"
	@echo "  （前端改动由 Vite 热更新，后端改动 go run 需重启）"

test:
	cd $(BACKEND_DIR) && go test ./... -count=1
	cd $(FRONTEND_DIR) && pnpm vitest run

clean:
	rm -rf $(BIN) $(WEBDIST) $(FRONTEND_DIR)/dist $(FRONTEND_DIR)/node_modules data
	@echo "已清理"
