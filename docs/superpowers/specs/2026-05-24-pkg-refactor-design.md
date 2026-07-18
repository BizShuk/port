# 專案套件結構重構設計文件 (Project Package Structure Refactoring Design Document)

## 目的與背景 (Objective and Background)

為了解耦命令列介面 (CLI) 的定義與核心業務邏輯，本專案將進行套件結構重構。我們將建立 `cmd` 套件以存放 Cobra 指令定義，建立 `svc` 套件以存放實際的連接埠檢查、指標更新與儀表板監控的核心業務邏輯，並將模組名稱變更為 `github.com/bizshuk/port`。

## 系統架構與套件規劃 (System Architecture and Package Layout)

重構後的檔案組織如下：

- `main.go`：主進入點，只導入 `cmd` 並啟動命令執行。
- `go.mod`：模組宣告變更為 `github.com/bizshuk/port`。
- `cmd/`：命令解析層，存放與 CLI 指令相關之定義。
    - `root.go`：全域根命令與配置初始化。
    - `port.go`：父命令 `port` 定義。
    - `port_check.go`：子命令 `port check` 定義，解析參數後調用 `svc.RunOneTimeCheck`。
    - `monitor.go`：父命令 `monitor` 定義。
    - `monitor_port.go`：子命令 `monitor port` 定義，解析參數後調用 `svc.RunMonitor`。
- `svc/`：核心業務邏輯層。
    - `checker.go`：連接埠檢查與指標匯出的核心實作 (重命名自 `checker/checker.go`)。
    - `check.go`：單次檢查服務邏輯實作。
    - `monitor.go`：持續監控服務邏輯實作。

## 介面與資料結構定義 (Interfaces and Data Structures)

### `svc` 核心類型與設定 (Core Types and Configs)

原 `checker` 的 `Config`、`PortEntry`、`PortStatus` 與 `Checker` 維持在 `svc` 套件中，類型名稱不變。

### 單次檢查服務介面 (One-Time Check Service API)

```go
package svc

import "io"

type CheckConfig struct {
 PortsToCheck []PortEntry
 TimeoutVal   string
 Writer       io.Writer
}

func RunOneTimeCheck(cfg *CheckConfig, globalConfig *Config) error
```

### 持續監控服務介面 (Continuous Monitor Service API)

```go
package svc

import "context"

type MonitorConfig struct {
 Interval    string
 Timeout     string
 MetricsPort int
 MimirURL    string
}

func RunMonitor(ctx context.Context, cfg *MonitorConfig, globalConfig *Config) error
```

## 測試策略與驗證方式 (Testing Strategy and Verification)

- 編譯驗證：重構後執行 `go build -o port_listenor .` 確保無編譯錯誤。
- 功能驗證：
    - 執行 `go run . check`，驗證終端機能正確輸出各連接埠的狀態表格。
        - 執行 `go run . monitor`，驗證定時監控儀表板運作正常，且 metrics 伺服器能正常被 Prometheus 抓取。
