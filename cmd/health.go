package cmd

import (
	"fmt"

	"github.com/bizshuk/port/svc"
	"github.com/spf13/cobra"
)

// HealthCmd 對設定中宣告了 health 的 entry 做一次健康檢查。
// 任一項失敗即以非零碼結束，可直接當作 CI / 啟動後的驗證關卡。
var HealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Probe health endpoints of configured services",
	Long: `Check every configured port entry that declares a "health" field.

An HTTP(S) URL is probed with GET and passes on 2xx. The literal value "tcp"
falls back to a TCP connect, for services without an HTTP health endpoint.
Entries without a "health" field are skipped. Exits non-zero if any check fails.`,
	// Execute() 已負責印出錯誤；不 silence 會讓失敗訊息重複兩次
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		entries, timeout, err := ResolvePorts(nil)
		if err != nil {
			return err
		}

		results := svc.CheckHealth(cmd.Context(), entries, timeout)
		if len(results) == 0 {
			return fmt.Errorf(`no port entry declares a "health" field`)
		}

		failed := 0
		for _, result := range results {
			verdict := "OK  "
			if !result.OK {
				verdict = "FAIL"
				failed++
			}
			fmt.Printf("  %s  %-16s %-6d %-52s %s (%dms)\n",
				verdict,
				result.Entry.Name,
				result.Entry.Port,
				result.Probe,
				result.Detail,
				result.Elapsed.Milliseconds(),
			)
		}

		fmt.Printf("\n%d/%d healthy\n", len(results)-failed, len(results))
		if failed > 0 {
			return fmt.Errorf("%d health check(s) failed", failed)
		}
		return nil
	},
}

func init() {
	RootCmd.AddCommand(HealthCmd)
}
