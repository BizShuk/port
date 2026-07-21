package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	gosdkcmd "github.com/bizshuk/gosdk/cmd"
	"github.com/bizshuk/port/svc"
	"github.com/spf13/cobra"
)

var checkPorts string

// RootCmd 是CLI的根命令；裸執行時直接進行一次性的連接埠檢查。
var RootCmd = &cobra.Command{
	Use:   "port",
	Short: "Port Health Checker CLI",
	Long:  `A CLI tool to check the status of specific ports and export metrics.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var ports []int
		if checkPorts != "" {
			for _, p := range strings.Split(checkPorts, ",") {
				p = strings.TrimSpace(p)
				port, err := strconv.Atoi(p)
				if err != nil {
					return fmt.Errorf("invalid port: %s", p)
				}
				ports = append(ports, port)
			}
		}
		entries, timeout, err := ResolvePorts(ports)
		if err != nil {
			return err
		}
		return svc.RunOneTimeCheck(cmd.Context(), entries, timeout)
	},
}

// Execute 執行根命令
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.Flags().StringVarP(
		&checkPorts,
		"ports",
		"p",
		"",
		"comma-separated list of ports to check (e.g. 80,443,3000)",
	)
	RootCmd.AddCommand(gosdkcmd.ConfigCmd, MonitorCmd)
}
