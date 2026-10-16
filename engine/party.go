package engine

// Party menu (O0EF:01A1) and the list display O0ED:07A2.

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
		e.displayHP(c, 0x24+dx, y, false, false)
		y++
	}
	e.ClearCells(x, y, 0x26, y)
}

// The party menu's entries (DS 06C0, 42-byte records: name, enabled flag).
// Picking one gives its first letter.
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
	"Begin Adventuring",
	"Initialize Mouse/Joystick",
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
	menuInitInput
	menuExit
)

// The party menu's list window (text cells, edges included).
const (
	partyMenuLeft, partyMenuTop     = 9, 12
	partyMenuRight, partyMenuBottom = 38, 22
)

// PartyMenu = O0EF:01A1. Returns when the party begins adventuring.
func (e *Engine) PartyMenu() {
	stateBefore := e.gameState
	e.gameState = stateStartMenu
	enabled := [len(partyMenuEntries)]bool{menuCreate: true, menuAdd: true, menuLoad: true, menuInitInput: true, menuExit: true}
	e.freeTraining = false
	index := 0
	for {
		e.drawScreenFrame()
		party := len(e.party) > 0
		if party {
			e.playerSummary(e.selected())
		}
		training := e.trainingClasses() > 0 || e.freeTraining
		enabled[menuDrop], enabled[menuModify] = party, party
		enabled[menuTrain], enabled[menuChangeClass] = party && training, party && training
		enabled[menuView], enabled[menuRemove], enabled[menuSave], enabled[menuBegin] = party, party, party, party
		enabled[menuLoad], enabled[menuInitInput] = !party, !party
		var list []*ListNode
		for i, name := range partyMenuEntries {
			if enabled[i] {
				list = append(list, &ListNode{Text: name})
			}
		}
		redraw := true
		var sel *ListNode
		e.ListMenu("Choose a FUNCTION ", "Select ", colourLightMagenta, colourGreen, colourWhite,
			partyMenuLeft, partyMenuTop, partyMenuRight, partyMenuBottom, list, false, &redraw, &index, &sel)
		if sel == nil {
			continue
		}
		k := sel.Text[0]
		if k != 'E' && k != 'S' {
			e.saved = false
		}
		switch k {
		case 'C':
			e.CreateCharacter()
		case 'D':
			e.DropCharacter()
		case 'H':
			if e.pickCharacter() {
				c := e.selected()
				if class := e.ChangeClass(c); class != noClass {
					// the old class's protections go; a paladin or ranger gets its own
					for _, typ := range []byte{affectProtFromEvil, affectRangerVsGiants} {
						if e.FindAffect(c, typ) != nil {
							e.RemoveAffect(c, typ, nil)
						}
					}
					switch class {
					case clsPaladin:
						e.AddAffect(c, affectProtFromEvil, 0, permanent, 0)
					case clsRanger:
						e.AddAffect(c, affectRangerVsGiants, 0, permanent, 0)
					}
				}
			}
		case 'M':
			e.ModifyCharacter()
		case 'T':
			e.TrainCharacter()
		case 'V':
			if e.pickCharacter() {
				e.viewPlayer()
			}
		case 'A':
			e.AddCharacter()
		case 'R':
			if e.pickCharacter() {
				if c := e.selected(); !c.NPC() {
					e.SaveCharacter(c, "")
					e.freePlayers(false, true)
				} else {
					e.DropCharacter() // non-player characters can only be dropped
				}
			}
		case 'L':
			e.LoadGame()
		case 'S':
			e.SaveGame()
		case 'B':
			if !party && !e.demo {
				break
			}
			e.gameState = stateBefore
			if !e.loadedGame || e.mapMode == mapPicture {
				if e.gameState == stateWilderness && wordAt(e.area, 2*areaWordPartyOnMap) == 0 {
					e.drawTextWindowFrame()
				} else {
					e.drawAdventureFrame()
				}
				e.playerSummary(e.selected())
			} else if wordAt(e.area, 2*areaWordLastECL) == 0 || wordAt(e.area, 2*areaWordPartyOnMap) != 0 {
				e.drawAdventureFrame()
				e.playerSummary(e.selected())
			}
			e.ClearRow24()
			putWord(e.game, trainingMaskByte, 0)
			return
		case 'I':
			e.initMouseJoystick()
		case 'E':
			if e.yesNoFrom(answerNo, colourYellow, colourGreen, colourWhite, "Quit to DOS ") == 'Y' {
				if party && !e.saved &&
					e.yesNoFrom(answerNo, colourYellow, colourGreen, colourWhite, "Game NOT saved.  Quit anyway? ") == 'N' {
					e.SaveGame()
				}
				e.Exit()
			}
		}
	}
}

// initMouseJoystick = O0EF:5508: switch the mouse or the joystick on or
// off and keep the choice as the input line of BLADES.CFG.
// ponytail: only the setting is kept; the port reads mouse and gamepad either way, no joystick calibration.
func (e *Engine) initMouseJoystick() {
	index := 0
	for {
		state := func(on bool) string {
			if on {
				return "ON"
			}
			return "OFF"
		}
		list := []*ListNode{
			{Text: "mouse " + state(e.Cfg.Input == 'M')},
			{Text: "joystick " + state(e.Cfg.Input == 'J')},
			{Text: "disable both"},
		}
		redraw := true
		var sel *ListNode
		if e.ListMenu("Init Mouse/Joystick: ", "Select", colourLightMagenta, colourGreen, colourWhite,
			partyMenuLeft, partyMenuTop, partyMenuRight, partyMenuBottom, list, true, &redraw, &index, &sel) != 'S' {
			return
		}
		switch {
		case index == 0 && e.Cfg.Input != 'M':
			e.Cfg.Input = 'M'
		case index == 1 && e.Cfg.Input != 'J':
			e.Cfg.Input = 'J'
		default:
			e.Cfg.Input = 'N'
		}
		e.writeConfig()
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
