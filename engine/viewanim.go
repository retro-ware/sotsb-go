package engine

// The party's blinking cursor on the wilderness map (unit O109): a white
// cell (O060's init_dax_block 8x1 filled with 255) over the place the
// area word names; what it covers is saved (DS:65E4) and put back.

const (
	areaWordMapPlace = 417 // the party's place on the wilderness map
	mapCursorPx      = 8
	mapCursorColour  = 15
)

// mapCursorPlace = O109:0005: the cell of the party's place (DS:8C8C/8C8E).
func (e *Engine) mapCursorPlace() {
	p := int(wordAt(e.area, 2*areaWordMapPlace)) % len(wildMapCursorX)
	e.mapCursorX, e.mapCursorY = int(wildMapCursorX[p]), int(wildMapCursorY[p])
}

// mapCursorShow = O109:002E: save the cell (ega_01) and draw the cursor.
func (e *Engine) mapCursorShow() {
	x0, y0 := e.mapCursorX*mapCursorPx, e.mapCursorY*mapCursorPx
	e.S.Lock()
	for r := 0; r < mapCursorPx; r++ {
		for c := 0; c < mapCursorPx; c++ {
			e.mapCursorUnder[r*mapCursorPx+c] = e.S.Get(x0+c, y0+r)
			e.S.Set(x0+c, y0+r, mapCursorColour)
		}
	}
	e.S.Unlock()
	e.Flip()
}

// mapCursorHide = O109:005D: put the saved cell back.
func (e *Engine) mapCursorHide() {
	x0, y0 := e.mapCursorX*mapCursorPx, e.mapCursorY*mapCursorPx
	e.S.Lock()
	for r := 0; r < mapCursorPx; r++ {
		for c := 0; c < mapCursorPx; c++ {
			e.S.Set(x0+c, y0+r, e.mapCursorUnder[r*mapCursorPx+c])
		}
	}
	e.S.Unlock()
	e.Flip()
}
