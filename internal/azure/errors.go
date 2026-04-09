package azure

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrAzNotInstalled indicates the az CLI binary was not found in PATH.
	ErrAzNotInstalled = errors.New("az CLI not found in PATH")

	// ErrNotLoggedIn indicates the user needs to run 'az login'.
	ErrNotLoggedIn = errors.New("not logged in to Azure (run 'az login')")

	// ErrSubscriptionNotFound indicates the requested subscription does not exist.
	ErrSubscriptionNotFound = errors.New("subscription not found")
)

// CLIError wraps an az CLI execution failure with exit code and stderr.
type CLIError struct {
	Command  string
	ExitCode int
	Stderr   string
}

func (e *CLIError) Error() string {
	return fmt.Sprintf("az %s: exit %d: %s", e.Command, e.ExitCode, e.Stderr)
}

// IsNotLoggedIn checks if an error indicates the user needs to run `az login`.
func IsNotLoggedIn(err error) bool {
	if errors.Is(err, ErrNotLoggedIn) {
		return true
	}
	var cliErr *CLIError
	if errors.As(err, &cliErr) {
		return strings.Contains(cliErr.Stderr, "az login") ||
			strings.Contains(cliErr.Stderr, "AADSTS") ||
			strings.Contains(cliErr.Stderr, "Please run 'az login'")
	}
	return false
}

// IsNotInstalled checks if an error indicates the az CLI is missing.
func IsNotInstalled(err error) bool {
	return errors.Is(err, ErrAzNotInstalled)
}
