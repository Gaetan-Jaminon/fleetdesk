// Package notes provides a file-based note storage engine.
// Notes are plain text files stored at
// {fleet_dir}/notes/{fleet}/{segments...}/{timestamp}_note.txt
// and associated with resources (hosts, services, containers, VMs,
// AKS clusters, K8s namespaces/workloads/pods, fleets themselves).
package notes

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/Gaetan-Jaminon/fleetdesk/internal/fspath"
)

// ResourceRef identifies a resource within a fleet by stable name-based path.
// Segments are joined under the fleet name to form a directory, with each
// component sanitized for filesystem safety.
//
// Examples:
//
//	{Fleet: "aap-prod", Segments: ["hosts", "aap-ctrl-01", "services", "automation-controller-web"]}
//	{Fleet: "azure-dev", Segments: ["azure", "APP-DEV", "rg-app", "vm", "vm-ctrl-01"]}
//	{Fleet: "aks-dev", Segments: ["k8s", "AKS-APP-DEV-BLUE", "ctx-aks-dev-blue", "default", "nginx"]}
//	{Fleet: "aap-prod", Segments: nil} // fleet-level notes
type ResourceRef struct {
	Fleet    string
	Segments []string
}

// Dir returns the absolute directory path for this resource's notes, rooted
// under the provided base (typically fleet_dir/notes). Each segment is
// sanitized via fspath.Sanitize.
func (r ResourceRef) Dir(base string) string {
	parts := make([]string, 0, len(r.Segments)+2)
	parts = append(parts, base, fspath.Sanitize(r.Fleet))
	for _, seg := range r.Segments {
		parts = append(parts, fspath.Sanitize(seg))
	}
	return filepath.Join(parts...)
}

// Key returns a stable, compact string suitable for use as a map key (e.g.
// for caching note counts in the UI). Two refs with equal Fleet and equal
// Segments produce the same Key.
func (r ResourceRef) Key() string {
	var b strings.Builder
	b.WriteString(r.Fleet)
	for _, seg := range r.Segments {
		b.WriteString("/")
		b.WriteString(seg)
	}
	return b.String()
}

// Note represents a single persisted note.
type Note struct {
	// Path is the absolute path to the note file.
	Path string
	// CreatedAt is parsed from the timestamped filename (UTC).
	CreatedAt time.Time
	// Preview is the first non-empty line of the note, truncated to a
	// reasonable length for display in list views.
	Preview string
}
