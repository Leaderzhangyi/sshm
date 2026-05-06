package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPath_Tilde(t *testing.T) {
	home, _ := os.UserHomeDir()
	got := ExpandPath("~/test/path")
	want := filepath.Join(home, "test/path")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExpandPath_NoTilde(t *testing.T) {
	got := ExpandPath("/absolute/path")
	if got != "/absolute/path" {
		t.Errorf("got %q, want %q", got, "/absolute/path")
	}
}

func TestParseJumpHost_Full(t *testing.T) {
	user, host, port := ParseJumpHost("root@192.168.1.1:2222")
	if user != "root" {
		t.Errorf("user: got %q, want %q", user, "root")
	}
	if host != "192.168.1.1" {
		t.Errorf("host: got %q, want %q", host, "192.168.1.1")
	}
	if port != 2222 {
		t.Errorf("port: got %d, want %d", port, 2222)
	}
}

func TestParseJumpHost_DefaultPort(t *testing.T) {
	_, _, port := ParseJumpHost("root@192.168.1.1")
	if port != 22 {
		t.Errorf("default port: got %d, want %d", port, 22)
	}
}

func TestParseJumpHost_HostOnly(t *testing.T) {
	user, host, port := ParseJumpHost("192.168.1.1")
	if user != "" {
		t.Errorf("user: got %q, want empty", user)
	}
	if host != "192.168.1.1" {
		t.Errorf("host: got %q, want %q", host, "192.168.1.1")
	}
	if port != 22 {
		t.Errorf("port: got %d, want %d", port, 22)
	}
}

func TestCleanUserPath_StripQuotes(t *testing.T) {
	got := CleanUserPath(`"C:\Users\test\conn"`)
	want := filepath.Clean(`C:\Users\test\conn`)
	if got != want {
		t.Errorf("strip double quotes: got %q, want %q", got, want)
	}

	got = CleanUserPath(`'C:\Users\test\conn'`)
	want = filepath.Clean(`C:\Users\test\conn`)
	if got != want {
		t.Errorf("strip single quotes: got %q, want %q", got, want)
	}
}

func TestCleanUserPath_TrimSpace(t *testing.T) {
	got := CleanUserPath("  /tmp/conn  ")
	want := filepath.Clean("/tmp/conn")
	if got != want {
		t.Errorf("trim space: got %q, want %q", got, want)
	}
}

func TestCleanUserPath_ExpandEnv(t *testing.T) {
	t.Setenv("SSHM_TEST_DIR", "/tmp/sshm_test")
	got := CleanUserPath("$SSHM_TEST_DIR/conn")
	want := filepath.Clean("/tmp/sshm_test/conn")
	if got != want {
		t.Errorf("expand env: got %q, want %q", got, want)
	}
}

func TestCleanUserPath_Tilde(t *testing.T) {
	home, _ := os.UserHomeDir()
	got := CleanUserPath("~/FinalShell/conn")
	want := filepath.Join(home, "FinalShell/conn")
	if got != want {
		t.Errorf("expand tilde: got %q, want %q", got, want)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "connections.json")

	original := &Config{
		Connections: []Connection{
			{Name: "test1", Host: "10.0.0.1", Port: 22, User: "root", Auth: "password", Password: "pw1"},
			{Name: "test2", Host: "10.0.0.2", Port: 22, User: "admin", Auth: "key", KeyPath: "/tmp/id_rsa"},
		},
	}

	data, _ := json.MarshalIndent(original, "", "  ")
	if err := os.WriteFile(cfgPath, data, 0600); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	var loaded Config
	if err := json.Unmarshal(raw, &loaded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(loaded.Connections) != 2 {
		t.Fatalf("expected 2 connections, got %d", len(loaded.Connections))
	}
	if loaded.Connections[0].Password != "pw1" {
		t.Errorf("password: got %q, want %q", loaded.Connections[0].Password, "pw1")
	}
	if loaded.Connections[1].KeyPath != "/tmp/id_rsa" {
		t.Errorf("key_path: got %q, want %q", loaded.Connections[1].KeyPath, "/tmp/id_rsa")
	}
}
