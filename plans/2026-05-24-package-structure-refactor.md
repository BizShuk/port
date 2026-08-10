# 專案套件結構重構執行計畫 (Package Refactor Implementation Plan)

> `對於 Agent 執行者：` 必須使用 `superpowers:subagent-driven-development` (推薦) 或 `superpowers:executing-plans` 來逐步執行此計畫。步驟使用核取方塊 (`- [ ]`) 語法進行追蹤。

`目標：` 將專案的命令列指令移至 `cmd` 套件，將核心邏輯移至 `svc` 套件，變更 Go 模組名稱為 `github.com/bizshuk/port_listenor`。

`架構設計：` 將原有主目錄下的指令定義檔案 (`root.go`, `check.go`, `monitor.go`) 重新設計並移入 `cmd/` 套件；原有的 `checker/checker.go` 重新命名並移入 `svc/` 套件，並在 `svc/` 內新增 `check.go` 與 `monitor.go` 用於處理具體業務邏輯。

`技術棧：` `Go 1.26.0`, `github.com/spf13/cobra`, `github.com/prometheus/client_golang`

---

### 任務 1：更新 `go.mod` 模組宣告

`檔案：`

- 修改：`go.mod`

- [ ] `步驟 1：修改 go.mod 中的 module 名稱`
      將第 1 行的 `module port-checker` 變更為 `module github.com/bizshuk/port_listenor`。

    修改後的 `go.mod` 前幾行應如下所示：

    ```go
    module github.com/bizshuk/port_listenor

    go 1.26.0
    ```

- [ ] `步驟 2：提交變更`
      執行：

    ```bash
    git add go.mod
    git commit -m "refactor: rename module to github.com/bizshuk/port_listenor"
    ```

---

### 任務 2：遷移 `checker/checker.go` 至 `svc/checker.go`

`檔案：`

- 新增：`svc/checker.go`
- 測試：透過編譯檢查

- [ ] `步驟 1：建立 svc 目錄並將 checker/checker.go 複製/移動至 svc/checker.go`
      在 `svc/` 下建立 `checker.go`。其 package 宣告應為 `package svc`。

    修改後的 `svc/checker.go` 前 30 行代碼如下：

    ```go
    package svc

    import (
     "context"
     "encoding/json"
     "fmt"
     "net"
     "net/url"
     "os"
     "os/exec"
     "strings"
     "sync"
     "time"

     "github.com/prometheus/client_golang/prometheus"
     "go.opentelemetry.io/otel"
     "go.opentelemetry.io/otel/attribute"
     "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
     "go.opentelemetry.io/otel/metric"
     sdkmetric "go.opentelemetry.io/otel/sdk/metric"
     "go.opentelemetry.io/otel/sdk/resource"
     semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
    )
    ```

    後續的類型與函式定義 (如 `Config`、`PortEntry`、`PortStatus`、`Checker` 等) 均保持不變。

- [ ] `步驟 2：提交變更`
      執行：

    ```bash
    git add svc/checker.go
    git commit -m "refactor: move checker.go to svc package"
    ```

---

### 任務 3：實作單次檢查業務邏輯 `svc/check.go`

`檔案：`

- 新增：`svc/check.go`

- [ ] `步驟 1：建立 svc/check.go`
      寫入單次檢查的核心執行程式碼：

    ```go
    package svc

    import (
     "fmt"
     "io"
     "os"
     "sort"
     "sync"
     "text/tabwriter"
    )

    // CheckConfig 定義單次檢查的參數
    type CheckConfig struct {
     PortsToCheck []PortEntry
     TimeoutVal   string
     Writer       io.Writer
    }

    // RunOneTimeCheck 執行單次檢查邏輯，印出表格結果
    func RunOneTimeCheck(cfg *CheckConfig, globalConfig *Config) error {
     ports := cfg.PortsToCheck
     if len(ports) == 0 {
      ports = globalConfig.Ports
     }
     if len(ports) == 0 {
      return fmt.Errorf("no ports specified to check. Use -c to specify a config file or --ports to specify ports directly")
     }

     timeoutVal := globalConfig.Timeout
     if cfg.TimeoutVal != "" {
      timeoutVal = cfg.TimeoutVal
     }
     timeout := ParseDuration(timeoutVal)

     c := NewChecker(globalConfig)
     var results []PortStatus
     var resultsLock sync.Mutex
     var wg sync.WaitGroup

     for _, entry := range ports {
      wg.Add(1)
      go func(e PortEntry) {
       defer wg.Done()
       status := c.CheckPortWithProcess(e, timeout)
       resultsLock.Lock()
       results = append(results, status)
       resultsLock.Unlock()
      }(entry)
     }
     wg.Wait()

     sort.Slice(results, func(i, j int) bool {
      return results[i].Port < results[j].Port
     })

     w := tabwriter.NewWriter(cfg.Writer, 0, 0, 3, ' ', 0)
     fmt.Fprintln(w, "PORT\tSERVICE\tSTATUS\tLATENCY\tPID\tPROCESS NAME")
     for _, s := range results {
      statusStr := "CLOSED"
      if s.IsOpen {
       statusStr = "OPEN"
      }
      latencyStr := "-"
      if s.IsOpen {
       latencyStr = fmt.Sprintf("%.2fms", s.LatencyMs)
      }
      pidStr := "-"
      if s.IsOpen && s.PID != "" {
       pidStr = s.PID
      }
      procStr := "-"
      if s.IsOpen && s.ProcessName != "" {
       procStr = s.ProcessName
      }
      fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n",
       s.Port, s.Service, statusStr, latencyStr, pidStr, procStr)
     }
     w.Flush()
     return nil
    }
    ```

- [ ] `步驟 2：提交變更`
      執行：

    ```bash
    git add svc/check.go
    git commit -m "feat: implement one-time check service logic"
    ```

---

### 任務 4：實作持續監控業務邏輯 `svc/monitor.go`

`檔案：`

- 新增：`svc/monitor.go`

- [ ] `步驟 1：建立 svc/monitor.go`
      寫入監控循環與 OTel 初始化程式碼：

    ```go
    package svc

    import (
     "context"
     "fmt"
     "log"
     "net/http"
     "os"
     "sort"
     "sync"
     "text/tabwriter"
     "time"

     "github.com/prometheus/client_golang/prometheus/promhttp"
    )

    // MonitorConfig 定義持續監控的參數
    type MonitorConfig struct {
     Interval    string
     Timeout     string
     MetricsPort int
     MimirURL    string
    }

    // RunMonitor 啟動持續監控循環與指標服務
    func RunMonitor(ctx context.Context, cfg *MonitorConfig, globalConfig *Config) error {
     if cfg.Interval != "" {
      globalConfig.CheckInterval = cfg.Interval
     }
     if cfg.Timeout != "" {
      globalConfig.Timeout = cfg.Timeout
     }
     if cfg.MetricsPort != 0 {
      globalConfig.MetricsPort = cfg.MetricsPort
     }
     if cfg.MimirURL != "" {
      globalConfig.MimirEndpoint = cfg.MimirURL
     }

     c := NewChecker(globalConfig)

     if globalConfig.MimirEndpoint != "" {
      if err := c.InitOTel(ctx); err != nil {
       log.Printf("Warning: Failed to initialize OpenTelemetry: %v", err)
      } else {
       defer func() {
        shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer shutdownCancel()
        if err := c.ShutdownOTel(shutdownCtx); err != nil {
         log.Printf("Error shutting down MeterProvider: %v", err)
        }
       }()
      }
     }

     go func() {
      http.Handle("/metrics", promhttp.HandlerFor(c.Registry, promhttp.HandlerOpts{}))
      addr := fmt.Sprintf(":%d", globalConfig.MetricsPort)
      log.Printf("Starting prometheus metrics server on %s", addr)
      if err := http.ListenAndServe(addr, nil); err != nil {
       log.Printf("Failed to start metrics server: %v", err)
      }
     }()

     timeout := ParseDuration(globalConfig.Timeout)
     interval := ParseDuration(globalConfig.CheckInterval)
     var wg sync.WaitGroup

     for {
      select {
      case <-ctx.Done():
       log.Println("Received shutdown signal, shutting down gracefully...")
       return nil
      default:
      }

      var currentStatuses []PortStatus
      var currentLock sync.Mutex

      for _, entry := range globalConfig.Ports {
       wg.Add(1)
       go func(e PortEntry) {
        defer wg.Done()
        status := c.CheckPortWithProcess(e, timeout)
        c.UpdateMetrics(status)

        currentLock.Lock()
        currentStatuses = append(currentStatuses, status)
        currentLock.Unlock()
       }(entry)
      }

      wg.Wait()

      sort.Slice(currentStatuses, func(i, j int) bool {
       return currentStatuses[i].Port < currentStatuses[j].Port
      })

      c.StatusLock.Lock()
      c.LatestStatuses = currentStatuses
      c.StatusLock.Unlock()

      RenderDashboard(currentStatuses, interval)

      select {
      case <-time.After(interval):
      case <-ctx.Done():
       log.Println("Received shutdown signal, shutting down gracefully...")
       return nil
      }
     }
    }

    // RenderDashboard 渲染儀表板
    func RenderDashboard(statuses []PortStatus, interval time.Duration) {
     fmt.Print("\033[H\033[2J")
     fmt.Println("================================================================================")
     fmt.Printf(" PORT HEALTH CHECKER - Last Update: %s (Interval: %v)\n",
      time.Now().Format("2006-01-02 15:04:05"), interval)
     fmt.Println("================================================================================")

     w := tabwriter.NewWriter(os.Stdout, 0, 8, 3, ' ', 0)
     fmt.Fprintln(w, "PORT\tSERVICE\tSTATUS\tLATENCY\tPID\tPROCESS NAME")

     for _, s := range statuses {
      statusStr := "\033[31mCLOSED\033[0m"
      if s.IsOpen {
       statusStr = "\033[32mOPEN\033[0m"
      }

      latencyStr := "-"
      if s.IsOpen {
       latencyStr = fmt.Sprintf("%.2fms", s.LatencyMs)
      }

      pidStr := "-"
      if s.IsOpen && s.PID != "" {
       pidStr = s.PID
      }

      procStr := "-"
      if s.IsOpen && s.ProcessName != "" {
       procStr = s.ProcessName
      }

      fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n",
       s.Port, s.Service, statusStr, latencyStr, pidStr, procStr)
     }
     w.Flush()
     fmt.Println("================================================================================")
    }
    ```

- [ ] `步驟 2：提交變更`
      執行：

    ```bash
    git add svc/monitor.go
    git commit -m "feat: implement continuous monitoring service logic"
    ```

---

### 任務 5：建立 `cmd/root.go`

`檔案：`

- 新增：`cmd/root.go`

- [ ] `步驟 1：建立 cmd 目錄並寫入 cmd/root.go`
      在 `cmd/` 下建立 `root.go`，負責指令基礎、全域參數與配置載入：

    ```go
    package cmd

    import (
     "errors"
     "fmt"
     "os"

     "github.com/spf13/cobra"
     "github.com/bizshuk/port_listenor/svc"
    )

    var (
     cfgFile string
     config  svc.Config
    )

    var RootCmd = &cobra.Command{
     Use:   "port-checker",
     Short: "Port Health Checker CLI",
     Long:  `A CLI tool to check the status of specific ports and export metrics.`,
    }

    func init() {
     cobra.OnInitialize(initConfig)
     RootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "settings.json", "config file path")
    }

    func initConfig() {
     if cfgFile == "" {
      cfgFile = "settings.json"
     }
     cfg, err := svc.LoadConfig(cfgFile)
     if err != nil {
      if errors.Is(err, os.ErrNotExist) && cfgFile == "settings.json" {
       config = svc.Config{
        CheckInterval: "30s",
        Timeout:       "5s",
        MetricsPort:   10235,
       }
       return
      }
      fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
      os.Exit(1)
     }
     config = *cfg
    }

    func Execute() {
     if err := RootCmd.Execute(); err != nil {
      fmt.Println(err)
      os.Exit(1)
     }
    }

    // GetGlobalConfig 提供給同套件的其他 command 檔案獲取全域設定
    func GetGlobalConfig() *svc.Config {
     return &config
    }
    ```

- [ ] `步驟 2：提交變更`
      執行：

    ```bash
    git add cmd/root.go
    git commit -m "feat: add cmd/root.go"
    ```

---

### 任務 6：建立 `cmd/port.go` 及 `cmd/port_check.go`

`檔案：`

- 新增：`cmd/port.go`
- 新增：`cmd/port_check.go`

- [ ] `步驟 1：建立 cmd/port.go`
      編寫 `port` 父指令：

    ```go
    package cmd

    import (
     "github.com/spf13/cobra"
    )

    var portCmd = &cobra.Command{
     Use:   "port",
     Short: "Port operations",
     Long:  `Subcommands for checking specific ports.`,
    }

    func init() {
     RootCmd.AddCommand(portCmd)
    }
    ```

- [ ] `步驟 2：建立 cmd/port_check.go`
      編寫 `port check` 子指令，呼叫 `svc.RunOneTimeCheck`：

    ```go
    package cmd

    import (
     "fmt"
     "os"
     "strconv"
     "strings"

     "github.com/spf13/cobra"
     "github.com/bizshuk/port_listenor/svc"
    )

    var (
     checkPorts   string
     checkTimeout string
    )

    var checkCmd = &cobra.Command{
     Use:   "check",
     Short: "Run a one-time port status check",
     Long:  `Check defined ports once and print the results immediately to the console.`,
     RunE: func(cmd *cobra.Command, args []string) error {
      var portsToCheck []svc.PortEntry

      if checkPorts != "" {
       parts := strings.Split(checkPorts, ",")
       for _, pStr := range parts {
        pStr = strings.TrimSpace(pStr)
        portNum, err := strconv.Atoi(pStr)
        if err != nil {
         return fmt.Errorf("invalid port number: %s", pStr)
        }
        portsToCheck = append(portsToCheck, svc.PortEntry{
         Port: portNum,
         Name: fmt.Sprintf("port-%d", portNum),
        })
       }
      }

      cfg := &svc.CheckConfig{
       PortsToCheck: portsToCheck,
       TimeoutVal:   checkTimeout,
       Writer:       os.Stdout,
      }

      return svc.RunOneTimeCheck(cfg, GetGlobalConfig())
     },
    }

    func init() {
     checkCmd.Flags().StringVarP(&checkPorts, "ports", "p", "", "comma-separated list of ports to check (e.g. 80,443,3000)")
     checkCmd.Flags().StringVar(&checkTimeout, "timeout", "", "connection timeout (e.g. 2s, 5s)")
     portCmd.AddCommand(checkCmd)
    }
    ```

- [ ] `步驟 3：提交變更`
      執行：

    ```bash
    git add cmd/port.go cmd/port_check.go
    git commit -m "feat: add cmd/port.go and cmd/port_check.go"
    ```

---

### 任務 7：建立 `cmd/monitor.go` 及 `cmd/monitor_port.go`

`檔案：`

- 新增：`cmd/monitor.go`
- 新增：`cmd/monitor_port.go`

- [ ] `步驟 1：建立 cmd/monitor.go`
      編寫 `monitor` 父指令：

    ```go
    package cmd

    import (
     "github.com/spf13/cobra"
    )

    var monitorCmd = &cobra.Command{
     Use:   "monitor",
     Short: "Continuous monitoring operations",
     Long:  `Subcommands for continuous port monitoring and metrics exporting.`,
    }

    func init() {
     RootCmd.AddCommand(monitorCmd)
    }
    ```

- [ ] `步驟 2：建立 cmd/monitor_port.go`
      編寫 `monitor port` 子指令，呼叫 `svc.RunMonitor`：

    ```go
    package cmd

    import (
     "context"
     "os"
     "os/signal"
     "syscall"

     "github.com/spf13/cobra"
     "github.com/bizshuk/port_listenor/svc"
    )

    var (
     monitorInterval string
     monitorTimeout  string
     metricsPort     int
     mimirEndpoint   string
    )

    var monitorPortCmd = &cobra.Command{
     Use:   "port",
     Short: "Start continuous port health monitoring",
     Long:  `Run in daemon mode to check port status periodically and expose metrics.`,
     RunE: func(cmd *cobra.Command, args []string) error {
      ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
      defer cancel()

      cfg := &svc.MonitorConfig{
       Interval:    monitorInterval,
       Timeout:     monitorTimeout,
       MetricsPort: metricsPort,
       MimirURL:    mimirEndpoint,
      }

      if !cmd.Flags().Changed("metrics-port") {
       cfg.MetricsPort = 0 // 代表使用設定檔的值
      }

      return svc.RunMonitor(ctx, cfg, GetGlobalConfig())
     },
    }

    func init() {
     monitorPortCmd.Flags().StringVar(&monitorInterval, "interval", "", "check interval (e.g. 10s, 1m)")
     monitorPortCmd.Flags().StringVar(&monitorTimeout, "timeout", "", "connection timeout (e.g. 2s, 5s)")
     monitorPortCmd.Flags().IntVar(&metricsPort, "metrics-port", 0, "prometheus metrics port")
     monitorPortCmd.Flags().StringVar(&mimirEndpoint, "mimir-endpoint", "", "OTLP mimir endpoint URL")
     monitorCmd.AddCommand(monitorPortCmd)
    }
    ```

- [ ] `步驟 3：提交變更`
      執行：

    ```bash
    git add cmd/monitor.go cmd/monitor_port.go
    git commit -m "feat: add cmd/monitor.go and cmd/monitor_port.go"
    ```

---

### 任務 8：更新 `main.go`

`檔案：`

- 修改：`main.go`

- [ ] `步驟 1：重構 main.go 呼叫 cmd.Execute`
      將原內容替換為：

    ```go
    package main

    import "github.com/bizshuk/port_listenor/cmd"

    func main() {
     cmd.Execute()
    }
    ```

- [ ] `步驟 2：提交變更`
      執行：

    ```bash
    git add main.go
    git commit -m "refactor: update main.go to use new cmd package"
    ```

---

### 任務 9：移除舊程式碼與清理

`檔案：`

- 刪除：`root.go`
- 刪除：`check.go`
- 刪除：`monitor.go`
- 刪除：`checker/checker.go`

- [ ] `步驟 1：移除主目錄下的舊指令檔案`
      執行：

    ```bash
    rm root.go check.go monitor.go
    ```

- [ ] `步驟 2：移除 checker 目錄`
      執行：

    ```bash
    rm -rf checker
    ```

- [ ] `步驟 3：提交變更`
      執行：

    ```bash
    git rm root.go check.go monitor.go
    git rm -r checker
    git commit -m "refactor: remove old files and clean up legacy packages"
    ```

---

### 任務 10：編譯與功能驗證

`檔案：`

- 無

- [ ] `步驟 1：執行 go mod tidy 清理依賴`
      執行：

    ```bash
    go mod tidy
    ```

- [ ] `步驟 2：編譯測試`
      執行：

    ```bash
    go build -o port-checker .
    ```

    預期輸出：無編譯錯誤，生成 `port-checker` 可執行檔。

- [ ] `步驟 3：單次檢查功能測試`
      執行：

    ```bash
    ./port-checker port check
    ```

    預期輸出：印出 `PORT SERVICE STATUS LATENCY PID PROCESS NAME` 連接埠列表。

- [ ] `步驟 4：持續監控功能測試`
      執行 (按 Ctrl+C 退出)：

    ```bash
    ./port-checker monitor port --interval 5s
    ```

    預期輸出：儀表板能夠正常跳動更新。

- [ ] `步驟 5：提交最終變更`
      執行：

    ```bash
    git add .
    git commit -m "refactor: finalize refactoring and tidy up go mod"
    ```
