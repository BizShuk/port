package svc

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bizshuk/port/config"
)

func TestCheckHealthHTTPStatusCodes(t *testing.T) {
	cases := []struct {
		name    string
		code    int
		wantOK  bool
		wantMsg string
	}{
		{name: "ok", code: http.StatusOK, wantOK: true, wantMsg: "HTTP 200"},
		{name: "no content", code: http.StatusNoContent, wantOK: true, wantMsg: "HTTP 204"},
		{name: "service unavailable", code: http.StatusServiceUnavailable, wantOK: false, wantMsg: "HTTP 503"},
		{name: "not found", code: http.StatusNotFound, wantOK: false, wantMsg: "HTTP 404"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.code)
			}))
			defer server.Close()

			entries := []config.PortEntry{{Port: 1, Name: tc.name, Health: server.URL}}
			results := CheckHealth(context.Background(), entries, time.Second)

			if len(results) != 1 {
				t.Fatalf("got %d results, want 1", len(results))
			}
			if results[0].OK != tc.wantOK {
				t.Errorf("OK = %v, want %v", results[0].OK, tc.wantOK)
			}
			if results[0].Detail != tc.wantMsg {
				t.Errorf("Detail = %q, want %q", results[0].Detail, tc.wantMsg)
			}
		})
	}
}

func TestCheckHealthSkipsEntriesWithoutHealth(t *testing.T) {
	entries := []config.PortEntry{
		{Port: 22, Name: "ssh"},
		{Port: 11434, Name: "ollama"},
	}

	results := CheckHealth(context.Background(), entries, time.Second)
	if len(results) != 0 {
		t.Fatalf("got %d results, want 0 — entries without health must be skipped", len(results))
	}
}

func TestCheckHealthTCPFallback(t *testing.T) {
	// httptest 的 listener 就是一個真的 TCP listener，足以驗證 tcp 分支
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	port := server.Listener.Addr().(*net.TCPAddr).Port
	entries := []config.PortEntry{{Port: port, Name: "tcp-service", Health: HEALTH_TCP}}

	results := CheckHealth(context.Background(), entries, time.Second)
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if !results[0].OK {
		t.Errorf("OK = false, want true: %s", results[0].Detail)
	}
	if results[0].Probe[:6] != "tcp://" {
		t.Errorf("Probe = %q, want tcp:// prefix", results[0].Probe)
	}
}

func TestCheckHealthTCPClosedPort(t *testing.T) {
	// 綁一個 listener 再關掉，確保這個 port 當下沒有東西在聽
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	port := server.Listener.Addr().(*net.TCPAddr).Port
	server.Close()

	entries := []config.PortEntry{{Port: port, Name: "dead", Health: HEALTH_TCP}}
	results := CheckHealth(context.Background(), entries, time.Second)

	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].OK {
		t.Error("OK = true, want false for a closed port")
	}
}

func TestCheckHealthSortsByPort(t *testing.T) {
	entries := []config.PortEntry{
		{Port: 9000, Name: "c", Health: HEALTH_TCP},
		{Port: 1000, Name: "a", Health: HEALTH_TCP},
		{Port: 5000, Name: "b", Health: HEALTH_TCP},
	}

	results := CheckHealth(context.Background(), entries, 50*time.Millisecond)
	if len(results) != 3 {
		t.Fatalf("got %d results, want 3", len(results))
	}
	for i, want := range []int{1000, 5000, 9000} {
		if results[i].Entry.Port != want {
			t.Errorf("results[%d].Port = %d, want %d", i, results[i].Entry.Port, want)
		}
	}
}
