---
title: witch
description: which, but better
---

which, but better.

Resolve names the way your shell does: aliases, then functions, then
executables on $PATH. Aliases and functions are read from the files and
directories listed under [[sources]] in your config; with no sources
configured, witch behaves like which.

With a name and an exact match, witch prints the result and exits.
With no name, or no exact match on a terminal, it opens an interactive list.

## Usage

```bash
witch
```

## Flags

### Lookup

Control which entries are printed and how.

| Flag | Type | Description |
|------|------|-------------|
| -a, --all | bool | print all matches, including shadowed ones |
| -t, --type | stringslice | only match these types: alias, function, executable (repeatable or comma-separated) |
| --json | bool | print matches as JSON |
| -i, --interactive | bool | always open the interactive list |
| -o, --open | bool | open the file defining the alias or function in your editor, at its line |

### Config

Define a config.toml file to use instead of the global one.

| Flag | Type | Description |
|------|------|-------------|
| -c, --config | string | config file path |

### Meta



| Flag | Type | Description |
|------|------|-------------|
| -v, --version | bool | print version information |


## Available Commands


- [`completion`](/commands/completion) - Generate shell completion scripts
- [`config`](/commands/config) - View or edit configuration
- [`config-init`](/commands/config-init) - Generate a default config file
- [`init`](/commands/init) - Print shell integration so enter runs the selection in your shell

## Source

See [root.go](https://github.com/imdevan/witch/blob/main/cmd/witch/root.go) for implementation details.
