package tui

import (
	"strings"
	"testing"
)

func TestComposeOverlay_PlacesModalCentered(t *testing.T) {
	// 8x6 grid of dots; modal is a single 4-wide line.
	base := strings.Join([]string{
		"........",
		"........",
		"........",
		"........",
		"........",
		"........",
	}, "\n")
	modal := "XXXX"
	got := composeOverlay(base, modal, 8, 6)
	lines := strings.Split(got, "\n")
	if len(lines) != 6 {
		t.Fatalf("got %d lines, want 6", len(lines))
	}
	// Modal should land on row (6-1)/2 = 2, column (8-4)/2 = 2.
	wantRow := "  XXXX  "
	if lines[2] != wantRow {
		t.Errorf("row 2 = %q, want %q", lines[2], wantRow)
	}
	// Other rows should still be the padded base. Width matches.
	for i, l := range lines {
		if i == 2 {
			continue
		}
		if l != "........" {
			t.Errorf("row %d = %q, want %q", i, l, "........")
		}
	}
}

func TestComposeOverlay_FallsBackWithoutSize(t *testing.T) {
	m := &Model{}
	got := m.overlay("base", "modal")
	if got != "base\nmodal" {
		t.Errorf("overlay without size = %q, want %q", got, "base\nmodal")
	}
}

func TestSplitLines_PadsToHeight(t *testing.T) {
	got := splitLines("a\nb", 5)
	if len(got) != 5 {
		t.Fatalf("len=%d, want 5", len(got))
	}
	if got[0] != "a" || got[1] != "b" || got[2] != "" {
		t.Errorf("got %v, want [a b   ]", got)
	}
}

func TestSplitLines_TruncatesToHeight(t *testing.T) {
	got := splitLines("a\nb\nc\nd", 2)
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("got %v, want [a b]", got)
	}
}
