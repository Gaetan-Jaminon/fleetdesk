package ssh

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ProbeInfo holds the results of a host probe.
type ProbeInfo struct {
	FQDN             string
	OS               string
	UpSince          string
	ServiceCount     int
	ServiceRunning   int
	ServiceFailed    int
	ContainerCount   int
	ContainerRunning int
	CronCount        int
	ErrorCount       int
	UpdateCount      int
	DiskCount        int
	DiskHighCount    int
	LastUpdate       string
	LastSecurity     string
	SystemdMode      string
}

// FormatDateEU converts a date string to DD/MM/YYYY format.
func FormatDateEU(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || s == "—" || s == "unknown" {
		return s
	}
	for _, layout := range []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("02/01/2006")
		}
	}
	return s
}

// ParseProbeOutput parses the probe command output.
func ParseProbeOutput(output string, systemdMode string) (ProbeInfo, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 7 {
		return ProbeInfo{}, fmt.Errorf("unexpected probe output (%d lines): %s", len(lines), output)
	}

	getInt := func(idx int) int {
		if idx < len(lines) {
			v, _ := strconv.Atoi(strings.TrimSpace(lines[idx]))
			return v
		}
		return 0
	}

	getDate := func(idx int) string {
		if idx < len(lines) {
			if v := strings.TrimSpace(lines[idx]); v != "" && v != "unknown" {
				return FormatDateEU(v)
			}
		}
		return "—"
	}

	return ProbeInfo{
		FQDN:             strings.TrimSpace(lines[0]),
		UpSince:          FormatDateEU(strings.TrimSpace(lines[1])),
		OS:               strings.TrimSpace(lines[2]),
		ServiceCount:     getInt(3),
		ServiceRunning:   getInt(4),
		ServiceFailed:    getInt(5),
		ContainerRunning: getInt(6),
		ContainerCount:   getInt(7),
		LastUpdate:       getDate(8),
		LastSecurity:     getDate(9),
		CronCount:        getInt(10),
		ErrorCount:       getInt(11),
		UpdateCount:      getInt(12),
		DiskCount:        getInt(13),
		DiskHighCount:    getInt(14),
		SystemdMode:      systemdMode,
	}, nil
}
