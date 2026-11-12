# witch

which, but better.

`witch` resolves a name the way your shell does: **aliases**, then **functions**, then **executables on `$PATH`**. It also tells you where each one is defined and what it shadows.

```console
$ witch ls
/usr/bin/ls

$ witch -a cat
~/.aliases/aliases.zsh:12
alias cat="bat" # cat with syntax highlighting
/usr/bin/cat

$ witch G
~/.aliases/aliases.zsh:39
alias -g G="| rg" # grep output

$ witch fcd
~/.functions/fcd.sh:6
fcd() {
  cd "$(fd --type d | fzf)"
}

$ witch nope; echo $?
witch: nope not found
1
```

## Features

- Works as a drop-in for `which`. Piped or scripted, `witch <name>` prints the result and exits 1 when nothing is found.
- Parses your dotfiles for aliases, including zsh global (`-g`) and suffix (`-s`) aliases and brace lists like `alias -s {js,ts}=nvim`, and for functions (`function f()`, `function f {`, `f() {`).
- Treats a trailing `# comment` on an alias, or the comment above a function, as its description.
- Shows the file, line and source of every alias and function definition.
- Like `which -a`, `--all` shows every match, with shadowed entries after the winner.
- Opens an inline, fuzzy-filterable [Bubble Tea](https://github.com/charmbracelet/bubbletea) browser when you run `witch` with no name, or when a name has no exact match. On wide terminals the list sits next to a preview of the defining file, scrolled to the definition with its lines highlighted.
- `--json` output for scripts and `fzf` pipelines.

## Usage

```bash
witch <name...>              # resolve one or more names
witch -a <name>              # show all matches, including shadowed ones
witch -t alias,function <n>  # only match some types (alias, function, executable)
witch --json <name>          # JSON output
witch                        # browse everything interactively
witch -i <query>             # always open the interactive list, pre-filtered
witch -o <name>              # open the defining file at the alias/function's line
witch -o                     # pick an entry interactively, then open it
```

### Flags

| Flag | Description |
|------|-------------|
| `-a`, `--all` | Print all matches, including shadowed ones (like `which -a`) |
| `-t`, `--type` | Only match these types: `alias`, `function`, `executable` (also `func`, `bin`, `path`). Repeatable or comma-separated |
| `--json` | Print matches as a JSON array |
| `-i`, `--interactive` | Always open the interactive list; arguments pre-fill the filter |
| `-o`, `--open` | Open the file that defines the alias or function in your `editor`, at its line. Takes one name (or none to pick interactively). Not supported for executables or with `--json` |
| `-c`, `--config` | Use this config file instead of the global/local ones |
| `-v`, `--version` | Print the version |

### Behavior

| Situation | Result |
|-----------|--------|
| `witch <name>` with an exact match | Prints the winning match (or every match with `-a`) and exits 0 |
| `witch <name>` with no match, stdout is a terminal, and something fuzzy-matches | Opens the interactive list, pre-filtered with `<name>` |
| `witch <name>` with no match, otherwise | Prints `witch: <name> not found` to stderr and exits 1 |
| `witch` on a terminal | Opens the interactive list |
| `witch` piped | Prints one `name  type  location` row per name |

Set `interactive_default = false` to never open the interactive list unless you pass `-i`.

Executables print their path, as `which` does. Aliases and functions print where they are defined (`file:line`, with your home directory shortened to `~`) followed by their source: the full `alias` line, or the whole function body. `--json` includes the same information as separate fields (`file`, `line`, `definition`).

### Interactive browser

At 80 columns or wider, the list is on the left and a preview of the selected entry's file is on the right. The definition is highlighted and scrolled into view. Narrower terminals show only the list, and `enter` opens a full-width preview (`esc` goes back).

In the preview, every whole-word occurrence of the entry's name is highlighted, and so is your search term (case-insensitive) when it differs from the name. Set the color with `highlight` in your config.

Executables that are scripts are previewed as text. Binaries show their path, symlink target, mode, and size.

The browser opens with the search focused: just type to filter (fuzzy), and the list and preview follow each keystroke. The search is pre-filled with the name you passed. A line under the search shows the type filter (`all aliases funcs bin`, current one emphasized) and the match count.

Press `esc` or `↓` to move into the list to scroll and use single-key actions; `/`, `esc`, or `↑` on the first item goes back to the search.

| Key | Search focused | List focused |
|-----|----------------|--------------|
| `enter` | Print the top/selected match and exit | Print the selection and exit |
| typing | Filter the list | — |
| `esc`, `↓` | Move to the list | — |
| `/`, `esc` | — | Back to the search (`↑` on the first item too) |
| `↑`/`↓`, `j`/`k`, `pgup`/`pgdn`, `g`/`G` | — (`ctrl+n`/`ctrl+p` move the selection) | Move through the list |
| `tab` / `shift+tab` | Cycle type: all → aliases → funcs → bin | same |
| `o` | — (`ctrl+o`) | Open the defining file at the definition's line in your `editor` |
| `y` | — (`ctrl+y`) | Copy the definition (an executable's path) |
| `Y` / `c` | — | Copy `path:line` / the name |
| `shift+↑`/`shift+↓` | Scroll the preview | Scroll the preview (also `J`/`K`, `ctrl+d`/`ctrl+u`) |
| `?` | — | Toggle full help |
| `q` | — | Quit |
| `ctrl+c` | Quit | Quit |

On narrow terminals `enter` opens a full-width preview first; there, `enter` prints, `esc` goes back, and `j`/`k` scroll.

The browser is inline, so it draws below your prompt and leaves only the printed result behind when it exits.

### Precedence

Matches for the same name are ordered the way your shell resolves them:

1. Aliases. If a name is defined more than once, the last definition wins, with sources read in config order.
2. Functions, also last definition wins.
3. Executables, in `$PATH` order. Later duplicates are shadowed.

Suffix aliases (`alias -s md=nvim`) apply only to file names, so they never shadow a command named `md`.

Names of subcommands (`config`, `completion`) run those subcommands. To look one up, run `witch -t executable config`.

## Configuration

With no config, `witch` only searches `$PATH`, like `which`. To include aliases and functions, list the files or directories that define them:

```toml
# ~/.config/witch/config.toml

[[sources]]
path = "~/.zshrc"

[[sources]]
path = "~/.functions"   # directories are searched recursively

[[ignore]]
path = "*.zwc"          # a glob without a slash matches file names anywhere
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `[[sources]] path` | string | none | Shell file or directory to parse for aliases and functions |
| `[[ignore]] path` | string | none | File, directory, or glob (`*`, `?`, `[...]`, `**`) to skip inside sources |
| `index_path` | bool | `true` | Index executables on `$PATH` |
| `interactive_default` | bool | `true` | Open the interactive list when no name is given (or none matches) on a terminal |
| `editor` | string | `nvim` | Editor used by `witch config` and `witch -o` |
| `list_spacing` | string | `space` | List density: `compact`, `tight`, `space` |
| `highlight` | string | `03` | Background color for the name and search term in the preview |

Paths accept `~` and `$VARS`. Relative paths resolve against the directory of the config file that declares them. A project-local `.witch/config.toml` adds its sources and ignores to the global ones.

See [`CONFIG.md`](CONFIG.md) for the full reference, including colors, and [`example-config.toml`](example-config.toml) for a ready-to-copy example.

```bash
witch config init            # write a commented default config
witch config init --force    # overwrite an existing one
witch config                 # open the config in your editor
```

## Installation

See [`INSTALL.md`](INSTALL.md).

## Development

Requires [Go](https://go.dev/), [Just](https://just.systems/), and [Bun](https://bun.sh/) (docs only).

```bash
just build           # build ./bin/witch
just build-run       # build and run
just test            # run all tests
just install         # install to /usr/local/bin
just docs-dev        # docs site with live reload
```

### Layout

```
cmd/witch/               CLI entrypoint, root lookup command, config + completion subcommands
internal/
  domain/                Config and Entry types
  config/                config.toml loading (global < local, or --config)
  sources/               alias/function parsers, PATH scanner, ignore globs, collector/index
  ui/                    Bubble Tea components (list, theme, responsive, entry picker)
  adapters/              editor, clipboard, tty
  package/package.toml   project metadata (run `just sync` after editing)
```

witch started from [go-cli-template](https://github.com/imdevan/go-cli-template).
