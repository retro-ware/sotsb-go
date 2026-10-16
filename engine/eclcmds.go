package engine

import "strconv"

// The script commands of unit O03C, in command order. Each reads its own
// operands (readArgs) and leaves pc on the next command.

// Command numbers (the first byte of each command).
const (
	cmdExit        = 0
	cmdGoto        = 1
	cmdGosub       = 2
	cmdCompare     = 3
	cmdAdd         = 4
	cmdSubtract    = 5
	cmdDivide      = 6
	cmdMultiply    = 7
	cmdRandom      = 8
	cmdSave        = 9
	cmdLoadChar    = 10
	cmdLoadMonster = 11
	cmdSetupMon    = 12
	cmdApproach    = 13
	cmdPicture     = 14
	cmdInputNumber = 15
	cmdInputText   = 16
	cmdPrint       = 17
	cmdPrintClear  = 18
	cmdReturn      = 19
	cmdCompareAnd  = 20
	cmdVMenu       = 21
	cmdIfEqual     = 22 // .. 27: skip the next command unless the flag is set
	cmdIfLast      = 27
	cmdClearMon    = 28
	cmdPartyStr    = 29
	cmdCheckParty  = 30
	cmdNewECL      = 32
	cmdLoadFiles   = 33
	cmdPartySurp   = 34
	cmdSurprise    = 35
	cmdCombat      = 36
	cmdOnGoto      = 37
	cmdOnGosub     = 38
	cmdTreasure    = 39
	cmdRob         = 40
	cmdEncounter   = 41
	cmdGetTable    = 42
	cmdHMenu       = 43
	cmdParlay      = 44
	cmdCall        = 45
	cmdDamage      = 46
	cmdAnd         = 47
	cmdOr          = 48
	cmdSpriteOff   = 49
	cmdFindItem    = 50
	cmdPrintReturn = 51
	cmdClock       = 52
	cmdSaveTable   = 53
	cmdAddNPC      = 54
	cmdLoadPieces  = 55
	cmdProgram     = 56
	cmdWho         = 57
	cmdDelay       = 58
	cmdSpell       = 59
	cmdProtection  = 60
	cmdClearBox    = 61
	cmdDump        = 62
	cmdFindSpecial = 63
	cmdDestroy     = 64
)

// The text window the scripts print in.
const (
	textLeft   = 1
	textTop    = 17
	textRight  = 38
	textBottom = 22
)

// commandTable = O03C:32F9: run the current command.
func (e *Engine) commandTable() {
	if h := scriptCommands[e.sc.command]; h != nil {
		h(e)
	}
}

var scriptCommands [cmdDestroy + 1]func(*Engine)

func init() {
	c := &scriptCommands
	c[cmdExit] = (*Engine).cmdExit
	c[cmdGoto] = (*Engine).cmdGoto
	c[cmdGosub] = (*Engine).cmdGosub
	c[cmdCompare] = (*Engine).cmdCompare
	for k := cmdAdd; k <= cmdMultiply; k++ {
		c[k] = (*Engine).cmdArithmetic
	}
	c[cmdRandom] = (*Engine).cmdRandom
	c[cmdSave] = (*Engine).cmdSave
	c[cmdLoadChar] = (*Engine).cmdLoadCharacter
	c[cmdLoadMonster] = (*Engine).cmdLoadMonster
	c[cmdSetupMon] = (*Engine).cmdSetupMonster
	c[cmdApproach] = (*Engine).cmdApproach
	c[cmdPicture] = (*Engine).cmdPicture
	c[cmdInputNumber] = (*Engine).cmdInputNumber
	c[cmdInputText] = (*Engine).cmdInputText
	c[cmdPrint] = (*Engine).cmdPrint
	c[cmdPrintClear] = (*Engine).cmdPrint
	c[cmdReturn] = (*Engine).cmdReturn
	c[cmdCompareAnd] = (*Engine).cmdCompareAnd
	c[cmdVMenu] = (*Engine).cmdVerticalMenu
	for k := cmdIfEqual; k <= cmdIfLast; k++ {
		c[k] = (*Engine).cmdIf
	}
	c[cmdClearMon] = (*Engine).cmdClearMonsters
	c[cmdPartyStr] = (*Engine).cmdPartyStrength
	c[cmdCheckParty] = (*Engine).cmdCheckParty
	c[cmdNewECL] = (*Engine).cmdNewECL
	c[cmdLoadFiles] = (*Engine).cmdLoadFiles
	c[cmdLoadPieces] = (*Engine).cmdLoadFiles
	c[cmdPartySurp] = (*Engine).cmdPartySurprise
	c[cmdSurprise] = (*Engine).cmdSurprise
	c[cmdCombat] = (*Engine).cmdCombat
	c[cmdOnGoto] = (*Engine).cmdOnGotoGosub
	c[cmdOnGosub] = (*Engine).cmdOnGotoGosub
	c[cmdTreasure] = (*Engine).cmdTreasure
	c[cmdRob] = (*Engine).cmdRob
	c[cmdEncounter] = (*Engine).cmdEncounter
	c[cmdGetTable] = (*Engine).cmdGetTable
	c[cmdHMenu] = (*Engine).cmdHorizontalMenu
	c[cmdParlay] = (*Engine).cmdParlay
	c[cmdCall] = (*Engine).cmdCall
	c[cmdDamage] = (*Engine).cmdDamage
	c[cmdAnd] = (*Engine).cmdAndOr
	c[cmdOr] = (*Engine).cmdAndOr
	c[cmdSpriteOff] = (*Engine).cmdSpriteOff
	c[cmdFindItem] = (*Engine).cmdFindItem
	c[cmdPrintReturn] = (*Engine).cmdPrintReturn
	c[cmdClock] = (*Engine).cmdClock
	c[cmdSaveTable] = (*Engine).cmdSaveTable
	c[cmdAddNPC] = (*Engine).cmdAddNPC
	c[cmdProgram] = (*Engine).cmdProgram
	c[cmdWho] = (*Engine).cmdWho
	c[cmdDelay] = (*Engine).cmdDelay
	c[cmdSpell] = (*Engine).cmdSpell
	c[cmdProtection] = (*Engine).cmdProtection
	c[cmdClearBox] = (*Engine).cmdClearBox
	c[cmdDump] = (*Engine).cmdDump
	c[cmdFindSpecial] = (*Engine).cmdFindSpecial
	c[cmdDestroy] = (*Engine).cmdDestroyItems
}

// cmdExit = O03C:0050: end the event; unwinds the call stack.
func (e *Engine) cmdExit() {
	s := &e.sc
	if s.clearPicture {
		e.clearViewArea()
	}
	if s.restoreSelected {
		e.selectMember(s.savedSelected)
		s.restoreSelected = false
	}
	s.pictureShown, s.pictureBig = false, false
	s.pictureOn = false
	s.clearPicture = false
	s.stop = true
	s.pc++
	s.calls = nil
	e.msgRow, e.msgCol = textTop, textLeft
	s.inSpecial = false
}

// cmdGoto = O03C:00EB.
func (e *Engine) cmdGoto() {
	e.readArgs(1)
	e.sc.pc = varAddr(e.sc.argLo[1], e.sc.argHi[1])
}

// cmdGosub = O03C:0108.
func (e *Engine) cmdGosub() {
	e.readArgs(1)
	e.scriptCall(1)
}

// cmdCompare = O03C:011D: texts when either operand is text.
func (e *Engine) cmdCompare() {
	e.readArgs(2)
	if e.sc.argType[1] >= argText || e.sc.argType[2] >= argText {
		e.compareTexts(e.sc.text[1], e.sc.text[2])
		return
	}
	e.compareValues(e.arg(1), e.arg(2))
}

// areaWordRemainder: area2 word the divide leaves its remainder in.
const areaWordRemainder = 831

// cmdArithmetic = O03C:019A (add, subtract, divide, multiply).
func (e *Engine) cmdArithmetic() {
	e.readArgs(3)
	a, b := e.arg(1), e.arg(2)
	dst := varAddr(e.sc.argLo[3], e.sc.argHi[3])
	var r uint16
	switch e.sc.command {
	case cmdAdd:
		r = a + b
	case cmdSubtract:
		r = b - a
	case cmdDivide:
		if b != 0 { // the original stops with a division error
			r = a / b
			putWord(e.game, 2*areaWordRemainder, a%b)
		}
	case cmdMultiply:
		r = a * b
	}
	e.setVar(dst, r)
}

// cmdRandom = O03C:023D: 0 .. operand.
func (e *Engine) cmdRandom() {
	e.readArgs(2)
	n := byte(e.arg(1))
	if n < 255 {
		n++
	}
	dst := varAddr(e.sc.argLo[2], e.sc.argHi[2])
	e.setVar(dst, uint16(e.Random(int(n))))
}

// cmdSave = O03C:029B: store a value or text.
func (e *Engine) cmdSave() {
	e.readArgs(2)
	dst := varAddr(e.sc.argLo[2], e.sc.argHi[2])
	if e.sc.argType[1] < argText {
		e.setVar(dst, e.arg(1))
		return
	}
	e.setTextVar(dst, e.sc.text[1])
}

// partyMemberFlag: in LOAD CHARACTER's operand, set to drop the member
// when the script has blanked it out.
const partyMemberFlag = 128

// cmdLoadCharacter = O03C:02E9: select party member n (0-based).
func (e *Engine) cmdLoadCharacter() {
	s := &e.sc
	s.restoreSelected = true
	e.readArgs(1)
	n := byte(e.arg(1))
	drop := n&partyMemberFlag != 0
	n &^= partyMemberFlag
	if int(n) < len(e.party) {
		e.partySel = int(n)
		s.justJoined = false
	} else {
		s.justJoined = true
	}
	if drop && s.activeCleared && s.nameCleared {
		if s.savedSelected == e.selected() {
			s.restoreSelected = false
		}
		e.freeSelected()
		e.playerSummary(e.selected())
		s.activeCleared, s.nameCleared = false, false
	}
}

// cmdInputNumber = O03C:095E.
func (e *Engine) cmdInputNumber() {
	e.readArgs(2)
	dst := varAddr(e.sc.argLo[2], e.sc.argHi[2])
	e.setVar(dst, e.InputNumber("", colourGreen))
}

// cmdInputText = O03C:09A4: a blank answer is one space.
func (e *Engine) cmdInputText() {
	e.readArgs(2)
	dst := varAddr(e.sc.argLo[2], e.sc.argHi[2])
	t := e.InputString("", colourGreen, screenLastCol+1)
	if t == "" {
		t = " "
	}
	e.setTextVar(dst, t)
}

// cmdPrint = O03C:0A1C (PRINT, PRINTCLEAR): text into the text window,
// typed out slowly.
func (e *Engine) cmdPrint() {
	e.readArgs(1)
	e.sc.waitedKey = false
	e.slowText = true
	if e.sc.argType[1] < argText {
		e.sc.text[1] = strconv.Itoa(int(e.arg(1)))
	}
	clear := e.sc.command != cmdPrint
	if clear {
		e.msgRow, e.msgCol = textTop, textLeft
	}
	e.DisplayTextBlock(textLeft, textTop, textRight, textBottom, colourGreen, clear, e.sc.text[1])
	e.slowText = false
}

// cmdReturn = O03C:0AB8: back from a GOSUB, or EXIT when there is none.
func (e *Engine) cmdReturn() {
	s := &e.sc
	s.pc++
	if n := len(s.calls); n > 0 {
		s.pc = s.calls[n-1]
		s.calls = s.calls[:n-1]
		return
	}
	e.cmdExit()
}

// cmdCompareAnd = O03C:0B0C: equal when both pairs are.
func (e *Engine) cmdCompareAnd() {
	e.sc.flags = [6]bool{}
	e.readArgs(4)
	a, b, c, d := e.arg(1), e.arg(2), e.arg(3), e.arg(4)
	if a == b && c == d {
		e.sc.flags[0] = true
		return
	}
	e.sc.flags[1] = true
}

// cmdIf = O03C:0B71: skip the next command unless the flag holds.
func (e *Engine) cmdIf() {
	e.sc.pc++
	if !e.sc.flags[e.sc.command-cmdIfEqual] {
		e.skipCommand()
	}
}

// cmdClearMonsters = O03C:1240.
func (e *Engine) cmdClearMonsters() {
	s := &e.sc
	s.pc++
	s.monsterCount = 0
	s.monstersLoaded = false
	s.iconSlot = firstMonsterIcon
	e.clearTreasure()
}

// cmdPartyStrength = O03C:12A9: the sum over the party of
// (hp + 5*(AC better than 0) + 5*(THAC0 worse than 0) + 8*mage + 4*cleric) / 10.
func (e *Engine) cmdPartyStrength() {
	e.readArgs(1)
	var sum byte
	for _, c := range e.party {
		hp, ac, thac0 := int(c.HP), int(c.AC[0]), int(c.THAC0)
		old := 0
		if oldClassUsable(c) {
			old = 1
		}
		mage := int(int8(c.Levels[clsMage])) + int(int8(c.OldLevels[clsMage]))*old
		cleric := int(int8(c.Levels[clsCleric])) + int(int8(c.OldLevels[clsCleric]))*old
		if ac > acZero {
			ac -= acZero
		} else {
			ac = 0
		}
		if thac0 > thac0Zero {
			thac0 -= thac0Zero
		} else {
			thac0 = 0
		}
		sum += byte((cleric*4 + hp + ac*5 + thac0*5 + mage*8) / 10)
	}
	e.setVar(varAddr(e.sc.argLo[1], e.sc.argHi[1]), uint16(sum))
}

// AC and THAC0 are stored offset: 60 is AC 0, 39 is THAC0 0... as kept in
// the record (see charRecord).
const (
	acZero    = 60
	thac0Zero = 39
)

// cmdNewECL = O03C:0BE7: load another area's script and start it.
func (e *Engine) cmdNewECL() {
	s := &e.sc
	e.readArgs(1)
	putWord(e.area, 2*areaWordLastECL, uint16(s.block))
	n := byte(e.arg(1))
	s.block = n
	e.loadScript(n)
	e.initScript()
	s.stop = true
	s.restart = true
	s.pictureShown, s.pictureBig = false, false
}

// cmdAndOr = O03C:0DD0.
func (e *Engine) cmdAndOr() {
	e.readArgs(3)
	a, b := e.arg(1), e.arg(2)
	dst := varAddr(e.sc.argLo[3], e.sc.argHi[3])
	var r byte
	if e.sc.command == cmdAnd {
		r = byte(a & b)
	} else {
		r = byte(a | b)
	}
	e.compareValues(0, uint16(r))
	e.setVar(dst, uint16(r))
}

// cmdGetTable = O03C:0E3F: dst := [base + index].
func (e *Engine) cmdGetTable() {
	e.readArgs(3)
	base := varAddr(e.sc.argLo[1], e.sc.argHi[1])
	i := byte(e.arg(2))
	dst := varAddr(e.sc.argLo[3], e.sc.argHi[3])
	e.setVar(dst, e.getVar(uint16(i)+base))
}

// cmdSaveTable = O03C:0E9D: [base + index] := value.
func (e *Engine) cmdSaveTable() {
	e.readArgs(3)
	v := e.arg(1)
	base := varAddr(e.sc.argLo[2], e.sc.argHi[2])
	i := e.arg(3)
	e.setVar(base+i, v)
}

// cmdHorizontalMenu = O03C:10B4: the items on the bottom row; the item's
// number goes to the first operand's address. A single item is "press a
// key to continue".
func (e *Engine) cmdHorizontalMenu() {
	s := &e.sc
	e.readArgs(2)
	dst := varAddr(s.argLo[1], s.argHi[1])
	n := byte(e.arg(2))
	s.pc--
	e.readArgs(int(n))
	enterOK, highlight := byte(0), colourGreen
	if n == 1 {
		enterOK, highlight = 1, colourWhite
		if s.text[1] == "PRESS BUTTON OR RETURN TO CONTINUE." {
			s.text[1] = "PRESS <ENTER>/<RETURN> TO CONTINUE"
		}
	}
	animate := s.pictureShown && s.pictureBig
	menu := ""
	for k := 1; k < int(n); k++ {
		menu = cut(menu+"~"+s.text[k]+" ", menuTextMax)
	}
	menu = cut(menu+"~"+s.text[n], menuTextMax)
	r := e.scriptMenu("", menu, colourLightRed, highlight, colourWhite, enterOK != 0, animate)
	e.setVar(dst, uint16(r))
	e.ClearCells(0, statusRow, screenLastCol, statusRow)
}

const menuTextMax = 50

// scriptMenu = O04C:17AA (prompt, menu, colours, enterOK, animate): the
// horizontal menu; the chosen item's number, 0 for Enter, 255 otherwise.
func (e *Engine) scriptMenu(prompt, menu string, promptColour, normal, highlight int, enterOK, animate bool) byte {
	menu, hot := menuHotkeys(menu)
	var k byte
	for {
		var extra bool
		k, extra = e.DisplayInput(prompt, menu, promptColour, normal, highlight, true, animate)
		if extra {
			e.cycleSelected(k)
			e.playerSummary(e.selected())
			k = 0
		}
		if k >= '0' && k <= '9' || k >= 'A' && k <= 'Z' || k == keyEnter && enterOK {
			break
		}
	}
	if k == keyEnter {
		return 0
	}
	for i := 0; i < len(hot); i++ {
		if hot[i] == k {
			return byte(i)
		}
	}
	return 255
}

// menuHotkeys = O04C:1693: drop each "~" and collect the upcased letter after
// it; every other letter or digit gets +32 (lower case; digits turn into
// "P".."Y", as in the original).
func menuHotkeys(menu string) (string, string) {
	var b [256]byte
	n := copy(b[1:], menu)
	length := n
	var hot []byte
	for i := 1; i <= n; i++ {
		if c := b[i]; c >= '0' && c <= '9' || c >= 'A' && c <= 'Z' {
			b[i] += 32
		}
		if b[i] == '~' {
			copy(b[i:], b[i+1:length+1])
			length--
			if b[i] >= 'a' && b[i] <= 'z' {
				b[i] -= 32
			}
			hot = append(hot, b[i])
		}
	}
	return string(b[1 : length+1]), string(hot)
}

// cycleSelected = O0AF:32CF (key): Home selects the previous member, End
// the next, anything else the first.
func (e *Engine) cycleSelected(k byte) {
	if len(e.party) == 0 {
		return
	}
	switch k {
	case scanHome:
		if e.partySel == 0 {
			e.partySel = len(e.party) - 1
		} else {
			e.partySel--
		}
	case scanEnd:
		e.partySel++
		if e.partySel >= len(e.party) {
			e.partySel = 0
		}
	default:
		e.partySel = 0
	}
}
