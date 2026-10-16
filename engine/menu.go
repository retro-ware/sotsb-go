package engine

// The bottom-row menu (displayInput) and the yes/no question (unit O104 of
// the reconstruction).

// isHotkey: the characters that start a menu item ([0-9A-Z]).
func isHotkey(c byte) bool { return c >= '0' && c <= '9' || c >= 'A' && c <= 'Z' }

// menuItem is the span of one menu item: from its hotkey to two cells
// before the next item's hotkey (1-based positions in the menu text).
type menuItem struct{ first, last int }

// menuItemsOf = O104:00DA (nested in displayInput): split the menu text into
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

// keypadMap = DS 10192: the digits 1..9 as the scan codes of the keypad keys
// (End Down PgDn Left - Right Home Up PgUp).
var keypadMap = []byte("OPQK MGHI")

// The wilderness map cursor blinks while a menu waits.
const (
	cursorShowDelay = 30 // ticks
	cursorHideDelay = 50
	areaPictureWord = 1022 // area word: a picture changed, restart the animation
	wildernessArea  = 'y'  // b72AA: the area that has a blinking map cursor
	notOnMap        = 'P'
)

// drawMenuBar = O104:01E9 (nested in displayInput): the menu text after the
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

// DisplayInput = O104:03D8 displayInput (prompt, menu, promptColour,
// normal, highlight, allowExtra, animate): the prompt and the menu on the
// bottom row; returns the key (the item's hotkey for Enter, 0 for Escape)
// and whether it was a keypad / extended key. The selected item
// (menuSel) is kept between calls.
func (e *Engine) DisplayInput(prompt, menu string, promptColour, normal, highlight int, allowExtra, animate bool) (byte, bool) {
	if len(prompt) > 40 {
		prompt = prompt[:40]
	}
	if len(menu) > 40 {
		menu = menu[:40]
	}
	e.menuExtra = false
	hasHighlight := normal != 0 || highlight != 0
	items := menuItemsOf(menu)
	if e.menuSel > len(items) || e.menuSel < 1 {
		e.menuSel = 1
	}
	key, extra, done := byte(0), false, false
	since := e.Ticks()
	cursorShow := since + cursorShowDelay
	cursorHide := cursorShow + cursorHideDelay
	if prompt != "" {
		e.displayStringBg(0, statusRow, promptColour, colourBlack, prompt)
	}
	x0 := len(prompt)
	e.drawMenuBar(x0, menu, items, e.menuSel, normal, highlight)
	mapCursor := e.gameState == stateWilderness && e.areaKind == wildernessArea && e.mapMode != notOnMap
	if mapCursor {
		e.mapCursorPlace()
		e.mapCursorShow()
		e.mapCursorHide()
	}
	for !done {
		if mapCursor && e.Ticks() >= cursorShow {
			e.mapCursorShow()
			cursorShow = cursorHide + cursorShowDelay
		}
		if a := &e.pic; (wordAt(e.area, areaPictureWord) != 0 || animate) && a.cur > 0 && a.cur <= len(a.frames) {
			e.drawViewPicture(viewPicX, viewPicY, animate, a.cur-1)
			if e.Ticks()-since >= int64(a.frames[a.cur-1].Delay)*10 || wordAt(e.area, areaPictureWord) != 0 {
				a.cur++
				if a.cur > len(a.frames) {
					a.cur = 1
				}
				since = e.Ticks()
			}
		}
		switch {
		case e.inputTimeout > 0 && e.Ticks()-since >= e.inputTimeout:
			key, done = e.inputTimeoutKey, true
		case e.keyWaiting():
			key = e.ReadKeyByte()
			if key == 0 {
				key = e.ReadKeyByte()
				if allowExtra {
					extra, done = true, true
				}
				break
			}
			switch key {
			case 27:
				key, done = 0, true
			case 13:
				if hasHighlight && e.menuSel > 0 && e.menuSel <= len(items) && items[e.menuSel-1].first > 0 {
					key = menu[items[e.menuSel-1].first-1]
					if key == 0 {
						key = 13
					}
					done = true
				}
			case ',':
				if e.menuSel == 1 {
					e.menuSel = len(items)
				} else {
					e.menuSel--
				}
				e.drawMenuBar(x0, menu, items, e.menuSel, normal, highlight)
			case '.':
				if e.menuSel == len(items) {
					e.menuSel = 1
				} else {
					e.menuSel++
				}
				e.drawMenuBar(x0, menu, items, e.menuSel, normal, highlight)
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
				if allowExtra && (key >= '1' && key <= '9' || key == '\\') {
					if key == '\\' {
						key = '7' // the original stores '7' itself, not the keypad code (sic)
					} else {
						key = keypadMap[key-'1']
					}
					extra, done = true, true
				}
			}
		default:
			e.idle()
		}
		if mapCursor && e.Ticks() >= cursorHide {
			e.mapCursorHide()
			cursorHide = cursorShow + cursorHideDelay
		}
	}
	putWord(e.area, areaPictureWord, 0)
	if mapCursor {
		e.mapCursorHide()
	}
	e.menuExtra = extra
	return key, extra
}

// YesNo = O104:11E4 (prompt, c1, c2, c3): "Yes No" on the bottom row with
// No preselected, until Y or N is chosen.
func (e *Engine) YesNo(c1, c2, c3 int, prompt string) byte {
	e.menuSel = 2
	for {
		k, _ := e.DisplayInput(prompt, "Yes No", c1, c2, c3, false, false)
		if k == 'N' || k == 'Y' {
			return k
		}
	}
}
