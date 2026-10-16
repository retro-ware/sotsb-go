package engine

// The list menu (O16B:13EE and its nested procedures; the node count of
// O16B:0062 is len(list)).

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

// newList = O16B:18F5: n empty nodes (n<2 gives 1 node). freeList is a no-op.
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

// nth = O16B:0000 (nil when i >= count or negative).
func (c *listMenuCtx) nth(i int) *ListNode {
	if i < 0 || i >= len(c.list) {
		return nil
	}
	return c.list[i]
}

// drawRows = O16B:0DAC: full redraw of the list area from node first.
// (lastHeaderBefore, O16B:0D20, is computed there but never used.)
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

// firstNonSpace = O16B:0F22, lastNonSpace = O16B:0F6E (1-based).
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

// lmDrawRow = O16B:0FB4 (highlighted: black on the highlight colour) /
// O16B:1056 (normal): redraw only the trimmed text of row i.
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

// lmStep moves the cursor one row with the wrap rules of O16B:1319 (without
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

// lmSkipHeaders = O16B:1157: up to rows times, step past header rows.
func (e *Engine) lmSkipHeaders(c *listMenuCtx, forward bool) {
	for k := 0; k < c.rows; k++ {
		if p := c.nth(*c.index); p != nil && !p.skipped() {
			break
		}
		e.lmStep(c, forward)
	}
}

// lmMoveCursor = O16B:1319.
func (e *Engine) lmMoveCursor(c *listMenuCtx, forward bool) {
	e.lmStep(c, forward)
	e.lmSkipHeaders(c, forward)
}

// lmPageScroll = O16B:1287.
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

// ListMenu = O16B:13EE. index is 0-based in/out (persists across calls),
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
	c := &listMenuCtx{list: list, index: index, x1: x1, y1: y1, x2: x2, y2: y2,
		promptColour: promptColour, normal: normal, hi: hi,
		width: x2 - x1 + 1, rows: y2 - y1 + 1, total: len(list)}
	for c.leadingHdr < len(list) && list[c.leadingHdr].Flag != 0 {
		c.leadingHdr++
	}
	if c.total <= c.rows {
		e.listTop = 0
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
			case scanUp:
				e.lmMoveCursor(c, false)
			case scanDown:
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
