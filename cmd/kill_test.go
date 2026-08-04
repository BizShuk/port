package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestKillCmdKillsProcessListeningOnPort(t *testing.T) {
	var gotPort int
	kill := func(_ context.Context, port int) (int, error) {
		gotPort = port
		return 4321, nil
	}

	command := newKillCmd(kill)
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"8080"})

	if err := command.Execute(); err != nil {
		t.Fatalf("execute kill command: %v", err)
	}
	if gotPort != 8080 {
		t.Fatalf("port = %d, want 8080", gotPort)
	}
	want := "Killed process 4321 listening on port 8080\n"
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
}

func TestKillCmdRejectsPortOutsideValidRange(t *testing.T) {
	for _, port := range []string{"0", "65536"} {
		t.Run(port, func(t *testing.T) {
			called := false
			kill := func(_ context.Context, _ int) (int, error) {
				called = true
				return 0, nil
			}

			command := newKillCmd(kill)
			command.SetArgs([]string{port})

			err := command.Execute()
			if err == nil {
				t.Fatal("execute kill command succeeded, want invalid port error")
			}
			if !strings.Contains(err.Error(), "must be between 1 and 65535") {
				t.Fatalf("error = %q, want valid range message", err)
			}
			if called {
				t.Fatal("kill function called for invalid port")
			}
		})
	}
}
