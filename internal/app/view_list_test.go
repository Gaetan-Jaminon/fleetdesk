package app

import (
	"strings"
	"testing"
)

func TestRenderList_EmptyRows_DefaultMessage(t *testing.T) {
	cfg := ListConfig{
		Columns:    []ListColumn{{Label: "NAME"}},
		RowCount:   0,
		InnerWidth: 40,
	}
	out := renderList(cfg)
	if !strings.Contains(out, "No items found") {
		t.Errorf("expected default empty message, got:\n%s", out)
	}
}

func TestRenderList_EmptyRows_CustomMessage(t *testing.T) {
	cfg := ListConfig{
		Columns:      []ListColumn{{Label: "NAME"}},
		RowCount:     0,
		EmptyMessage: "  No services found.",
		InnerWidth:   40,
	}
	out := renderList(cfg)
	if !strings.Contains(out, "No services found") {
		t.Errorf("expected custom empty message, got:\n%s", out)
	}
	if strings.Contains(out, "No items found") {
		t.Errorf("expected custom message to override default, got:\n%s", out)
	}
}

func TestRenderList_SingleRow(t *testing.T) {
	rows := [][]string{{"foo", "running"}}
	cfg := ListConfig{
		Columns: []ListColumn{
			{Label: "NAME"},
			{Label: "STATE"},
		},
		RowCount:   1,
		RowBuilder: func(i int) []string { return rows[i] },
		Cursor:     0,
		MaxVisible: 10,
		InnerWidth: 60,
	}
	out := renderList(cfg)
	if !strings.Contains(out, "NAME") || !strings.Contains(out, "STATE") {
		t.Errorf("expected header labels in output, got:\n%s", out)
	}
	if !strings.Contains(out, "foo") || !strings.Contains(out, "running") {
		t.Errorf("expected row values in output, got:\n%s", out)
	}
}

func TestRenderList_CursorMarker(t *testing.T) {
	rows := [][]string{{"a"}, {"b"}, {"c"}}
	cfg := ListConfig{
		Columns:    []ListColumn{{Label: "X"}},
		RowCount:   3,
		RowBuilder: func(i int) []string { return rows[i] },
		Cursor:     1,
		MaxVisible: 10,
		InnerWidth: 40,
	}
	out := renderList(cfg)
	// ▸ marker should be on the cursor row only
	cursorCount := strings.Count(out, "\u25b8")
	if cursorCount != 1 {
		t.Errorf("expected exactly 1 cursor marker, got %d in:\n%s", cursorCount, out)
	}
}

func TestRenderList_RowPrefix(t *testing.T) {
	rows := [][]string{{"ok-row"}, {"failed-row"}}
	cfg := ListConfig{
		Columns:    []ListColumn{{Label: "NAME"}},
		RowCount:   2,
		RowBuilder: func(i int) []string { return rows[i] },
		RowPrefix: func(i int) string {
			if i == 1 {
				return "\u2717 "
			}
			return ""
		},
		MaxVisible: 10,
		InnerWidth: 40,
	}
	out := renderList(cfg)
	if !strings.Contains(out, "\u2717 failed-row") {
		t.Errorf("expected prefix on failed-row, got:\n%s", out)
	}
	// ok-row should not have the prefix
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "ok-row") && strings.Contains(line, "\u2717") {
			t.Errorf("ok-row should not have ✗ prefix, got line: %s", line)
		}
	}
}

func TestRenderList_GroupHeader(t *testing.T) {
	rows := [][]string{{"pod-a"}, {"pod-b"}, {"deploy-x"}}
	cfg := ListConfig{
		Columns:    []ListColumn{{Label: "NAME"}},
		RowCount:   3,
		RowBuilder: func(i int) []string { return rows[i] },
		GroupHeader: func(i int) (string, bool) {
			if i == 0 {
				return "Pods", true
			}
			if i == 2 {
				return "Deployments", true
			}
			return "", false
		},
		MaxVisible: 10,
		InnerWidth: 40,
	}
	out := renderList(cfg)
	if !strings.Contains(out, "── Pods ──") {
		t.Errorf("expected Pods group header, got:\n%s", out)
	}
	if !strings.Contains(out, "── Deployments ──") {
		t.Errorf("expected Deployments group header, got:\n%s", out)
	}
}

func TestRenderList_SortIndicator(t *testing.T) {
	rows := [][]string{{"foo"}}
	cfg := ListConfig{
		Columns:    []ListColumn{{Label: "NAME", SortIndex: 1}},
		RowCount:   1,
		RowBuilder: func(i int) []string { return rows[i] },
		SortIndicator: func(key int) string {
			if key == 1 {
				return "\u25b2"
			}
			return ""
		},
		MaxVisible: 10,
		InnerWidth: 40,
	}
	out := renderList(cfg)
	if !strings.Contains(out, "NAME\u25b2") {
		t.Errorf("expected NAME▲ in header, got:\n%s", out)
	}
}

func TestRenderList_RightAlign(t *testing.T) {
	rows := [][]string{{"foo", "5"}, {"barbaz", "100"}}
	cfg := ListConfig{
		Columns: []ListColumn{
			{Label: "NAME"},
			{Label: "COUNT", Width: 8, RightAlign: true},
		},
		RowCount:   2,
		RowBuilder: func(i int) []string { return rows[i] },
		MaxVisible: 10,
		InnerWidth: 60,
	}
	out := renderList(cfg)
	// right-aligned "5" in a width-8 column should have leading spaces before it
	if !strings.Contains(out, "       5") {
		t.Errorf("expected right-aligned 5 with leading spaces, got:\n%s", out)
	}
}

func TestRenderList_Viewport_ScrollsWithCursor(t *testing.T) {
	rows := make([][]string, 20)
	for i := range rows {
		rows[i] = []string{"item-" + string(rune('a'+i))}
	}
	cfg := ListConfig{
		Columns:    []ListColumn{{Label: "NAME"}},
		RowCount:   20,
		RowBuilder: func(i int) []string { return rows[i] },
		Cursor:     15,
		MaxVisible: 5,
		InnerWidth: 40,
	}
	out := renderList(cfg)
	// With cursor=15 and MaxVisible=5, offset = 15 - 5 + 1 = 11, rendered rows 11..15
	if strings.Contains(out, "item-a") {
		t.Errorf("rows before offset should not be rendered, got:\n%s", out)
	}
	if !strings.Contains(out, "item-p") { // index 15 = 'p'
		t.Errorf("cursor row (item-p) should be rendered, got:\n%s", out)
	}
	if strings.Contains(out, "item-q") { // index 16 - out of viewport
		t.Errorf("rows after viewport should not be rendered, got:\n%s", out)
	}
}

func TestRenderList_ComputedWidth_IncludesTrailingPadding(t *testing.T) {
	rows := [][]string{{"foo", "bar"}}
	cfg := ListConfig{
		Columns: []ListColumn{
			{Label: "NAME"},
			{Label: "VALUE"},
		},
		RowCount:   1,
		RowBuilder: func(i int) []string { return rows[i] },
		MaxVisible: 10,
		InnerWidth: 80,
	}
	out := renderList(cfg)
	// Computed width = max(label, data) + 2. For NAME col: max(4, 3) + 2 = 6.
	// So "NAME  " (4 chars + 2 padding) followed by 2-space gap = "NAME    " before "VALUE"
	if !strings.Contains(out, "NAME    VALUE") {
		t.Errorf("expected NAME padded to width 6 then 2-space gap before VALUE, got:\n%s", out)
	}
}

func TestRenderList_RowOverride_ReplacesCells(t *testing.T) {
	cfg := ListConfig{
		Columns: []ListColumn{
			{Label: "HOST"},
			{Label: "OS"},
			{Label: "UPTIME"},
		},
		RowCount: 2,
		RowBuilder: func(i int) []string {
			return []string{"host" + string(rune('0'+i)), "RHEL 9", "5d"}
		},
		RowOverride: func(i int) string {
			if i == 1 {
				return "host1  connecting..."
			}
			return ""
		},
		MaxVisible: 10,
		InnerWidth: 80,
	}
	out := renderList(cfg)
	if !strings.Contains(out, "host1  connecting...") {
		t.Errorf("expected override line for row 1, got:\n%s", out)
	}
	if !strings.Contains(out, "RHEL 9") {
		t.Errorf("expected normal row for row 0 (RHEL 9), got:\n%s", out)
	}
}

func TestRenderList_FixedWidth_UsedDirectly(t *testing.T) {
	rows := [][]string{{"x", "ready"}}
	cfg := ListConfig{
		Columns: []ListColumn{
			{Label: "NAME", Width: 10},
			{Label: "STATE"},
		},
		RowCount:   1,
		RowBuilder: func(i int) []string { return rows[i] },
		MaxVisible: 10,
		InnerWidth: 60,
	}
	out := renderList(cfg)
	// Fixed width 10: "NAME      " (NAME + 6 pad) then 2-space gap then "STATE"
	if !strings.Contains(out, "NAME        STATE") {
		t.Errorf("expected NAME padded to fixed width 10 then gap, got:\n%s", out)
	}
}
