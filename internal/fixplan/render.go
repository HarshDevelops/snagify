package fixplan

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	headerStyle = lipgloss.NewStyle().Bold(true)
	safeStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true) // green
	guidedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))            // yellow
	manualStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))             // gray
	cmdStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))            // blue
)

// Terminal renders the fix plan to w.
func Terminal(w io.Writer, p FixPlan) {
	fmt.Fprintf(w, "%s\n\n", headerStyle.Render("Project: "+p.Project))

	if len(p.Safe) > 0 {
		fmt.Fprintln(w, safeStyle.Render("Safe fixes available"))
		fmt.Fprintln(w, strings.Repeat("─", 20))
		for i, a := range p.Safe {
			fmt.Fprintf(w, "[%d] %s\n", i+1, a.Title)
			if a.Description != "" {
				fmt.Fprintf(w, "    %s\n", a.Description)
			}
		}
		fmt.Fprintln(w)
		fmt.Fprintln(w, safeStyle.Render("Run `snagify fix --safe` to apply."))
		fmt.Fprintln(w)
	}

	if len(p.Guided) > 0 {
		fmt.Fprintln(w, guidedStyle.Render("Guided fixes"))
		fmt.Fprintln(w, strings.Repeat("─", 20))
		for _, a := range p.Guided {
			fmt.Fprintf(w, "%s\n", guidedStyle.Render(a.Title))
			if a.Description != "" {
				fmt.Fprintf(w, "  %s\n", a.Description)
			}
			for _, c := range a.Commands {
				fmt.Fprintf(w, "    %s\n", cmdStyle.Render(c))
			}
			fmt.Fprintln(w)
		}
	}

	if len(p.Unfixable) > 0 {
		fmt.Fprintln(w, manualStyle.Render("Informational / manual"))
		fmt.Fprintln(w, strings.Repeat("─", 20))
		for _, a := range p.Unfixable {
			fmt.Fprintf(w, "  %s\n", a.Title)
			if a.Description != "" {
				fmt.Fprintf(w, "    %s\n", manualStyle.Render(a.Description))
			}
		}
		fmt.Fprintln(w)
	}

	if !p.HasFixes() && len(p.Unfixable) == 0 {
		fmt.Fprintln(w, safeStyle.Render("No blockers detected. Setup looks good."))
	}
}

// JSON renders the fix plan as indented JSON.
func JSON(w io.Writer, p FixPlan) error {
	type view struct {
		Project   string      `json:"project"`
		Safe      []FixAction `json:"safe"`
		Guided    []FixAction `json:"guided"`
		Manual    []FixAction `json:"manual"`
		Unfixable []FixAction `json:"unfixable"`
	}
	v := view{Project: p.Project, Safe: p.Safe, Guided: p.Guided, Manual: p.Manual, Unfixable: p.Unfixable}
	if v.Safe == nil {
		v.Safe = []FixAction{}
	}
	if v.Guided == nil {
		v.Guided = []FixAction{}
	}
	if v.Manual == nil {
		v.Manual = []FixAction{}
	}
	if v.Unfixable == nil {
		v.Unfixable = []FixAction{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// Render dispatches by format.
func Render(w io.Writer, p FixPlan, format string) error {
	switch strings.ToLower(format) {
	case "json":
		return JSON(w, p)
	default:
		Terminal(w, p)
		return nil
	}
}

// RenderResult renders an ApplyResult to w.
func RenderResult(w io.Writer, r ApplyResult) {
	for _, s := range r.Applied {
		fmt.Fprintf(w, "  %s %s\n", safeStyle.Render("✓"), s)
	}
	for _, s := range r.Skipped {
		fmt.Fprintf(w, "  %s %s\n", manualStyle.Render("·"), s)
	}
	for _, e := range r.Errors {
		fmt.Fprintf(w, "  %s %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("✗"), e)
	}
}
