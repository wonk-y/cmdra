package attachterm

import (
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
)

func TestPlainTextAndCarriageReturn(t *testing.T) {
	m := New(12, 4)
	m.Write([]byte("hello\rworld"))

	got := stripANSI(m.View())
	if !strings.Contains(got, "world") {
		t.Fatalf("expected overwritten line to contain world, got %q", got)
	}
	if strings.Contains(got, "hello") {
		t.Fatalf("expected carriage return overwrite to replace hello, got %q", got)
	}
}

func TestAltScreenPreservesPrimaryBuffer(t *testing.T) {
	m := New(20, 4)
	m.Write([]byte("main"))
	m.Write([]byte("\x1b[?1049hALT"))

	if !strings.Contains(stripANSI(m.View()), "ALT") {
		t.Fatalf("expected alternate screen content while alt screen active, got %q", stripANSI(m.View()))
	}

	m.Write([]byte("\x1b[?1049l"))
	got := stripANSI(m.View())
	if !strings.Contains(got, "main") {
		t.Fatalf("expected primary screen content after leaving alt screen, got %q", got)
	}
}

func TestCursorMotionAndErase(t *testing.T) {
	m := New(10, 4)
	m.Write([]byte("abcdef"))
	m.Write([]byte("\x1b[1;1H"))
	m.Write([]byte("Z"))
	m.Write([]byte("\x1b[2K"))

	for x := 0; x < 10; x++ {
		cell := m.active().CellAt(x, 0)
		if cell == nil || strings.TrimSpace(cell.Content) == "" {
			continue
		}
		t.Fatalf("expected erase line to clear first row, found cell %q at column %d", cell.Content, x)
	}
}

func TestSGRAppliesCellStyle(t *testing.T) {
	m := New(10, 2)
	m.Write([]byte("\x1b[31;1mR"))

	cell := m.active().CellAt(0, 0)
	if cell == nil {
		t.Fatal("expected rendered cell")
	}
	if cell.Style.Attrs&uv.AttrBold == 0 {
		t.Fatalf("expected bold attribute on rendered cell, got %+v", cell.Style)
	}
	if cell.Style.Fg == nil {
		t.Fatalf("expected foreground color on rendered cell, got %+v", cell.Style)
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case !inEsc && c == 0x1b:
			inEsc = true
		case inEsc && c >= 0x40 && c <= 0x7e:
			inEsc = false
		case !inEsc:
			b.WriteByte(c)
		}
	}
	return b.String()
}
