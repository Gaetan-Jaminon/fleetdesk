package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// logWriter manages saving streamed log output to a file.
// Usage: create with newLogWriter, write lines with WriteLine, close with Close.
type logWriter struct {
	file *os.File
}

// newLogWriter creates a log save file at:
//
//	<fleetDir>/logs/<fleet>/<host>/<source>-<datetime>.log
//
// Returns nil (no error) if the file cannot be created — log saving is best-effort.
func newLogWriter(fleetDir, fleetName, hostName, sourceName string) *logWriter {
	dir := filepath.Join(fleetDir, "logs", sanitizePathComponent(fleetName), sanitizePathComponent(hostName))
	os.MkdirAll(dir, 0755)

	fileName := fmt.Sprintf("%s-%s.log", sanitizePathComponent(sourceName), time.Now().Format("2006-01-02_150405"))
	f, err := os.Create(filepath.Join(dir, fileName))
	if err != nil {
		return nil
	}
	return &logWriter{file: f}
}

// WriteLine writes a single line to the log file.
func (lw *logWriter) WriteLine(line string) {
	if lw == nil || lw.file == nil {
		return
	}
	lw.file.WriteString(line + "\n")
}

// WriteLines writes multiple lines to the log file.
func (lw *logWriter) WriteLines(lines []string) {
	if lw == nil || lw.file == nil {
		return
	}
	for _, line := range lines {
		lw.file.WriteString(line + "\n")
	}
}

// Close closes the log file. Safe to call on nil.
func (lw *logWriter) Close() {
	if lw == nil || lw.file == nil {
		return
	}
	lw.file.Close()
	lw.file = nil
}

// Path returns the file path, or empty if no file is open.
func (lw *logWriter) Path() string {
	if lw == nil || lw.file == nil {
		return ""
	}
	return lw.file.Name()
}

// sanitizePathComponent replaces characters unsafe for directory/file names.
func sanitizePathComponent(s string) string {
	r := strings.NewReplacer("/", "_", "\\", "_", " ", "-", ":", "_")
	return r.Replace(s)
}
