package attachterm

import (
	"image/color"
	"strings"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	chansi "github.com/charmbracelet/x/ansi"
)

const tabWidth = 8

// Model is a limited ANSI terminal emulator for PTY-backed attach mode.
// It is intentionally scoped to the common shell/session control sequences
// needed for prompt redraws and basic full-screen updates.
type Model struct {
	width  int
	height int

	primary uv.ScreenBuffer
	alt     uv.ScreenBuffer
	useAlt  bool

	parser *chansi.Parser

	cursorX int
	cursorY int

	savedX int
	savedY int

	scrollTop    int
	scrollBottom int

	style uv.Style

	showCursor bool

	stats Stats
}

// Stats provides lightweight emulator timing counters for PTY attach debugging.
type Stats struct {
	Writes         uint64
	Bytes          uint64
	Frames         uint64
	LastWrite      time.Duration
	LastRender     time.Duration
	LastWriteBytes int
}

// New returns a terminal model sized for the given view area.
func New(width, height int) *Model {
	width = max(1, width)
	height = max(1, height)
	m := &Model{
		width:        width,
		height:       height,
		primary:      uv.NewScreenBuffer(width, height),
		alt:          uv.NewScreenBuffer(width, height),
		scrollBottom: height - 1,
		showCursor:   true,
	}
	m.parser = chansi.NewParser()
	m.parser.SetHandler(chansi.Handler{
		Print:     m.printRune,
		Execute:   m.execute,
		HandleCsi: m.handleCsi,
		HandleEsc: m.handleEsc,
		HandleOsc: m.handleOsc,
	})
	return m
}

// Resize resizes the primary and alternate buffers and clamps terminal state.
func (m *Model) Resize(width, height int) {
	width = max(1, width)
	height = max(1, height)
	if width == m.width && height == m.height {
		return
	}
	m.width = width
	m.height = height
	m.primary.Resize(width, height)
	m.alt.Resize(width, height)
	m.cursorX = clamp(m.cursorX, 0, width-1)
	m.cursorY = clamp(m.cursorY, 0, height-1)
	m.savedX = clamp(m.savedX, 0, width-1)
	m.savedY = clamp(m.savedY, 0, height-1)
	if m.scrollTop >= height {
		m.scrollTop = 0
	}
	if m.scrollBottom >= height || m.scrollBottom < m.scrollTop {
		m.scrollBottom = height - 1
	}
}

// Write feeds remote PTY bytes into the ANSI parser.
func (m *Model) Write(data []byte) {
	start := time.Now()
	for _, b := range data {
		m.parser.Advance(b)
	}
	m.stats.Writes++
	m.stats.Bytes += uint64(len(data))
	m.stats.LastWrite = time.Since(start)
	m.stats.LastWriteBytes = len(data)
}

// View renders the currently active terminal screen.
func (m *Model) View() string {
	start := time.Now()
	scr := m.active()
	buf := scr.Buffer.Clone()
	if m.showCursor {
		x := clamp(m.cursorX, 0, m.width-1)
		y := clamp(m.cursorY, 0, m.height-1)
		cell := buf.CellAt(x, y)
		if cell == nil || cell.IsZero() {
			cell = uv.EmptyCell.Clone()
		} else {
			cell = cell.Clone()
		}
		cell.Style.Attrs |= uv.AttrReverse
		buf.SetCell(x, y, cell)
	}

	lines := make([]string, 0, m.height)
	for y := 0; y < m.height; y++ {
		line := buf.Line(y)
		if line == nil {
			lines = append(lines, "")
			continue
		}
		lines = append(lines, line.Render())
	}
	rendered := strings.Join(lines, "\n")
	m.stats.Frames++
	m.stats.LastRender = time.Since(start)
	return rendered
}

// Stats returns a snapshot of the current emulator counters.
func (m *Model) Stats() Stats {
	return m.stats
}

func (m *Model) active() *uv.ScreenBuffer {
	if m.useAlt {
		return &m.alt
	}
	return &m.primary
}

func (m *Model) reset() {
	m.primary = uv.NewScreenBuffer(m.width, m.height)
	m.alt = uv.NewScreenBuffer(m.width, m.height)
	m.useAlt = false
	m.cursorX = 0
	m.cursorY = 0
	m.savedX = 0
	m.savedY = 0
	m.scrollTop = 0
	m.scrollBottom = m.height - 1
	m.style = uv.Style{}
	m.showCursor = true
}

func (m *Model) printRune(r rune) {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	cell := uv.NewCell(m.active().WidthMethod(), string(r))
	if cell == nil || cell.Width <= 0 {
		return
	}
	cell.Style = m.style

	if m.cursorX >= m.width {
		m.newLine()
	}
	if cell.Width > m.width {
		return
	}
	if m.cursorX+cell.Width > m.width {
		m.newLine()
	}
	m.active().SetCell(m.cursorX, m.cursorY, cell)
	m.cursorX += max(1, cell.Width)
	if m.cursorX >= m.width {
		m.cursorX = m.width
	}
}

func (m *Model) execute(b byte) {
	switch b {
	case '\a':
		return
	case '\b':
		if m.cursorX > 0 {
			m.cursorX--
		}
	case '\t':
		next := ((m.cursorX / tabWidth) + 1) * tabWidth
		for m.cursorX < min(next, m.width) {
			m.printRune(' ')
		}
	case '\n', '\v', '\f':
		m.newLine()
	case '\r':
		m.cursorX = 0
	}
}

func (m *Model) handleEsc(cmd chansi.Cmd) {
	switch cmd.Final() {
	case '7':
		m.saveCursor()
	case '8':
		m.restoreCursor()
	case 'D':
		m.index()
	case 'E':
		m.cursorX = 0
		m.index()
	case 'M':
		m.reverseIndex()
	case 'c':
		m.reset()
	}
}

func (m *Model) handleOsc(_ int, _ []byte) {
	// Title/OSC handling is not needed for the phase-1 attach spike.
}

func (m *Model) handleCsi(cmd chansi.Cmd, params chansi.Params) {
	switch cmd.Final() {
	case 'A':
		m.cursorY = clamp(m.cursorY-paramOrDefault(params, 0, 1), 0, m.height-1)
	case 'B':
		m.cursorY = clamp(m.cursorY+paramOrDefault(params, 0, 1), 0, m.height-1)
	case 'C':
		m.cursorX = clamp(m.cursorX+paramOrDefault(params, 0, 1), 0, m.width-1)
	case 'D':
		m.cursorX = clamp(m.cursorX-paramOrDefault(params, 0, 1), 0, m.width-1)
	case 'E':
		m.cursorY = clamp(m.cursorY+paramOrDefault(params, 0, 1), 0, m.height-1)
		m.cursorX = 0
	case 'F':
		m.cursorY = clamp(m.cursorY-paramOrDefault(params, 0, 1), 0, m.height-1)
		m.cursorX = 0
	case 'G':
		m.cursorX = clamp(paramOrDefault(params, 0, 1)-1, 0, m.width-1)
	case 'H', 'f':
		row := paramOrDefault(params, 0, 1)
		col := paramOrDefault(params, 1, 1)
		m.moveCursor(col-1, row-1)
	case 'J':
		m.eraseDisplay(paramOrDefault(params, 0, 0))
	case 'K':
		m.eraseLine(paramOrDefault(params, 0, 0))
	case 'L':
		m.active().InsertLineArea(m.cursorY, paramOrDefault(params, 0, 1), nil, m.scrollRegion())
	case 'M':
		m.active().DeleteLineArea(m.cursorY, paramOrDefault(params, 0, 1), nil, m.scrollRegion())
	case '@':
		m.active().InsertCellArea(m.cursorX, m.cursorY, paramOrDefault(params, 0, 1), nil, m.lineRegion())
	case 'P':
		m.active().DeleteCellArea(m.cursorX, m.cursorY, paramOrDefault(params, 0, 1), nil, m.lineRegion())
	case 'X':
		m.eraseCharacters(paramOrDefault(params, 0, 1))
	case 'S':
		m.scrollUp(paramOrDefault(params, 0, 1))
	case 'T':
		m.scrollDown(paramOrDefault(params, 0, 1))
	case 'm':
		m.handleSGR(params)
	case 's':
		m.saveCursor()
	case 'u':
		m.restoreCursor()
	case 'r':
		m.setScrollRegion(params)
	case 'h', 'l':
		m.handleModeSet(cmd, params)
	}
}

func (m *Model) handleModeSet(cmd chansi.Cmd, params chansi.Params) {
	set := cmd.Final() == 'h'
	if cmd.Prefix() != '?' {
		return
	}
	for _, raw := range paramsToInts(params) {
		switch raw {
		case 25:
			m.showCursor = set
		case 47, 1047, 1049:
			if set {
				m.enterAltScreen()
			} else {
				m.leaveAltScreen()
			}
		}
	}
}

func (m *Model) handleSGR(params chansi.Params) {
	values := paramsToInts(params)
	if len(values) == 0 {
		values = []int{0}
	}
	for i := 0; i < len(values); i++ {
		switch values[i] {
		case 0:
			m.style = uv.Style{}
		case 1:
			m.style.Attrs |= uv.AttrBold
		case 2:
			m.style.Attrs |= uv.AttrFaint
		case 3:
			m.style.Attrs |= uv.AttrItalic
		case 4:
			m.style.Underline = uv.UnderlineSingle
		case 22:
			m.style.Attrs &^= uv.AttrBold | uv.AttrFaint
		case 23:
			m.style.Attrs &^= uv.AttrItalic
		case 24:
			m.style.Underline = uv.UnderlineNone
		case 7:
			m.style.Attrs |= uv.AttrReverse
		case 27:
			m.style.Attrs &^= uv.AttrReverse
		case 9:
			m.style.Attrs |= uv.AttrStrikethrough
		case 29:
			m.style.Attrs &^= uv.AttrStrikethrough
		case 30, 31, 32, 33, 34, 35, 36, 37:
			m.style.Fg = chansi.BasicColor(values[i] - 30)
		case 39:
			m.style.Fg = color.Color(nil)
		case 40, 41, 42, 43, 44, 45, 46, 47:
			m.style.Bg = chansi.BasicColor(values[i] - 40)
		case 49:
			m.style.Bg = color.Color(nil)
		case 90, 91, 92, 93, 94, 95, 96, 97:
			m.style.Fg = chansi.BasicColor(values[i] - 90 + 8)
		case 100, 101, 102, 103, 104, 105, 106, 107:
			m.style.Bg = chansi.BasicColor(values[i] - 100 + 8)
		case 38, 48:
			if i+1 >= len(values) {
				continue
			}
			targetFG := values[i] == 38
			mode := values[i+1]
			switch mode {
			case 5:
				if i+2 >= len(values) {
					i = len(values)
					break
				}
				if targetFG {
					m.style.Fg = chansi.IndexedColor(values[i+2])
				} else {
					m.style.Bg = chansi.IndexedColor(values[i+2])
				}
				i += 2
			case 2:
				if i+4 >= len(values) {
					i = len(values)
					break
				}
				rgb := chansi.RGBColor{R: uint8(values[i+2]), G: uint8(values[i+3]), B: uint8(values[i+4])}
				if targetFG {
					m.style.Fg = rgb
				} else {
					m.style.Bg = rgb
				}
				i += 4
			}
		}
	}
}

func (m *Model) moveCursor(x, y int) {
	m.cursorX = clamp(x, 0, m.width-1)
	m.cursorY = clamp(y, 0, m.height-1)
}

func (m *Model) saveCursor() {
	m.savedX = m.cursorX
	m.savedY = m.cursorY
}

func (m *Model) restoreCursor() {
	m.moveCursor(m.savedX, m.savedY)
}

func (m *Model) enterAltScreen() {
	if m.useAlt {
		return
	}
	m.saveCursor()
	m.useAlt = true
	m.alt.Clear()
	m.cursorX = 0
	m.cursorY = 0
	m.scrollTop = 0
	m.scrollBottom = m.height - 1
}

func (m *Model) leaveAltScreen() {
	if !m.useAlt {
		return
	}
	m.useAlt = false
	m.restoreCursor()
	m.scrollTop = 0
	m.scrollBottom = m.height - 1
}

func (m *Model) lineRegion() uv.Rectangle {
	return uv.Rect(0, m.cursorY, m.width, 1)
}

func (m *Model) scrollRegion() uv.Rectangle {
	return uv.Rect(0, m.scrollTop, m.width, m.scrollBottom-m.scrollTop+1)
}

func (m *Model) setScrollRegion(params chansi.Params) {
	top := paramOrDefault(params, 0, 1) - 1
	bottom := paramOrDefault(params, 1, m.height) - 1
	if top < 0 || bottom < top || bottom >= m.height {
		m.scrollTop = 0
		m.scrollBottom = m.height - 1
		return
	}
	m.scrollTop = top
	m.scrollBottom = bottom
	m.moveCursor(0, 0)
}

func (m *Model) eraseCharacters(count int) {
	if count <= 0 {
		count = 1
	}
	for x := m.cursorX; x < min(m.width, m.cursorX+count); x++ {
		m.active().SetCell(x, m.cursorY, nil)
	}
}

func (m *Model) eraseLine(mode int) {
	switch mode {
	case 1:
		for x := 0; x <= m.cursorX && x < m.width; x++ {
			m.active().SetCell(x, m.cursorY, nil)
		}
	case 2:
		for x := 0; x < m.width; x++ {
			m.active().SetCell(x, m.cursorY, nil)
		}
	default:
		for x := m.cursorX; x < m.width; x++ {
			m.active().SetCell(x, m.cursorY, nil)
		}
	}
}

func (m *Model) eraseDisplay(mode int) {
	switch mode {
	case 1:
		for y := 0; y < m.cursorY; y++ {
			for x := 0; x < m.width; x++ {
				m.active().SetCell(x, y, nil)
			}
		}
		for x := 0; x <= m.cursorX && x < m.width; x++ {
			m.active().SetCell(x, m.cursorY, nil)
		}
	case 2, 3:
		m.active().Clear()
		m.moveCursor(0, 0)
	default:
		for x := m.cursorX; x < m.width; x++ {
			m.active().SetCell(x, m.cursorY, nil)
		}
		for y := m.cursorY + 1; y < m.height; y++ {
			for x := 0; x < m.width; x++ {
				m.active().SetCell(x, y, nil)
			}
		}
	}
}

func (m *Model) newLine() {
	m.cursorX = 0
	m.index()
}

func (m *Model) index() {
	if m.cursorY == m.scrollBottom {
		m.scrollUp(1)
		return
	}
	m.cursorY = clamp(m.cursorY+1, 0, m.height-1)
}

func (m *Model) reverseIndex() {
	if m.cursorY == m.scrollTop {
		m.scrollDown(1)
		return
	}
	m.cursorY = clamp(m.cursorY-1, 0, m.height-1)
}

func (m *Model) scrollUp(count int) {
	if count <= 0 {
		count = 1
	}
	m.active().DeleteLineArea(m.scrollTop, count, nil, m.scrollRegion())
}

func (m *Model) scrollDown(count int) {
	if count <= 0 {
		count = 1
	}
	m.active().InsertLineArea(m.scrollTop, count, nil, m.scrollRegion())
}

func paramsToInts(params chansi.Params) []int {
	out := make([]int, 0, len(params))
	params.ForEach(0, func(_ int, param int, _ bool) {
		out = append(out, param)
	})
	return out
}

func paramOrDefault(params chansi.Params, idx, def int) int {
	value, _, ok := params.Param(idx, def)
	if !ok {
		return def
	}
	return value
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clamp(v, low, high int) int {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}
