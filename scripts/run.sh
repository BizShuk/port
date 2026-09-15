#!/usr/bin/env bash
# run:setup — 冪等 (idempotent) 的前置作業，不啟動任何服務。
# 設定的唯一位置是 ~/.config/port，repo 內的 tmp/config 只是指過去的 symlink。

set -euo pipefail

PROJECT_ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIG_DIR="${XDG_CONFIG_HOME:-${HOME}/.config}/port"
SOURCE_PATH="$PROJECT_ROOT/config/default_settings.json"
TARGET_PATH="$CONFIG_DIR/settings.json"
WORKSPACE_LINK="$PROJECT_ROOT/tmp/config"

if [ ! -f "$SOURCE_PATH" ]; then
    echo "錯誤：找不到來源設定檔 $SOURCE_PATH" >&2
    exit 1
fi

mkdir -p "$CONFIG_DIR/data" "$CONFIG_DIR/logs" "$PROJECT_ROOT/tmp"

# 已存在的設定`不覆寫`：使用者手改過的 ports 陣列必須留著。
if [ ! -f "$TARGET_PATH" ]; then
    cp "$SOURCE_PATH" "$TARGET_PATH"
    echo "成功將設定檔複製至目標目錄：$TARGET_PATH"
fi

ln -sfn "$CONFIG_DIR" "$WORKSPACE_LINK"
echo "成功建立軟連結：$WORKSPACE_LINK -> $CONFIG_DIR"
