package probes

import (
	"net/url"
	"strings"
)

// RedactProxyURL returns the proxy URL with the password replaced by "***".
// Returns the input unchanged if there is no password or the URL is empty.
func RedactProxyURL(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "(invalid proxy URL)"
	}
	if u.User == nil {
		return raw
	}
	_, hasPw := u.User.Password()
	if !hasPw {
		return raw
	}
	// Replace the raw userinfo section to avoid URL re-encoding artifacts.
	// URL format: scheme://userinfo@host/...
	// Extract raw userinfo from the original string.
	schemeEnd := strings.Index(raw, "://")
	if schemeEnd < 0 {
		return raw
	}
	afterScheme := raw[schemeEnd+3:]
	atIdx := strings.Index(afterScheme, "@")
	if atIdx < 0 {
		return raw
	}
	rawUserinfo := afterScheme[:atIdx]
	colonIdx := strings.Index(rawUserinfo, ":")
	if colonIdx < 0 {
		return raw
	}
	redactedUserinfo := rawUserinfo[:colonIdx] + ":***"
	return raw[:schemeEnd+3] + redactedUserinfo + raw[schemeEnd+3+atIdx:]
}

// RedactError returns the error message with any proxy password replaced by "***".
// Returns empty string for nil errors. Safe to call with empty proxyURL.
func RedactError(err error, proxyURL string) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	pw := extractPassword(proxyURL)
	if pw == "" {
		return msg
	}
	return strings.ReplaceAll(msg, pw, "***")
}

// extractPassword extracts the raw password from a proxy URL string.
func extractPassword(proxyURL string) string {
	if proxyURL == "" {
		return ""
	}
	u, err := url.Parse(proxyURL)
	if err != nil || u.User == nil {
		return ""
	}
	pw, _ := u.User.Password()
	return pw
}
