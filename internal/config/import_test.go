package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestImportFinalShell_BasicImport(t *testing.T) {
	tmp := t.TempDir()
	stgDir := filepath.Join(tmp, "stg1")
	os.MkdirAll(stgDir, 0755)

	conn := map[string]interface{}{
		"host":                "10.0.0.1",
		"port":                22,
		"user_name":           "root",
		"name":                "test-server",
		"password":            "secret123",
		"authentication_type": 0,
	}
	data, _ := json.Marshal(conn)
	os.WriteFile(filepath.Join(stgDir, "server.json"), data, 0644)

	cfg := &Config{Connections: []Connection{}}
	imported, _, err := ImportFinalShell(tmp, cfg)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if imported != 1 {
		t.Fatalf("expected 1 imported, got %d", imported)
	}
	c := cfg.Connections[0]
	if c.Host != "10.0.0.1" {
		t.Errorf("host: got %q, want %q", c.Host, "10.0.0.1")
	}
	if c.User != "root" {
		t.Errorf("user: got %q, want %q", c.User, "root")
	}
	if c.Password != "secret123" {
		t.Errorf("password: got %q, want %q", c.Password, "secret123")
	}
	if c.Auth != "password" {
		t.Errorf("auth: got %q, want %q", c.Auth, "password")
	}
	if c.Group != "stg1" {
		t.Errorf("group: got %q, want %q", c.Group, "stg1")
	}
}

func TestImportFinalShell_KeyAuthWithKeyPath(t *testing.T) {
	tmp := t.TempDir()
	conn := map[string]interface{}{
		"host":                "10.0.0.2",
		"port":                22,
		"user_name":           "admin",
		"name":                "key-server",
		"secret_key_id":       "/home/user/.ssh/id_rsa",
		"authentication_type": 0,
	}
	data, _ := json.Marshal(conn)
	os.WriteFile(filepath.Join(tmp, "key.json"), data, 0644)

	cfg := &Config{Connections: []Connection{}}
	imported, _, err := ImportFinalShell(tmp, cfg)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if imported != 1 {
		t.Fatalf("expected 1, got %d", imported)
	}
	c := cfg.Connections[0]
	if c.Auth != "key" {
		t.Errorf("auth: got %q, want %q", c.Auth, "key")
	}
	if c.KeyPath != "/home/user/.ssh/id_rsa" {
		t.Errorf("key_path: got %q, want %q", c.KeyPath, "/home/user/.ssh/id_rsa")
	}
	if c.Password != "" {
		t.Errorf("password should be empty for key auth with key, got %q", c.Password)
	}
}

func TestImportFinalShell_KeyAuthButNoKeyFallsBackToPassword(t *testing.T) {
	tmp := t.TempDir()
	conn := map[string]interface{}{
		"host":                "10.0.0.3",
		"port":                50022,
		"user_name":           "mgmt",
		"name":                "fallback-server",
		"password":            "MyPassword",
		"secret_key_id":       "",
		"authentication_type": 0,
	}
	data, _ := json.Marshal(conn)
	os.WriteFile(filepath.Join(tmp, "fallback.json"), data, 0644)

	cfg := &Config{Connections: []Connection{}}
	imported, _, err := ImportFinalShell(tmp, cfg)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if imported != 1 {
		t.Fatalf("expected 1, got %d", imported)
	}
	c := cfg.Connections[0]
	if c.Auth != "password" {
		t.Errorf("auth should fall back to password, got %q", c.Auth)
	}
	if c.Password != "MyPassword" {
		t.Errorf("password: got %q, want %q", c.Password, "MyPassword")
	}
	if c.Port != 50022 {
		t.Errorf("port: got %d, want %d", c.Port, 50022)
	}
}

func TestImportFinalShell_SkipNoHost(t *testing.T) {
	tmp := t.TempDir()
	conn := map[string]interface{}{
		"name":                "folder-config",
		"authentication_type": 0,
	}
	data, _ := json.Marshal(conn)
	os.WriteFile(filepath.Join(tmp, "folder.json"), data, 0644)

	cfg := &Config{Connections: []Connection{}}
	imported, _, err := ImportFinalShell(tmp, cfg)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if imported != 0 {
		t.Errorf("expected 0 for no-host entry, got %d", imported)
	}
}

func TestImportFinalShell_Deduplicate(t *testing.T) {
	tmp := t.TempDir()
	conn := map[string]interface{}{
		"host":      "10.0.0.1",
		"port":      22,
		"user_name": "root",
		"name":      "server-a",
	}
	data, _ := json.Marshal(conn)
	os.WriteFile(filepath.Join(tmp, "a.json"), data, 0644)
	os.WriteFile(filepath.Join(tmp, "b.json"), data, 0644)

	cfg := &Config{Connections: []Connection{}}
	imported, skipped, err := ImportFinalShell(tmp, cfg)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if imported != 1 {
		t.Errorf("should deduplicate identical host+port+user, got %d", imported)
	}
	if skipped != 1 {
		t.Errorf("should skip 1 duplicate, got %d", skipped)
	}
}

func TestImportFinalShell_AllExist(t *testing.T) {
	tmp := t.TempDir()
	conn := map[string]interface{}{
		"host":      "10.0.0.1",
		"port":      22,
		"user_name": "root",
		"name":      "existing",
	}
	data, _ := json.Marshal(conn)
	os.WriteFile(filepath.Join(tmp, "exist.json"), data, 0644)

	cfg := &Config{Connections: []Connection{
		{Host: "10.0.0.1", Port: 22, User: "root"},
	}}
	imported, skipped, err := ImportFinalShell(tmp, cfg)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if imported != 0 {
		t.Errorf("imported should be 0, got %d", imported)
	}
	if skipped != 1 {
		t.Errorf("skipped should be 1, got %d", skipped)
	}
}

func TestImportFinalShell_NonexistentDir(t *testing.T) {
	cfg := &Config{Connections: []Connection{}}
	_, _, err := ImportFinalShell("/nonexistent/path/12345", cfg)
	if err == nil {
		t.Error("expected error for nonexistent dir")
	}
}

func TestImportFinalShell_NotDir(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "file.txt")
	os.WriteFile(f, []byte("test"), 0644)

	cfg := &Config{Connections: []Connection{}}
	_, _, err := ImportFinalShell(f, cfg)
	if err == nil {
		t.Error("expected error when path is a file, not dir")
	}
}

func TestImportFinalShell_QuotedPath(t *testing.T) {
	tmp := t.TempDir()
	conn := map[string]interface{}{
		"host":      "10.0.0.5",
		"port":      22,
		"user_name": "root",
		"name":      "quoted",
	}
	data, _ := json.Marshal(conn)
	os.WriteFile(filepath.Join(tmp, "q.json"), data, 0644)

	cfg := &Config{Connections: []Connection{}}
	imported, _, err := ImportFinalShell(`"`+tmp+`"`, cfg)
	if err != nil {
		t.Fatalf("import with quoted path failed: %v", err)
	}
	if imported != 1 {
		t.Errorf("expected 1, got %d", imported)
	}
}

func TestImportFinalShell_PortDefault22(t *testing.T) {
	tmp := t.TempDir()
	conn := map[string]interface{}{
		"host":      "10.0.0.6",
		"port":      0,
		"user_name": "root",
		"name":      "no-port",
	}
	data, _ := json.Marshal(conn)
	os.WriteFile(filepath.Join(tmp, "noport.json"), data, 0644)

	cfg := &Config{Connections: []Connection{}}
	imported, _, err := ImportFinalShell(tmp, cfg)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if imported != 1 {
		t.Fatalf("expected 1, got %d", imported)
	}
	if cfg.Connections[0].Port != 22 {
		t.Errorf("port should default to 22, got %d", cfg.Connections[0].Port)
	}
}
