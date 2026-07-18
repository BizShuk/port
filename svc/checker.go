package svc

import (
	"fmt"
	"net"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bizshuk/port/config"
)

func CheckPortWithProcess(entry config.PortEntry, timeout time.Duration) config.PortStatus {
	status := config.PortStatus{
		Host:    "localhost",
		Port:    entry.Port,
		Service: entry.Name,
		Domain:  "",
	}

	isOpen, latency, _ := checkPort(status.Host, entry.Port, timeout)
	status.IsOpen = isOpen
	status.LatencyMs = latency
	status.LastCheckTime = time.Now()

	if isOpen {
		pid, procName := getProcessInfo(entry.Port)
		status.PID = pid
		status.ProcessName = procName
	}

	return status
}

func checkPort(host string, port int, timeout time.Duration) (bool, float64, error) {
	start := time.Now()
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		elapsed := time.Since(start).Seconds() * 1000
		return false, elapsed, nil
	}
	defer conn.Close()
	elapsed := time.Since(start).Seconds() * 1000
	return true, elapsed, nil
}

// getProcessInfo 透過 lsof -F 結構化輸出取得 PID 與短 process name。
//
// 原本用 ps -p PID -o comm= 二次呼叫，在 macOS 上會回傳完整執行檔路徑
// (e.g. /Applications/OrbStack.app/.../OrbStack Helper)，造成 PROCESS NAME
// 欄位被應用程式路徑污染。改用 lsof -F pc 一次拿到 p<PID> 與 c<COMMAND>。
func getProcessInfo(port int) (pid string, processName string) {
	cmd := exec.Command("lsof", "-nP", "-i", fmt.Sprintf(":%d", port), "-F", "pc")
	output, err := cmd.Output()
	if err != nil {
		return "", ""
	}

	for line := range strings.SplitSeq(string(output), "\n") {
		if len(line) < 2 {
			continue
		}
		switch line[0] {
		case 'p':
			pid = line[1:]
		case 'c':
			if processName == "" {
				processName = line[1:]
			}
		}
		if pid != "" && processName != "" {
			return pid, processName
		}
	}
	return pid, processName
}

// CheckPorts 併發檢查多個連接埠，並依連接埠號碼排序回傳結果
func CheckPorts(entries []config.PortEntry, timeout time.Duration) []config.PortStatus {
	var results []config.PortStatus
	var resultsLock sync.Mutex
	var wg sync.WaitGroup

	for _, entry := range entries {
		wg.Add(1)
		go func(e config.PortEntry) {
			defer wg.Done()
			status := CheckPortWithProcess(e, timeout)
			resultsLock.Lock()
			results = append(results, status)
			resultsLock.Unlock()
		}(entry)
	}
	wg.Wait()

	sort.Slice(results, func(i, j int) bool {
		return results[i].Port < results[j].Port
	})

	return results
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 5 * time.Second
	}
	return d
}
