package svc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

type commandOutput func(ctx context.Context, name string, args ...string) ([]byte, error)
type processSignal func(pid int, signal os.Signal) error

// ErrNoListeningProcess indicates that no process is listening on the requested port.
var ErrNoListeningProcess = errors.New("no listening process found")

// KillPortProcess terminates the process listening on port with SIGTERM.
func KillPortProcess(ctx context.Context, port int) (int, error) {
	return killPortProcess(ctx, port, runCommand, signalProcess)
}

func killPortProcess(ctx context.Context, port int, run commandOutput, signal processSignal) (int, error) {
	output, err := run(
		ctx,
		"lsof",
		"-nP",
		fmt.Sprintf("-iTCP:%d", port),
		"-sTCP:LISTEN",
		"-t",
	)
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return 0, fmt.Errorf("%w on port %d", ErrNoListeningProcess, port)
		}
		return 0, fmt.Errorf("find process listening on port %d: %w", port, err)
	}
	if len(strings.TrimSpace(string(output))) == 0 {
		return 0, fmt.Errorf("%w on port %d", ErrNoListeningProcess, port)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return 0, fmt.Errorf("parse listener PID for port %d: %w", port, err)
	}
	if err := signal(pid, syscall.SIGTERM); err != nil {
		return 0, fmt.Errorf("terminate process %d on port %d: %w", pid, port, err)
	}

	return pid, nil
}

func runCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

func signalProcess(pid int, signal os.Signal) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find process %d: %w", pid, err)
	}
	return process.Signal(signal)
}
