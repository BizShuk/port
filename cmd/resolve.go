package cmd

import (
	"fmt"
	"time"

	"github.com/bizshuk/port/config"
)

// ResolvePorts 解析傳入的 port 列表，若為空則從設定檔讀取，並回傳 PortEntry 列表、Timeout 期間與錯誤。
func ResolvePorts(ports []int) ([]config.PortEntry, time.Duration, error) {
	globalConfig := config.GlobalSettings
	timeout := ParseDuration(globalConfig.Timeout)

	var entries []config.PortEntry
	if len(ports) > 0 {
		for _, port := range ports {
			entries = append(entries, config.PortEntry{
				Port: port,
				Name: fmt.Sprintf("port-%d", port),
			})
		}
	} else {
		entries = globalConfig.Ports
	}

	if len(entries) == 0 {
		return nil, 0, fmt.Errorf("no ports specified in config or arguments")
	}

	return entries, timeout, nil
}

// ParseDuration parses a duration string, returning a default of 5s on error.
func ParseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 5 * time.Second
	}
	return d
}
