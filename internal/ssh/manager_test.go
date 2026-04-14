package ssh

import (
	"log/slog"
	"testing"
)

func TestSudoPasswordCache(t *testing.T) {
	sm := NewManager(slog.Default())

	// Initially empty
	if got := sm.GetSudoPassword(0); got != "" {
		t.Errorf("GetSudoPassword(0) = %q, want empty", got)
	}

	// Set and retrieve
	sm.SetSudoPassword(0, "secret")
	if got := sm.GetSudoPassword(0); got != "secret" {
		t.Errorf("GetSudoPassword(0) = %q, want %q", got, "secret")
	}

	// Per-host isolation
	if got := sm.GetSudoPassword(1); got != "" {
		t.Errorf("GetSudoPassword(1) = %q, want empty (different host)", got)
	}
	sm.SetSudoPassword(1, "other")
	if got := sm.GetSudoPassword(0); got != "secret" {
		t.Errorf("GetSudoPassword(0) = %q, want %q after setting host 1", got, "secret")
	}

	// Clear with empty string
	sm.SetSudoPassword(0, "")
	if got := sm.GetSudoPassword(0); got != "" {
		t.Errorf("GetSudoPassword(0) = %q, want empty after clear", got)
	}

	// Close clears all
	sm.SetSudoPassword(1, "stillset")
	sm.Close()
	if got := sm.GetSudoPassword(1); got != "" {
		t.Errorf("GetSudoPassword(1) = %q, want empty after Close()", got)
	}
}

func TestGetCachedPassword(t *testing.T) {
	sm := NewManager(slog.Default())

	if got := sm.GetCachedPassword(); got != "" {
		t.Errorf("GetCachedPassword() = %q, want empty initially", got)
	}

	sm.SetCachedPassword("mypassword")
	if got := sm.GetCachedPassword(); got != "mypassword" {
		t.Errorf("GetCachedPassword() = %q, want %q", got, "mypassword")
	}

	sm.ClearPassword()
	if got := sm.GetCachedPassword(); got != "" {
		t.Errorf("GetCachedPassword() = %q, want empty after ClearPassword", got)
	}
}

func TestRewriteSudoCmd(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		password string
		want     string
	}{
		{
			name:     "simple sudo",
			cmd:      "sudo journalctl -u sshd",
			password: "secret",
			want:     "echo 'secret' | sudo -S 2>/dev/null journalctl -u sshd",
		},
		{
			name:     "multiple sudo",
			cmd:      "sudo systemctl status && sudo journalctl",
			password: "pass",
			want:     "echo 'pass' | sudo -S 2>/dev/null systemctl status && echo 'pass' | sudo -S 2>/dev/null journalctl",
		},
		{
			name:     "password with single quote",
			cmd:      "sudo true",
			password: "it's a secret",
			want:     `echo 'it'\''s a secret' | sudo -S 2>/dev/null true`,
		},
		{
			name:     "no sudo in command",
			cmd:      "systemctl status",
			password: "irrelevant",
			want:     "systemctl status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rewriteSudoCmd(tt.cmd, tt.password)
			if got != tt.want {
				t.Errorf("rewriteSudoCmd(%q, %q)\n  got  %q\n  want %q", tt.cmd, tt.password, got, tt.want)
			}
		})
	}
}
