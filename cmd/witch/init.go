package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init <zsh|bash|fish>",
		Short: "Print shell integration so enter runs the selection in your shell",
		Long: `Print a shell wrapper function for witch.

With it, pressing enter in the interactive browser runs the selected alias,
function, or executable in your current shell (and adds it to history), so
aliases that cd or set variables work. Without it, witch runs the selection
in an interactive subshell.

The wrapper only acts on interactive selections: witch <name> output is
unchanged, so it still works as a drop-in for which.

Examples:
  # ~/.zshrc
  eval "$(witch init zsh)"

  # ~/.bashrc
  eval "$(witch init bash)"

  # ~/.config/fish/config.fish
  witch init fish | source`,
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"zsh", "bash", "fish"},
		RunE: func(cmd *cobra.Command, args []string) error {
			script, err := initScript(args[0])
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), script)
			return nil
		},
	}
}

func initScript(shell string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(shell)) {
	case "zsh":
		return posixInit("print -s -- \"$__witch_cmd\""), nil
	case "bash":
		return posixInit("history -s -- \"$__witch_cmd\""), nil
	case "fish":
		return fishInit, nil
	default:
		return "", fmt.Errorf("unsupported shell %q (expected zsh, bash, or fish)", shell)
	}
}

// posixInit builds the zsh/bash wrapper; addHistory records the command.
func posixInit(addHistory string) string {
	return `# witch shell integration
witch() {
  local __witch_cmd_file __witch_cmd __witch_status
  __witch_cmd_file="$(mktemp "${TMPDIR:-/tmp}/witch.XXXXXX")" || { command witch "$@"; return $?; }
  ` + cmdFileEnv + `="$__witch_cmd_file" command witch "$@"
  __witch_status=$?
  __witch_cmd="$(cat -- "$__witch_cmd_file")"
  command rm -f -- "$__witch_cmd_file"
  if [ -n "$__witch_cmd" ]; then
    ` + addHistory + `
    eval "$__witch_cmd"
    return $?
  fi
  return $__witch_status
}
`
}

const fishInit = `# witch shell integration
function witch
    set -l __witch_cmd_file (mktemp)
    or begin; command witch $argv; return $status; end
    ` + cmdFileEnv + `=$__witch_cmd_file command witch $argv
    set -l __witch_status $status
    set -l __witch_cmd (cat -- $__witch_cmd_file | string collect)
    command rm -f -- $__witch_cmd_file
    if test -n "$__witch_cmd"
        eval $__witch_cmd
        return $status
    end
    return $__witch_status
end
`
