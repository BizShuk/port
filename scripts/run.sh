#!/bin/bash

# 設定路徑
CONFIG_DIR="$HOME/.config/port_listenor"

SOURCE_PATH="$(pwd)/config/default_settings.json"
TARGET_PATH="$CONFIG_DIR/settings.json"
WORKSPACE_PATH="$(pwd)/tmp/settings.json"

# 建立設定檔目錄
mkdir -p "$CONFIG_DIR"

if [ -f "$SOURCE_PATH" ]; then
    # 複製來源至目標目錄
    cp "$SOURCE_PATH" "$TARGET_PATH"
    echo "成功將設定檔複製至目標目錄：$TARGET_PATH"
    
    # 建立軟連結從工作區指向目標目錄的檔案
    ln -sf "$TARGET_PATH" "$WORKSPACE_PATH"
    echo "成功建立軟連結：$WORKSPACE_PATH -> $TARGET_PATH"
else
    echo "錯誤：找不到來源設定檔 $SOURCE_PATH"
    exit 1
fi
