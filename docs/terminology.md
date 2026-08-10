# port_listenor — 術語表 (Terminology)

本檔是領域名詞、狀態值與縮寫的單一定義來源。CLI、設定檔與文件使用同一組正名。

## 核心實體 (Core Entities)

| 術語 (Term) | 英文 (English) | 定義 (Definition) | 出處 (Source) |
| --- | --- | --- | --- |
| 埠口項目 | Port Entry | 設定檔中的一筆待檢查目標，含 port 與可選的健康端點 | `PortEntry` |
| 埠口狀態 | Port Status | 一次檢查的結果，含開啟與否、延遲與監聽行程資訊 | `PortStatus` |
| 檢查器 | Checker | 執行連線測試與行程檢索的元件 | `Checker`、`svc/` |
| 延遲 | Latency | TCP 握手所花的時間 | `net.DialTimeout` 量測 |
| 監聽行程 | Listener Process | 目前綁定該 port 的行程，以 PID 與名稱表示 | `lsof` + `ps` |

## 檢查語意 (Check Semantics)

`埠口狀態檢查`與`服務健康檢查`回答的是不同問題，混用會讓命令失去意義：

| 術語 (Term) | 英文 (English) | 回答的問題 | 判定方式 |
| --- | --- | --- | --- |
| 埠口狀態檢查 | Port Status Check | 這個 port 上有東西在聽嗎 | TCP 握手成功即為開啟 |
| 服務健康檢查 | Service Health Check | 這個服務健康嗎 | `health` 為 HTTP(S) URL 時取 2xx；為字面值 `"tcp"` 時退回 TCP 判定 |
| 明確 opt-in | Explicit Opt-in | `port health` 只檢查設定中`宣告了 health 欄位`的 entry | 未宣告者一律不參與健康判定 |

> `設計理由`：對整份 port 清單無差別做健康檢查，會讓 `ssh`、`ollama` 這類
> 非常駐項目永遠紅燈，命令便無法當作啟動後驗證或 CI 關卡使用。
> 沒有 HTTP 健康端點的服務（資料庫等）以 `"health": "tcp"` 退回 TCP 判定，
> 而不是被排除在外。

## 命令 (Commands)

| 命令 (Command) | 定義 (Definition) | 出處 (Source) |
| --- | --- | --- |
| `port` | 裸命令，對設定中所有 entry 做一次埠口狀態檢查 | `RootCmd.RunE` |
| `port kill <port>` | 找出監聽該 port 的 PID 並發送 `SIGTERM` | `KillCmd` |
| `port health` | 對宣告了 `health` 的 entry 做通過/不通過判定，以結束碼表態 | `HealthCmd` |

## 觀測 (Observability)

| 術語 (Term) | 英文 (English) | 定義 (Definition) | 出處 (Source) |
| --- | --- | --- | --- |
| 監控指標 | Metrics | 匯出給 Prometheus 的檢查結果 | `prometheus/client_golang` |
| 遙測 | Telemetry | OpenTelemetry 追蹤資料 | `go.opentelemetry.io/otel` |
| 監控循環 | Monitoring Loop | 定期重複執行檢查的常駐模式，與手動單次觸發相對 | `ecosystem.config.js` |

## 實作慣例 (Implementation Conventions)

- `併發檢查`：每個 port 一個 goroutine，以 `sync.WaitGroup` 與 `sync.Mutex`
  同步與收集結果。
- `系統命令整合`：PID 與行程名稱來自 `lsof` 與 `ps`，不自行解析 `/proc`。
  macOS 上 `lsof` 的 `COMMAND` 欄位會截斷至 8 字元，需要完整名稱時
  改用 `lsof -nP -i :PORT -F pc` 取結構化欄位。
