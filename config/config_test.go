package config

import (
	"encoding/json"
	"testing"
)

func TestDefaultSettingsIncludeVitePorts(t *testing.T) {
	var settings Settings
	if err := json.Unmarshal([]byte(defaultSettingsJSON), &settings); err != nil {
		t.Fatalf("unmarshal default settings: %v", err)
	}

	want := map[int]string{
		5173: "vite",
		4173: "vite-preview",
	}
	got := make(map[int]string, len(settings.Ports))
	for _, entry := range settings.Ports {
		got[entry.Port] = entry.Name
	}

	for port, name := range want {
		if got[port] != name {
			t.Errorf("default port %d = %q, want %q", port, got[port], name)
		}
	}
}
