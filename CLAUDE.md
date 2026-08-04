# port_listenor — 技術脈絡 (Technical Context)

## 專案結構 (Project Structure)

```tree
.
├── CLAUDE.md
├── README.md
├── cmd/                # CLI commands (Cobra)
├── config/             # Configuration loader with viper
├── svc/                # Service layer (core checker and dashboard logic)
├── docs/               # Documentation
├── plans/              # Implementation plans
├── go.mod
├── scripts/run.sh      # Setup and execution helper script
└── main.go             # Program entry point
```

## 技術棧 (Tech Stack)

- Language: `Go 1.26.0`
- CLI Framework: `github.com/spf13/cobra`
- SDK: `github.com/bizshuk/gosdk v1.2.7`
- Metrics: `github.com/prometheus/client_golang`
- Telemetry: `go.opentelemetry.io/otel`

## 關鍵決策 (Key Decisions)

- `併發檢查`：在執行連接埠檢查時，為每個連接埠啟動一個獨立的 goroutine 進行併發檢測，並使用 `sync.WaitGroup` 與互斥鎖 `sync.Mutex` 進行同步與結果收集，以加速多連接埠偵測。
- `系統命令集成`：當連接埠開啟時，透過執行系統內建的 `lsof` 與 `ps` 命令取得監聽該連接埠的 PID 與進程名稱，提供更豐富的診斷資訊。

## 模組對應 (Module Mapping)

| 業務領域 (Domain)                   | 套件/模組 (Package/Module) | 進入點 (Entry Point)      |
| ----------------------------------- | -------------------------- | ------------------------- |
| 埠口狀態檢查 (Port Status Check)    | `svc`, `cmd`               | `RootCmd.RunE` (裸指令)、`CheckPortWithProcess()` |
| 監聽程序終止 (Listener Process Termination) | `svc`, `cmd` | `KillCmd`、`KillPortProcess()` |
| 指標與監控 (Metrics and Monitoring) | `svc`, `cmd`               | `MonitorCmd` 執行邏輯 |
| 設定管理 (Configuration Management) | `config`, `cmd`             | `gosdk/cmd.ConfigCmd` |

## 開發指南 (Development Guide)

### 前置需求 (Prerequisites)

- `Go SDK (v1.26.0+)`
- 作業系統支援 `lsof` 與 `ps` 命令 (如 macOS 或 Linux)

### 安裝依賴 (Installation)

```bash
go mod download
```

### 建置 (Build)

```bash
go build -o port_listenor .
```

### 執行 (Run)

```bash
# 執行單次檢查（裸指令即為 check）
go run .

# 終止監聽指定 port 的程序
go run . kill 8080

# 執行持續監控
go run . monitor

# 檢視合併後的設定
go run . config
```

### 測試 (Test)

```bash
go test ./...
```

目前測試涵蓋設定預設值、module path 與 CLI command 註冊；`svc` 核心邏輯仍待補齊單元測試。

## 慣例 (Conventions)

- Naming：變數與函式命名遵循 Go 官方風格指南，CLI 指令使用 package-level exported var，並以 `Cmd` 結尾；旗標於 `init()` 綁定。
- Error handling：錯誤訊息應包含上下文資訊，使用 `fmt.Errorf` 進行包裝並回傳至指令層統一輸出。
- Telemetry：指標暴露使用 Prometheus Registry 進行註冊，遠端 OpenTelemetry 連接則使用 HTTP 導出器。
