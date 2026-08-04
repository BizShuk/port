package svc

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"reflect"
	"syscall"
	"testing"
)

func TestKillPortProcessFindsListenerAndSendsSIGTERM(t *testing.T) {
	var gotCommand string
	var gotArgs []string
	run := func(_ context.Context, name string, args ...string) ([]byte, error) {
		gotCommand = name
		gotArgs = args
		return []byte("4321\n"), nil
	}

	var gotPID int
	var gotSignal os.Signal
	signal := func(pid int, sig os.Signal) error {
		gotPID = pid
		gotSignal = sig
		return nil
	}

	got, err := killPortProcess(context.Background(), 8080, run, signal)
	if err != nil {
		t.Fatalf("kill port process: %v", err)
	}
	if got != 4321 {
		t.Fatalf("PID = %d, want 4321", got)
	}
	if gotCommand != "lsof" {
		t.Fatalf("command = %q, want lsof", gotCommand)
	}
	wantArgs := []string{"-nP", "-iTCP:8080", "-sTCP:LISTEN", "-t"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("args = %q, want %q", gotArgs, wantArgs)
	}
	if gotPID != 4321 {
		t.Fatalf("signaled PID = %d, want 4321", gotPID)
	}
	if gotSignal != syscall.SIGTERM {
		t.Fatalf("signal = %v, want %v", gotSignal, syscall.SIGTERM)
	}
}

func TestKillPortProcessTreatsLSOFNoMatchAsMissingListener(t *testing.T) {
	run := func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return exec.Command("sh", "-c", "exit 1").Output()
	}
	signal := func(_ int, _ os.Signal) error {
		t.Fatal("signal called without a listening process")
		return nil
	}

	_, err := killPortProcess(context.Background(), 8080, run, signal)
	if !errors.Is(err, ErrNoListeningProcess) {
		t.Fatalf("error = %v, want ErrNoListeningProcess", err)
	}
}

func TestKillPortProcessReportsMissingListener(t *testing.T) {
	run := func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return nil, nil
	}
	signal := func(_ int, _ os.Signal) error {
		t.Fatal("signal called without a listening process")
		return nil
	}

	_, err := killPortProcess(context.Background(), 8080, run, signal)
	if !errors.Is(err, ErrNoListeningProcess) {
		t.Fatalf("error = %v, want ErrNoListeningProcess", err)
	}
}
