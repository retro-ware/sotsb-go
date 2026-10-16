package engine

// The adventuring screen: the view frame (R133:0137), the party summary
// (O0ED:0798), the status line (O0ED:2B67), redrawing the view (O10D:00BA),
// the command bar (O099:08DF) and stepping through doors (O099).

// View frame layout.
const (
	viewRuleRow    = 16 // the rule under the view and the summary
	viewRuleCol    = 16 // the rule between the view and the summary
	viewBoxFirst   = 2  // the 3D view's own frame: rows and columns 2..14
	viewBoxLast    = 14
	viewFrameBase  = 276 // symbol of entry 0 of the view frame tables
	frameRuleBase  = 286 // symbol of entry 0 of the rule tables
	summaryTopRow  = 2
	summaryLeftCol = 17
	summaryRight   = 38
	statusLineRow  = 15
)

// drawAdventureFrame = R133:0137 draw8x8_03: the screen frame, the rules
// beside and under the view and the view's own frame.
func (e *Engine) drawAdventureFrame() {
	e.drawScreenFrame()
	for x := 0; x <= screenLastCol; x++ {
		e.Put8x8Symbol(x, viewRuleRow, int(frameMiddleRow[x])+frameRuleBase)
	}
	for y := 0; y <= viewRuleRow; y++ {
		e.Put8x8Symbol(viewRuleCol, y, int(viewDivider[y])+frameRuleBase)
	}
	for x := viewBoxFirst; x <= viewBoxLast; x++ {
		e.Put8x8Symbol(x, viewBoxFirst, int(viewFrameTop[x])+viewFrameBase)
		e.Put8x8Symbol(x, viewBoxLast, int(viewFrameBottom[x])+viewFrameBase)
	}
	for y := viewBoxFirst; y <= viewBoxLast; y++ {
		e.Put8x8Symbol(viewBoxFirst, y, int(viewFrameLeft[y])+viewFrameBase)
		e.Put8x8Symbol(viewBoxLast, y, int(viewFrameRight[y])+viewFrameBase)
	}
}

// clearViewArea is drawAdventureFrame under the name the scripts use.
func (e *Engine) clearViewArea() { e.drawAdventureFrame() }

// playerSummary = O0ED:0798 Player_Summary (selected): the party with AC
// and HP; the selected member's name in white.
func (e *Engine) playerSummary(sel *Character) {
	if e.gameState == stateWilderness {
		return
	}
	x := summaryLeftCol
	if e.gameState == stateStartMenu {
		x = 1
	}
	y := summaryTopRow
	e.displayStringBg(x, y, colourWhite, colourBlack, "Name")
	e.displayStringBg(acColumn, y, colourWhite, colourBlack, "AC  HP")
	y += 2
	for _, c := range e.party {
		e.ClearCells(x, y, summaryRight, y)
		if c == sel {
			e.displayStringBg(x, y, colourWhite, colourBlack, c.Name())
		} else {
			e.displayName(c, x, y, false)
		}
		dx := 0
		switch ac := c.AC[0]; {
		case ac <= 50:
			dx = 1
		case ac <= 60:
			dx = 2
		case ac <= 69:
			dx = 1
		}
		e.displayAC(c, acColumn-1+dx, y)
		dx = 0
		switch {
		case c.HP <= 9:
			dx = 2
		case c.HP <= 99:
			dx = 1
		}
		e.displayHP(c, hpColumn+dx, y, false)
		y++
	}
	e.ClearCells(x, y, summaryRight, y)
}

// Columns of the party summary.
const (
	acColumn = 33
	hpColumn = 36
)

// selectMember makes c the selected member (when it is still in the party).
func (e *Engine) selectMember(c *Character) {
	for i, p := range e.party {
		if p == c {
			e.partySel = i
			return
		}
	}
}

// freeSelected = O0A7:3A25 free_players (0, 1).
func (e *Engine) freeSelected() { e.freePlayers(false, true) }

// statusLine = O0ED:2B67 camping_search: position, facing, time and
// "camping" / "search" under the view.
func (e *Engine) statusLine() {
	if e.gameState == stateWilderness {
		return
	}
	s := &e.sc
	hours := itoa(int(wordAt(e.area, 2*areaWordHours)))
	if len(hours) < 2 {
		hours = "0" + hours
	}
	minutes := itoa(int(10*wordAt(e.area, 2*areaWordTensOfMinutes) + wordAt(e.area, 2*areaWordMinutes)))
	if len(minutes) < 2 {
		minutes = "0" + minutes
	}
	t := ""
	if wordAt(e.area, 2*areaWordNoMap) == 0 {
		t = itoa(int(s.posX)) + "," + itoa(int(s.posY)) + " "
	}
	t += directionNames[s.facing] + " "
	t += hours + ":" + minutes
	if e.debug {
		t += "*"
	}
	if e.gameState == stateCamp {
		t += " camping"
	} else if wordAt(e.game, 2*gameWordLook)&1 > 0 {
		t += " search"
	}
	e.ClearCells(summaryLeftCol, statusLineRow, summaryRight, statusLineRow)
	e.displayStringBg(summaryLeftCol, statusLineRow, colourGreen, colourBlack, cut(t, screenLastCol+1))
}

// Area words of the clock and the map.
const (
	areaWordMinutes       = 199
	areaWordTensOfMinutes = 200
	areaWordHours         = 201
	areaWordWallsetA      = 253
	areaWordWallsetB      = 254
	areaWordNoMap         = 251 // the overhead map is not allowed here
	areaWordMapShown      = 197
)

// mapPicture: the value of mapMode while a picture is up instead of the view.
const mapPicture = 'P'

// drawView = O10D:00BA: draw the 3D view (or the overhead map) in a
// dungeon, or the wilderness picture.
func (e *Engine) drawView() {
	s := &e.sc
	if e.mapMode == mapPicture {
		e.viewReset = false
	}
	if e.partyDead {
		return
	}
	if wordAt(e.area, 2*areaWordFreeSteps) != 0 {
		s.cellHere = e.MapCell(int(s.posX), int(s.posY))
		var sky byte
		if s.cellHere > 127 {
			sky = skyColours[wordAt(e.area, 2*areaWordWallsetB)%16]
		} else {
			sky = skyColours[wordAt(e.area, 2*areaWordWallsetA)%16]
		}
		horizon, band, ground := e.gs.sky[1], e.gs.sky[2], e.gs.sky[3]
		if e.videoMode == 0 {
			if s.redraw {
				horizon = lookHorizon(s.look2)
				band = lookBand(s.look2)
				ground = lookGround(s.look1)
			}
		} else {
			ground = 8
		}
		e.SetSkyColours(sky, horizon, band, ground)
		if wordAt(e.area, 2*areaWordNoMap) != 0 {
			e.gs.overhead = false
		}
		e.viewOrMap(int(s.posX), int(s.posY), int(s.facing))
	} else if e.viewReset {
		e.wildernessPicture()
	}
	e.viewReset = false
}

// lookHorizon, lookBand, lookGround = O10D:0014, 0048, 007C.
func lookHorizon(v byte) byte {
	if v == 8 || v == 'A' {
		return 0
	}
	return 15
}

func lookBand(v byte) byte {
	if v == 8 {
		return 8
	}
	return 7
}

func lookGround(v byte) byte {
	switch v {
	case 11:
		return 8
	case 6:
		return 1
	}
	return 6
}

// viewOrMap = O116:0820: the 3D view, or the overhead map when it is on.
func (e *Engine) viewOrMap(x, y, dir int) {
	e.Draw3DView(x, y, dir)
}

// wildernessPicture = O110:087A: the frame and the picture of the area.
func (e *Engine) wildernessPicture() {
	e.drawWildernessFrame()
	e.drawBigPic()
}

// gameDelay = R4CE:0B19: the message pause for the game speed.
func (e *Engine) gameDelay() { e.Delay(int(e.gameSpeed) * 100) }

// clearTreasure = O03C:1240's part: the treasure's money and items.
func (e *Engine) clearTreasure() {
	e.treasureCoins = [7]int32{}
	e.treasureItems = nil
}

// adventureInput = O099:08DF: the command bar in a dungeon; returns the key.
func (e *Engine) adventureInput() byte {
	s := &e.sc
	if f := e.DevAction; f != nil { // cmd/snap's scenes
		e.DevAction = nil
		f()
	}
	putWord(e.game, 2*gameWordSteps, 0)
	var k byte
	if e.gameState == stateDungeon {
		done := false
		for !done {
			var extra bool
			k, extra = e.DisplayInput("", commandBar, colourLightMagenta, colourGreen, colourWhite, true, false)
			if !extra {
				switch k {
				case 'A':
					if wordAt(e.area, 2*areaWordNoMap) == 0 || e.cheatArg {
						e.gs.overhead = !e.gs.overhead
						e.viewOrMap(int(s.posX), int(s.posY), int(s.facing))
					} else {
						e.flashMessage("Not Here", colourYellow, colourBlack)
					}
				case 'C':
					if e.selected().Status == 0 {
						e.menuSel = 1
						e.castMenu()
					}
				case 'V':
					e.menuSel = 1
					e.CharacterSheet()
				case 'E':
					done = true
					e.menuSel = 1
				case 'S':
					putWord(e.game, 2*gameWordLook, wordAt(e.game, 2*gameWordLook)^1)
				case 'L':
					putWord(e.game, 2*gameWordLook, wordAt(e.game, 2*gameWordLook)|2)
					e.AdvanceClock(1, clockMinute)
					s.pc = s.events[1]
					done = true
				}
			} else {
				switch k {
				case scanUp:
					e.stepAhead()
					done = true
				case scanDown:
					s.facing = (s.facing + 4) % 8
					s.wallHere = e.WallAt(int(s.posX), int(s.posY), int(s.facing))
					e.viewOrMap(int(s.posX), int(s.posY), int(s.facing))
				case scanLeft:
					s.facing = (s.facing + 6) % 8
					e.PlaySound(sndStep)
					s.wallHere = e.WallAt(int(s.posX), int(s.posY), int(s.facing))
					e.viewOrMap(int(s.posX), int(s.posY), int(s.facing))
				case scanRight:
					s.facing = (s.facing + 2) % 8
					e.PlaySound(sndStep)
					s.wallHere = e.WallAt(int(s.posX), int(s.posY), int(s.facing))
					e.viewOrMap(int(s.posX), int(s.posY), int(s.facing))
				default:
					e.cycleSelected(k)
					e.playerSummary(e.selected())
				}
			}
			e.statusLine()
		}
	}
	if !s.waitedKey {
		e.ClearCells(textLeft, textTop, textRight, textBottom)
		s.waitedKey = true
	}
	return k
}

// The dungeon command bar (DS:03D8).
const commandBar = "Area Cast View Encamp Search Look"

// gameWordSteps: cleared before each command; set by a step that left the map.
const gameWordSteps = 713

// stepAhead = O099:0765: through an open side the party would leave the
// map: clamp and flag it (the step itself is taken by lockedDoor).
func (e *Engine) stepAhead() {
	s := &e.sc
	x, y, d := int(int8(s.posX)), int(int8(s.posY)), int(s.facing)
	putWord(e.game, 2*gameWordSearchFlag, 0)
	if e.DoorAt(x, y, d) == 0 {
		return
	}
	x += int(stepX[d])
	y += int(stepY[d])
	if x > 15 {
		s.posX = 15
		putWord(e.game, 2*gameWordSearchFlag, 1)
	}
	if x < 0 {
		s.posX = 0
		putWord(e.game, 2*gameWordSearchFlag, 1)
	}
	if y > 15 {
		s.posY = 15
		putWord(e.game, 2*gameWordSearchFlag, 1)
	}
	if y < 0 {
		s.posY = 0
		putWord(e.game, 2*gameWordSearchFlag, 1)
	}
}

// Door states (O116:0573).
const (
	doorNone    = 0
	doorOpen    = 1
	doorLocked  = 2
	doorWizard  = 3
	pickSkill   = 6 // thief level index PICK checks for
	knockSpell  = 31
	maxStepCode = 255
)

// lockedDoor = O099:0B80: after a forward step command, walk through an
// open side, or offer Bash / Pick / Knock at a locked door.
func (e *Engine) lockedDoor() {
	s := &e.sc
	move := false
	if e.gameState == stateDungeon {
		if wordAt(e.game, 2*gameWordSteps) < maxStepCode {
			e.viewReset = true
			x, y, d := int(s.posX), int(s.posY), int(s.facing)
			switch door := e.DoorAt(x, y, d); door {
			case doorOpen:
				move = true
			case doorLocked, doorWizard:
				menu := ""
				if e.canBash {
					menu = "Bash"
				}
				if e.canPick && e.partyHasClassLevel(pickSkill) {
					menu += " Pick"
				}
				if e.canKnock && e.memorizedBy(knockSpell) != nil {
					menu += " Knock"
				}
				menu += " Exit"
				if menu != " Exit" {
					k, _ := e.DisplayInput("Locked. ", menu, colourLightMagenta, colourGreen, colourWhite, false, false)
					switch k {
					case 'B':
						move = e.bashDoor()
					case 'P':
						if door == doorLocked {
							move = e.pickLock()
						} else {
							e.canPick = false
						}
					case 'K':
						move = e.castKnock()
					}
				}
			}
			if move {
				e.walkForward()
			}
			e.statusLine()
		} else {
			putWord(e.game, 2*gameWordEncDist+1, 0)
		}
	}
	e.restoreBackground()
	e.port = portrait{headID: noPicture, bodyID: noPicture}
}

// partyHasClassLevel = O099:0275 (class): someone has levels in the class
// (or usable old levels).
func (e *Engine) partyHasClassLevel(class int) bool {
	for _, c := range e.party {
		if int8(c.Levels[class]) > 0 || int8(c.OldLevels[class]) > 0 && oldClassUsable(c) {
			return true
		}
	}
	return false
}

// memorizedSlot = O099:0683 (spell, c): the memorized slot holding it, or 255.
func memorizedSlot(spell byte, c *Character) byte {
	for i := byte(0); i < byte(len(c.Memorized)); i++ {
		if c.Memorized[i] == spell {
			return i
		}
	}
	return 255
}

// memorizedBy = O099:06C6 (spell): the first member with it memorized.
func (e *Engine) memorizedBy(spell byte) *Character {
	for _, c := range e.party {
		if memorizedSlot(spell, c) != 255 {
			return c
		}
	}
	return nil
}

// castKnock = O099:0720 find_knock_spell: spend a memorized Knock.
func (e *Engine) castKnock() bool {
	if c := e.memorizedBy(knockSpell); c != nil {
		c.Memorized[memorizedSlot(knockSpell, c)] = 0
		return true
	}
	return false
}

// openDoor = O099:0148 (x, y, dir): set the door bits of that side open.
func (e *Engine) openDoor(x, y, dir int) {
	if x < 0 || x > 15 || y < 0 || y > 15 {
		return
	}
	i := doorPlane + y*16 + x
	switch dir {
	case 6:
		e.gs.geo[i] = e.gs.geo[i]&63 | 64
	case 4:
		e.gs.geo[i] = e.gs.geo[i]&207 | 16
	case 2:
		e.gs.geo[i] = e.gs.geo[i]&243 | 4
	case 0:
		e.gs.geo[i] = e.gs.geo[i]&252 | 1
	}
}

// doorPlane: the map's fourth 256-byte plane holds the door bits.
const doorPlane = 768

// openBothSides opens the door ahead and its other side.
func (e *Engine) openBothSides() {
	s := &e.sc
	x, y, d := int(int8(s.posX)), int(int8(s.posY)), int(s.facing)
	e.openDoor(x, y, d)
	e.openDoor(x+int(stepX[d]), y+int(stepY[d]), (d+4)%8)
}

// bashDoor = O099:02F3: each member tries by strength until one succeeds.
func (e *Engine) bashDoor() bool {
	s := &e.sc
	ok := false
	for _, c := range e.party {
		if ok {
			break
		}
		str, str00 := c.Stats[statStr].Value, c.Str00.Base
		if e.DoorAt(int(s.posX), int(s.posY), int(s.facing)) == doorWizard {
			switch str {
			case 18:
				switch {
				case str00 >= 91 && str00 <= 99:
					ok = e.Dice(1, 6) == 1
				case str00 == 100:
					ok = e.Dice(1, 6) <= 2
				default:
					e.canBash = false
				}
			case 19, 20:
				ok = e.Dice(1, 6) <= 3
			case 21, 22:
				ok = e.Dice(1, 6) <= 4
			case 23:
				ok = e.Dice(1, 6) <= 5
			case 24:
				ok = e.Dice(1, 8) <= 7
			case 25:
				ok = true
			default:
				e.canBash = false
			}
			continue
		}
		switch {
		case str >= 3 && str <= 7:
			ok = e.Dice(1, 6) == 1
		case str >= 8 && str <= 15:
			ok = e.Dice(1, 6) <= 2
		case str >= 16 && str <= 17:
			ok = e.Dice(1, 6) <= 3
		case str == 18:
			switch {
			case str00 <= 50:
				ok = e.Dice(1, 6) <= 3
			case str00 <= 99:
				ok = e.Dice(1, 6) <= 4
			case str00 == 100:
				ok = e.Dice(1, 6) <= 5
			}
		case str == 19 || str == 20:
			ok = e.Dice(1, 8) <= 7
		case str == 21:
			ok = e.Dice(1, 10) <= 9
		case str == 22 || str == 23:
			ok = e.Dice(1, 12) <= 11
		case str == 24:
			ok = e.Dice(1, 20) <= 19
		case str == 25:
			ok = true
		}
	}
	if ok {
		e.openBothSides()
	}
	return ok
}

// pickLock = O099:05B6: a d100 under someone's open-locks skill.
func (e *Engine) pickLock() bool {
	ok := false
	for _, c := range e.party {
		if ok {
			break
		}
		if byte(e.Dice(1, 100)) <= c.Thief[1] && c.Status == 0 {
			ok = true
		}
	}
	e.canPick = false
	if ok {
		e.openBothSides()
	}
	return ok
}

// walkForward = O099:0813: one step ahead, wrapping at the map's edges,
// then the clock (twice as long while searching).
func (e *Engine) walkForward() {
	s := &e.sc
	e.PlaySound(sndStep)
	e.Delay(stepDelay)
	x := int(int8(s.posX)) + int(stepX[s.facing])
	y := int(int8(s.posY)) + int(stepY[s.facing])
	if x < 0 {
		x = 15
	}
	if x > 15 {
		x = 0
	}
	if y < 0 {
		y = 15
	}
	if y > 15 {
		y = 0
	}
	s.posX, s.posY = byte(x), byte(y)
	s.wallHere = e.WallAt(x, y, int(s.facing))
	e.canBash, e.canPick, e.canKnock = true, true, true
	s.cellHere = e.MapCell(x, y)
	if wordAt(e.game, 2*gameWordLook)&1 > 0 {
		e.AdvanceClock(1, clockTenMinutes)
		return
	}
	e.AdvanceClock(1, clockMinute)
}

const stepDelay = 50

// Clock units of AdvanceClock.
const (
	clockMinute     = 1
	clockTenMinutes = 2
)

// keyEnter is the Enter key's code.
const keyEnter = 13

// flashMessage = R4CE:0ABF (text, fg, bg): the text on the bottom row for
// a moment.
func (e *Engine) flashMessage(text string, fg, bg int) {
	e.ClearRow24()
	e.displayStringBg(0, statusRow, fg, bg, text)
	e.gameDelay()
	e.ClearRow24()
}

// campEvents = O03C:3094: the camp menu between the area's camp events.
func (e *Engine) campEvents() {
	e.runEvent(e.sc.events[2])
	ok := e.makeCamp()
	if ok {
		e.loadPicture()
		e.runEvent(e.sc.events[3])
	}
	e.viewReset = true
	e.drawView()
	e.textWaiting = false
}

// makeCamp: the camp menu; true when an encounter interrupted the rest.
func (e *Engine) makeCamp() bool { return e.campMenu() }

// The EGA sky pictures (SKY.DAX 250..252, masked on colour 13).
const (
	skySunBlock   = 250
	skyMoonBlock  = 251
	skyFloorBlock = 252
	skyMask       = 13
	skyDay        = 11 // the sky colour of daytime (cyan)
	skyFloorX     = 2
	skyFloorY     = 7
)

// loadSkyPictures = O060:0054's part for EGA.
func (e *Engine) loadSkyPictures() {
	e.skySun = e.LoadPicMasked("sky", skySunBlock, skyMask)
	e.skyMoon = e.LoadPicMasked("sky", skyMoonBlock, skyMask)
	e.skyFloor = e.LoadPicMasked("sky", skyFloorBlock, skyMask)
}

// drawSkyEGA = O116:0184 in EGA: three colour bands, the sun or moon by
// the hour (outdoors-lit squares in daylight), then the floor picture.
func (e *Engine) drawSkyEGA() {
	s := &e.sc
	sky, band, ground := e.gs.sky[0], e.gs.sky[2], e.gs.sky[3]
	e.FillRect(viewX-1, skyTop, skyW, skyRows, sky)
	e.FillRect(viewX-1, skyBandTop, skyW, skyBandRows, band)
	e.FillRect(viewX-1, groundTop, skyW, groundRows, ground)
	if e.MapCell(int(s.posX), int(s.posY)) < 128 && sky == skyDay {
		const left, top, right = 2, 2, 12
		h := int(wordAt(e.area, 2*areaWordHours))
		switch {
		case h >= 1 && h <= 5:
			switch s.facing {
			case dirEast:
				e.drawPage(right-3, top+5-h, e.skyMoon, 0, true, false)
			case dirSouth:
				if h > 2 {
					e.drawPage(left+h-3, top+5-h, e.skyMoon, 0, true, false)
				}
			}
		case h >= 13 && h <= 18:
			switch s.facing {
			case dirWest:
				e.drawPage(left, top+h-13, e.skyMoon, 0, true, false)
			case dirSouth:
				if h < 16 {
					e.drawPage(left+h-8, top+h-13, e.skyMoon, 0, true, false)
				}
			}
		}
		if s.facing == dirNorth {
			e.drawPage(left, top, e.skySun, 0, true, false)
		}
	}
	e.drawPage(skyFloorX, skyFloorY, e.skyFloor, 0, true, false)
}

// The view's colour bands (page rows).
const (
	skyW        = 11
	skyTop      = 16
	skyRows     = 44
	skyBandTop  = 60
	skyBandRows = 2
	groundTop   = 62
	groundRows  = 42
)

// displayAndWait = R4CE:094D displayAndDebug (fg, bg, text): the text on
// the bottom row, then a key (debug_txt; in the demo only a waiting key
// is taken).
func (e *Engine) displayAndWait(fg, bg int, text string) {
	e.ClearRow24()
	e.displayStringBg(0, statusRow, fg, bg, cut(text, screenLastCol+1))
	if e.demo {
		if e.KeyPressed() {
			e.ReadKeyByte()
		}
		return
	}
	e.ReadKeyByte()
}
