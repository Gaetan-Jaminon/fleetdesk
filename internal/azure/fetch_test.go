package azure

import "testing"

func TestParseJSONCount(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  int
	}{
		{"positive number", []byte("12"), 12},
		{"zero", []byte("0"), 0},
		{"large number", []byte("1523"), 1523},
		{"with newline", []byte("42\n"), 42},
		{"with whitespace", []byte(" 7 "), 7},
		{"empty", []byte(""), 0},
		{"nil", nil, 0},
		{"invalid", []byte("not a number"), 0},
		{"json null", []byte("null"), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseJSONCount(tt.input)
			if got != tt.want {
				t.Errorf("ParseJSONCount(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
