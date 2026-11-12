//go:build unix

package manual

import (
	"os/exec"
	"syscall"
)

// detach starts c in its own session: it has no controlling terminal, so it
// can't read from or draw on the TTY, and it can be killed as a group.
func detach(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

// kill stops c and anything it started.
func kill(c *exec.Cmd) {
	if c.Process == nil {
		return
	}
	_ = syscall.Kill(-c.Process.Pid, syscall.SIGKILL)
	_ = c.Process.Kill()
}
