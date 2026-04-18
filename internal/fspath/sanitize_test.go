package fspath

import "testing"

func TestSanitize(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "fleet-a", "fleet-a"},
		{"slash", "a/b", "a_b"},
		{"backslash", `a\b`, "a_b"},
		{"space", "Azure DEV", "Azure-DEV"},
		{"colon", "host:22", "host_22"},
		{"multiple", "a/b c:d", "a_b-c_d"},
		{"empty", "", ""},
		{"no-op ascii", "alphaNumeric_123.txt", "alphaNumeric_123.txt"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Sanitize(tc.in)
			if got != tc.want {
				t.Errorf("Sanitize(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
