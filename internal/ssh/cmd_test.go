package ssh

import (
	"testing"

	"sshm/internal/config"
)

func TestBuildArgs_BasicPassword(t *testing.T) {
	c := &config.Connection{
		Host: "10.0.0.1",
		Port: 22,
		User: "root",
		Auth: "password",
	}
	args := BuildArgs(c)
	want := "root@10.0.0.1"
	if args[len(args)-1] != want {
		t.Errorf("last arg: got %q, want %q", args[len(args)-1], want)
	}
	for _, a := range args {
		if a == "-p" {
			t.Error("default port 22 should not appear in args")
		}
	}
}

func TestBuildArgs_CustomPort(t *testing.T) {
	c := &config.Connection{
		Host: "10.0.0.1",
		Port: 50022,
		User: "root",
		Auth: "password",
	}
	args := BuildArgs(c)
	found := false
	for i, a := range args {
		if a == "-p" && i+1 < len(args) && args[i+1] == "50022" {
			found = true
		}
	}
	if !found {
		t.Error("expected -p 50022 in args")
	}
}

func TestBuildArgs_KeyAuth(t *testing.T) {
	c := &config.Connection{
		Host:    "10.0.0.1",
		Port:    22,
		User:    "root",
		Auth:    "key",
		KeyPath: "/home/user/.ssh/id_rsa",
	}
	args := BuildArgs(c)
	found := false
	for i, a := range args {
		if a == "-i" && i+1 < len(args) && args[i+1] == "/home/user/.ssh/id_rsa" {
			found = true
		}
	}
	if !found {
		t.Error("expected -i /home/user/.ssh/id_rsa in args")
	}
}

func TestBuildArgs_ProxyJump(t *testing.T) {
	c := &config.Connection{
		Host:      "10.0.0.1",
		Port:      22,
		User:      "root",
		Auth:      "password",
		ProxyJump: "jump@192.168.1.1:22",
	}
	args := BuildArgs(c)
	found := false
	for i, a := range args {
		if a == "-J" && i+1 < len(args) && args[i+1] == "jump@192.168.1.1:22" {
			found = true
		}
	}
	if !found {
		t.Error("expected -J jump@192.168.1.1:22 in args")
	}
}

func TestBuildArgs_KeyAuthNoKeyPath(t *testing.T) {
	c := &config.Connection{
		Host: "10.0.0.1",
		Port: 22,
		User: "root",
		Auth: "key",
	}
	args := BuildArgs(c)
	for _, a := range args {
		if a == "-i" {
			t.Error("should not include -i when KeyPath is empty")
		}
	}
}
