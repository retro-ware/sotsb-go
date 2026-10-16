package engine

// Messages on the bottom row (unit R4CE / O0ED of the reconstruction).

const (
	statusRow       = 24
	statusDelayStep = 100 // milliseconds per game speed step
)

// pauseForMessage = R4CE:0B19: wait game speed × 100 ms.
func (e *Engine) pauseForMessage() { e.Delay(int(e.gameSpeed) * statusDelayStep) }

// statusMessage = R4CE:0ABF: show text on the bottom row in the given
// colours, pause, clear the row again.
func (e *Engine) statusMessage(text string, fg, bg int) {
	e.ClearRow24()
	e.displayStringBg(0, statusRow, fg, bg, text)
	e.pauseForMessage()
	e.ClearRow24()
}

// printMessage = O0ED:172C string_print01 (S40): a green bottom-row message.
func (e *Engine) printMessage(text string) {
	e.statusMessage(cut(text, screenLastCol+1), colourGreen, colourBlack)
}

// displayStringBg = R4CE:0349 (x, y, fg, bg, text): the characters on a
// background colour, upcased, cut at the right edge.
func (e *Engine) displayStringBg(x, y, fg, bg int, text string) {
	if x > screenLastCol || y > screenLastCol {
		return
	}
	for i := 0; i < len(text) && i < 255; i++ {
		e.displayCharBg(x+i, y, fg, bg, text[i], true)
	}
}

// displayCharsBg = R4CE:01DF display_char01 (x, y, fg, bg, count, ch, upcase):
// the character count times from x on, upcased (the font has 64 glyphs).
func (e *Engine) displayCharsBg(x, y, fg, bg, count int, ch byte) {
	if x > screenLastCol || y > statusRow {
		return
	}
	for i := 0; i < count; i++ {
		e.drawChar(x+i, y, fg, bg, ch, true)
	}
}
