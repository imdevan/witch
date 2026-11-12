package main

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/imdevan/witch/internal/domain"
	"github.com/imdevan/witch/internal/utils"
)

// formatEntry renders an entry for non-interactive output. Executables print
// their path, like which. Aliases and functions print where they are defined
// followed by the source that defines them:
//
//	~/.aliases/aliases.zsh:447
//	alias asdf="cowsay 'hi'"
func formatEntry(e domain.Entry) string {
	if e.Type == domain.TypeExecutable {
		return e.File
	}
	definition := e.Definition
	if definition == "" {
		definition = fallbackDefinition(e)
	}
	return utils.DisplayPath(e.Location()) + "\n" + definition
}

// fallbackDefinition reconstructs source for entries without raw text.
func fallbackDefinition(e domain.Entry) string {
	if e.Type == domain.TypeFunction {
		if e.Value != "" {
			return e.Value
		}
		return e.Name + "() { ... }"
	}
	flag := ""
	switch e.AliasKind {
	case domain.AliasGlobal:
		flag = "-g "
	case domain.AliasSuffix:
		flag = "-s "
	}
	return fmt.Sprintf("alias %s%s=%q", flag, e.Name, e.Value)
}

// writeList prints one tab-separated row per entry: name, type, location.
func writeList(w io.Writer, entries []domain.Entry) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, e := range entries {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", e.Name, e.TypeLabel(), e.Location())
	}
	_ = tw.Flush()
}

func writeJSON(w io.Writer, entries []domain.Entry) error {
	if entries == nil {
		entries = []domain.Entry{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(entries)
}
