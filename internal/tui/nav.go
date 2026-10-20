package tui

// cursorList — курсор и прокрутка списка: ↑/↓, PgUp/PgDn, Home/End.
type cursorList struct{ n, pos, top, h int }

func (c *cursorList) SetLen(n int)    { c.n = n; c.clamp() }
func (c *cursorList) SetHeight(h int) { c.h = max(h, 1); c.clamp() }

// Pos — позиция курсора (0 при пустом списке).
func (c *cursorList) Pos() int { return c.pos }

// Set ставит курсор на позицию i.
func (c *cursorList) Set(i int) { c.pos = i; c.clamp() }

// Move сдвигает курсор на d строк.
func (c *cursorList) Move(d int) { c.pos += d; c.clamp() }

// Key обрабатывает клавишу навигации; false — клавиша не навигационная.
func (c *cursorList) Key(k string) bool {
	switch k {
	case "up":
		c.Move(-1)
	case "down":
		c.Move(1)
	case "pgup":
		c.Move(-max(c.h-1, 1))
	case "pgdown":
		c.Move(max(c.h-1, 1))
	case "home":
		c.Set(0)
	case "end":
		c.Set(c.n - 1)
	default:
		return false
	}
	return true
}

// Range — полуинтервал видимых строк [from, to).
func (c *cursorList) Range() (from, to int) { return c.top, min(c.top+c.h, c.n) }

func (c *cursorList) clamp() {
	c.pos = max(0, min(c.pos, c.n-1))
	if c.h < 1 {
		c.h = 1
	}
	if c.pos < c.top {
		c.top = c.pos
	}
	if c.pos >= c.top+c.h {
		c.top = c.pos - c.h + 1
	}
	c.top = max(0, min(c.top, c.n-c.h))
}
