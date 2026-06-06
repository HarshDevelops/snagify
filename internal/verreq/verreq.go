// Package verreq implements small, dependency-free version requirement
// checking for Snagify config files.
//
// Supported requirement syntax (space-separated comparators, all ANDed):
//
//	"17"          -> major version must equal 17
//	"17.0.9"      -> all provided components must match exactly
//	">=20"        -> at least 20 (major)
//	">=20 <23"    -> at least 20 and below 23
//	">=3.8"       -> at least 3.8
//	"required"    -> handled by the caller (presence only); not a version
//
// If a requirement or found version cannot be parsed, Check returns
// ok=false and parsed=false so callers can downgrade to a warning rather
// than crashing.
package verreq

import (
	"regexp"
	"strconv"
	"strings"
)

// numRe extracts a dotted numeric version token from arbitrary text.
var numRe = regexp.MustCompile(`\d+(?:\.\d+)*`)

// Result reports the outcome of checking a found version against a requirement.
type Result struct {
	OK     bool // requirement satisfied
	Parsed bool // both requirement and version were parseable
}

// IsRequiredKeyword reports whether a requirement value just means "must be
// present" rather than a version constraint.
func IsRequiredKeyword(req string) bool {
	switch strings.ToLower(strings.TrimSpace(req)) {
	case "required", "present", "any", "*", "":
		return true
	}
	return false
}

// Check evaluates found against the requirement string.
func Check(found, requirement string) Result {
	fv := parse(found)
	if fv == nil {
		return Result{OK: false, Parsed: false}
	}

	comparators := strings.Fields(strings.TrimSpace(requirement))
	if len(comparators) == 0 {
		return Result{OK: true, Parsed: true}
	}

	for _, c := range comparators {
		ok, parsed := evalComparator(fv, c)
		if !parsed {
			return Result{OK: false, Parsed: false}
		}
		if !ok {
			return Result{OK: false, Parsed: true}
		}
	}
	return Result{OK: true, Parsed: true}
}

// evalComparator evaluates a single comparator like ">=20", "<23", "17", or
// "17.0.9" against the parsed found version.
func evalComparator(found []int, comp string) (ok, parsed bool) {
	op, rest := splitOp(comp)
	want := parse(rest)
	if want == nil {
		return false, false
	}

	switch op {
	case "=", "==":
		return compareN(found, want, len(want)) == 0, true
	case ">":
		return compareN(found, want, len(want)) > 0, true
	case ">=":
		return compareN(found, want, len(want)) >= 0, true
	case "<":
		return compareN(found, want, len(want)) < 0, true
	case "<=":
		return compareN(found, want, len(want)) <= 0, true
	case "":
		// Bare version: match only the components specified in want.
		return compareN(found, want, len(want)) == 0, true
	default:
		return false, false
	}
}

// splitOp separates a leading comparison operator from the version text.
func splitOp(s string) (op, rest string) {
	s = strings.TrimSpace(s)
	for _, prefix := range []string{">=", "<=", "==", ">", "<", "="} {
		if strings.HasPrefix(s, prefix) {
			return prefix, strings.TrimSpace(s[len(prefix):])
		}
	}
	return "", s
}

// parse extracts numeric version components from text, or nil if none found.
func parse(s string) []int {
	tok := numRe.FindString(strings.TrimSpace(s))
	if tok == "" {
		return nil
	}
	parts := strings.Split(tok, ".")
	nums := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			break
		}
		nums = append(nums, n)
	}
	if len(nums) == 0 {
		return nil
	}
	return nums
}

// compareN compares a and b considering only the first n components of b.
// Missing components in a are treated as zero. Returns -1, 0, or 1.
func compareN(a, b []int, n int) int {
	if n > len(b) {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		av := 0
		if i < len(a) {
			av = a[i]
		}
		bv := b[i]
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}
