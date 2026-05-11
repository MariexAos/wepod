#!/usr/bin/env bash
#
# wepod 卸载脚本
#
# 用法：
#   curl -fsSL https://raw.githubusercontent.com/MariexAos/wepod/main/scripts/uninstall.sh | bash
#   PREFIX=$HOME/.local ./uninstall.sh
#
set -euo pipefail

PREFIX="${PREFIX:-/usr/local}"
BIN="$PREFIX/bin/wepod"
STATE_DIR="${XDG_STATE_HOME:-$HOME/.local/state}/wepod"

red()   { printf '\033[31m%s\033[0m\n' "$*"; }
green() { printf '\033[32m%s\033[0m\n' "$*"; }
info()  { printf '\033[36m→\033[0m %s\n' "$*"; }

# 移除二进制
if [ -e "$BIN" ]; then
    info "移除 $BIN"
    if [ -w "$PREFIX/bin" ]; then
        rm -f "$BIN"
    else
        sudo rm -f "$BIN"
    fi
    green "✓ 二进制已删除"
else
    info "$BIN 不存在，跳过"
fi

# 调试日志目录（可选清理）
if [ -d "$STATE_DIR" ]; then
    if [ -t 0 ]; then
        read -r -p "同时删除调试日志目录 $STATE_DIR? [y/N]: " yn
    else
        yn="n"
    fi
    if [[ "$yn" =~ ^[Yy]$ ]]; then
        rm -rf "$STATE_DIR"
        green "✓ 已清理 $STATE_DIR"
    else
        info "保留 $STATE_DIR"
    fi
fi

# 提示：副本本身不动
cat <<'EOF'

注意：wepod 创建的 WeChat 副本（/Applications/WeChat2.app 等）和数据目录
（~/Library/Containers/com.tencent.xinWeChat2 等）不会被这个脚本删除。
要清理副本，先重装 wepod 并在 TUI 里 `d` 删除；或手动 rm。
EOF
