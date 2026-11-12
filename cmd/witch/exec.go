package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/imdevan/witch/internal/domain"
)

// cmdFileEnv is set by the shell integration (`witch init <shell>`). When
// present, witch writes the command to run into this file and the shell
// wrapper evals it in the user's current shell.
const cmdFileEnv = "WITCH_CMD_FILE"

// runInShell runs a command line in the user's interactive shell so aliases
// and functions from their rc files resolve. It is a variable so tests can
// stub it out.
var runInShell = func(line string) error {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "sh"
	}
	c := exec.Command(shell, "-ic", line)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	return c.Run()
}

// commandFor returns the shell command that runs e, or "" when e can't be
// run as a command on its own (global and suffix aliases).
func commandFor(e domain.Entry) string {
	switch e.Type {
	case domain.TypeExecutable:
		// Use the path so a shadowing alias or function doesn't run instead.
		return shellQuote(e.File)
	case domain.TypeFunction:
		return e.Name
	case domain.TypeAlias:
		if e.AliasKind == "" || e.AliasKind == domain.AliasRegular {
			return e.Name
		}
	}
	return ""
}

// executeEntry runs the selected entry. With shell integration it hands the
// command to the wrapper (so aliases, functions, and cd run in the current
// shell); otherwise it runs it in an interactive subshell.
func executeEntry(cmd *cobra.Command, e domain.Entry) error {
	line := commandFor(e)
	if line == "" {
		// Not runnable by itself: show what it is instead.
		fmt.Fprintln(cmd.OutOrStdout(), formatEntry(e))
		return nil
	}
	if file := os.Getenv(cmdFileEnv); file != "" {
		return os.WriteFile(file, []byte(line+"\n"), 0o600)
	}
	err := runInShell(line)
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		// Propagate the command's exit status quietly.
		return &exitError{code: exitErr.ExitCode()}
	}
	return err
}

// shellQuote single-quotes s for POSIX shells when it contains anything
// beyond safe path characters.
func shellQuote(s string) string {
	if s != "" && strings.Trim(s, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789/._-+@:,%") == "" {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
