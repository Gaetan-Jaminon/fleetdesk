package azure

import (
	"errors"
	"fmt"
	"testing"
)

func TestIsNotLoggedIn(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"sentinel error", ErrNotLoggedIn, true},
		{"wrapped sentinel", fmt.Errorf("check failed: %w", ErrNotLoggedIn), true},
		{"cli error with az login", &CLIError{Command: "account list", ExitCode: 1, Stderr: "Please run 'az login' to setup account."}, true},
		{"cli error with AADSTS", &CLIError{Command: "vm list", ExitCode: 1, Stderr: "AADSTS700082: The refresh token has expired"}, true},
		{"unrelated cli error", &CLIError{Command: "vm list", ExitCode: 1, Stderr: "ResourceGroupNotFound"}, false},
		{"unrelated error", errors.New("connection timeout"), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNotLoggedIn(tt.err)
			if got != tt.want {
				t.Errorf("IsNotLoggedIn() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNotInstalled(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"sentinel error", ErrAzNotInstalled, true},
		{"wrapped sentinel", fmt.Errorf("prereq: %w", ErrAzNotInstalled), true},
		{"unrelated error", errors.New("some error"), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNotInstalled(tt.err)
			if got != tt.want {
				t.Errorf("IsNotInstalled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCLIErrorMessage(t *testing.T) {
	err := &CLIError{Command: "vm list", ExitCode: 2, Stderr: "something failed"}
	want := "az vm list: exit 2: something failed"
	if err.Error() != want {
		t.Errorf("CLIError.Error() = %q, want %q", err.Error(), want)
	}
}
