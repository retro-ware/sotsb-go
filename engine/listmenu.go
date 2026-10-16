package engine

// The list menu (O104:0E9F and its nested procedures; the node count of
// O104:0076 is len(list)).

// ListNode is the ovr26 list node: 0x2E bytes, text string[40] at +0, flag at
// +0x29 (non-zero = header row, drawn in promptColour and skipped by the
// cursor). The next pointer (+0x2A) is the slice order.
type ListNode struct {
	Text string
	Flag byte
	Dim  bool // (extra) greyed out and passed over by the cursor, as headers
}

// dimColour draws a greyed-out line.
const dimColour = 8

// skipped: the cursor passes over the node.
func (p *ListNode) skipped() bool { return p.Flag != 0 || p.Dim }

// newList = O104:1253: n empty nodes (n<2 gives 1 node). freeList is a no-op.
func newList(n int) []*ListNode {
	if n < 1 {
		n = 1
	}
	l := make([]*ListNode, n)
	for i := range l {
		l[i] = &ListNode{}
	}
	return l
}

// listMenuCtx holds the "frame" the ovr26 helpers read from listMenu's locals.
type listMenuCtx struct {
	list                           []*ListNode
	index                          *int
	x1, y1, x2, y2                 int
	promptColour, normal, hi       int
	width, rows, total, leadingHdr int
}

// nth = O104:000F (nil when i >= count or negative).
func (c *listMenuCtx) nth(i int) *ListNode {
	if i < 0 || i >= len(c.list) {
		return nil
	}
	return c.list[i]
}

// drawRows = O104:0897: full redraw of the list area from node first.
// (lastHeaderBefore, O104:0804, is computed there but never used.)
func (e *Engine) lmDrawRows(c *listMenuCtx, first int) {
	e.ClearCells(c.x1, c.y1, c.x2, c.y2)
	y := c.y1
	for i := first; i >= 0 && i < len(c.list) && y <= c.y2; i++ {
		p := c.list[i]
		col := c.normal
		if p.Flag != 0 {
			col = c.promptColour
		}
		e.displayStringBg(c.x1, y, col, colourBlack, p.Text)
		if len(p.Text) < c.width {
			e.displayCharsBg(c.x1+len(p.Text), y, colourBlack, colourBlack, c.width-len(p.Text), ' ')
		}
		y++
	}
}

// firstNonSpace = O104:09D3, lastNonSpace = O104:0A1F (1-based).
func firstNonSpace(s string) int {
	i := 1
	for i < len(s) && s[i-1] == ' ' {
		i++
	}
	return i
}
func lastNonSpace(s string) int {
	i := len(s)
	for i > 1 && s[i-1] == ' ' {
		i--
	}
	return i
}

// lmDrawRow = O104:0A65 (highlighted: black on the highlight colour) /
// O104:0B07 (normal): redraw only the trimmed text of row i.
func (e *Engine) lmDrawRow(c *listMenuCtx, i int, highlighted bool) {
	p := c.nth(i)
	if p == nil {
		return
	}
	a, b := firstNonSpace(p.Text), lastNonSpace(p.Text)
	if b < a {
		return
	}
	x, y, text := c.x1+a-1, c.y1+i-e.listTop, p.Text[a-1:b]
	switch {
	case highlighted:
		e.displayStringBg(x, y, colourBlack, c.hi, text)
	case p.Flag != 0:
		e.displayStringBg(x, y, c.promptColour, colourBlack, text)
	default:
		e.displayStringBg(x, y, c.normal, colourBlack, text)
	}
}

// lmStep moves the cursor one row with the wrap rules of O104:0DCA (without
// the header skip).
func (e *Engine) lmStep(c *listMenuCtx, forward bool) {
	top := e.listTop
	if forward {
		*c.index++
		if *c.index > top+c.rows-1 {
			*c.index = top
		}
		if *c.index > c.total-1 {
			*c.index = top
		}
	} else {
		*c.index--
		if *c.index < top {
			*c.index = top + c.rows - 1
		}
		if *c.index > c.total-1 {
			*c.index = c.total - 1
		}
	}
}

// lmSkipHeaders = O104:0C08: up to rows times, step past header rows.
func (e *Engine) lmSkipHeaders(c *listMenuCtx, forward bool) {
	for k := 0; k < c.rows; k++ {
		if p := c.nth(*c.index); p != nil && !p.skipped() {
			break
		}
		e.lmStep(c, forward)
	}
}

// lmMoveCursor = O104:0DCA.
func (e *Engine) lmMoveCursor(c *listMenuCtx, forward bool) {
	e.lmStep(c, forward)
	e.lmSkipHeaders(c, forward)
}

// lmPageScroll = O104:0D38.
func (e *Engine) lmPageScroll(c *listMenuCtx, forward bool) {
	rel := *c.index - e.listTop
	if forward {
		e.listTop += c.rows
		if e.listTop > c.total-c.rows {
			e.listTop = c.total - c.rows
		}
	} else {
		e.listTop -= c.rows
	}
	if e.listTop < 0 { // ponytail: original word arithmetic; clamp keeps Go indexes sane
		e.listTop = 0
	}
	*c.index = e.listTop + rel
	e.lmSkipHeaders(c, forward)
	e.lmDrawRows(c, e.listTop)
}

// ListMenu = O104:0E9F. index is 0-based in/out (persists across calls),
// redraw is set to force a full redraw and is cleared here. Returns the
// DisplayInput key (0 on exit) and sets *selected to the node under the
// cursor (nil on exit).
func (e *Engine) ListMenu(title, menu string, promptColour, normal, hi, x1, y1, x2, y2 int,
	list []*ListNode, showExit bool, redraw *bool, index *int, selected **ListNode) byte {
	if len(title) > 40 {
		title = title[:40]
	}
	if len(menu) > 40 {
		menu = menu[:40]
	}
	if len(list) == 0 {
		*index = 0
		*selected = nil
		return 0
	}
	e.menuSel = 1 // DS:6B2B
	c := &listMenuCtx{list: list, index: index, x1: x1, y1: y1, x2: x2, y2: y2,
		promptColour: promptColour, normal: normal, hi: hi,
		width: x2 - x1 + 1, rows: y2 - y1 + 1, total: len(list)}
	for c.leadingHdr < len(list) && list[c.leadingHdr].Flag != 0 {
		c.leadingHdr++
	}
	if e.listTop > *index {
		e.listTop = *index
		*redraw = true
	}
	if e.listTop > c.total {
		e.listTop = 0
		*redraw = true
	}
	*index++
	e.lmMoveCursor(c, false)
	if *redraw {
		e.lmDrawRows(c, e.listTop)
	}
	for {
		*redraw = false
		e.lmDrawRow(c, *index, true)
		m := menu
		hasNext, hasPrev := false, false
		if c.total-c.rows > e.listTop {
			m += " Next"
			hasNext = true
		}
		if e.listTop > c.leadingHdr {
			m += " Prev"
			hasPrev = true
		}
		if showExit {
			m += " Exit"
		}
		key, extra := e.DisplayInput(title, m, promptColour, normal, hi, true, false)
		e.lmDrawRow(c, *index, false)
		if extra {
			switch key {
			case scanHome: // keypad 7: one line up
				e.lmMoveCursor(c, false)
			case scanEnd: // keypad 1: one line down
				e.lmMoveCursor(c, true)
			case scanPageUp:
				if hasPrev {
					e.lmPageScroll(c, false)
				}
			case scanPageDown:
				if hasNext {
					e.lmPageScroll(c, true)
				}
			}
			continue
		}
		switch key {
		case 'P':
			e.lmPageScroll(c, false)
		case 'N':
			e.lmPageScroll(c, true)
		case 27, 'E', 0:
			*selected = nil
			return 0
		default:
			*selected = c.nth(*index)
			return key
		}
	}
}
