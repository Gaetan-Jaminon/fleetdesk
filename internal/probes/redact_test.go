package probes

import (
	"fmt"
	"testing"
)

func TestRedactProxyURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"with credentials", "http://user:secret@proxy.corp:8080", "http://user:***@proxy.corp:8080"},
		{"user only no password", "http://user@proxy.corp:8080", "http://user@proxy.corp:8080"},
		{"no userinfo", "http://proxy.corp:8080", "http://proxy.corp:8080"},
		{"empty string", "", ""},
		{"https with credentials", "https://admin:p4ss@proxy.corp:3128", "https://admin:***@proxy.corp:3128"},
		{"special chars in password", "http://user:p%40ss%21@proxy.corp:8080", "http://user:***@proxy.corp:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RedactProxyURL(tt.raw)
			if got != tt.want {
				t.Errorf("RedactProxyURL(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestRedactError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		proxyURL string
		wantSafe bool // true = output must not contain the password
	}{
		{
			"error with proxy password",
			fmt.Errorf("proxyconnect tcp: dial tcp user:secret@proxy.corp:8080: connection refused"),
			"http://user:secret@proxy.corp:8080",
			true,
		},
		{
			"error without password",
			fmt.Errorf("connection refused"),
			"http://proxy.corp:8080",
			true,
		},
		{
			"empty proxy URL",
			fmt.Errorf("connection refused"),
			"",
			true,
		},
		{
			"nil error",
			nil,
			"http://user:secret@proxy.corp:8080",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RedactError(tt.err, tt.proxyURL)
			if tt.wantSafe && tt.proxyURL != "" {
				if pw := extractPassword(tt.proxyURL); pw != "" {
					if contains(got, pw) {
						t.Errorf("RedactError() = %q, still contains password %q", got, pw)
					}
				}
			}
			if tt.err == nil && got != "" {
				t.Errorf("RedactError(nil, ...) = %q, want empty", got)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) && (s == substr || findSubstring(s, substr))
}

func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
