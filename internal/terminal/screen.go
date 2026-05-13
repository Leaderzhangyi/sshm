package terminal

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

type Cell struct {
	Char  rune
	Style string
}

type Screen struct {
	cells  [][]Cell
	Width  int
	Height int
	curRow int
	curCol int
	style  string

	scrollTop int
	scrollBot int

	state  byte
	csiBuf []byte

	altCells [][]Cell
	altStyle string
	usingAlt bool
	savedRow int
	savedCol int

	utf8Buf [4]byte
	utf8Len int
}

func NewScreen(w, h int) *Screen {
	s := &Screen{Width: w, Height: h, state: 'n'}
	s.cells = makeGrid(w, h)
	s.scrollBot = h - 1
	return s
}

func makeGrid(w, h int) [][]Cell {
	g := make([][]Cell, h)
	for i := range g {
		g[i] = make([]Cell, w)
	}
	return g
}

func (s *Screen) grid() [][]Cell {
	if s.usingAlt {
		return s.altCells
	}
	return s.cells
}

func (s *Screen) Process(data []byte) {
	g := s.grid()
	for _, b := range data {
		switch s.state {
		case 'n':
			s.normal(b, g)
		case 'e':
			s.esc(b)
		case 'c':
			s.csi(b, g)
		case 'o':
			s.osc(b)
		}
	}
}

func (s *Screen) normal(b byte, g [][]Cell) {
	if s.utf8Len > 0 {
		s.utf8Buf[s.utf8Len] = b
		s.utf8Len++
		r, sz := utf8.DecodeRune(s.utf8Buf[:s.utf8Len])
		if sz == s.utf8Len {
			s.put(r, g)
			s.utf8Len = 0
		} else if s.utf8Len >= 4 {
			s.utf8Len = 0
		}
		return
	}

	switch b {
	case '\n':
		s.newLine(g)
	case '\r':
		s.curCol = 0
	case '\b':
		if s.curCol > 0 {
			s.curCol--
		}
	case '\t':
		s.curCol = (s.curCol + 8) &^ 7
		if s.curCol >= s.Width {
			s.curCol = s.Width - 1
		}
	case 0x1b:
		s.utf8Len = 0
		s.state = 'e'
	default:
		if b >= 0xC0 {
			s.utf8Buf[0] = b
			s.utf8Len = 1
		} else if b >= 0x20 {
			s.put(rune(b), g)
		}
	}
}

func (s *Screen) put(r rune, g [][]Cell) {
	if s.curCol >= s.Width {
		s.curCol = 0
		s.newLine(g)
	}
	if s.curRow < s.Height && s.curCol < s.Width {
		g[s.curRow][s.curCol] = Cell{Char: r, Style: s.style}
	}
	s.curCol++
}

func (s *Screen) newLine(g [][]Cell) {
	if s.curRow == s.scrollBot {
		for i := s.scrollTop; i < s.scrollBot; i++ {
			g[i] = g[i+1]
		}
		g[s.scrollBot] = make([]Cell, s.Width)
		s.curRow = s.scrollBot
	} else {
		s.curRow++
		if s.curRow >= s.Height {
			s.curRow = s.Height - 1
		}
	}
}

func (s *Screen) esc(b byte) {
	switch b {
	case '[':
		s.state = 'c'
		s.csiBuf = s.csiBuf[:0]
	case ']':
		s.state = 'o'
	case '7':
		s.savedRow, s.savedCol = s.curRow, s.curCol
		s.state = 'n'
	case '8':
		s.curRow, s.curCol = s.savedRow, s.savedCol
		s.state = 'n'
	default:
		s.state = 'n'
	}
}

func (s *Screen) csi(b byte, g [][]Cell) {
	if (b >= '0' && b <= '9') || b == ';' || b == '?' {
		s.csiBuf = append(s.csiBuf, b)
		return
	}
	s.execCSI(b, string(s.csiBuf), g)
	s.state = 'n'
}

func (s *Screen) execCSI(cmd byte, raw string, g [][]Cell) {
	private := false
	params := raw
	if len(params) > 0 && params[0] == '?' {
		private = true
		params = params[1:]
	}

	switch cmd {
	case 'A':
		s.curRow -= csiInt(params, 1)
	case 'B':
		s.curRow += csiInt(params, 1)
	case 'C':
		s.curCol += csiInt(params, 1)
	case 'D':
		s.curCol -= csiInt(params, 1)
	case 'H', 'f':
		r, c := csiPos(params)
		s.curRow, s.curCol = r-1, c-1
	case 'J':
		s.eraseDisplay(params, g)
	case 'K':
		s.eraseLine(params, g)
	case 'L':
		n := csiInt(params, 1)
		for i := s.scrollBot; i >= s.curRow+n; i-- {
			g[i] = g[i-n]
		}
		for i := s.curRow; i < s.curRow+n && i <= s.scrollBot; i++ {
			g[i] = make([]Cell, s.Width)
		}
	case 'M':
		n := csiInt(params, 1)
		for i := s.curRow; i <= s.scrollBot-n; i++ {
			g[i] = g[i+n]
		}
		for i := s.scrollBot - n + 1; i <= s.scrollBot; i++ {
			g[i] = make([]Cell, s.Width)
		}
	case 'P':
		n := csiInt(params, 1)
		for i := s.curCol + n; i < s.Width; i++ {
			g[s.curRow][i-n] = g[s.curRow][i]
		}
		for i := s.Width - n; i < s.Width; i++ {
			g[s.curRow][i] = Cell{}
		}
	case 'm':
		if params == "0" || params == "" {
			s.style = ""
		} else {
			s.style += "\x1b[" + params + "m"
		}
	case 'r':
		top, bot := csiPos(params)
		s.scrollTop = top - 1
		s.scrollBot = bot - 1
		if s.scrollTop < 0 {
			s.scrollTop = 0
		}
		if s.scrollBot >= s.Height {
			s.scrollBot = s.Height - 1
		}
		s.curRow, s.curCol = 0, 0
	case 'h':
		if private && params == "1049" {
			s.altCells = makeGrid(s.Width, s.Height)
			s.altStyle = s.style
			s.usingAlt = true
			s.curRow, s.curCol = 0, 0
			s.style = ""
			s.scrollTop = 0
			s.scrollBot = s.Height - 1
		}
	case 'l':
		if private && params == "1049" {
			s.usingAlt = false
			s.style = s.altStyle
			s.scrollTop = 0
			s.scrollBot = s.Height - 1
		}
	case 'G':
		s.curCol = csiInt(params, 1) - 1
	case 'd':
		s.curRow = csiInt(params, 1) - 1
	}
	s.clamp()
}

func (s *Screen) eraseDisplay(params string, g [][]Cell) {
	switch params {
	case "0", "":
		for i := s.curCol; i < s.Width; i++ {
			g[s.curRow][i] = Cell{}
		}
		for r := s.curRow + 1; r < s.Height; r++ {
			g[r] = make([]Cell, s.Width)
		}
	case "1":
		for r := 0; r < s.curRow; r++ {
			g[r] = make([]Cell, s.Width)
		}
		for i := 0; i <= s.curCol && i < s.Width; i++ {
			g[s.curRow][i] = Cell{}
		}
	case "2":
		for r := 0; r < s.Height; r++ {
			g[r] = make([]Cell, s.Width)
		}
	}
}

func (s *Screen) eraseLine(params string, g [][]Cell) {
	switch params {
	case "0", "":
		for i := s.curCol; i < s.Width; i++ {
			g[s.curRow][i] = Cell{}
		}
	case "1":
		for i := 0; i <= s.curCol && i < s.Width; i++ {
			g[s.curRow][i] = Cell{}
		}
	case "2":
		g[s.curRow] = make([]Cell, s.Width)
	}
}

func (s *Screen) osc(b byte) {
	switch b {
	case 0x07:
		s.state = 'n'
	case 0x1b:
		s.state = 'e'
	}
}

func (s *Screen) clamp() {
	if s.curRow < 0 {
		s.curRow = 0
	}
	if s.curCol < 0 {
		s.curCol = 0
	}
	if s.curRow >= s.Height {
		s.curRow = s.Height - 1
	}
	if s.curCol >= s.Width {
		s.curCol = s.Width - 1
	}
}

func (s *Screen) String() string {
	g := s.grid()
	var sb strings.Builder
	last := ""
	for row := 0; row < s.Height; row++ {
		for col := 0; col < s.Width; col++ {
			c := g[row][col]
			if c.Style != last {
				sb.WriteString("\x1b[0m")
				if c.Style != "" {
					sb.WriteString(c.Style)
				}
				last = c.Style
			}
			if c.Char != 0 {
				sb.WriteRune(c.Char)
			} else {
				sb.WriteByte(' ')
			}
		}
		if last != "" {
			sb.WriteString("\x1b[0m")
			last = ""
		}
		if row < s.Height-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

func (s *Screen) Resize(w, h int) {
	ng := makeGrid(w, h)
	old := s.grid()
	for r := 0; r < min(s.Height, h); r++ {
		for c := 0; c < min(s.Width, w); c++ {
			ng[r][c] = old[r][c]
		}
	}
	s.Width, s.Height = w, h
	s.scrollBot = h - 1
	if s.usingAlt {
		s.altCells = ng
	} else {
		s.cells = ng
	}
	s.clamp()
}

func csiInt(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n == 0 {
		return def
	}
	return n
}

func csiPos(s string) (int, int) {
	a, b := 1, 1
	parts := strings.SplitN(s, ";", 2)
	if len(parts) >= 1 && parts[0] != "" {
		if v, err := strconv.Atoi(parts[0]); err == nil {
			a = v
		}
	}
	if len(parts) >= 2 && parts[1] != "" {
		if v, err := strconv.Atoi(parts[1]); err == nil {
			b = v
		}
	}
	return a, b
}
