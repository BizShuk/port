package svc

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/bizshuk/port_listenor/config"
)

// monitorInterval 套件級變數，記錄監控間隔時間（預設為 0）
var monitorInterval time.Duration = 0

// RunMonitor 啟動持續監控循環與指標服務
func RunMonitor(ctx context.Context, entries []config.PortEntry, timeout time.Duration) error {
	globalConfig := config.GlobalSettings
	monitorInterval = parseDuration(globalConfig.CheckInterval)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		if err := RunOneTimeCheck(ctx, entries, timeout); err != nil {
			return err
		}

		select {
		case <-time.After(monitorInterval):
		case <-ctx.Done():
			return nil
		}
	}
}

// RunOneTimeCheck 執行單次檢查邏輯，印出表格結果。
func RunOneTimeCheck(ctx context.Context, entries []config.PortEntry, timeout time.Duration) error {
	results := CheckPorts(entries, timeout)
	config.UpdateStatuses(ctx, results)
	RenderDashboard(results)
	return nil
}

// RenderDashboard 渲染儀表板
//
// 改用手動計算欄寬 + padding，不再依賴 text/tabwriter：
//   - tabwriter 會把 ANSI escape codes 算進 cell 寬度，造成 STATUS 欄位
//     header 與 value 錯位（OPEN 帶 \033[32m...\033[0m 變成 15 字元視覺寬度 4）
//   - 同時在 PROCESS NAME 為 "OrbStack Helper" 時補上 service 名稱前綴
//     (e.g. "OrbStack: grafana")，避免 6 個 row 都長一樣難以辨識
func RenderDashboard(statuses []config.PortStatus) {
	fmt.Print("\033[H\033[2J")
	fmt.Println("================================================================================")
	fmt.Printf(" PORT HEALTH CHECKER - Last Update: %s (Interval: %v)\n",
		time.Now().Format("2006-01-02 15:04:05"), monitorInterval)
	fmt.Println("================================================================================")

	const colGap = "  "

	// cell 紀錄「要印出的字串（含 ANSI）」與「可見寬度（去除 ANSI）」
	type cell struct {
		display string
		visible int
	}

	makeCells := func(port, service, status, latency, pid, process string) []cell {
		return []cell{
			{display: port, visible: len(port)},
			{display: service, visible: len(service)},
			{display: status, visible: ansiVisibleLen(status)},
			{display: latency, visible: len(latency)},
			{display: pid, visible: len(pid)},
			{display: process, visible: ansiVisibleLen(process)},
		}
	}

	rows := [][]cell{
		makeCells("PORT", "SERVICE", "STATUS", "LATENCY", "PID", "PROCESS NAME"),
	}

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
			// OrbStack 內部服務從 host 角度看都是同一個 PID + 同名 process，
			// 把 service 補進前綴讓表格可讀
			procStr = fmt.Sprintf("\033[33mOrbStack:\033[0m %s", s.Service)
			if s.ProcessName != "OrbStack Helper" {
				procStr = s.ProcessName
			}
		}
		rows = append(rows, makeCells(
			strconv.Itoa(s.Port),
			s.Service,
			statusStr,
			latencyStr,
			pidStr,
			procStr,
		))
	}

	// 計算每欄最大可見寬度
	widths := make([]int, 6)
	for _, row := range rows {
		for i, c := range row {
			if c.visible > widths[i] {
				widths[i] = c.visible
			}
		}
	}

	// 印出：display + 右側補空白到 widths[i]
	for _, row := range rows {
		for i, c := range row {
			if i > 0 {
				fmt.Print(colGap)
			}
			fmt.Print(c.display)
			for pad := widths[i] - c.visible; pad > 0; pad-- {
				fmt.Print(" ")
			}
		}
		fmt.Println()
	}

	fmt.Println("================================================================================")
}

// ansiVisibleLen 計算字串的「可見長度」：忽略 CSI/SGR escape sequence
// (e.g. \033[31m, \033[0m) 中的字元，只計算實際顯示的字。
func ansiVisibleLen(s string) int {
	n := 0
	inEscape := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == 0x1b {
			inEscape = true
			continue
		}
		if inEscape {
			if c == 'm' {
				inEscape = false
			}
			continue
		}
		n++
	}
	return n
}
