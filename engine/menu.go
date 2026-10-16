package engine

// The bottom-row menu (displayInput) and the yes/no question (unit O16B of
// the reconstruction).

// isHotkey: the characters that start a menu item ([0-9A-Z]).
func isHotkey(c byte) bool { return c >= '0' && c <= '9' || c >= 'A' && c <= 'Z' }

// menuItem is the span of one menu item: from its hotkey to two cells
// before the next item's hotkey (1-based positions in the menu text).
type menuItem struct{ first, last int }

// menuItemsOf = O16B:00C3 (nested in displayInput): split the menu text into
// its items; every hotkey starts one.
func menuItemsOf(menu string) []menuItem {
	items := []menuItem{{}}
	for i := 1; i <= len(menu); i++ {
		if !isHotkey(menu[i-1]) {
			continue
		}
		cur := &items[len(items)-1]
		if cur.first == 0 {
			cur.first = i
		} else {
			cur.last = i - 2
			items = append(items, menuItem{first: i})
		}
	}
	items[len(items)-1].last = len(menu)
	return items
}

// keypadMap = DS 341E: the digits 1..9 as the scan codes of the keypad keys
// (End Down PgDn Left - Right Home Up PgUp).
var keypadMap = []byte("OPQK MGHI")

// drawMenuBar = O16B:01D0 (nested in displayInput): the menu text after the
// prompt; the selected item black on the highlight colour, hotkeys in the
// highlight colour, the rest in the normal colour; the row cleared after it.
func (e *Engine) drawMenuBar(x0 int, menu string, items []menuItem, sel, normal, highlight int) {
	if len(menu) == 0 {
		return
	}
	for i := 1; i <= len(menu); i++ {
		c := menu[i-1]
		switch {
		case items[sel-1].first <= i && i <= items[sel-1].last && highlight != 0:
			e.displayCharsBg(x0+i-1, statusRow, colourBlack, highlight, 1, c)
		case isHotkey(c):
			e.displayCharsBg(x0+i-1, statusRow, highlight, colourBlack, 1, c)
		default:
			e.displayCharsBg(x0+i-1, statusRow, normal, colourBlack, 1, c)
		}
	}
	if x0+len(menu) < screenLastCol {
		e.displayCharsBg(x0+len(menu), statusRow, colourBlack, colourBlack, screenLastCol-len(menu)-x0+1, ' ')
	}
}

// DisplayInput = O16B:03C9 displayInput (prompt, menu, promptColour,
// normal, highlight, allowKeypad, animate): the prompt and the menu on the
// bottom row; returns the key (the item's hotkey for Enter, 0 for Escape)
// and whether it was an extended key. Left/Right and the keypad 4/6 move
// through the items (menuSel, kept between calls); with allowKeypad and a
// single item, 1..9 give the keypad's keys. The mouse is not supported.
func (e *Engine) DisplayInput(prompt, menu string, promptColour, normal, highlight int, allowKeypad, animate bool) (byte, bool) {
	if len(prompt) > 40 {
		prompt = prompt[:40]
	}
	if len(menu) > 40 {
		menu = menu[:40]
	}
	e.menuExtra = false
	items := menuItemsOf(menu)
	if e.menuSel > len(items) || e.menuSel < 1 {
		e.menuSel = 1
	}
	key, extra, done := byte(0), false, false
	since := e.Ticks()
	if prompt != "" {
		e.displayStringBg(0, statusRow, promptColour, colourBlack, prompt)
	}
	x0 := len(prompt)
	e.drawMenuBar(x0, menu, items, e.menuSel, normal, highlight)
	a := &e.pic
	animate = animate && a.cur > 0 && a.cur <= len(a.frames)
	if animate {
		e.drawViewPicture(viewPicX, viewPicY, true, a.cur-1)
		if len(a.frames) < 2 {
			animate = false
		}
	}
	step := func(d int) {
		e.menuSel = (e.menuSel-1+d+len(items))%len(items) + 1
		e.drawMenuBar(x0, menu, items, e.menuSel, normal, highlight)
	}
	for !done {
		if animate && e.Ticks()-since >= int64(a.frames[a.cur-1].Delay)*10 {
			a.cur++
			if a.cur > len(a.frames) {
				a.cur = 1
			}
			e.drawViewPicture(viewPicX, viewPicY, true, a.cur-1)
			since = e.Ticks()
		}
		if e.inputTimeout > 0 && e.Ticks()-since >= e.inputTimeout {
			key, done = e.inputTimeoutKey, true
		}
		if !e.keyWaiting() {
			if !done {
				e.idle()
			}
			continue
		}
		key = e.ReadKeyByte()
		if key == 0 {
			key = e.ReadKeyByte()
			extra = true
			switch {
			case len(items) > 1 && key == scanLeft:
				step(-1)
			case len(items) > 1 && key == scanRight:
				step(1)
			default:
				done = true
			}
			continue
		}
		extra = false
		switch key {
		case keyEsc:
			key, done = 0, true
		case keyEnter:
			key = byte(len(menu)) // menu[0], the length byte, when nothing is a hotkey
			if f := items[e.menuSel-1].first; f > 0 {
				key = menu[f-1]
			}
			if key == 0 {
				key = keyEnter
			}
			done = true
		case '4', '6':
			if len(items) > 1 {
				if key == '4' {
					step(-1)
				} else {
					step(1)
				}
			} else {
				if allowKeypad {
					key, extra = keypadMap[key-'1'], true
				}
				done = true
			}
		default:
			key = upper(key)
			if key == ' ' {
				done = true
			} else if isHotkey(key) {
				for i := 1; i <= len(menu); i++ {
					if menu[i-1] != key {
						continue
					}
					done = true
					j := 1
					for items[j-1].first != i {
						j++
					}
					e.menuSel = j
					e.drawMenuBar(x0, menu, items, e.menuSel, normal, highlight)
				}
			}
			if allowKeypad && (key >= '1' && key <= '9' || key == '\\') {
				if key == '\\' {
					key = '7' // the original gives '7' itself, not the keypad code (sic)
				} else {
					key = keypadMap[key-'1']
				}
				extra, done = true, true
			}
		}
	}
	e.ClearKeyboard()
	e.menuExtra = extra
	return key, extra
}

// menuChoice: DisplayInput for menus that only take letters; extended keys
// are passed over (callers that ignore the extended flag would read Up as
// 'H').
func (e *Engine) menuChoice(prompt, menu string, promptColour, normal, highlight int, allowKeypad, animate bool) (byte, bool) {
	for {
		if k, extra := e.DisplayInput(prompt, menu, promptColour, normal, highlight, allowKeypad, animate); !extra {
			return k, false
		}
	}
}

// YesNo: yesNoFrom with No preselected.
func (e *Engine) YesNo(c1, c2, c3 int, prompt string) byte {
	return e.yesNoFrom(answerNo, c1, c2, c3, prompt)
}

// The items of "Yes No".
const (
	answerYes = 1
	answerNo  = 2
)

// yesNoFrom = O16B:1885 (prompt, c1, c2, c3, sel): "Yes No" on the bottom
// row with item sel preselected, until Y or N is chosen.
func (e *Engine) yesNoFrom(sel, c1, c2, c3 int, prompt string) byte {
	e.menuSel = sel
	for {
		k, _ := e.menuChoice(prompt, "Yes No", c1, c2, c3, false, false)
		if k == 'N' || k == 'Y' {
			return k
		}
	}
}
