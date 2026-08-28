SHELL := /bin/bash
FRONTEND_DIR := frontend
BACKEND_DIR := backend
BIN := bin
WEBDIST := $(BACKEND_DIR)/cmd/server/webdist

# 纯 Go 驱动（modernc.org/sqlite），CGO_ENABLED=0 可交叉编译各平台静态单文件
# 体积优化（仅构建参数，不改源码）：
#   -tags nomsgpack      去掉 gin 内置但用不到的 msgpack 绑定（连带移除 ugorji codec ~6MB）
#   -buildvcs=false      不嵌入 VCS 版本信息
#   -trimpath -ldflags -s -w   裁剪路径与 DWARF/符号表
CGO_ENABLED := 0
GOFLAGS := -tags nomsgpack -buildvcs=false -trimpath -ldflags "-s -w"

.PHONY: help bootstrap frontend webdist backend build run dev test clean build-all upx \
        build-linux-amd64 build-linux-arm64 build-windows-amd64

help:
	@echo "本地电气台账 — 常用命令"
	@echo "  make bootstrap               安装 Go 工具链（仅首次/新机器需要）"
	@echo "  make build                   构建本机二进制 $(BIN)/ledger"
	@echo "  make build-all               构建三平台二进制（linux x64 / linux arm64 / windows x64）"
	@echo "  make upx                     用 UPX 进一步压缩产物（可选，需先安装 upx）"
	@echo "  make run                     构建并启动，访问 http://127.0.0.1:5288"
	@echo "  make dev                     开发模式（见下方提示，需两个终端）"
	@echo "  make test                    后端 go test + 前端 vitest"
	@echo "  make clean                   清理构建产物与运行数据(data/)"

bootstrap:
	@bash scripts/bootstrap-go.sh

frontend:
	cd $(FRONTEND_DIR) && pnpm install && pnpm build

# 将前端产物复制进后端 go:embed 目录
webdist:
	rm -rf $(WEBDIST) && mkdir -p $(WEBDIST)
	cp -r $(FRONTEND_DIR)/dist/. $(WEBDIST)/

# 后端交叉编译模板：$(1)=GOOS $(2)=GOARCH $(3)=输出文件名
define build-backend
	cd $(BACKEND_DIR) && CGO_ENABLED=$(CGO_ENABLED) GOOS=$(1) GOARCH=$(2) \
	  go build $(GOFLAGS) -o ../$(BIN)/$(3) ./cmd/server
endef

backend: webdist
	cd $(BACKEND_DIR) && CGO_ENABLED=$(CGO_ENABLED) go build $(GOFLAGS) -o ../$(BIN)/ledger ./cmd/server

build: frontend backend
	@echo "✅ 构建完成: $(BIN)/ledger"

build-linux-amd64: webdist
	$(call build-backend,linux,amd64,ledger-linux-amd64)
	@echo "✅ $(BIN)/ledger-linux-amd64"

build-linux-arm64: webdist
	$(call build-backend,linux,arm64,ledger-linux-arm64)
	@echo "✅ $(BIN)/ledger-linux-arm64"

build-windows-amd64: webdist
	$(call build-backend,windows,amd64,ledger-windows-amd64.exe)
	@echo "✅ $(BIN)/ledger-windows-amd64.exe"

build-all: frontend build-linux-amd64 build-linux-arm64 build-windows-amd64
	@echo "✅ 三平台构建完成，产物位于 $(BIN)/"

# 可选：UPX 后处理压缩（注意：Windows 版压缩后可能被杀软误报，按需使用）
upx:
	@command -v upx >/dev/null 2>&1 || { echo "未安装 upx，请先：apt-get install upx-ucl 或从 https://upx.github.io 下载"; exit 1; }
	@for f in $(BIN)/ledger $(BIN)/ledger-linux-amd64 $(BIN)/ledger-linux-arm64 $(BIN)/ledger-windows-amd64.exe; do \
	  if [ -f "$$f" ]; then upx -9 "$$f"; fi; \
	done

run: build
	cd $(BACKEND_DIR) && ../$(BIN)/ledger

dev: webdist
	@echo "==== 开发模式 ===="
	@echo "  终端 1: cd $(BACKEND_DIR) && go run ./cmd/server   # API :5288"
	@echo "  终端 2: cd $(FRONTEND_DIR) && pnpm dev             # 页面 http://localhost:5173（已代理 /api）"
	@echo "  （前端改动由 Vite 热更新，后端改动 go run 需重启）"

test:
	cd $(BACKEND_DIR) && go test ./... -count=1
	cd $(FRONTEND_DIR) && pnpm vitest run

clean:
	rm -rf $(BIN) $(WEBDIST) $(FRONTEND_DIR)/dist $(FRONTEND_DIR)/node_modules data
	@echo "已清理"
