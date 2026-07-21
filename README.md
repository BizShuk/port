# port_listenor (Port Listenor)

一個用於檢查特定連接埠狀態、獲取監聽進程資訊並匯出監控指標的命令列工具。

## 業務領域 (Business Domains)

### 埠口狀態檢查 (Port Status Check)

負責對指定的連接埠執行單次或定期的 TCP 連線測試，並在連接埠處於開啟狀態時，檢索該連接埠的系統進程資訊。

`領域流程 (Domain Flow):`

1. 進入點觸發：使用者直接執行 `port` (裸指令) 手動觸發，或由監控循環定期觸發。
2. 連線測試：使用 `net.DialTimeout` 對目標連接埠進行 TCP 握手，計算連線延遲 (Latency)。
3. 進程檢索：若連接埠為開啟狀態，則執行系統命令 `lsof` 取得該連接埠的進程識別碼 (PID)，再透過 `ps` 取得對應的進程名稱。
4. 結果彙整：包裝為 `PortStatus` 實體並返回。

`核心實體 (Key Entities):` `PortEntry`, `PortStatus`, `Checker`

`相關處理器 (Related Handlers):` `RootCmd.RunE`

---

### 指標與監控 (Metrics and Monitoring)

負責提供持續的連接埠健康狀態監控，將檢查結果轉換為監控指標，並透過 Prometheus HTTP 伺服器或 OpenTelemetry 協定發送至遠端監控平台。

`領域流程 (Domain Flow):`

1. 初始化：使用者透過 `monitor port` 指令啟動，系統初始化 Prometheus 註冊表與 OpenTelemetry 指標提供者。
2. 啟動伺服器：在背景啟動 HTTP 伺服器以暴露 `/metrics` 端點。
3. 監控循環：定期調用 `埠口狀態檢查`，更新內部的指標數值。
4. 儀表板渲染：在終端機中即時輸出格式化後的連接埠狀態儀表板。

`核心實體 (Key Entities):` `Config`, `Checker`

`相關處理器 (Related Handlers):` `MonitorCmd`

---

## 領域關聯 (Domain Relationships)

`指標與監控 (Metrics and Monitoring)` 領域依賴 `埠口狀態檢查 (Port Status Check)` 領域來獲取最新連接埠狀態。監控服務定期執行檢查，並將產生的 `PortStatus` 數據轉換為指標更新至 Prometheus 註冊表。

## 使用方式 (Usage)

### 埠口狀態檢查 (Port Status Check)

```bash
# 檢查特定連接埠
go run . --ports 80,443,3000
```

### 指標與監控 (Metrics and Monitoring)

```bash
# 啟動持續監控儀表板與指標伺服器
go run . monitor --interval 10s --metrics-port 10235
```

### 設定管理 (Configuration Management)

```bash
# 檢視合併後的設定
go run . config

# 顯示每個設定值的來源
go run . config --source

# 更新使用者層級的 settings.local.json
go run . config --update timeout=2s
```

## 改善建議 (Improvement Suggestions)

- [ ] `解耦指令與核心邏輯 (Decouple commands and core logic)`：目前命令列的執行邏輯直接編寫於 `RootCmd.RunE` 與 `MonitorCmd.RunE` 函式中，建議將具體業務邏輯抽離至服務層 `svc` 套件。
- [ ] `使用統一的日誌記錄器 (Use a unified logger)`：專案目前混用標準輸出與標準日誌套件，應設計統一的 Logger 介面以利維護。
- [ ] `增加核心單元測試 (Add core unit tests)`：目前測試涵蓋設定與指令註冊，但仍應為 `checker` 核心邏輯補齊單元測試。
