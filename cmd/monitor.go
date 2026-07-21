package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/bizshuk/port/svc"
	"github.com/spf13/cobra"
)

var (
	monitorInterval string
	monitorTimeout  string
	metricsPort     int
	mimirEndpoint   string
)

// MonitorCmd starts continuous port health monitoring.
var MonitorCmd = &cobra.Command{
	Use:     "monitor",
	Aliases: []string{"m"},
	Short:   "Start continuous port health monitoring",
	Long:    `Run in daemon mode to check port status periodically and expose metrics.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		entries, timeout, err := ResolvePorts(nil)
		if err != nil {
			return err
		}

		return svc.RunMonitor(ctx, entries, timeout)
	},
}

func init() {
	MonitorCmd.Flags().StringVar(&monitorInterval, "interval", "", "check interval (e.g. 10s, 1m)")
	MonitorCmd.Flags().StringVar(&monitorTimeout, "timeout", "", "connection timeout (e.g. 2s, 5s)")
	MonitorCmd.Flags().IntVar(&metricsPort, "metrics-port", 0, "prometheus metrics port")
	MonitorCmd.Flags().StringVar(&mimirEndpoint, "mimir-endpoint", "", "OTLP mimir endpoint URL")
}
