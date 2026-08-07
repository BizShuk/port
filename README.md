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

### 監聽程序終止 (Listener Process Termination)

負責尋找監聽指定 TCP port 的程序，並以 `SIGTERM` 終止該程序。

`領域流程 (Domain Flow):`

1. 使用者執行 `port kill <port>`，指定一個 `1-65535` 範圍內的 port。
2. 系統以 `lsof` 尋找該 port 上處於 `LISTEN` 狀態的 PID。
3. 系統對該 PID 發送 `SIGTERM`，並輸出已終止的 PID 與 port。

`相關處理器 (Related Handlers):` `KillCmd`

---

### 服務健康檢查 (Service Health Check)

負責對設定中`明確宣告 health 的 entry` 做一次通過/不通過的健康判定，供啟動後驗證或 CI 關卡使用。與`埠口狀態檢查`的差別在於它回答的是「服務健康嗎」而非「這個 port 有東西在聽嗎」，並以行程結束碼表態。

`領域流程 (Domain Flow):`

1. 使用者執行 `port health`，系統讀取設定並`只取出`宣告了 `health` 欄位的 entry。
2. `health` 為 HTTP(S) URL 時發送 GET，狀態碼落在 2xx 視為健康；為字面值 `"tcp"` 時退回 TCP 連線測試，供沒有 HTTP 健康端點的服務（資料庫等）使用。
3. 憑證：宣告 `insecure: true` 的 entry 跳過 TLS 驗證，供使用自簽憑證的本機服務 opt-in。
4. 逐行輸出 `OK` / `FAIL`、探測目標與耗時，最後印出健康比例；`任一項失敗即以非零碼結束`。

未宣告 `health` 的 entry 一律略過——若對整份清單無差別檢查，`ssh`、`ollama` 之類非常駐項目會讓結果永遠是紅的，這個指令也就失去把關的意義。

`核心實體 (Key Entities):` `PortEntry.Health`, `HealthResult`

`相關處理器 (Related Handlers):` `HealthCmd`、`svc.CheckHealth()`

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

### 監聽程序終止 (Listener Process Termination)

```bash
# 終止監聽 8080 port 的程序
port kill 8080
```

### 指標與監控 (Metrics and Monitoring)

```bash
# 啟動持續監控儀表板與指標伺服器
go run . monitor --interval 10s --metrics-port 10235
```

### 服務健康檢查 (Service Health Check)

```bash
# 檢查所有宣告了 health 的服務；任一失敗以非零碼結束
port health
```

設定範例（`config/default_settings.json` 或 `~/.config/port/settings.json`）：

```json
{
  "port": 3000,
  "name": "grafana",
  "health": "https://localhost:3000/api/health",
  "insecure": true
}
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
