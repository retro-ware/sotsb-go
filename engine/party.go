package engine

// Party menu (O0EF:01A1).

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

// freePlayers = O0EF:38A4 (keepCount, freeIcon): the selected member
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
		e.freeIcon(e.party[i].Order) // O195:014E
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
