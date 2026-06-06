package diff

import (
	"strconv"
	"strings"
)

// majorMinor splits a version string into numeric components.
func parseComponents(v string) []int {
	parts := strings.Split(v, ".")
	nums := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			break
		}
		nums = append(nums, n)
	}
	return nums
}

// majorDiffers reports whether two versions differ at the major version level.
// Missing components are treated as zero. If either string is unparseable, it
// falls back to a plain string inequality being treated as a minor difference.
func majorDiffers(a, b string) bool {
	ca, cb := parseComponents(a), parseComponents(b)
	if len(ca) == 0 || len(cb) == 0 {
		return false
	}
	return ca[0] != cb[0]
}
