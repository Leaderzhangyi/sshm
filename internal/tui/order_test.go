package tui

import (
	"testing"

	"sshm/internal/config"
)

func TestBuildOrder_GroupSort(t *testing.T) {
	cfg := &config.Config{
		Connections: []config.Connection{
			{Name: "beta", Group: "z-group"},
			{Name: "alpha", Group: "a-group"},
			{Name: "middle", Group: "m-group"},
		},
	}
	order := buildOrder(cfg)
	if len(order) != 3 {
		t.Fatalf("expected 3, got %d", len(order))
	}
	if cfg.Connections[order[0]].Group != "a-group" {
		t.Errorf("first should be a-group, got %q", cfg.Connections[order[0]].Group)
	}
	if cfg.Connections[order[2]].Group != "z-group" {
		t.Errorf("last should be z-group, got %q", cfg.Connections[order[2]].Group)
	}
}

func TestBuildOrder_SameGroupSortByName(t *testing.T) {
	cfg := &config.Config{
		Connections: []config.Connection{
			{Name: "charlie", Group: "grp"},
			{Name: "alpha", Group: "grp"},
			{Name: "bravo", Group: "grp"},
		},
	}
	order := buildOrder(cfg)
	names := []string{
		cfg.Connections[order[0]].Name,
		cfg.Connections[order[1]].Name,
		cfg.Connections[order[2]].Name,
	}
	want := []string{"alpha", "bravo", "charlie"}
	for i, n := range names {
		if n != want[i] {
			t.Errorf("pos %d: got %q, want %q", i, n, want[i])
		}
	}
}
