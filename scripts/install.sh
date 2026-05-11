#!/usr/bin/env bash
#
# wepod 安装脚本
#
# 处理 macOS Gatekeeper 警告（下载来源未知）：剥离 quarantine xattr +
# ad-hoc 重签，再安装到 PREFIX/bin。
#
# 用法：
#   curl -fsSL https://raw.githubusercontent.com/MariexAos/wepod/main/scripts/install.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/MariexAos/wepod/main/scripts/install.sh | bash -s v0.1.0
#   PREFIX=$HOME/.local ./install.sh
#
set -euo pipefail

REPO="MariexAos/wepod"
PREFIX="${PREFIX:-/usr/local}"
VERSION="${1:-latest}"

red()   { printf '\033[31m%s\033[0m\n' "$*"; }
green() { printf '\033[32m%s\033[0m\n' "$*"; }
info()  { printf '\033[36m→\033[0m %s\n' "$*"; }

# 平台检测
if [ "$(uname -s)" != "Darwin" ]; then
    red "wepod 只支持 macOS（当前: $(uname -s)）"
    exit 1
fi

case "$(uname -m)" in
    arm64|aarch64) arch="arm64" ;;
    x86_64)        arch="amd64" ;;
    *) red "不支持的架构: $(uname -m)"; exit 1 ;;
esac

asset="wepod-darwin-${arch}.tar.gz"

if [ "$VERSION" = "latest" ]; then
    url="https://github.com/$REPO/releases/latest/download/$asset"
    sha_url="$url.sha256"
else
    url="https://github.com/$REPO/releases/download/$VERSION/$asset"
    sha_url="$url.sha256"
fi

info "下载 wepod $VERSION (darwin-$arch)"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

curl --fail --silent --show-error --location -o "$tmp/$asset"        "$url"
curl --fail --silent --show-error --location -o "$tmp/$asset.sha256" "$sha_url"

# 校验 sha256
expected=$(awk '{print $1}' < "$tmp/$asset.sha256")
actual=$(shasum -a 256 "$tmp/$asset" | awk '{print $1}')
if [ "$expected" != "$actual" ]; then
    red "✗ sha256 校验失败"
    echo "  期望: $expected"
    echo "  实得: $actual"
    exit 1
fi
green "✓ sha256 校验通过"

# 解压
tar -C "$tmp" -xzf "$tmp/$asset"
[ -f "$tmp/wepod" ] || { red "归档里没有 wepod 二进制"; exit 1; }

# 关键步骤：清除 quarantine + ad-hoc 重签，避免 Gatekeeper 拦截
xattr -d com.apple.quarantine "$tmp/wepod" 2>/dev/null || true
codesign --force --sign - "$tmp/wepod" >/dev/null 2>&1 || true

# 安装
dest="$PREFIX/bin/wepod"
mkdir -p "$PREFIX/bin" 2>/dev/null || true

if [ -w "$PREFIX/bin" ]; then
    install -m 0755 "$tmp/wepod" "$dest"
else
    info "需要 sudo 写入 $dest"
    sudo install -m 0755 "$tmp/wepod" "$dest"
fi

green "✓ wepod 已安装到 $dest"
echo
echo "运行：  wepod"
echo "卸载：  curl -fsSL https://raw.githubusercontent.com/$REPO/main/scripts/uninstall.sh | bash"
