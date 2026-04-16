package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMultiSelectContent_InitialState(t *testing.T) {
	opts := []MultiSelectOption{
		{Key: "--allowerasing", Label: "--allowerasing", Description: "replace conflicts"},
		{Key: "--skip-broken", Label: "--skip-broken", Description: "skip failures"},
	}
	ms := NewMultiSelectContent(opts)
	result := ms.Result().([]string)
	if len(result) != 0 {
		t.Errorf("initial Result() should be empty, got %v", result)
	}
}

func TestMultiSelectContent_SpaceToggles(t *testing.T) {
	opts := []MultiSelectOption{
		{Key: "--allowerasing", Label: "--allowerasing", Description: "replace conflicts"},
		{Key: "--skip-broken", Label: "--skip-broken", Description: "skip failures"},
	}
	ms := NewMultiSelectContent(opts)
	// Space toggles first item
	ms, _, _ = ms.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	result := ms.Result().([]string)
	if len(result) != 1 || result[0] != "--allowerasing" {
		t.Errorf("after Space, Result() = %v, want [--allowerasing]", result)
	}
}

func TestMultiSelectContent_SpaceUntoggles(t *testing.T) {
	opts := []MultiSelectOption{
		{Key: "--allowerasing", Label: "--allowerasing", Description: "replace conflicts"},
	}
	ms := NewMultiSelectContent(opts)
	// Toggle on then off
	ms, _, _ = ms.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	ms, _, _ = ms.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	result := ms.Result().([]string)
	if len(result) != 0 {
		t.Errorf("after double Space, Result() should be empty, got %v", result)
	}
}

func TestMultiSelectContent_CursorDown(t *testing.T) {
	opts := []MultiSelectOption{
		{Key: "a", Label: "a", Description: ""},
		{Key: "b", Label: "b", Description: ""},
		{Key: "c", Label: "c", Description: ""},
	}
	ms := NewMultiSelectContent(opts)
	ms, _, _ = ms.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	if ms.(*MultiSelectContent).cursor != 1 {
		t.Errorf("cursor = %d, want 1", ms.(*MultiSelectContent).cursor)
	}
}

func TestMultiSelectContent_CursorDownAtEnd(t *testing.T) {
	opts := []MultiSelectOption{
		{Key: "a", Label: "a", Description: ""},
		{Key: "b", Label: "b", Description: ""},
	}
	ms := NewMultiSelectContent(opts)
	ms, _, _ = ms.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	ms, _, _ = ms.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	ms, _, _ = ms.HandleKey(tea.KeyMsg{Type: tea.KeyDown}) // past end
	if ms.(*MultiSelectContent).cursor != 1 {
		t.Errorf("cursor = %d, want 1 (clamped)", ms.(*MultiSelectContent).cursor)
	}
}

func TestMultiSelectContent_CursorUpWithK(t *testing.T) {
	opts := []MultiSelectOption{
		{Key: "a", Label: "a", Description: ""},
		{Key: "b", Label: "b", Description: ""},
	}
	ms := NewMultiSelectContent(opts)
	ms, _, _ = ms.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	ms, _, _ = ms.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if ms.(*MultiSelectContent).cursor != 0 {
		t.Errorf("cursor = %d, want 0", ms.(*MultiSelectContent).cursor)
	}
}

func TestMultiSelectContent_CursorUpAtZero(t *testing.T) {
	opts := []MultiSelectOption{
		{Key: "a", Label: "a", Description: ""},
	}
	ms := NewMultiSelectContent(opts)
	ms, _, _ = ms.HandleKey(tea.KeyMsg{Type: tea.KeyUp})
	if ms.(*MultiSelectContent).cursor != 0 {
		t.Errorf("cursor = %d, want 0", ms.(*MultiSelectContent).cursor)
	}
}

func TestMultiSelectContent_EnterDone(t *testing.T) {
	opts := []MultiSelectOption{
		{Key: "a", Label: "a", Description: ""},
	}
	ms := NewMultiSelectContent(opts)
	_, _, done := ms.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !done {
		t.Error("Enter should return done=true")
	}
}

func TestMultiSelectContent_EnterWithNoneSelected(t *testing.T) {
	opts := []MultiSelectOption{
		{Key: "a", Label: "a", Description: ""},
		{Key: "b", Label: "b", Description: ""},
	}
	ms := NewMultiSelectContent(opts)
	ms, _, done := ms.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !done {
		t.Error("Enter with 0 selected should still return done=true")
	}
	result := ms.Result().([]string)
	if len(result) != 0 {
		t.Errorf("Result() should be empty, got %v", result)
	}
}

func TestMultiSelectContent_ResultOrder(t *testing.T) {
	opts := []MultiSelectOption{
		{Key: "first", Label: "first", Description: ""},
		{Key: "second", Label: "second", Description: ""},
		{Key: "third", Label: "third", Description: ""},
	}
	ms := NewMultiSelectContent(opts)
	// Select third, then first (out of order)
	ms, _, _ = ms.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	ms, _, _ = ms.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	ms, _, _ = ms.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}) // third
	ms, _, _ = ms.HandleKey(tea.KeyMsg{Type: tea.KeyUp})
	ms, _, _ = ms.HandleKey(tea.KeyMsg{Type: tea.KeyUp})
	ms, _, _ = ms.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}) // first

	result := ms.Result().([]string)
	if len(result) != 2 {
		t.Fatalf("len(Result()) = %d, want 2", len(result))
	}
	if result[0] != "first" {
		t.Errorf("result[0] = %q, want %q", result[0], "first")
	}
	if result[1] != "third" {
		t.Errorf("result[1] = %q, want %q", result[1], "third")
	}
}

func TestMultiSelectContent_MultipleSelected(t *testing.T) {
	opts := []MultiSelectOption{
		{Key: "a", Label: "a", Description: ""},
		{Key: "b", Label: "b", Description: ""},
		{Key: "c", Label: "c", Description: ""},
	}
	ms := NewMultiSelectContent(opts)
	// Select all three
	ms, _, _ = ms.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	ms, _, _ = ms.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	ms, _, _ = ms.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	ms, _, _ = ms.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	ms, _, _ = ms.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})

	result := ms.Result().([]string)
	if len(result) != 3 {
		t.Errorf("len(Result()) = %d, want 3", len(result))
	}
}

func TestMultiSelectContent_EscNotDone(t *testing.T) {
	opts := []MultiSelectOption{
		{Key: "a", Label: "a", Description: ""},
	}
	ms := NewMultiSelectContent(opts)
	_, cmd, done := ms.HandleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if done {
		t.Error("Esc should return done=false")
	}
	if cmd != nil {
		t.Error("Esc should return cmd=nil")
	}
}

func TestMultiSelectContent_ViewContainsLabels(t *testing.T) {
	opts := []MultiSelectOption{
		{Key: "a", Label: "Alpha", Description: "first letter"},
		{Key: "b", Label: "Beta", Description: "second letter"},
	}
	ms := NewMultiSelectContent(opts)
	view := ms.View(80)
	if !strings.Contains(view, "Alpha") {
		t.Error("View should contain label Alpha")
	}
	if !strings.Contains(view, "Beta") {
		t.Error("View should contain label Beta")
	}
	if !strings.Contains(view, "first letter") {
		t.Error("View should contain description")
	}
}

func TestMultiSelectContent_ViewCheckedMarker(t *testing.T) {
	opts := []MultiSelectOption{
		{Key: "a", Label: "Alpha", Description: ""},
	}
	ms := NewMultiSelectContent(opts)
	ms, _, _ = ms.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	view := ms.View(80)
	if !strings.Contains(view, "[x]") {
		t.Error("checked item should show [x]")
	}
}

func TestMultiSelectContent_ViewUncheckedMarker(t *testing.T) {
	opts := []MultiSelectOption{
		{Key: "a", Label: "Alpha", Description: ""},
	}
	ms := NewMultiSelectContent(opts)
	view := ms.View(80)
	if !strings.Contains(view, "[ ]") {
		t.Error("unchecked item should show [ ]")
	}
}

func TestMultiSelectContent_JKNavigation(t *testing.T) {
	opts := []MultiSelectOption{
		{Key: "a", Label: "a", Description: ""},
		{Key: "b", Label: "b", Description: ""},
		{Key: "c", Label: "c", Description: ""},
	}
	ms := NewMultiSelectContent(opts)
	ms, _, _ = ms.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if ms.(*MultiSelectContent).cursor != 1 {
		t.Errorf("j: cursor = %d, want 1", ms.(*MultiSelectContent).cursor)
	}
	ms, _, _ = ms.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if ms.(*MultiSelectContent).cursor != 2 {
		t.Errorf("j: cursor = %d, want 2", ms.(*MultiSelectContent).cursor)
	}
	ms, _, _ = ms.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if ms.(*MultiSelectContent).cursor != 1 {
		t.Errorf("k: cursor = %d, want 1", ms.(*MultiSelectContent).cursor)
	}
}
