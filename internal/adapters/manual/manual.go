// Package manual finds help text for executables: the man page first, then
// the program's own --help output.
package manual

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"
)

// Timeout is the most time a man or --help lookup may take before it is
// killed. It is a variable so tests can shorten it.
var Timeout = 2 * time.Second

// maxOutput caps how much help text is kept.
const maxOutput = 512 * 1024

// ErrNoHelp is returned when an executable has no man page and doesn't
// appear to support --help.
var ErrNoHelp = errors.New("no man page or --help output")

// Page is help text for an executable.
type Page struct {
	// Source is the command that produced the text, e.g. "man ls".
	Source string
	Text   string
}

// Lookup returns the man page for name, falling back to running path
// --help. --help only runs when the file contains the text "--help", so
// programs that ignore their arguments aren't run by accident. width is the
// column count to format for.
func Lookup(name, path string, width int) (Page, error) {
	if text, err := manPage(name, width); err == nil {
		return Page{Source: "man " + name, Text: text}, nil
	}
	if !mentionsHelp(path) {
		return Page{}, ErrNoHelp
	}
	text, err := helpOutput(path, width)
	if err != nil {
		return Page{}, err
	}
	return Page{Source: name + " --help", Text: text}, nil
}

func manPage(name string, width int) (string, error) {
	if strings.ContainsAny(name, "/\x00") || strings.HasPrefix(name, "-") {
		return "", ErrNoHelp
	}
	env := []string{"MANPAGER=cat", "PAGER=cat", "GROFF_NO_SGR=1"}
	if width > 0 {
		env = append(env, fmt.Sprintf("MANWIDTH=%d", width))
	}
	return run(exec.Command("man", name), env, true)
}

func helpOutput(path string, width int) (string, error) {
	env := []string{"NO_COLOR=1", "PAGER=cat", "MANPAGER=cat", "TERM=dumb"}
	if width > 0 {
		env = append(env, fmt.Sprintf("COLUMNS=%d", width))
	}
	return run(exec.Command(path, "--help"), env, false)
}

// run executes c with no stdin, detached from the terminal so it can't draw
// over the UI, and returns its cleaned output. With strict, only stdout
// from a successful run counts (man prints "No manual entry" to stderr and
// fails). Otherwise stdout and stderr are used even on a non-zero exit,
// since many programs print help to stderr or exit 1 or 2 after --help.
func run(c *exec.Cmd, env []string, strict bool) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	var out bytes.Buffer
	w := &limitWriter{w: &out, n: maxOutput}
	c.Stdin = nil
	c.Stdout = w
	if !strict {
		c.Stderr = w
	}
	c.Env = append(os.Environ(), env...)
	detach(c)
	c.WaitDelay = 200 * time.Millisecond

	if err := c.Start(); err != nil {
		return "", err
	}
	done := make(chan error, 1)
	go func() { done <- c.Wait() }()
	var err error
	select {
	case err = <-done:
	case <-ctx.Done():
		kill(c)
		<-done
		return "", fmt.Errorf("%s timed out after %s", c.Path, Timeout)
	}
	if strict && err != nil {
		return "", err
	}
	text := Clean(out.String())
	if strings.TrimSpace(text) == "" {
		if err == nil {
			err = ErrNoHelp
		}
		return "", err
	}
	return text, nil
}

// mentionsHelp reports whether the file at path contains "--help".
func mentionsHelp(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	needle := []byte("--help")
	r := bufio.NewReaderSize(f, 64*1024)
	buf := make([]byte, 64*1024)
	var tail []byte
	for {
		n, err := r.Read(buf)
		if n > 0 {
			chunk := append(tail, buf[:n]...)
			if bytes.Contains(chunk, needle) {
				return true
			}
			// Keep enough to catch a match split across reads.
			tail = append([]byte(nil), chunk[max(len(chunk)-len(needle)+1, 0):]...)
		}
		if err != nil {
			return false
		}
	}
}

// Clean removes terminal formatting from help text: ANSI escapes, man's
// backspace overstrikes (bold/underline), and other control characters.
func Clean(s string) string {
	s = ansi.Strip(s)
	var b strings.Builder
	b.Grow(len(s))
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		// "X\bX" (bold) and "_\bX" (underline): keep the character after \b.
		if i+2 < len(runes) && runes[i+1] == '\b' {
			b.WriteRune(runes[i+2])
			i += 2
			for i+2 < len(runes) && runes[i+1] == '\b' {
				i += 2
			}
			continue
		}
		if r == '\n' || r == '\t' || r >= ' ' && r != 0x7f {
			b.WriteRune(r)
		}
	}
	lines := strings.Split(strings.ReplaceAll(b.String(), "\r", ""), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return strings.Trim(strings.Join(lines, "\n"), "\n")
}

type limitWriter struct {
	w io.Writer
	n int
}

func (l *limitWriter) Write(p []byte) (int, error) {
	if l.n <= 0 {
		return len(p), nil
	}
	keep := p
	if len(keep) > l.n {
		keep = keep[:l.n]
	}
	l.n -= len(keep)
	if _, err := l.w.Write(keep); err != nil {
		return 0, err
	}
	return len(p), nil
}
