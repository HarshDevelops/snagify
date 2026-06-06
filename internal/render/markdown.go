package render

import (
	"fmt"
	"io"

	"github.com/HarshDevelops/snagify/internal/diff"
)

// Markdown writes a GitHub-friendly markdown report to w.
func Markdown(w io.Writer, r diff.Result) {
	fmt.Fprintf(w, "# Snagify diff: %s\n\n", r.ProjectName)
	fmt.Fprintf(w, "Comparing **%s** (A) vs **%s** (B).\n\n", r.NameA, r.NameB)

	if !r.HasDifferences() {
		fmt.Fprintln(w, "✅ No differences found. These environments match.")
		return
	}

	mdSection(w, "Critical differences", r.Critical(), r)
	mdSection(w, "Differences", r.Differences(), r)
	mdSection(w, "Informational", r.Informational(), r)

	blockers := r.Blockers()
	if len(blockers) > 0 {
		fmt.Fprintf(w, "## Likely blockers for %s\n\n", r.NameB)
		for _, b := range blockers {
			fmt.Fprintf(w, "- %s\n", b)
		}
		fmt.Fprintln(w)
	}
}

func mdSection(w io.Writer, title string, entries []diff.Entry, r diff.Result) {
	if len(entries) == 0 {
		return
	}
	fmt.Fprintf(w, "## %s\n\n", title)
	fmt.Fprintf(w, "| Item | %s | %s | Status |\n", r.NameA, r.NameB)
	fmt.Fprintln(w, "| --- | --- | --- | --- |")
	for _, e := range entries {
		flag := ""
		if e.Blocker != "" {
			flag = "⚠️ likely blocker"
		}
		fmt.Fprintf(w, "| %s | %s | %s | %s |\n", e.Name, e.A, e.B, flag)
	}
	fmt.Fprintln(w)
}
