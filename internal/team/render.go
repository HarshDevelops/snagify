package team

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	headerStyle = lipgloss.NewStyle().Bold(true)
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

// Terminal renders the team summary as an aligned table plus a blocker tally.
func Terminal(w io.Writer, s Summary) {
	fmt.Fprintln(w, headerStyle.Render("Team setup drift"))
	fmt.Fprintln(w)

	cols := append([]string{"Name"}, s.Columns...)
	widths := make([]int, len(cols))
	for i, c := range cols {
		widths[i] = len(c)
	}
	for _, m := range s.Machines {
		if len(m.Name) > widths[0] {
			widths[0] = len(m.Name)
		}
		for i, c := range s.Columns {
			if v := m.Cells[c]; displayWidth(v) > widths[i+1] {
				widths[i+1] = displayWidth(v)
			}
		}
	}

	writeRow(w, cols, widths)
	for _, m := range s.Machines {
		row := []string{m.Name}
		for _, c := range s.Columns {
			row = append(row, m.Cells[c])
		}
		writeRow(w, row, widths)
	}
	fmt.Fprintln(w)

	if len(s.CommonBlockers) > 0 {
		fmt.Fprintln(w, headerStyle.Render("Most common blockers:"))
		for _, b := range s.CommonBlockers {
			plural := "machine"
			if b.Count != 1 {
				plural = "machines"
			}
			fmt.Fprintf(w, "  • %s: %d %s\n", b.Blocker, b.Count, plural)
		}
	} else {
		fmt.Fprintln(w, dimStyle.Render("No blockers across the team."))
	}
}

func writeRow(w io.Writer, cells []string, widths []int) {
	var b strings.Builder
	for i, c := range cells {
		pad := widths[i] - displayWidth(c)
		if pad < 0 {
			pad = 0
		}
		b.WriteString(c)
		b.WriteString(strings.Repeat(" ", pad+2))
	}
	fmt.Fprintln(w, strings.TrimRight(b.String(), " "))
}

// displayWidth approximates terminal width counting the ❌ emoji as width 2.
func displayWidth(s string) int {
	w := 0
	for _, r := range s {
		if r == '❌' || r == '✅' {
			w += 2
		} else {
			w++
		}
	}
	return w
}

// Markdown renders the summary as a GitHub table.
func Markdown(w io.Writer, s Summary) {
	fmt.Fprintln(w, "# Team setup drift")
	fmt.Fprintln(w)

	cols := append([]string{"Name"}, s.Columns...)
	fmt.Fprintln(w, "| "+strings.Join(cols, " | ")+" |")
	fmt.Fprintln(w, "| "+strings.Repeat("--- | ", len(cols)))
	for _, m := range s.Machines {
		row := []string{m.Name}
		for _, c := range s.Columns {
			row = append(row, m.Cells[c])
		}
		fmt.Fprintln(w, "| "+strings.Join(row, " | ")+" |")
	}
	fmt.Fprintln(w)

	if len(s.CommonBlockers) > 0 {
		fmt.Fprintln(w, "## Most common blockers")
		fmt.Fprintln(w)
		for _, b := range s.CommonBlockers {
			fmt.Fprintf(w, "- %s: %d\n", b.Blocker, b.Count)
		}
		fmt.Fprintln(w)
	}
}

// JSON renders the summary as indented JSON.
func JSON(w io.Writer, s Summary) error {
	if s.Machines == nil {
		s.Machines = []MachineResult{}
	}
	if s.CommonBlockers == nil {
		s.CommonBlockers = []BlockerCount{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(s)
}

// Render dispatches by format.
func Render(w io.Writer, s Summary, format string) error {
	switch strings.ToLower(format) {
	case "markdown", "md":
		Markdown(w, s)
	case "json":
		return JSON(w, s)
	case "", "text", "terminal":
		Terminal(w, s)
	default:
		return fmt.Errorf("unknown format %q (use terminal, markdown, or json)", format)
	}
	return nil
}
