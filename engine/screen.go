package engine

import "strconv"

// The adventure screen: frames, the view window and the status line.
// Ported from Pool 00BA:272A/27D9/290B and ovr27.

// skyFg = O10D:0014, the horizon colour for DS:6F69.
func skyFg(v byte) byte {
	switch v {
	case 8, 0x41:
		return 0
	}
	return 0xF
}

// skyBand = O10D:0048, the band colour for DS:6F69.
func skyBand(v byte) byte {
	if v == 8 {
		return 8
	}
	return 7
}

// skyGround = O10D:007C, the ground colour for DS:6F68.
func skyGround(v byte) byte {
	switch v {
	case 0xB:
		return 8
	case 6:
		return 1
	}
	return 6
}

// Redraw = Pool ovr27:016D: redraw the view window for the current view mode.
func (e *Engine) Redraw() {
	mode := e.viewMode
	if wordAt(e.area, 0x1CC) != 0 {
		mode = 1
	}
	switch {
	case mode == 1:
		e.es.cell = e.MapCell(e.partyX(), e.partyY())
		sky := wordAt(e.area, 0x1FA)
		if e.es.cell > 0x7F {
			sky = wordAt(e.area, 0x1FC)
		}
		s := e.gs.sky
		s[0] = skyColours[sky]
		if e.es.f84E5 != 0 {
			s[1] = skyFg(e.es.v6F69)
			s[2] = skyBand(e.es.v6F69)
			s[3] = skyGround(e.es.v6F68)
		}
		e.SetSkyColours(s[0], s[1], s[2], s[3])
		if wordAt(e.area, 0x1F6) != 0 {
			e.gs.overhead = false
		}
		e.Draw3DView(e.partyX(), e.partyY(), e.viewFacing())
		e.viewDrawn()
	}
}

// DrawMainFrame = Pool 00BA:272A (inner): the screen, text and view-window
// frames, plus the inner picture frame when inner is set.
func (e *Engine) DrawMainFrame(inner bool) {
	e.DrawBoxTitled(1, 1, 0x26, 0x16, 0, 0xF, "")
	e.DrawBoxTitled(1, 0x11, 0x26, 0x16, 0, 0xF, "")
	e.DrawBoxTitled(1, 1, 0xF, 0xF, 0, 0xF, "")
	if inner {
		e.DrawBoxTitled(3, 3, 0xD, 0xD, 0, 0xF, "")
	}
}

// DrawMainScreen = Pool 00BA:27D9: rebuild the whole screen for DS:49F3.
func (e *Engine) DrawMainScreen() {
	switch e.gameState {
	case 0:
		e.DrawBoxTitled(1, 1, 0x26, 0x16, 0, 0xF, "")
	case 1:
		e.DrawMainFrame(true)
		e.LoadPortrait(int(e.es.f49FD), int(e.es.f49FE))
		e.DrawPortrait(3, 3, true)
	case 2:
		e.DrawMainFrame(true)
		e.LoadAnimPic("PIC", 0x1D, false)
		e.DisplayPartyList()
		e.StatusLine()
	case 4:
		e.DrawMainFrame(true)
		e.Redraw()
		e.DisplayPartyList()
		e.StatusLine()
	case 3:
		e.DrawMainFrame(false)
		e.Redraw()
		e.DisplayPartyList()
		e.StatusLine()
	case 6:
		e.DrawMainFrame(true)
		e.LoadAnimPic("PIC", 1, false)
		e.DisplayPartyList()
	}
}

// pad2 pads a number string with "0" to two characters (O0ED:2B66).
func pad2(n int) string {
	s := strconv.Itoa(n & 0xFFFF)
	if len(s) > 2 {
		s = s[:2]
	}
	if len(s) < 2 {
		s = "0" + s
	}
	return s
}

// StatusLine = O0ED:2B66: "x,y dir hh:mm" (+ camping/search) under the
// party list.
func (e *Engine) StatusLine() {
	x, y := e.partyX(), e.partyY()
	hh := pad2(int(wordAt(e.area, 0x192)))
	mm := pad2(10*int(wordAt(e.area, 0x190)) + int(wordAt(e.area, 0x18E)))
	s := ""
	if wordAt(e.area, 0x1F6) == 0 {
		s = strconv.Itoa(x) + "," + strconv.Itoa(y) + " "
	}
	s += directionNames[e.viewFacing()] + " "
	s += hh + ":" + mm
	if e.gameState == 2 {
		s += " camping"
	} else if wordAt(e.game, 0x594)&1 != 0 {
		s += " search"
	}
	if len(s) > 0x28 {
		s = s[:0x28]
	}
	e.ClearCells(0x11, 0xF, 0x26, 0xF)
	e.DisplayString(0x11, 0xF, 0xA, s)
}
