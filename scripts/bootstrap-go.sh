#!/usr/bin/env bash
# 安装 Go 工具链（已安装则跳过）。
# 从 go.dev 获取最新稳定版，下载到 /tmp 并解压到 /usr/local/go。
set -euo pipefail

if command -v go >/dev/null 2>&1; then
  echo "Go 已安装: $(go version)"
  exit 0
fi

echo "未检测到 Go，开始安装最新稳定版 ..."

# 1. 查询最新稳定版本号（优先使用本机已有工具，其次用 python3 解析 JSON）
if command -v python3 >/dev/null 2>&1; then
  VERSION=$(curl -fsSL "https://go.dev/dl/?mode=json" | python3 -c "
import json,sys
data=json.load(sys.stdin)
for r in data:
    if r.get('stable'):
        print(r['version']); break
")
else
  VERSION=$(curl -fsSL "https://go.dev/dl/?mode=json" | grep -oP '"version":\s*"go\K[0-9.]+' | head -1)
  VERSION="go${VERSION}"
fi

if [ -z "${VERSION:-}" ]; then
  echo "无法获取 Go 版本，请手动下载 https://go.dev/dl/ 后安装" >&2
  exit 1
fi

ARCH=$(uname -m)
case "$ARCH" in
  x86_64) GOARCH=amd64 ;;
  aarch64|arm64) GOARCH=arm64 ;;
  *) echo "不支持的架构: $ARCH" >&2; exit 1 ;;
esac

FILENAME="${VERSION}.linux-${GOARCH}.tar.gz"
echo "下载 ${FILENAME} ..."
curl -fsSL -o "/tmp/${FILENAME}" "https://go.dev/dl/${FILENAME}"

echo "解压到 /usr/local/go ..."
rm -rf /usr/local/go
tar -C /usr/local -xzf "/tmp/${FILENAME}"
rm -f "/tmp/${FILENAME}"

ln -sf /usr/local/go/bin/go /usr/local/bin/go
ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt

echo "安装完成: $(go version)"

# 2. 配置模块代理（默认 proxy.golang.org 在国内可能不可达）
go env -w GOPROXY=https://goproxy.cn,direct
go env -w GOFLAGS=-mod=mod
echo "GOPROXY 已设置为 goproxy.cn,direct"
