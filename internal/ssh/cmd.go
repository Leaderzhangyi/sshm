// Package ssh builds external SSH commands and provides SFTP transfer.
package ssh

import (
	"fmt"
	"os/exec"
	"strconv"

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
	}

	return exec.Command("ssh", args...)
}
