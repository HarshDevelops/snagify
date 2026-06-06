package diff

import (
	"fmt"
	"strings"

	"github.com/harshdevelops/snagify/internal/model"
)

// diffEnvFiles compares declared-but-missing env keys between machines. A key
// missing on one machine but not the other is a likely blocker.
func diffEnvFiles(a, b model.EnvFiles) []Entry {
	var entries []Entry

	onlyB := subtract(b.MissingKeys, a.MissingKeys)
	onlyA := subtract(a.MissingKeys, b.MissingKeys)

	if len(onlyB) > 0 {
		entries = append(entries, Entry{
			Category: "Env", Name: ".env keys",
			A:        "present",
			B:        "missing: " + strings.Join(onlyB, ", "),
			Severity: Critical,
			Blocker:  fmt.Sprintf("required environment variables absent on B: %s", strings.Join(onlyB, ", ")),
		})
	}
	if len(onlyA) > 0 {
		entries = append(entries, Entry{
			Category: "Env", Name: ".env keys",
			A:        "missing: " + strings.Join(onlyA, ", "),
			B:        "present",
			Severity: Critical,
			Blocker:  fmt.Sprintf("required environment variables absent on A: %s", strings.Join(onlyA, ", ")),
		})
	}

	return entries
}

// subtract returns elements in x that are not in y.
func subtract(x, y []string) []string {
	yset := map[string]bool{}
	for _, v := range y {
		yset[v] = true
	}
	var out []string
	for _, v := range x {
		if !yset[v] {
			out = append(out, v)
		}
	}
	return out
}
