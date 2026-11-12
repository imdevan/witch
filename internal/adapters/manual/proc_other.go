//go:build !unix

package manual

import "os/exec"

func detach(*exec.Cmd) {}

func kill(c *exec.Cmd) {
	if c.Process != nil {
		_ = c.Process.Kill()
	}
}
