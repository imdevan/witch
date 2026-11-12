package domain

import (
	"fmt"
	"strings"
)

// EntryType identifies what kind of command an entry resolves to.
type EntryType string

const (
	// TypeAlias is a shell alias.
	TypeAlias EntryType = "alias"
	// TypeFunction is a shell function.
	TypeFunction EntryType = "function"
	// TypeExecutable is an executable file found on $PATH.
	TypeExecutable EntryType = "executable"
)

// Precedence returns the shell resolution order for the type.
// Lower values win: alias > function > executable.
func (t EntryType) Precedence() int {
	switch t {
	case TypeAlias:
		return 0
	case TypeFunction:
		return 1
	case TypeExecutable:
		return 2
	default:
		return 3
	}
}

// ParseEntryType converts user input (e.g. from --type) into an EntryType.
// It accepts a few common shorthands.
func ParseEntryType(value string) (EntryType, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "alias", "aliases", "a":
		return TypeAlias, nil
	case "function", "functions", "func", "fn", "f":
		return TypeFunction, nil
	case "executable", "executables", "exec", "bin", "path", "command", "e":
		return TypeExecutable, nil
	default:
		return "", fmt.Errorf("unknown type %q (expected alias, function, or executable)", value)
	}
}

// AliasKind distinguishes zsh alias sub-types.
type AliasKind string

const (
	// AliasRegular is a plain `alias name=value`.
	AliasRegular AliasKind = "regular"
	// AliasGlobal is `alias -g name=value`, expanded anywhere on the line.
	AliasGlobal AliasKind = "global"
	// AliasSuffix is `alias -s ext=value`, used to open files by extension.
	AliasSuffix AliasKind = "suffix"
)

// Entry is a single resolvable name: an alias, a function, or an executable.
type Entry struct {
	Name        string    `json:"name"`
	Type        EntryType `json:"type"`
	AliasKind   AliasKind `json:"alias_kind,omitempty"`
	Value       string    `json:"value,omitempty"`
	Description string    `json:"description,omitempty"`
	// Definition is the source text that defines an alias (its full line)
	// or a function (its full body).
	Definition string `json:"definition,omitempty"`
	// File is the defining file for aliases and functions, or the
	// executable path for PATH entries.
	File string `json:"file"`
	// Line is the 1-indexed line the entry is defined on (0 for executables).
	Line int `json:"line,omitempty"`
	// Target is the resolved symlink destination for executables.
	Target string `json:"target,omitempty"`
	// Shadowed is true when another entry with the same name wins.
	Shadowed bool `json:"shadowed"`
}

// TypeLabel returns a human-readable type, including the alias kind.
func (e Entry) TypeLabel() string {
	if e.Type == TypeAlias && e.AliasKind != "" && e.AliasKind != AliasRegular {
		return fmt.Sprintf("%s alias", e.AliasKind)
	}
	return string(e.Type)
}

// Location returns "file:line" for definitions, or the file alone.
func (e Entry) Location() string {
	if e.Line > 0 {
		return fmt.Sprintf("%s:%d", e.File, e.Line)
	}
	return e.File
}
