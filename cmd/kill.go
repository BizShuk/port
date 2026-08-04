package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bizshuk/port/svc"
	"github.com/spf13/cobra"
)

type killPortProcess func(ctx context.Context, port int) (int, error)

// KillCmd terminates the process listening on one port.
var KillCmd = newKillCmd(svc.KillPortProcess)

func newKillCmd(kill killPortProcess) *cobra.Command {
	return &cobra.Command{
		Use:   "kill <port>",
		Short: "Kill the process listening on a port",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			port, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid port %q: must be an integer", args[0])
			}
			if port < 1 || port > 65535 {
				return fmt.Errorf("invalid port %d: must be between 1 and 65535", port)
			}

			pid, err := kill(cmd.Context(), port)
			if err != nil {
				return err
			}
			cmd.Printf("Killed process %d listening on port %d\n", pid, port)
			return nil
		},
	}
}

func init() {
	RootCmd.AddCommand(KillCmd)
}
