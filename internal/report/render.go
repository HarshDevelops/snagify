package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true)
	statusBad     = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	statusOK      = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	criticalStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	warnStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	infoStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

// Terminal writes a colorized human-readable report.
func Terminal(w io.Writer, r Report) {
	fmt.Fprintln(w, titleStyle.Render("Project: "+orValue(r.Project, "unknown")))
	if r.Failed() {
		fmt.Fprintln(w, statusBad.Render(orValue(r.Status, "Check failed")))
	} else {
		fmt.Fprintln(w, statusOK.Render(orValue(r.Status, "Check passed")))
	}
	fmt.Fprintln(w)

	termSection(w, criticalStyle.Render("Critical blockers"), r.Critical(), r)
	termSection(w, warnStyle.Render("Warnings"), r.Warnings(), r)
	termSection(w, infoStyle.Render("Info"), r.Informational(), r)

	blockers := r.Blockers()
	if len(blockers) > 0 {
		fmt.Fprintln(w, criticalStyle.Render("Likely blockers:"))
		for _, b := range blockers {
			fmt.Fprintln(w, criticalStyle.Render("  • "+b))
		}
	}
}

func termSection(w io.Writer, title string, items []Item, r Report) {
	if len(items) == 0 {
		return
	}
	fmt.Fprintln(w, title)
	fmt.Fprintln(w, strings.Repeat("─", 17))

	nameW := 0
	foundW := 0
	for _, it := range items {
		if len(it.Name) > nameW {
			nameW = len(it.Name)
		}
		fl := r.FoundLabel + ": " + it.Found
		if len(fl) > foundW {
			foundW = len(fl)
		}
	}
	for _, it := range items {
		found := fmt.Sprintf("%s: %s", r.FoundLabel, it.Found)
		fmt.Fprintf(w, "%-*s  %-*s  %s: %s\n",
			nameW, it.Name, foundW, found, r.ExpLabel, it.Expected)
	}
	fmt.Fprintln(w)
}

// Markdown writes a GitHub-friendly report.
func Markdown(w io.Writer, r Report) {
	fmt.Fprintf(w, "# Snagify check: %s\n\n", orValue(r.Project, "unknown"))
	fmt.Fprintf(w, "**%s**\n\n", orValue(r.Status, statusText(r)))

	mdSection(w, "Critical blockers", r.Critical(), r)
	mdSection(w, "Warnings", r.Warnings(), r)
	mdSection(w, "Info", r.Informational(), r)

	if b := r.Blockers(); len(b) > 0 {
		fmt.Fprintln(w, "## Likely blockers")
		fmt.Fprintln(w)
		for _, x := range b {
			fmt.Fprintf(w, "- %s\n", x)
		}
		fmt.Fprintln(w)
	}
}

func mdSection(w io.Writer, title string, items []Item, r Report) {
	if len(items) == 0 {
		return
	}
	fmt.Fprintf(w, "## %s\n\n", title)
	fmt.Fprintf(w, "| Item | %s | %s |\n", r.FoundLabel, r.ExpLabel)
	fmt.Fprintln(w, "| --- | --- | --- |")
	for _, it := range items {
		fmt.Fprintf(w, "| %s | %s | %s |\n", it.Name, it.Found, it.Expected)
	}
	fmt.Fprintln(w)
}

// JSON writes the report as indented JSON for scripting.
func JSON(w io.Writer, r Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(newJSONView(r))
}

type jsonView struct {
	Project  string     `json:"project"`
	Status   string     `json:"status"`
	Found    string     `json:"found_label"`
	Expected string     `json:"expected_label"`
	Failed   bool       `json:"failed"`
	Blockers []string   `json:"blockers"`
	Items    []jsonItem `json:"items"`
}

type jsonItem struct {
	Category string `json:"category"`
	Name     string `json:"name"`
	Found    string `json:"found"`
	Expected string `json:"expected"`
	Severity string `json:"severity"`
	Blocker  string `json:"blocker,omitempty"`
}

func newJSONView(r Report) jsonView {
	v := jsonView{
		Project:  r.Project,
		Status:   orValue(r.Status, statusText(r)),
		Found:    r.FoundLabel,
		Expected: r.ExpLabel,
		Failed:   r.Failed(),
		Blockers: r.Blockers(),
	}
	if v.Blockers == nil {
		v.Blockers = []string{}
	}
	for _, it := range r.Items {
		v.Items = append(v.Items, jsonItem{
			Category: it.Category,
			Name:     it.Name,
			Found:    it.Found,
			Expected: it.Expected,
			Severity: it.Severity.String(),
			Blocker:  it.Blocker,
		})
	}
	if v.Items == nil {
		v.Items = []jsonItem{}
	}
	return v
}

func statusText(r Report) string {
	if r.Failed() {
		return "Check failed"
	}
	return "Check passed"
}

func orValue(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// Render dispatches to the renderer matching format ("terminal"/"text",
// "markdown"/"md", "json").
func Render(w io.Writer, r Report, format string) error {
	switch strings.ToLower(format) {
	case "markdown", "md":
		Markdown(w, r)
	case "json":
		return JSON(w, r)
	case "", "text", "terminal":
		Terminal(w, r)
	default:
		return fmt.Errorf("unknown format %q (use terminal, markdown, or json)", format)
	}
	return nil
}
