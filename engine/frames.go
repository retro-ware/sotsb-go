package engine

// Screen frames drawn from 8x8 line symbols (unit R19D of the reconstruction).
// Silver Blades draws its frames from a handful of line pieces instead of
// Curse's per-cell frame tables.

// The frame symbols (8X8D set 1).
const (
	symbolLineTopEnd      = 308 // a vertical line's upper end, under a rule
	symbolLineVertical    = 309
	symbolLineBottomEnd   = 310 // a vertical line's lower end, over a rule
	symbolLineHorizontal  = 311
	symbolInnerTopEnd     = 312 // the 3D view's inner frame
	symbolInnerVertical   = 313
	symbolInnerBottomEnd  = 314
	symbolInnerHorizontal = 315
)

const (
	screenLastCol = 39
	screenLastRow = 23
	// The rows and columns of the adventure screen's dividers.
	viewDividerColumn   = 16 // between the 3D view and the party list
	textRuleRow         = 16 // above the text window
	viewInnerFirst      = 2  // the 3D view's inner frame, rows and columns 2..14
	viewInnerLast       = 14
	combatDividerColumn = 22 // the combat screen's divider between the map and the panel
	combatRuleRow       = 22 // the rule under the map
	listRuleRow1        = 8  // the three rules of the shop and training lists
	listRuleRow2        = 16
	listRuleRow3        = 20
	listDividerColumn   = 19
)

// verticalSymbol is the piece of a vertical line at row y between first and
// last: the upper end, the middle or the lower end.
func verticalSymbol(y, first, last int) int {
	switch y {
	case first:
		return symbolLineTopEnd
	case last:
		return symbolLineBottomEnd
	}
	return symbolLineVertical
}

// drawScreenFrame = R19D:0000: clear the inside and draw the frame around
// the whole screen.
func (e *Engine) drawScreenFrame() {
	e.ClearCells(1, 1, screenLastCol-1, screenLastRow-1)
	for x := 0; x <= screenLastCol; x++ {
		e.Put8x8Symbol(x, 0, symbolLineHorizontal)
		e.Put8x8Symbol(x, screenLastRow, symbolLineHorizontal)
	}
	for y := 1; y <= screenLastRow-1; y++ {
		piece := verticalSymbol(y, 1, screenLastRow-1)
		e.Put8x8Symbol(0, y, piece)
		e.Put8x8Symbol(screenLastCol, y, piece)
	}
}

// drawRule = R19D:00C0: a horizontal rule across the screen at row y, with
// the side lines ending above and below it.
func (e *Engine) drawRule(y int) {
	for x := 0; x <= screenLastCol; x++ {
		e.Put8x8Symbol(x, y, symbolLineHorizontal)
	}
	e.Put8x8Symbol(0, y-1, symbolLineBottomEnd)
	e.Put8x8Symbol(0, y+1, symbolLineTopEnd)
	e.Put8x8Symbol(screenLastCol, y-1, symbolLineBottomEnd)
	e.Put8x8Symbol(screenLastCol, y+1, symbolLineTopEnd)
}

// drawLowerFrame = R19D:0156: the side lines from row 2 down and the bottom
// line; the top rows belong to whatever the caller draws there.
func (e *Engine) drawLowerFrame() {
	for y := 2; y <= screenLastRow-2; y++ {
		e.Put8x8Symbol(0, y, symbolLineVertical)
		e.Put8x8Symbol(screenLastCol, y, symbolLineVertical)
	}
	e.Put8x8Symbol(0, screenLastRow-1, symbolLineBottomEnd)
	e.Put8x8Symbol(screenLastCol, screenLastRow-1, symbolLineBottomEnd)
	for x := 0; x <= screenLastCol; x++ {
		e.Put8x8Symbol(x, screenLastRow, symbolLineHorizontal)
	}
}

// drawCreditsFrame = R19D:01EA: clear the screen, a line along rows 1 and 23
// and plain side lines between them.
func (e *Engine) drawCreditsFrame() {
	e.ClearCells(0, 0, screenLastCol, screenLastRow)
	for y := 1; y <= screenLastRow; y++ {
		if y == 1 || y == screenLastRow {
			for x := 0; x <= screenLastCol; x++ {
				e.Put8x8Symbol(x, y, symbolLineHorizontal)
			}
			continue
		}
		e.Put8x8Symbol(0, y, symbolLineVertical)
		e.Put8x8Symbol(screenLastCol, y, symbolLineVertical)
	}
}

// drawAdventureFrame = R19D:0299: the text-window frame, the divider between
// the 3D view and the party list, and the view's inner frame.
func (e *Engine) drawAdventureFrame() {
	e.drawTextWindowFrame()
	for y := 1; y <= textRuleRow-1; y++ {
		e.Put8x8Symbol(viewDividerColumn, y, verticalSymbol(y, 1, textRuleRow-1))
	}
	for x := viewInnerFirst; x <= viewInnerLast; x++ {
		e.Put8x8Symbol(x, viewInnerFirst, symbolInnerHorizontal)
		e.Put8x8Symbol(x, viewInnerLast, symbolInnerHorizontal)
	}
	for y := viewInnerFirst + 1; y <= viewInnerLast-1; y++ {
		piece := symbolInnerVertical
		switch y {
		case viewInnerFirst + 1:
			piece = symbolInnerTopEnd
		case viewInnerLast - 1:
			piece = symbolInnerBottomEnd
		}
		e.Put8x8Symbol(viewInnerFirst, y, piece)
		e.Put8x8Symbol(viewInnerLast, y, piece)
	}
}

// drawTextWindowFrame = R19D:038F: clear the inside, the lower frame and the
// rule over the text window at row 16.
func (e *Engine) drawTextWindowFrame() {
	e.ClearCells(1, 1, screenLastCol-1, screenLastRow-1)
	e.drawLowerFrame()
	e.drawRule(textRuleRow)
}

// drawMemoriseFrame = R19D:03BE: as drawTextWindowFrame, but only the part
// above the rule is cleared (the spell list above, the memorised spells below).
func (e *Engine) drawMemoriseFrame() {
	e.ClearCells(1, 1, screenLastCol-1, textRuleRow)
	e.drawLowerFrame()
	e.drawRule(textRuleRow)
}

// drawCombatFrame = R19D:03ED: the lower frame with a divider at column 22
// down to a rule at row 22, and the bottom row cleared.
func (e *Engine) drawCombatFrame() {
	e.ClearCells(1, 1, screenLastCol-1, combatRuleRow-1)
	e.drawLowerFrame()
	for y := 1; y <= combatRuleRow-1; y++ {
		e.Put8x8Symbol(combatDividerColumn, y, verticalSymbol(y, 1, combatRuleRow-1))
	}
	e.drawRule(combatRuleRow)
	e.ClearCells(0, screenLastRow, screenLastCol, screenLastRow)
}

// drawShopFrame = R19D:0470: rules at rows 8, 16 and 20 with a divider at
// column 19 between the first two.
func (e *Engine) drawShopFrame() {
	e.ClearCells(1, 1, screenLastCol-1, screenLastRow-1)
	e.drawLowerFrame()
	e.drawRule(listRuleRow1)
	for y := listRuleRow1 + 1; y <= screenLastRow-4; y++ {
		e.Put8x8Symbol(listDividerColumn, y, verticalSymbol(y, listRuleRow1+1, screenLastRow-4))
	}
	e.drawRule(listRuleRow2)
	e.drawRule(listRuleRow3)
}

// drawListFrame = R19D:04F1: clear the upper part and draw the lower frame.
func (e *Engine) drawListFrame() {
	e.ClearCells(1, 1, screenLastCol-1, textRuleRow)
	e.drawLowerFrame()
}
