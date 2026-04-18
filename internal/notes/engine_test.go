package notes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResourceRef_Dir_AllFleetTypes(t *testing.T) {
	base := "/tmp/fleetdesk/notes"
	tests := []struct {
		name string
		ref  ResourceRef
		want string
	}{
		{
			"fleet-level",
			ResourceRef{Fleet: "aap-prod"},
			"/tmp/fleetdesk/notes/aap-prod",
		},
		{
			"vm-host",
			ResourceRef{Fleet: "aap-prod", Segments: []string{"hosts", "aap-ctrl-01"}},
			"/tmp/fleetdesk/notes/aap-prod/hosts/aap-ctrl-01",
		},
		{
			"vm-service",
			ResourceRef{Fleet: "aap-prod", Segments: []string{"hosts", "aap-ctrl-01", "services", "automation-controller-web"}},
			"/tmp/fleetdesk/notes/aap-prod/hosts/aap-ctrl-01/services/automation-controller-web",
		},
		{
			"azure-sub",
			ResourceRef{Fleet: "azure-dev", Segments: []string{"azure", "APP-DEV"}},
			"/tmp/fleetdesk/notes/azure-dev/azure/APP-DEV",
		},
		{
			"azure-vm",
			ResourceRef{Fleet: "azure-dev", Segments: []string{"azure", "APP-DEV", "rg-app", "vm", "vm-ctrl-01"}},
			"/tmp/fleetdesk/notes/azure-dev/azure/APP-DEV/rg-app/vm/vm-ctrl-01",
		},
		{
			"k8s-namespace",
			ResourceRef{Fleet: "aks-dev", Segments: []string{"k8s", "AKS-APP-DEV-BLUE", "ctx", "default"}},
			"/tmp/fleetdesk/notes/aks-dev/k8s/AKS-APP-DEV-BLUE/ctx/default",
		},
		{
			"sanitizes-unsafe-chars",
			ResourceRef{Fleet: "dev / fleet", Segments: []string{"foo:bar", "a b"}},
			"/tmp/fleetdesk/notes/dev-_-fleet/foo_bar/a-b",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.ref.Dir(base)
			if got != tc.want {
				t.Errorf("Dir = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResourceRef_Key_StableAcrossEqualRefs(t *testing.T) {
	a := ResourceRef{Fleet: "f", Segments: []string{"hosts", "h1"}}
	b := ResourceRef{Fleet: "f", Segments: []string{"hosts", "h1"}}
	if a.Key() != b.Key() {
		t.Errorf("equal refs should have equal keys: %q vs %q", a.Key(), b.Key())
	}

	c := ResourceRef{Fleet: "f", Segments: []string{"hosts", "h2"}}
	if a.Key() == c.Key() {
		t.Errorf("different refs should have different keys: %q == %q", a.Key(), c.Key())
	}
}

func TestEngine_Create_WritesEmptyFile(t *testing.T) {
	dir := t.TempDir()
	e := New(dir)
	ref := ResourceRef{Fleet: "f", Segments: []string{"hosts", "h1"}}

	path, err := e.Create(ref)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("note file should exist: %v", err)
	}
	if info.Size() != 0 {
		t.Errorf("note file should be empty, got %d bytes", info.Size())
	}

	// Parent dir must be created under <base>/notes/f/hosts/h1
	expectedDir := filepath.Join(dir, "notes", "f", "hosts", "h1")
	if filepath.Dir(path) != expectedDir {
		t.Errorf("note path parent = %q, want %q", filepath.Dir(path), expectedDir)
	}
}

func TestEngine_Create_FilenameFormat(t *testing.T) {
	dir := t.TempDir()
	e := New(dir)
	ref := ResourceRef{Fleet: "f", Segments: []string{"h"}}

	path, err := e.Create(ref)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	name := filepath.Base(path)
	if !strings.HasSuffix(name, "_note.txt") {
		t.Errorf("filename must end with _note.txt, got %q", name)
	}
	// Format: 2006-01-02T15-04-05.NNN_note.txt
	stem := strings.TrimSuffix(name, "_note.txt")
	if len(stem) != len("2006-01-02T15-04-05.000") {
		t.Errorf("filename stem length = %d, want %d, stem=%q", len(stem), len("2006-01-02T15-04-05.000"), stem)
	}
	if _, ok := parseTimestamp(name); !ok {
		t.Errorf("parseTimestamp could not re-parse filename %q", name)
	}
}

func TestEngine_Create_MillisecondResolution_NoCollision(t *testing.T) {
	dir := t.TempDir()
	e := New(dir)
	ref := ResourceRef{Fleet: "f", Segments: []string{"h"}}

	// Create 20 notes as fast as possible; expect unique filenames at ms
	// resolution. On any reasonable machine, two consecutive os.Create
	// calls will differ by well under 1ms — if they collide, we get
	// "file exists" behavior (os.Create truncates). So instead: parse
	// the timestamps and ensure uniqueness.
	paths := make(map[string]bool)
	for range 20 {
		path, err := e.Create(ref)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if paths[path] {
			t.Errorf("duplicate path generated: %s", path)
		}
		paths[path] = true
		// Tiny sleep to guarantee ms tick between iterations.
		time.Sleep(2 * time.Millisecond)
	}
}

func TestEngine_List_SortedNewestFirst(t *testing.T) {
	dir := t.TempDir()
	e := New(dir)
	ref := ResourceRef{Fleet: "f", Segments: []string{"h"}}

	p1, _ := e.Create(ref)
	time.Sleep(5 * time.Millisecond)
	p2, _ := e.Create(ref)
	time.Sleep(5 * time.Millisecond)
	p3, _ := e.Create(ref)

	notes, err := e.List(ref)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(notes) != 3 {
		t.Fatalf("expected 3 notes, got %d", len(notes))
	}
	if notes[0].Path != p3 || notes[1].Path != p2 || notes[2].Path != p1 {
		t.Errorf("notes not sorted newest-first: got %v %v %v (want %v %v %v)",
			notes[0].Path, notes[1].Path, notes[2].Path, p3, p2, p1)
	}
}

func TestEngine_List_MissingDir_ReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	e := New(dir)
	ref := ResourceRef{Fleet: "nonexistent", Segments: []string{"also", "missing"}}

	notes, err := e.List(ref)
	if err != nil {
		t.Errorf("List on missing dir should not error, got %v", err)
	}
	if len(notes) != 0 {
		t.Errorf("List on missing dir should return empty, got %d notes", len(notes))
	}
}

func TestEngine_Count_MissingDir_ReturnsZero(t *testing.T) {
	dir := t.TempDir()
	e := New(dir)
	ref := ResourceRef{Fleet: "nope"}
	if n := e.Count(ref); n != 0 {
		t.Errorf("Count on missing dir = %d, want 0", n)
	}
}

func TestEngine_Count_MatchesListLength(t *testing.T) {
	dir := t.TempDir()
	e := New(dir)
	ref := ResourceRef{Fleet: "f", Segments: []string{"h"}}

	for range 4 {
		if _, err := e.Create(ref); err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * time.Millisecond)
	}

	if n := e.Count(ref); n != 4 {
		t.Errorf("Count = %d, want 4", n)
	}
}

func TestEngine_Delete_RemovesFile(t *testing.T) {
	dir := t.TempDir()
	e := New(dir)
	ref := ResourceRef{Fleet: "f", Segments: []string{"h"}}

	path, _ := e.Create(ref)

	if err := e.Delete(path); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("file should be gone, stat err = %v", err)
	}
}

func TestEngine_Delete_MissingFile_IsNoOp(t *testing.T) {
	dir := t.TempDir()
	e := New(dir)
	if err := e.Delete(filepath.Join(dir, "does-not-exist.txt")); err != nil {
		t.Errorf("Delete on missing file should not error, got %v", err)
	}
}

func TestEngine_List_ReadsPreview(t *testing.T) {
	dir := t.TempDir()
	e := New(dir)
	ref := ResourceRef{Fleet: "f", Segments: []string{"h"}}

	path, _ := e.Create(ref)
	if err := os.WriteFile(path, []byte("\n\n  First real line of content  \n\nsecond line\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	notes, err := e.List(ref)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("expected 1 note, got %d", len(notes))
	}
	if notes[0].Preview != "First real line of content" {
		t.Errorf("preview = %q, want 'First real line of content'", notes[0].Preview)
	}
}

func TestEngine_List_PreviewTruncates(t *testing.T) {
	dir := t.TempDir()
	e := New(dir)
	ref := ResourceRef{Fleet: "f", Segments: []string{"h"}}

	path, _ := e.Create(ref)
	longLine := strings.Repeat("x", previewMaxLen+20)
	if err := os.WriteFile(path, []byte(longLine), 0o644); err != nil {
		t.Fatal(err)
	}

	notes, _ := e.List(ref)
	if len(notes) != 1 {
		t.Fatalf("expected 1 note, got %d", len(notes))
	}
	if !strings.HasSuffix(notes[0].Preview, "\u2026") {
		t.Errorf("preview should end with ellipsis for long lines, got %q", notes[0].Preview)
	}
	if len(notes[0].Preview) > previewMaxLen+5 {
		t.Errorf("preview should be truncated to ~%d chars, got %d", previewMaxLen, len(notes[0].Preview))
	}
}

func TestEngine_List_EmptyFile_EmptyPreview(t *testing.T) {
	dir := t.TempDir()
	e := New(dir)
	ref := ResourceRef{Fleet: "f", Segments: []string{"h"}}

	_, _ = e.Create(ref)

	notes, _ := e.List(ref)
	if len(notes) != 1 {
		t.Fatalf("expected 1 note, got %d", len(notes))
	}
	if notes[0].Preview != "" {
		t.Errorf("empty file should have empty preview, got %q", notes[0].Preview)
	}
}

func TestEngine_List_IgnoresNonNoteFiles(t *testing.T) {
	dir := t.TempDir()
	e := New(dir)
	ref := ResourceRef{Fleet: "f", Segments: []string{"h"}}

	_, _ = e.Create(ref)

	// Drop a stray file in the same directory.
	noteDir := ref.Dir(e.base)
	if err := os.WriteFile(filepath.Join(noteDir, "something.txt"), []byte("not a note"), 0o644); err != nil {
		t.Fatal(err)
	}

	notes, _ := e.List(ref)
	if len(notes) != 1 {
		t.Errorf("expected 1 note (the stray file should be ignored), got %d", len(notes))
	}
}

func TestParseTimestamp_RoundTrip(t *testing.T) {
	now := time.Date(2026, 4, 18, 9, 15, 30, 123_000_000, time.UTC)
	name := filename(now)
	parsed, ok := parseTimestamp(name)
	if !ok {
		t.Fatalf("parseTimestamp(%q) = false", name)
	}
	if !parsed.Equal(now) {
		t.Errorf("round-trip mismatch: got %v, want %v", parsed, now)
	}
}

func TestParseTimestamp_InvalidInputs(t *testing.T) {
	cases := []string{
		"not-a-note.log",
		"garbage_note.txt",
		"abc.123_note.txt",
		"2026-04-18T09-15-00_note.txt", // missing milliseconds
	}
	for _, c := range cases {
		if _, ok := parseTimestamp(c); ok {
			t.Errorf("parseTimestamp(%q) should be false", c)
		}
	}
}
