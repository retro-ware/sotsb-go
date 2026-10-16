package engine

// Party menu, ported from Pool ovr17:0161 and the list display O0ED:07A2.

// DisplayPartyList = O0ED:07A2 (selected): the roster with AC and HP.
// While adventuring with the minimap on (extras.go) the panel shows the
// map instead.
func (e *Engine) DisplayPartyList() {
	if e.gameState != 0 && e.ext.minimap && e.minimapAllowed() {
		e.DrawMinimap()
		return
	}
	e.drawPartyList()
}

func (e *Engine) drawPartyList() {
	x := 1
	if e.gameState != 0 {
		x = 0x11
	}
	y := 2
	e.DisplayString(x, y, 0xF, "Name")
	e.DisplayString(0x21, y, 0xF, "AC  HP")
	y += 2
	hi := e.partySel
	if e.rosterNone {
		hi = -1
	} else if e.rosterHi != nil {
		hi = e.partyIndex(e.rosterHi)
	}
	for i, c := range e.party {
		e.ClearCells(x, y, 0x26, y)
		if i == hi && e.ext.choosing {
			// extra: the member being chosen gets the menu selection background
			for j := 0; j < len(c.Name()); j++ {
				e.displayCharBg(x+j, y, 0xF, selBg(0xB, 0xF), c.Name()[j], true)
			}
		} else if i == hi {
			e.DisplayString(x, y, 0xF, c.Name())
		} else {
			e.displayName(c, x, y, false)
		}
		ac := c.AC[0]
		dx := 0
		switch {
		case ac <= 0x32, ac >= 0x3D && ac <= 0x45:
			dx = 1
		case ac >= 0x33 && ac <= 0x3C:
			dx = 2
		}
		e.displayAC(c, 0x20+dx, y)
		hp := c.HP
		dx = 0
		switch {
		case hp <= 9:
			dx = 2
		case hp <= 0x63:
			dx = 1
		}
		e.displayHP(c, 0x24+dx, y, false)
		y++
	}
	e.ClearCells(x, y, 0x26, y)
}

// partyKeypad = O0AF:32CF (key): Home selects the previous character
// (wrapping to the last), End the next (wrapping to the first: the original
// leaves its walker on the head when there is no next), anything else the first.
func (e *Engine) partyKeypad(k byte) {
	if len(e.party) == 0 {
		return
	}
	switch k {
	case 'G':
		if e.partySel == 0 {
			e.partySel = len(e.party) - 1
		} else {
			e.partySel--
		}
	case 'O':
		if e.partySel+1 < len(e.party) {
			e.partySel++
		} else {
			e.partySel = 0
		}
	default:
		e.partySel = 0
	}
}

// The party menu's entries (DS 1363, 42-byte records: name, enabled flag).
// The first letter of each name is its key.
var partyMenuEntries = [...]string{
	"Create New Character",
	"Drop Character",
	"Modify Character",
	"Train Character",
	"Human Change Classes",
	"View Character",
	"Add Character to Party",
	"Remove Character from Party",
	"Load Saved Game",
	"Save Current Game",
	"BEGIN Adventuring",
	"Exit to DOS",
}

const (
	menuCreate = iota
	menuDrop
	menuModify
	menuTrain
	menuChangeClass
	menuView
	menuAdd
	menuRemove
	menuLoad
	menuSave
	menuBegin
	menuExit
)

// The party menu's own rows: the function list starts at row 12, column 2.
const (
	partyMenuListRow = 12
	partyMenuListCol = 2
)

// PartyMenu = O0A7:01CD startGameMenu. Returns when the party begins
// adventuring (or a loaded game starts play).
func (e *Engine) PartyMenu() {
	e.symDAX = "8X8D" + itoa(int(e.gameArea))
	stateBefore := e.gameState
	e.gameState = stateStartMenu
	var enabled [len(partyMenuEntries)]bool
	enabled[menuCreate], enabled[menuAdd], enabled[menuLoad], enabled[menuExit] = true, true, true, true
	redraw := true
	e.freeTraining = false
	for {
		if redraw {
			e.drawScreenFrame()
			if len(e.party) > 0 {
				e.DisplayPartyList()
				enabled[menuDrop], enabled[menuModify] = true, true
				if e.trainingClasses() > 0 || e.freeTraining {
					enabled[menuTrain] = true // like the original, never cleared while there is a party
				}
				enabled[menuChangeClass] = (e.trainingClasses() > 0 || e.freeTraining) && e.mayChangeClass(e.selected())
				enabled[menuView], enabled[menuRemove] = true, true
				enabled[menuLoad] = false
				enabled[menuSave], enabled[menuBegin] = true, true
			} else {
				for _, m := range []int{menuDrop, menuModify, menuTrain, menuChangeClass, menuView, menuRemove, menuSave, menuBegin} {
					enabled[m] = false
				}
				enabled[menuLoad] = true
			}
			row := 0
			for i, name := range partyMenuEntries {
				if enabled[i] {
					e.DisplayString(partyMenuListCol, partyMenuListRow+row, colourWhite, name[:1])
					e.DisplayString(partyMenuListCol+1, partyMenuListRow+row, colourGreen, name[1:])
					row++
				}
			}
			redraw = false
		}
		// The menu bar itself is hidden (colours 0): the list above shows
		// the functions, the keys of disabled ones do nothing.
		k, extra := e.DisplayInput("Choose a function ", "C D M T H V A R L S B E J", colourLightMagenta, colourBlack, colourBlack, true, false)
		e.ClearRow24()
		if extra {
			if k == 'G' || k == 'O' { // keypad up/down: select another party member
				couldChange := e.mayChangeClass(e.selected())
				e.partyKeypad(k)
				e.DisplayPartyList()
				redraw = e.mayChangeClass(e.selected()) != couldChange && e.trainingClasses() > 0
			}
			continue
		}
		if k != 'E' && k != 'S' {
			e.saved = false
		}
		switch k {
		case 'C':
			if enabled[menuCreate] {
				e.CreateCharacter()
			}
		case 'D':
			if enabled[menuDrop] {
				e.DropCharacter()
			}
		case 'M':
			if enabled[menuModify] {
				e.ModifyCharacter()
			}
		case 'T':
			if enabled[menuTrain] {
				e.TrainCharacter()
			}
		case 'H':
			if enabled[menuChangeClass] {
				e.ChangeClass(e.selected())
			}
		case 'V':
			if enabled[menuView] {
				e.viewPlayer()
			}
		case 'A':
			if enabled[menuAdd] {
				e.AddCharacter()
			}
		case 'R':
			if enabled[menuRemove] && len(e.party) > 0 {
				if c := e.selected(); !c.NPC() {
					e.SaveCharacter(c, "")
					e.freePlayers(false, true)
				} else {
					e.DropCharacter() // non-player characters can only be dropped
				}
			}
		case 'L':
			if enabled[menuLoad] {
				e.LoadGame()
				if e.gameState != 0 {
					e.loadedGame = true
					return
				}
			}
		case 'S':
			if enabled[menuSave] && len(e.party) > 0 {
				e.SaveGame()
			}
		case 'B':
			if enabled[menuBegin] && (len(e.party) > 0 && !e.gameFinished() || e.demo) {
				e.gameState = stateBefore
				if !e.loadedGame && e.mapMode != mapPicture {
					if e.gameState == stateWilderness {
						e.drawWildernessFrame()
					} else {
						e.drawAdventureFrame()
					}
					e.playerSummary(e.selected())
				} else if wordAt(e.area, 2*areaWordLastECL) == 0 {
					e.drawAdventureFrame()
				}
				e.ClearRow24()
				putWord(e.game, trainingMaskByte, 0)
				return
			}
		case 'E':
			if enabled[menuExit] && e.YesNo(colourYellow, colourGreen, colourWhite, "Quit to DOS ") == 'Y' {
				if len(e.party) > 0 && !e.saved &&
					e.YesNo(colourYellow, colourGreen, colourWhite, "Game not saved.  Quit anyway? ") == 'N' {
					e.SaveGame()
				}
				e.Exit()
			}
		}
		redraw = true
	}
}

// SelectCharacter = O0ED:2E23 (prompt, exit, var who): pick a party member
// with Next/Prev (keypad) and Select; Exit (offered when exit is set)
// gives nil. The roster highlights the current choice.
func (e *Engine) SelectCharacter(prompt string, exit bool, who *Character) *Character {
	prompt = cut(prompt, 0x28)
	menu := "Select"
	if exit {
		menu += " Exit"
	}
	index := func(c *Character) int {
		for i, p := range e.party {
			if p == c {
				return i
			}
		}
		return -1
	}
	if who == nil && len(e.party) > 0 {
		who = e.party[0]
	}
	e.ext.choosing = true
	defer func() {
		e.ext.choosing = false
		e.DisplayPartyList() // (the original leaves the roster as drawn)
	}()
	for {
		// who stays highlighted while the menu waits (the view animation
		// redraws the roster).
		e.rosterHi, e.rosterNone = who, who == nil
		e.DisplayPartyList() // O0ED:07A2 (who)
		mark := e.regionMark()
		{ // extra (nav.go): click a member to select it
			e.addRegion(region{x0: 0, y0: 0, x1: 0x27, y1: 0x17, click: func(cx, cy int) uint16 {
				if i := e.rosterRow(cx, cy); i >= 0 {
					who = e.party[i]
					return 'S'
				}
				return 0
			}})
		}
		key, extra := e.DisplayInput(prompt+" ", menu, 0xD, 0xA, 0xF, true, e.gameState == 2 || e.gameState == 6) // O0ED:2E23 at 3844
		e.regionCut(mark)
		e.rosterHi, e.rosterNone = nil, false
		// The exit set (DS O0ED:2DFA) is checked before the extra flag.
		switch key {
		case 13, 'S', 0x1B:
			return who
		case 'E':
			if !extra {
				return nil
			}
			return who
		}
		if extra {
			i := index(who)
			switch key {
			case 'O', 'P': // next, wrapping to the first (P = cursor down, an extra)
				if who == nil {
					// nil^.next reads 0000:0104 (the INT 41h vector, garbage in
					// the original); the harness pins it to nil, and nil wraps to
					// the first member (O0ED:2F1B)
					who = e.party[0]
				} else if i+1 < len(e.party) {
					who = e.party[i+1]
				} else {
					who = e.party[0]
				}
			case 'G', 'H': // previous, wrapping to the last (H = cursor up, an extra)
				if i > 0 {
					who = e.party[i-1]
				} else {
					who = e.party[len(e.party)-1]
				}
			}
			if e.whoVar && who != nil { // opWho's var is DS:5D92 itself
				e.selectChar(who)
			}
			continue
		}
		if key == 0 { // (Esc) the original stores nil and keeps asking
			who = nil
		}
	}
}

// freePlayers = O0A7:3A25 (keepCount, freeIcon): the selected member
// leaves the party list; the one before it (or the new first) becomes
// selected. Unless keepCount, the party count (game +0x67C) drops.
func (e *Engine) freePlayers(keepCount, freeIcon bool) {
	if len(e.party) == 0 {
		return
	}
	i := e.partySel
	if i >= len(e.party) {
		return
	}
	if freeIcon {
		e.freeIcon(e.party[i].Order) // O12A:0156
	}
	e.party = append(e.party[:i], e.party[i+1:]...)
	if i > 0 {
		e.partySel = i - 1
	} else {
		e.partySel = 0
	}
	if !keepCount {
		putWord(e.game, gsPartyCount, wordAt(e.game, gsPartyCount)-1)
	}
}
