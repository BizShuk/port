package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bizshuk/port/svc"
	"github.com/spf13/cobra"
)

var checkPorts string

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Run a one-time port status check",
	Long:  `Check defined ports once and print the results immediately to the console.`,
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

func init() {
	checkCmd.Flags().StringVarP(
		&checkPorts,
		"ports",
		"p",
		"",
		"comma-separated list of ports to check (e.g. 80,443,3000)",
	)
	RootCmd.AddCommand(checkCmd)
}
