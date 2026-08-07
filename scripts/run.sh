#!/bin/bash

# 設定路徑
CONFIG_DIR="$HOME/.config/port"

SOURCE_PATH="$(pwd)/config/default_settings.json"
TARGET_PATH="$CONFIG_DIR/settings.json"
WORKSPACE_LINK="$(pwd)/tmp/port"

# 建立設定檔目錄與工作區目錄
mkdir -p "$CONFIG_DIR"
mkdir -p "$(pwd)/tmp"

if [ -f "$SOURCE_PATH" ]; then
    # 若目標設定檔不存在則複製預設設定
    if [ ! -f "$TARGET_PATH" ]; then
        cp "$SOURCE_PATH" "$TARGET_PATH"
        echo "成功將設定檔複製至目標目錄：$TARGET_PATH"
    fi
    
    # 清理舊連結並建立軟連結 tmp/port -> ~/.config/port
    rm -rf "$WORKSPACE_LINK"
    rm -f "$(pwd)/tmp/settings.json"
    ln -sfn "$CONFIG_DIR" "$WORKSPACE_LINK"
    echo "成功建立軟連結：$WORKSPACE_LINK -> $CONFIG_DIR"
else
    echo "錯誤：找不到來源設定檔 $SOURCE_PATH"
    exit 1
fi
