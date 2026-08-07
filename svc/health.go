package svc

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/bizshuk/port/config"
)

// HEALTH_TCP 是 health 欄位的特例值。沒有 HTTP 健康端點的服務（資料庫、
// 純 TCP 服務）以「連得上」作為健康判準，而不是被排除在檢查之外。
const HEALTH_TCP = "tcp"

// HealthResult 是單一 entry 的健康檢查結果。
type HealthResult struct {
	Entry   config.PortEntry
	Probe   string // 實際探測的目標：URL 或 tcp://host:port
	OK      bool
	Detail  string // 成功時為狀態碼，失敗時為原因
	Elapsed time.Duration
}

// CheckHealth 併發檢查所有宣告了 health 的 entry，依 port 排序回傳。
//
// 未宣告 health 的 entry 一律略過：health 是「這個服務該是活的」的明確
// 宣告，若對整份 port 清單無差別檢查，ssh / ollama 之類非常駐項目會讓
// 結果永遠是紅的，這個指令也就失去把關的意義。
func CheckHealth(ctx context.Context, entries []config.PortEntry, timeout time.Duration) []HealthResult {
	var results []HealthResult
	var resultsLock sync.Mutex
	var wg sync.WaitGroup

	for _, entry := range entries {
		if entry.Health == "" {
			continue
		}

		wg.Add(1)
		go func(e config.PortEntry) {
			defer wg.Done()
			result := checkOne(ctx, e, timeout)
			resultsLock.Lock()
			results = append(results, result)
			resultsLock.Unlock()
		}(entry)
	}
	wg.Wait()

	sort.Slice(results, func(i, j int) bool {
		return results[i].Entry.Port < results[j].Entry.Port
	})

	return results
}

func checkOne(ctx context.Context, entry config.PortEntry, timeout time.Duration) HealthResult {
	if entry.Health == HEALTH_TCP {
		return checkHealthTCP(entry, timeout)
	}
	return checkHealthHTTP(ctx, entry, timeout)
}

func checkHealthTCP(entry config.PortEntry, timeout time.Duration) HealthResult {
	addr := net.JoinHostPort("localhost", strconv.Itoa(entry.Port))
	result := HealthResult{Entry: entry, Probe: "tcp://" + addr}

	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, timeout)
	result.Elapsed = time.Since(start)
	if err != nil {
		result.Detail = err.Error()
		return result
	}
	conn.Close()

	result.OK = true
	result.Detail = "connected"
	return result
}

func checkHealthHTTP(ctx context.Context, entry config.PortEntry, timeout time.Duration) HealthResult {
	result := HealthResult{Entry: entry, Probe: entry.Health}

	client := &http.Client{Timeout: timeout}
	if entry.Insecure {
		// 本機服務常用自簽憑證（例如 Grafana 的 localhost.crt）。
		// 這是 entry 明確宣告的 opt-in，不是全域預設。
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, entry.Health, nil)
	if err != nil {
		result.Detail = err.Error()
		return result
	}

	start := time.Now()
	response, err := client.Do(request)
	result.Elapsed = time.Since(start)
	if err != nil {
		result.Detail = err.Error()
		return result
	}
	defer response.Body.Close()

	result.OK = response.StatusCode >= 200 && response.StatusCode < 300
	result.Detail = fmt.Sprintf("HTTP %d", response.StatusCode)
	return result
}
