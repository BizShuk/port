package cmd

import (
	"testing"

	gosdkcmd "github.com/bizshuk/gosdk/cmd"
	"github.com/spf13/cobra"
)

func TestRootCmdRegistersCommands(t *testing.T) {
	tests := []struct {
		name string
		want *cobra.Command
	}{
		{name: "config", want: gosdkcmd.ConfigCmd},
		{name: "kill", want: KillCmd},
		{name: "monitor", want: MonitorCmd},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := RootCmd.Find([]string{tt.name})
			if err != nil {
				t.Fatalf("find command %q: %v", tt.name, err)
			}
			if got != tt.want {
				t.Fatalf("command %q = %p, want %p", tt.name, got, tt.want)
			}
		})
	}
}
