package verreq

import "testing"

func TestCheck(t *testing.T) {
	cases := []struct {
		found  string
		req    string
		ok     bool
		parsed bool
	}{
		{"17.0.9", "17", true, true},        // exact major
		{"21.0.2", "17", false, true},       // major mismatch
		{"17.0.9", "17.0.9", true, true},    // exact full
		{"17.0.10", "17.0.9", false, true},  // patch differs
		{"20.11.1", ">=20", true, true},     // at least
		{"19.0.0", ">=20", false, true},     // below
		{"22.3.0", ">=20 <23", true, true},  // in range
		{"23.0.0", ">=20 <23", false, true}, // at upper bound (exclusive)
		{"3.9.6", ">=3.8", true, true},      // minor at-least
		{"3.7.0", ">=3.8", false, true},     // below minor
		{"v20.11.1", ">=20", true, true},    // tolerate v prefix
		{"weird", ">=20", false, false},     // unparseable found
		{"20.0.0", "garbage", false, false}, // unparseable req
	}
	for _, c := range cases {
		got := Check(c.found, c.req)
		if got.OK != c.ok || got.Parsed != c.parsed {
			t.Errorf("Check(%q,%q) = {OK:%v Parsed:%v}, want {OK:%v Parsed:%v}",
				c.found, c.req, got.OK, got.Parsed, c.ok, c.parsed)
		}
	}
}

func TestIsRequiredKeyword(t *testing.T) {
	for _, s := range []string{"required", "REQUIRED", "present", "any", "*", ""} {
		if !IsRequiredKeyword(s) {
			t.Errorf("IsRequiredKeyword(%q) = false, want true", s)
		}
	}
	for _, s := range []string{">=20", "17", "17.0.9"} {
		if IsRequiredKeyword(s) {
			t.Errorf("IsRequiredKeyword(%q) = true, want false", s)
		}
	}
}
