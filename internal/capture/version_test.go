package capture

import "testing"

func TestParseVersion(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"node", "v20.11.1", "20.11.1"},
		{"go", "go version go1.23.2 darwin/arm64", "1.23.2"},
		{"python", "Python 3.12.4", "3.12.4"},
		{"npm", "10.2.4", "10.2.4"},
		{"java stderr", `openjdk version "17.0.9" 2023-10-17`, "17.0.9"},
		{"docker", "Docker version 25.0.3, build 4debf41", "25.0.3"},
		{"two part", "Maven 3.9", "3.9"},
		{"no version", "command not found", "command not found"},
		{"empty", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := parseVersion(c.raw); got != c.want {
				t.Errorf("parseVersion(%q) = %q, want %q", c.raw, got, c.want)
			}
		})
	}
}
