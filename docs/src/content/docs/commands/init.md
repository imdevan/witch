---
title: init
description: Print shell integration so enter runs the selection in your shell
---

Print a shell wrapper function for witch.

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
  witch init fish | source

## Usage

```bash
witch init <zsh|bash|fish>
```

## Source

See [init.go](https://github.com/imdevan/witch/blob/main/cmd/witch/init.go) for implementation details.
