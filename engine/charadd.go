package engine

import (
	"os"
	"strings"
)

// Adding saved characters to the party (units O0A7 / O0A1 of the
// reconstruction).

// Where AddPlayer loads characters from (b759E).
const (
	fromCurse    = 0
	fromPool     = 1
	fromHillsfar = 2
)

// Party limits.
const (
	maxPlayerCharacters = 6
	maxPartyMembers     = 8
	maxRangers          = 3
	noOrder             = 255
)

// rosterEntry is one line of the "Add a character" list: the shown name and
// the file it comes from.
type rosterEntry struct {
	text, file string
}

func filepathExt(name string) string {
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		return name[i:]
	}
	return ""
}

// loadSavedCharacter = O0A1:22A0 import_char01 for the Curse source: the
// record, its items (.SWG) and affects (.FX); then the items and class values
// are worked out again.
func (e *Engine) loadSavedCharacter(file string, c *Character) bool {
	e.DisplayString(0, statusRow, colourGreen, "Loading...Please Wait")
	b, err := os.ReadFile(e.savePath(file))
	if err != nil {
		e.fileError("read", e.savePath(file))
		return false
	}
	c.setRecord(b)
	c.Items, c.Affects = nil, nil
	base := strings.TrimSuffix(file, filepathExt(file))
	if b, err := os.ReadFile(e.savePath(base + extItems)); err == nil {
		c.Items = readRecords[Item](b, itemRecSize)
	}
	if b, err := os.ReadFile(e.savePath(base + extAffects)); err == nil {
		c.Affects = readAffects(b)
	}
	e.ClearKeyboard()
	e.ApplyItems(c)
	e.recalcClassBonuses(c)
	return true
}

// AddToParty = O0A1:360A (c): append to the party, select it, give it the
// lowest free icon order 0..7 and count it; hired NPCs get their class values
// worked out.
func (e *Engine) AddToParty(c *Character) {
	c.Order = noOrder
	e.party = append(e.party, c)
	e.partySel = len(e.party) - 1
	var used [maxPartyMembers]bool
	for _, p := range e.party {
		if p.Order < maxPartyMembers {
			used[p.Order] = true
		}
	}
	c.Order = 0
	for c.Order < maxPartyMembers && used[c.Order] {
		c.Order++
	}
	e.setPartyCount(e.partyCount() + 1)
	if c.NPC() {
		e.recalcClassBonuses(c)
	}
}

// AddCharacter = O0A7:3648 AddPlayer: pick saved characters to join the
// party, with the party's rules: at most six player characters and eight in
// all, a paladin does not join evil company, no more than three rangers, no
// evil character joins a paladin.
func (e *Engine) AddCharacter() {
	var evil, paladin bool
	var paladinName string
	rangers := 0
	e.ClearCells(1, 1, 38, 22)
	k, _ := e.DisplayInput("Add from where? ", "Curse Pool Hillsfar Exit", colourLightMagenta, colourGreen, colourWhite, false, false)
	switch k {
	case 'C':
		e.addSource = fromCurse
	case 'P':
		e.addSource = fromPool
	case 'H':
		e.addSource = fromHillsfar
	case 'E', 0:
		return
	}
	roster := e.rosterFiles()
	if len(roster) == 0 {
		return
	}
	list := make([]*ListNode, len(roster))
	for i, r := range roster {
		list[i] = &ListNode{Text: r.text}
	}
	index, redraw := 0, true
	var sel *ListNode
	players := 0
	for {
		key := e.ListMenu("Add a character: ", "Add ", colourLightMagenta, colourGreen, colourWhite, 1, 2, 38, 22, list, true, &redraw, &index, &sel)
		if (key == 13 || key == 'A') && sel != nil && sel.Text[0] != '*' {
			e.ClearRow24()
			c := &Character{}
			if e.addSource == fromCurse {
				e.loadSavedCharacter(roster[index].file, c)
			} else {
				e.importCharacter(roster[index].file, c)
			}
			sel.Text = "* " + sel.Text
			players = 0
			if len(e.party) == 0 {
				e.setPartyCount(0)
				e.AddToParty(c)
				e.BuildCharIcon(c, true)
			} else {
				rangers = 0
				dup := false
				for _, p := range e.party {
					if p.Name() == c.Name() && p.RandomID == c.RandomID {
						dup = true
						break
					}
					if !p.NPC() {
						players++
					}
					if int8(p.Levels[clsRanger]) > 0 {
						rangers++
					}
					if isEvil(p) {
						evil = true
					}
					if int8(p.Levels[clsPaladin]) > 0 {
						paladin, paladinName = true, p.Name()
					}
				}
				ok := !dup && (!c.NPC() && players < maxPlayerCharacters || c.NPC() && e.partyCount() < maxPartyMembers) &&
					(c.Levels[clsPaladin] == 0 || !evil) && (c.Levels[clsRanger] == 0 || rangers < maxRangers) &&
					(!isEvil(c) || !paladin)
				if ok {
					e.AddToParty(c)
					e.BuildCharIcon(c, true)
					if !c.NPC() {
						players++
					}
				} else {
					sel.Text = sel.Text[2:]
					switch {
					case int8(c.Levels[clsPaladin]) > 0 && evil:
						e.printMessage("paladins do not join with evil scum")
						e.pauseForMessage()
					case int8(c.Levels[clsRanger]) > 0 && rangers > maxRangers-1:
						e.printMessage("too many rangers in party")
					case isEvil(c) && paladin:
						e.printMessage(paladinName + " will tolerate no evil!")
					}
				}
			}
		}
		if key == 'E' || key == 0 || players > maxPlayerCharacters-1 || e.partyCount() > maxPartyMembers-1 {
			return
		}
	}
}

// isEvil: lawful, neutral or chaotic evil ((alignment+1) mod 3 = 0, the
// alignment taken as a 0..127 value).
func isEvil(c *Character) bool { return (c.Alignment&127+1)%3 == 0 }
