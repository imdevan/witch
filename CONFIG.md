# Configuration

Global config: `$XDG_CONFIG_HOME/witch/config.toml` (usually `~/.config/witch/config.toml`).

A project-local `.witch/config.toml` in the current directory is layered on top. Scalar options override the global ones, and `[[sources]]` / `[[ignore]]` entries are appended. `--config <path>` uses only that file.

## Sources

witch has no default sources. Without any, it only searches `$PATH`, like `which`.

| Option | Type | Description |
|--------|------|-------------|
| `[[sources]] path` | string | Shell file or directory to parse for aliases and functions. Directories are searched recursively, skipping hidden directories and binary files. |
| `[[ignore]] path` | string | File, directory, or glob pattern to skip inside sources. |
| `index_path` | bool (default `true`) | Index executables on `$PATH`. Set `false` to search only your sources. |

Paths accept `~` and `$VARS`. Relative paths resolve against the directory of the config file that declares them.

Ignore patterns:

- A plain path ignores that file, or everything under that directory.
- A glob with a slash is matched against the full path. `*` and `?` stay within one path segment, and `**` spans directories (e.g. `~/.config/zsh/plugins/**`).
- A glob without a slash is matched against file names anywhere (e.g. `*.zwc`).

### What gets parsed

- Aliases: `alias [-g|-s] name="value" # comment`. A trailing comment becomes the description. Brace lists like `alias -s {js,ts}=nvim` expand to one entry per name, and lines with several aliases (`alias a=b c=d`) produce one entry each. Names built at runtime (`alias ${name}=...`) are skipped.
- Functions: `function name()`, `function name {`, and `name() {`. A trailing comment on the definition line, or the comment block directly above it, becomes the description.

## General

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `editor` | string | `nvim` | Editor used by `witch config` and `witch -o` (falls back to `$VISUAL` / `$EDITOR` when empty) |
| `interactive_default` | bool | `true` | On a terminal, open the interactive list when no name is given or no name matches exactly |
| `list_spacing` | string | `space` | List density: `compact` (title only), `tight` (title + description, no margin), `space` (with spacing) |

## Colors

Colors support named, numeric, or hex values (e.g., `7`, `13`, `"#ff8800"`).

| Option | Default | Description |
|--------|---------|-------------|
| `headings` | `15` | Headings |
| `primary` | `02` | Primary color |
| `secondary` | `06` | Secondary color |
| `text` | `07` | Text |
| `text_highlight` | `06` | Highlighted text |
| `description_highlight` | `05` | Highlighted description |
| `tags` | `13` | Tags |
| `flags` | `12` | Flags |
| `accent` | `13` | Accent (fallback for `tags`) |
| `muted` | `08` | Muted text |
| `border` | `08` | Borders |
| `highlight` | `03` | Background for the entry name and search term in the preview |

## Example Configuration

```toml
# General
editor = "nvim"

# CLI behavior
# Open the interactive list when run with no name (or no exact match) on a terminal.
interactive_default = true

# Index executables on $PATH (like which). Set false to only search sources.
index_path = true

# UI
# list_spacing options: compact (title only), tight (title + description, no margin), space (default, with spacing)
list_spacing = "space"

# Colors
# Colors support named, numeric, or hex values (ex: 7, 13, "#ff8800").
headings = "15"
primary = "02"
secondary = "06"
text = "07"
text_highlight = "06"
description_highlight = "05"
tags = "13"
flags = "12"
accent = "13"
muted = "08"
border = "08"
# Background for the entry name and search term in the preview
highlight = "03"

# Sources
# Shell files or directories to parse for aliases and functions.
# Directories are searched recursively. There are no default sources:
# without any, witch only searches $PATH, like which.
# Paths support ~ and $VARS; relative paths are relative to this file.
[[sources]]
path = "~/.zshrc"

[[sources]]
path = "~/.aliases/aliases.zsh"

[[sources]]
path = "~/.functions"

# Ignore
# Files, directories, or glob patterns to skip inside sources.
# A glob without a slash (like "*.zwc") matches file names anywhere.
[[ignore]]
path = "*.zwc"

[[ignore]]
path = "~/.functions/old"
```

## Commands

```bash
witch config init            # write a commented default config
witch config init --force    # overwrite an existing config
witch config init --editor   # write it and open it in your editor
witch config                 # open the config in your editor
```
