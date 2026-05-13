// Package ssh builds external SSH commands and provides SFTP transfer.
package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"sshm/internal/config"
)

func BuildArgs(c *config.Connection) []string {
	var args []string
	if c.ProxyJump != "" {
		args = append(args, "-J", c.ProxyJump)
	}
	if c.Port != 0 && c.Port != 22 {
		args = append(args, "-p", strconv.Itoa(c.Port))
	}
	if c.Auth == "key" && c.KeyPath != "" {
		args = append(args, "-i", c.KeyPath)
	}
	args = append(args, fmt.Sprintf("%s@%s", c.User, c.Host))
	return args
}

func BuildCmd(c *config.Connection) *exec.Cmd {
	args := BuildArgs(c)

	if c.Auth == "password" && c.Password != "" {
		if sp, err := exec.LookPath("sshpass"); err == nil {
			return exec.Command(sp, append([]string{"-p", c.Password, "ssh"}, args...)...)
		}
		if runtime.GOOS != "windows" {
			if cmd := buildAskPassCmd(c.Password, args); cmd != nil {
				return cmd
			}
		}
	}

	if runtime.GOOS == "windows" {
		if wt, err := exec.LookPath("wt"); err == nil {
			return exec.Command(wt, append([]string{"ssh"}, args...)...)
		}
		sshArgs := append([]string{"ssh"}, args...)
		return exec.Command("cmd", "/c", "start", strings.Join(sshArgs, " "))
	}

	return exec.Command("ssh", args...)
}

func buildAskPassCmd(password string, sshArgs []string) *exec.Cmd {
	tmpFile, err := os.CreateTemp("", "sshm-askpass-")
	if err != nil {
		return nil
	}
	script := fmt.Sprintf("#!/bin/sh\ncat << 'SSHM_EOF'\n%s\nSSHM_EOF\n", password)
	if _, err := tmpFile.WriteString(script); err != nil {
		tmpFile.Close()
		return nil
	}
	tmpFile.Close()
	os.Chmod(tmpFile.Name(), 0700)

	cmd := exec.Command("setsid", append([]string{"ssh"}, sshArgs...)...)
	cmd.Env = append(os.Environ(),
		"SSH_ASKPASS="+tmpFile.Name(),
		"DISPLAY=dummy:0",
	)
	return cmd
}
