package main

import (
	"os"
	"strings"
	"testing"
)

func TestModulePathMatchesPublicRepository(t *testing.T) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}

	const want = "module github.com/bizshuk/port"
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == want {
			return
		}
	}

	t.Fatalf("go.mod does not declare %q", want)
}
