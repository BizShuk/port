# Vite 常見連接埠加入設計文件 (Vite Common Ports Addition Design Document)

## 目的與背景 (Objective and Background)

將 Vite 開發與預覽伺服器的常見連接埠加入 `port_listenor` 預設檢查清單，讓不帶 `--ports` 的 `check` 與 `monitor` 指令能一併檢查本機前端服務。

## 設計 (Design)

- 在 `config/default_settings.json` 新增兩個 `PortEntry`：
  - `5173`，名稱為 `vite`
  - `4173`，名稱為 `vite-preview`
- 維持設定檔作為預設清單的唯一來源，不新增程式常數、CLI 選項或動態補值邏輯。
- 保留既有 `--ports` 行為：指定參數時仍只檢查使用者傳入的連接埠。

## 錯誤處理與相容性 (Error Handling and Compatibility)

這是向後相容的預設清單擴充。若 `5173` 或 `4173` 沒有服務監聽，既有檢查流程會將其標示為 `CLOSED`，不改變其他連接埠的處理方式。

## 測試與驗證 (Testing and Verification)

- 新增設定內容測試，確認嵌入的預設設定包含兩個連接埠與正確名稱。
- 先確認測試在設定尚未加入時失敗，再加入設定並確認測試通過。
- 執行完整 `go test ./...` 與 `go build -o port_listenor .`。
