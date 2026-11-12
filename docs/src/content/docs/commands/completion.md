---
title: completion
description: Generate shell completion scripts
---

Examples:
  witch completion bash > /etc/bash_completion.d/witch
  witch completion zsh > ~/.zsh/completion/_witch
  witch completion fish > ~/.config/fish/completions/witch.fish
  witch completion powershell > witch.ps1

## Usage

```bash
witch completion [bash|zsh|fish|powershell]
```

## Source

See [completion.go](https://github.com/imdevan/witch/blob/main/cmd/witch/completion.go) for implementation details.
