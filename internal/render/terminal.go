// Package render formats a diff.Result into terminal, markdown, or JSON output.
package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/harshdevelops/snagify/internal/diff"
)

var (
	headerStyle   = lipgloss.NewStyle().Bold(true)
	criticalStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)  // red
	diffStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))            // yellow
	infoStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))             // gray
	blockerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))             // red
	okStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true) // green
)

// Terminal writes a human-readable, colorized diff to w.
func Terminal(w io.Writer, r diff.Result) {
	fmt.Fprintln(w, headerStyle.Render("Project: "+r.ProjectName))
	fmt.Fprintf(w, "%s = %s   %s = %s\n",
		labelA(), r.NameA, labelB(), r.NameB)
	fmt.Fprintln(w)

	if !r.HasDifferences() {
		fmt.Fprintln(w, okStyle.Render("No differences found. These environments match."))
		return
	}

	section(w, criticalStyle.Render("Critical differences"), r.Critical(), r)
	section(w, diffStyle.Render("Differences"), r.Differences(), r)
	section(w, infoStyle.Render("Informational"), r.Informational(), r)

	blockers := r.Blockers()
	if len(blockers) > 0 {
		fmt.Fprintln(w, criticalStyle.Render(fmt.Sprintf("Likely blockers for %s:", r.NameB)))
		for _, b := range blockers {
			fmt.Fprintln(w, blockerStyle.Render("  • "+b))
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, infoStyle.Render("Run `snagify snapshot` on both machines to capture the latest state."))
}

func section(w io.Writer, title string, entries []diff.Entry, r diff.Result) {
	if len(entries) == 0 {
		return
	}
	fmt.Fprintln(w, title)
	fmt.Fprintln(w, strings.Repeat("─", 20))

	nameW, aW := columnWidths(entries)
	for _, e := range entries {
		marker := ""
		if e.Blocker != "" {
			marker = blockerStyle.Render("  ← likely blocker")
		}
		fmt.Fprintf(w, "%-*s  %s: %-*s  %s: %s%s\n",
			nameW, e.Name,
			r.NameA, aW, e.A,
			r.NameB, e.B,
			marker,
		)
	}
	fmt.Fprintln(w)
}

func columnWidths(entries []diff.Entry) (nameW, aW int) {
	for _, e := range entries {
		if len(e.Name) > nameW {
			nameW = len(e.Name)
		}
		if len(e.A) > aW {
			aW = len(e.A)
		}
	}
	return nameW, aW
}

func labelA() string { return "A" }
func labelB() string { return "B" }
