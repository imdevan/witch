## Feature 1: Finish template → witch conversion

- [x] 1.1 Package metadata
  - [x] 1.1.1 Update `internal/package/package.toml` description/short ("which, but better")
  - [x] 1.1.2 Module path → `github.com/imdevan/witch`
  - [x] 1.1.3 Sync `package.toml` `module` with `go.mod` and run `just sync`
- [x] 1.2 Remove template-only code
  - [x] 1.2.1 Strip directory-listing demo + stub actions from `cmd/witch/root.go`
  - [x] 1.2.2 Remove unused adapters (icon/tty/shell alias formatter) or keep only what witch uses
- [x] 1.3 Docs
  - [x] 1.3.1 Rewrite `README.md`, `INSTALL.md`, `example-config.toml` for witch
  - [x] 1.3.2 Update `config_init.go` defaults
- [x] 1.4 `just build` and `just test` pass after cleanup

## Feature 2: Sources 
- [x] 2.1 Domain model
  - [x] 2.1.1 `domain.Entry` { Name, Description, File, Line }
  - [x] 2.1.2 Alias sub-types: regular, global (`-g`), suffix (`-s`)
- [x] 2.2 Config sources
  - [x] 2.2.1 `[[sources]]` array in config.toml with `path` (file or dir)
  - [x] 2.2.2 No default location. Default behavior is just which fallback.
  - [x] 2.2.3 `[[ignore]]` array in config.toml can be direct file or blob pattern
  - [x] 2.2.4 Parse `alias [-g|-s] name="value" # comment`; comment → description
  - [x] 2.2.5 Expand brace suffix aliases (`alias -s {js,ts,...}=...`) into one entry each
  - [x] 2.2.6 Record file + line number
- [x] 2.3 Function parser 
  - [x] 2.3.1 Definitions: `function name()`, `function name {`, `name() {` → name + line
  - [x] 2.3.2 Aliases declared inside function files (e.g. `alias fd="fzf_cd"`) → alias entries
- [x] 2.4 PATH executables 
  - [x] 2.4.1 Walk `$PATH` in order, collect executable files
  - [x] 2.4.2 Keep every match; mark later duplicates as shadowed (like `which -a`)
  - [x] 2.4.3 Resolve symlinks for the preview
  - [x] 2.4.4 Config toggle to disable PATH indexing
- [x] 2.5 Collector merges all sources; precedence alias > function > PATH for exact-name lookups
- [x] 2.6 Unit tests per parser with fixture files copied from real dotfiles formats

## Feature 3: CLI behavior
- [x] 3.1 `witch <name>` non-interactive when exact match and stdout is not a TTY (drop-in for `which`)
- [x] 3.2 `witch` with no args / ambiguous match → interactive UI
- [x] 3.3 Flags: `-a/--all`, `-t/--type`, `--json`, `-i/--interactive`
- [x] 3.4 Exit code 1 when nothing found (match `which`)
- [x] 3.5 Update `README.md`, `example-config.toml`, `config_init.go` for new flags and `[[sources]]`
- [x] 3.6 cmd tests for non-interactive output

## Feature 4: Split UI
- [ ] 4.1 Split layout component in `internal/ui/split.go`
  - [ ] 4.1.1 List on left, preview viewport on right (lipgloss `JoinHorizontal`)
  - [ ] 4.1.2 Split only at `BreakpointLG`+; below that, list only + enter opens preview
  - [ ] 4.1.3 Inline (not alt-screen) per AGENTS.md
- [ ] 4.2 List pane
  - [ ] 4.2.1 Reuse `ui.NewListModel` / delegate; title = name, description = type + description
  - [ ] 4.2.2 Type badges colored from theme (`tags`, `flags`, `accent`)
  - [ ] 4.2.3 Fuzzy filtering (bubbles list built-in), pre-filled with CLI arg
  - [ ] 4.2.4 Type filter keys (e.g. `tab` to cycle all/alias/function/bookmark/bin)
- [ ] 4.3 Preview pane (bubbles `viewport`)
  - [ ] 4.3.1 preview pane should show file preview of given alias / function
  - [ ] 4.3.2 the user should see the contents of the file, with the alias / function highlighted so that it is clear what alias the user has searched and where it is in the file

- [ ] 4.4 Actions
  - [ ] 4.4.1 `enter` print selection and exit (shell-friendly)
  - [ ] 4.4.2 `y` copy name/path/definition (clipboard adapter)
  - [ ] 4.4.3 `o` open defining file at line in editor (editor adapter)
- [ ] 4.5 Responsive help keys, matching template patterns

## Feature 5: Release
- [ ] 5.1 Docs site via `just docs-generate`
- [ ] 5.2 `just release` / homebrew
