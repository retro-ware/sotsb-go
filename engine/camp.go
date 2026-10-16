package engine

// Encamp (unit O0E1) and resting (unit O10C): the camp menu, Magic (cast,
// memorize, scribe, display), Rest, Alter (order, drop, speed, icon, pics)
// and Fix, with the rest clock, healing and study.

// campState: the rest time and study counters (DS:7580..759x).
type campState struct {
	rest       [7]uint16                  // a7580: rest time left in clock units (1 minutes, 2 tens, 3 hours, 4 days)
	healSteps  uint16                     // w7596: five-minute steps toward the daily healing
	study      [maxPartyMembers + 1]uint8 // a758D: steps until member i (1-based) learns the next spell
	encounters uint16                     // w4FDC: steps since the last encounter roll
}

// Rest time units (as the clock's).
const (
	restMinutes = 1
	restTens    = 2
	restHours   = 3
	restDays    = 4
	restMonths  = 5
	maxRestDays = 99
)

func spellClass(sp byte) byte { return spellTableData[sp&127][0] }
func spellLevel(sp byte) byte { return spellTableData[sp&127][1] }

// studyMinutes = O0E1:0023 (c): minutes of study for the spells waiting to
// be memorized (+128, a level 0 spell counting as 1st) or scribed (a bag's
// scrolls too); the base hours (4, or 6 for spells above 2nd level) go to
// the record's SpellsToLearn.
func (e *Engine) studyMinutes(c *Character) uint16 {
	var maxMem, maxScr, sumMem, sumScr byte
	for i := 0; i < len(c.Memorized); i++ {
		if s := c.Memorized[i]; s > 127 {
			l := spellLevel(s)
			if l == 0 {
				l = 1
			}
			if l > maxMem {
				maxMem = l
			}
			sumMem += l
		}
	}
	for _, it := range c.Items {
		if !e.isScroll(it) {
			continue
		}
		for _, sc := range scrollsOf(it) {
			for k := 0; k < 3; k++ {
				if s := sc.Spells[k]; s > 127 {
					l := spellLevel(s)
					if l > maxScr {
						maxScr = l
					}
					sumScr += l
				}
			}
		}
	}
	var hours byte
	if sumMem > 0 || sumScr > 0 {
		hours = 4
	}
	if maxMem > 2 || maxScr > 2 {
		hours = 6
	}
	c.SpellsToLearn = hours
	return uint16(sumScr)*15 + uint16(hours)*60 + uint16(sumMem)*15
}

// scrollsOf: a scroll itself, or the scrolls in a bag.
func scrollsOf(it *Item) []*Item {
	if it.Type == itemTypeBag {
		return it.Contents
	}
	return []*Item{it}
}

// forgetPending = O0E1:05C9 (c): the spells still to be memorized go.
func forgetPending(c *Character) {
	for i := range c.Memorized {
		if c.Memorized[i] > 127 {
			c.Memorized[i] = 0
		}
	}
	c.SpellsToLearn = 0
}

// stopScribing = O0E1:060C (c): no scroll spell is being scribed.
func (e *Engine) stopScribing(c *Character) {
	for _, it := range c.Items {
		if !e.isScroll(it) {
			continue
		}
		for _, sc := range scrollsOf(it) {
			for k := range sc.Spells {
				sc.Spells[k] &= 127
			}
		}
	}
}

// stopStudy = O0E1:0700: nobody studies any more.
func (e *Engine) stopStudy() {
	for _, c := range e.party {
		forgetPending(c)
		e.stopScribing(c)
	}
}

// slotsLeft = O0E1:0744 (level, class): spells of the class and level the
// selected member may still memorize (never below none).
func (e *Engine) slotsLeft(level, class byte) byte {
	c := e.selected()
	var used byte
	for i := 0; i < len(c.Memorized); i++ {
		if s := c.Memorized[i]; s > 0 && spellLevel(s) == level && spellClass(s) == class {
			used++
		}
	}
	slots := c.SpellSlots[slotIndex(int(class), int(level))]
	if slots < used {
		used = slots
	}
	return slots - used
}

// Why a member cannot study (canStudy's mode).
const (
	studyCast     = 1
	studyMemorize = 2
	studyScribe   = 3
)

// canStudy = O0E1:089C (mode): not where spells are blocked, and not when
// out of action; otherwise says why.
func (e *Engine) canStudy(mode int) bool {
	msg := ""
	if mode == studyCast && wordAt(e.area, 2*areaWordNoMagic) > 0 {
		msg = "cannot cast spells in this area"
	} else if c := e.selected(); c.Status == statusAnimated || c.Okay == 0 {
		msg = "is in no condition to "
		switch mode {
		case studyCast:
			msg += "cast any spells"
		case studyMemorize:
			msg += "memorize spells"
		case studyScribe:
			msg += "scribe any scrolls"
		}
	}
	if msg == "" {
		return true
	}
	e.combatName(e.selected(), msg, 10, true)
	return false
}

// castMenu = O0E1:09D1: cast memorized spells until none is
// picked.
func (e *Engine) castMenu() {
	cast := false
	index := -1
	e.spellCaster = nil
	e.menuSel = 1
	if e.canStudy(studyCast) {
		for {
			var any bool
			sp := e.spellMenu2(0, 1, &index, &any)
			if sp == 0 {
				if any {
					cast = true
				} else {
					e.combatName(e.selected(), "has no spells memorized", 10, true)
				}
				break
			}
			cast = true
			e.ClearCells(1, 17, 38, 22)
			var quit bool
			e.CastSpell(sp, 0, 1, &quit)
		}
	}
	if cast {
		e.loadPicture()
	}
}

// The spell class headings of the memorize table (none for paladins and
// rangers).
var memorizeHeadings = [spellClassMage + 1]string{"    Cleric Spells:", "     Druid Spells:", "", "Magic-User Spells:"}

// memorizeInfo = O0E1:0AC3: the selected member's name over the free
// slots per class and level; false when there are none.
func (e *Engine) memorizeInfo() bool {
	c := e.selected()
	any := false
	var counts [spellLevels][spellClassMage + 1]string
	var has [spellClassMage + 1]bool
	for class := spellClassCleric; class <= spellClassMage; class++ {
		for level := 1; level <= spellLevels; level++ {
			counts[level-1][class] = itoa(int(e.slotsLeft(byte(level), byte(class))))
			if c.SpellSlots[slotIndex(class, level)] == 0 {
				counts[level-1][class] = " "
			} else {
				has[class] = true
				any = true
			}
		}
	}
	if any {
		e.combatName(c, "", 10, false)
		row := 1
		for class := spellClassCleric; class <= spellClassMage; class++ {
			if memorizeHeadings[class] == "" || !has[class] {
				continue
			}
			e.displayStringBg(1, row+17, colourGreen, colourBlack, memorizeHeadings[class])
			x := 19
			for level := 1; level <= spellLevels; level++ {
				e.displayStringBg(x+1, row+17, colourGreen, colourBlack, counts[level-1][class])
				x += 3
			}
			row++
		}
	}
	return any
}

// restMenu = O0E1:0CE7 (var interrupted): rest as long as the longest
// study takes, then the rest menu.
func (e *Engine) restMenu() bool {
	var longest uint16
	for _, c := range e.party {
		if m := e.studyMinutes(c); longest < m {
			longest = m
		}
	}
	r := &e.camp.rest
	r[restHours] = longest / 60
	r[restTens] = (longest - 60*r[restHours]) / 10
	r[restMinutes] = longest % 10
	interrupted := e.restFor(true)
	e.camp.rest = [7]uint16{}
	e.statusLine()
	return interrupted
}

// sortSpells = O0E1:0D8E: the selected member's memorized spells in spell
// order (a waiting one counts as its spell).
func (e *Engine) sortSpells() {
	m := &e.selected().Memorized
	for i := 0; i < len(m)-1; i++ {
		for j := i; j < len(m); j++ {
			if m[i]&127 > m[j]&127 {
				m[i], m[j] = m[j], m[i]
			}
		}
	}
}

// memorizeMenu = O0E1:0E3D.
func (e *Engine) memorizeMenu() {
	if !e.canStudy(studyMemorize) {
		return
	}
	done := false
	index := -1
	e.menuSel = 1
	var shown bool
	e.spellMenu2(5, 0, &index, &shown)
	redraw := true
	if shown {
		if e.yesNoFrom(answerYes, colourYellow, colourGreen, colourWhite, "memorize these spells?") == 'N' {
			forgetPending(e.selected())
		} else {
			done = true
		}
	} else {
		redraw = false
	}
	index = -1
	for !done {
		done = !e.memorizeInfo()
		if done {
			e.combatName(e.selected(), "cannot memorize any spells", 10, true)
			break
		}
		sp := e.spellMenu2(1, 2, &index, &shown)
		redraw = true
		if sp == 0 {
			done = true
		} else if e.slotsLeft(spellLevel(sp), spellClass(sp)) > 0 {
			c := e.selected()
			i := 0
			for c.Memorized[i] != 0 {
				i++
			}
			c.Memorized[i] = sp + 128
			e.sortSpells()
		}
	}
	if index != -1 {
		index = -1
		e.spellMenu2(5, 0, &index, &shown)
		if shown && e.yesNoFrom(answerYes, colourYellow, colourGreen, colourWhite, "memorize these spells?") == 'N' {
			forgetPending(e.selected())
		}
	}
	if redraw {
		e.loadPicture()
	}
}

// scribeMenu = O0E1:10A7.
func (e *Engine) scribeMenu() {
	if !e.canStudy(studyScribe) {
		return
	}
	c := e.selected()
	done := false
	index := -1
	var shown bool
	e.spellMenu2(6, 0, &index, &shown)
	redraw := true
	if shown {
		if e.yesNoFrom(answerYes, colourYellow, colourGreen, colourWhite, "Scribe these spells?") == 'N' {
			e.stopScribing(c)
		} else {
			done = true
		}
	} else {
		redraw = false
	}
	index = -1
	for !done {
		sp := e.spellMenu2(3, 3, &index, &shown)
		if sp == 0 {
			done = true
			if !shown {
				e.combatName(c, "has no copyable scrolls", 10, true)
			} else {
				redraw = true
			}
			continue
		}
		redraw = true
		if c.SpellBook[sp-1] != 0 {
			e.printMessage("You already know that spell")
			continue
		}
		busy := false
		for _, it := range c.Items {
			if busy {
				break
			}
			if !e.isScroll(it) {
				continue
			}
			for _, sc := range scrollsOf(it) {
				for k := 0; k < 3; k++ {
					if !busy && sc.Spells[k] > 127 && sc.Spells[k]&127 == sp {
						e.printMessage("You are already scribing that spell")
						busy = true
					}
				}
			}
		}
		if busy {
			continue
		}
		if c.SpellSlots[slotIndex(int(spellClass(sp)), int(spellLevel(sp)))] == 0 {
			e.printMessage("You can not scribe that spell.")
			continue
		}
		marked := false
		for _, it := range c.Items {
			if marked {
				break
			}
			for _, sc := range scrollsOf(it) {
				for k := 0; k < 3 && !marked; k++ {
					if sc.Spells[k] == sp {
						sc.Spells[k] |= 128
						marked = true
					}
				}
				if marked {
					break
				}
			}
		}
	}
	if index != -1 {
		index = -1
		e.spellMenu2(6, 0, &index, &shown)
		if shown && e.yesNoFrom(answerYes, colourYellow, colourGreen, colourWhite, "Scribe these spells?") == 'N' {
			e.stopScribing(c)
		}
	}
	if redraw {
		e.loadPicture()
	}
}

// The affects Display names that no spell's entry names.
var affectNames = map[byte]string{
	4: "Dispel Evil", 7: "Faerie Fire", 27: "Fumbling", 31: "Helpless", 35: "Confused", 44: "Cause Disease",
	50: "Hot Fire Shield", 54: "Cold Fire Shield", 55: "Poisoned", 59: "Regenerating", 61: "Fire Resistance",
	63: "Minor Globe of Invulnerability", 68: "enfeebled", 69: "invisible to animals", 71: "Invisible",
	72: "Camouflaged", 73: "protected from dragon breath", 77: "berserk", 89: "Displaced",
}

// spellAffect: Display names these by the spell whose table entry gives them.
func spellAffect(t byte) bool {
	switch {
	case t == 1 || t == 2 || t == 5 || t >= 8 && t <= 12 || t == 14 || t == 16 || t == 17 || t >= 19 && t <= 25,
		t == 28 || t == 29 || t >= 32 && t <= 34 || t >= 36 && t <= 39 || t == 41 || t == 42 || t == 45 || t == 46,
		t == 49 || t >= 51 && t <= 53:
		return true
	}
	return false
}

// spellEffectSpells: Display looks through the first 56 spells.
const spellEffectSpells = 56

// displayEffects = O0E1:158D: everyone's spell
// effects in a list.
func (e *Engine) displayEffects() {
	list := []*ListNode{{Text: " "}}
	add := func(flag byte, text string) { list = append(list, &ListNode{Text: cut(text, 40), Flag: flag}) }
	for _, c := range e.party {
		n := 0
		add(1, c.Name())
		for _, a := range c.Affects {
			t := a.Type
			name, ok := affectNames[t]
			if spellAffect(t) {
				ok = true
				name = ""
				for k := 1; k <= spellEffectSpells; k++ {
					if spellTableData[k][10] == t {
						name = spellNames[k]
						break
					}
				}
				if name == "" {
					name = "Funky--" + itoa(int(t))
				}
			}
			if ok {
				n++
				add(0, " "+name)
			}
		}
		if n == 0 {
			add(0, " <No Spell Effects>")
		}
		add(0, " ")
	}
	redraw := true
	index := 0
	var sel *ListNode
	e.drawScreenFrame()
	e.ListMenu("", "", colourCyan, colourGreen, colourWhite, 1, 4, 38, 22, list, true, &redraw, &index, &sel)
	e.loadPicture()
}

// selectStep: an extended key in the camp menus moves the selection
// (O151:0987 around O100:39AE): the old name is drawn plain, the new one
// white.
func (e *Engine) selectStep(key byte) {
	e.summaryRow(e.selected(), false)
	e.stepSelection(key)
	e.summaryRow(e.selected(), true)
}

// magicMenu = O0E1:1757 (var interrupted).
func (e *Engine) magicMenu(interrupted *bool) {
	key := byte(' ')
	for !*interrupted && key != 0 && key != 'E' {
		var extra bool
		key, extra = e.DisplayInput("", "Cast Memorize Scribe Display Rest Exit", colourLightMagenta, colourGreen, colourWhite, true, true)
		sel := e.menuSel
		if extra {
			e.selectStep(key)
		} else {
			switch key {
			case 'C':
				e.castMenu()
			case 'M':
				e.memorizeMenu()
			case 'S':
				e.scribeMenu()
			case 'D':
				e.displayEffects()
			case 'R':
				*interrupted = e.restMenu()
			}
		}
		e.menuSel = sel
	}
}

// movePrev = O0E1:182E: the selected member swaps places with the one
// before; the first goes last.
func (e *Engine) movePrev() {
	i := e.partySel
	n := len(e.party)
	if n < 2 {
		return
	}
	c := e.party[i]
	if i == 0 {
		e.party = append(e.party[1:], c)
		e.partySel = n - 1
		return
	}
	e.party[i-1], e.party[i] = e.party[i], e.party[i-1]
	e.partySel = i - 1
}

// moveNext = O0E1:196F: swaps with the one after; the last goes first.
func (e *Engine) moveNext() {
	i := e.partySel
	n := len(e.party)
	if n < 2 {
		return
	}
	c := e.party[i]
	if i == n-1 {
		e.party = append([]*Character{c}, e.party[:n-1]...)
		e.partySel = 0
		return
	}
	e.party[i+1], e.party[i] = e.party[i], e.party[i+1]
	e.partySel = i + 1
}

// orderMenu = O0E1:1AD8: Select a member, then Place it with Up / Down.
func (e *Engine) orderMenu() {
	placing := false
	key := byte(' ')
	for key != 0 && key != 'E' {
		menu := "Select Exit"
		if placing {
			menu = "Place Exit"
		}
		var extra bool
		key, extra = e.DisplayInput("Party Order: ", menu, colourLightMagenta, colourGreen, colourWhite, true, true)
		if extra {
			if !placing {
				e.selectStep(key)
			} else {
				switch key {
				case scanUp:
					e.movePrev()
				case scanDown:
					e.moveNext()
				}
				e.playerSummary(e.selected())
			}
			continue
		}
		if key == keyEnter || key == 'P' || key == 'S' {
			placing = !placing
			if placing {
				e.combatName(e.selected(), "has been selected", 10, false)
			} else {
				e.clearCombatText()
			}
		}
	}
}

// dropMember = O0E1:1C53: the selected member leaves the
// party (the last one quits the game).
func (e *Engine) dropMember() {
	c := e.selected()
	if len(e.party) == 1 {
		if e.YesNo(colourYellow, colourGreen, colourWhite, "quit TO DOS: ") == 'Y' {
			e.freeSelected()
			e.Exit()
		}
		return
	}
	e.combatName(c, "will be gone", 10, false)
	if e.YesNo(colourYellow, colourGreen, colourWhite, "Drop from party? ") == 'Y' {
		if c.Okay != 0 {
			e.combatName(c, "bids you farewell", 10, true)
		} else {
			e.combatName(c, "is dumped in a ditch", 10, true)
		}
		e.freeSelected()
		e.ClearCells(17, 1, 38, 11)
		e.playerSummary(e.selected())
		return
	}
	if c.Okay != 0 {
		e.combatName(c, "Breathes A sigh of relief", 10, true)
	}
}

// Game speed limits.
const maxGameSpeed = 9

// campSpeedMenu = O0E1:1ED8: Faster / Slower (the arrow keys too).
func (e *Engine) campSpeedMenu() {
	for {
		e.displayStringBg(1, 18, colourGreen, colourBlack, "Game Speed = "+itoa(int(e.gameSpeed))+" (0=fastest 9=slowest)")
		menu := ""
		if e.gameSpeed > 0 {
			menu += " Faster"
		}
		if e.gameSpeed < maxGameSpeed {
			menu += " Slower"
		}
		menu += " Exit"
		key, extra := e.DisplayInput("Game Speed:", menu, colourLightMagenta, colourGreen, colourWhite, true, true)
		if extra {
			switch key {
			case scanDown:
				if e.gameSpeed > 0 {
					e.gameSpeed--
				}
			case scanUp:
				if e.gameSpeed < maxGameSpeed {
					e.gameSpeed++
				}
			}
		} else {
			switch key {
			case 'F':
				e.gameSpeed--
			case 'S':
				e.gameSpeed++
			}
		}
		if key == 0 || key == 'E' {
			break
		}
	}
	e.clearCombatText()
}

// alterMenu = O0E1:2054: pictures and their animation are switched on
// for good.
func (e *Engine) alterMenu() {
	e.picturesOn = true
	e.loadingMessage = true
	key := byte(' ')
	for key != 0 && key != 'E' {
		var extra bool
		key, extra = e.DisplayInput("", "Order Drop Speed Icon Level Exit", colourLightMagenta, colourGreen, colourWhite, true, true)
		sel := e.menuSel
		if extra {
			e.selectStep(key)
		} else {
			switch key {
			case 'L':
				e.levelMenu()
			case 'O':
				e.orderMenu()
			case 'D':
				e.dropMember()
			case 'S':
				e.campSpeedMenu()
			case 'I':
				e.iconBuilder()
				e.loadPicture()
			}
		}
		e.menuSel = sel
	}
}

// areaWordDifficulty: the difficulty level, 0 Novice .. 4 Champion.
const areaWordDifficulty = 500

// levelMenu = O0E1:1DB0: pick the difficulty, the current one highlighted.
func (e *Engine) levelMenu() {
	level := uint16(byte(wordAt(e.area, 2*areaWordDifficulty)))
	e.menuSel = int(level) + 1
	key, extra := e.DisplayInput("", "Novice Squire Veteran Adept Champion", colourLightMagenta, colourGreen, colourWhite, true, true)
	if extra {
		e.selectStep(key)
	} else {
		switch key {
		case 'N':
			level = 0
		case 'S':
			level = 1
		case 'V':
			level = 2
		case 'A':
			level = 3
		case 'C':
			level = 4
		}
	}
	putWord(e.area, 2*areaWordDifficulty, level)
}

// fixCounts: what Fix's spells can do (cures by strength, and the other
// spells by what they mend).
type fixCounts struct {
	light, serious, critical, heal        int // cures rolled after the rest
	heals, neutralize, raise, removeCurse int // spells cast on the needy
	stoneToFlesh, cureBlind, cureDisease  int
}

// fixNeeds: how many members each spell could help (O0E1:2E41).
type fixNeeds struct {
	raise, removeCurse, cureBlind, cureDisease, heal, neutralize, stoneToFlesh int
}

// fixCaster = O0E1:2139: only the okay cast for Fix.
func fixCaster(c *Character) bool { return c.Status == statusOkay }

// woundsTotal = O0E1:2377: the party's hit points short of the maximum.
func (e *Engine) woundsTotal() int {
	w := 0
	for _, c := range e.party {
		w += int(c.MaxHP) - int(c.HP)
	}
	return w
}

// fixMenu = O0E1:2F86 (var interrupted): the casters use their memorized
// spells, rest to learn the cures again and cast them; then they rest for
// the other mending spells their slots hold and cast those.
func (e *Engine) fixMenu() bool {
	if e.woundsTotal() == 0 {
		return false
	}
	var n fixCounts
	healing := e.castCures(&n)
	saved := e.camp.rest
	e.fixRestTime(&n)
	if e.restFor(false) {
		return true
	}
	healing += e.rollCures(n)
	e.fixCure(&n)
	e.applyHealing(&healing)
	e.playerSummary(e.selected())
	e.statusLine()
	e.camp.rest = saved
	need := e.fixNeeds()
	e.mendRestTime(&n, need)
	if e.restFor(false) {
		return true
	}
	e.fixCure(&n)
	e.playerSummary(e.selected())
	e.statusLine()
	e.camp.rest = saved
	return false
}

// castCures = O0E1:215C: every caster casts its memorized cures (the hit
// points rolled) and counts its other mending spells.
func (e *Engine) castCures(n *fixCounts) int {
	total := 0
	for _, c := range e.party {
		if !fixCaster(c) {
			continue
		}
		for _, sp := range c.Memorized {
			switch sp {
			case spellCureLightWounds, spellCureLightWoundsDruid:
				total += e.Dice(1, 8)
			case spellCureSeriousWounds:
				total += e.Dice(2, 8) + 1
			case spellCureCriticalWounds:
				total += e.Dice(3, 8) + 3
			case spellHeal:
				total += e.Dice(4, 8) + 5
				n.heals++
			case spellNeutralizePoison:
				n.neutralize++
			case spellRaiseDead:
				n.raise++
			case spellRemoveCurse, spellRemoveCurseMU:
				n.removeCurse++
			case spellStoneToFlesh:
				n.stoneToFlesh++
			case spellCureBlindness:
				n.cureBlind++
			case spellCureDisease:
				n.cureDisease++
			}
		}
	}
	return total
}

// rollCures = O0E1:2291: the cures cast after the rest.
func (e *Engine) rollCures(n fixCounts) int {
	total := 0
	for k := 1; k <= n.light; k++ {
		total += e.Dice(1, 8)
	}
	for k := 1; k <= n.serious; k++ {
		total += e.Dice(2, 8) + 1
	}
	for k := 1; k <= n.critical; k++ {
		total += e.Dice(3, 8) + 3
	}
	for k := 1; k <= n.heal; k++ {
		total += e.Dice(4, 8) + 3
	}
	return total
}

// Minutes of study per cure slot.
const (
	cureLightMinutes    = 15
	druidCureMinutes    = 30
	cureSeriousMinutes  = 60
	cureCriticalMinutes = 75
	healMinutes         = 90
	minFixWisdom        = 3 // casters need more
)

// setRestMinutes puts minutes into the rest time.
func (e *Engine) setRestMinutes(mins int) {
	r := &e.camp.rest
	r[restHours] = uint16(mins / 60)
	r[restTens] = uint16((mins - int(r[restHours])*60) / 10)
	r[restMinutes] = uint16(mins % 10)
}

// fixRestTime = O0E1:23D1: how long the casters study their cures again,
// shortened when fewer rounds are needed. The slot minutes of the last
// caster carry over to members who cannot cast.
func (e *Engine) fixRestTime(n *fixCounts) {
	n.light, n.serious, n.critical, n.heal = 0, 0, 0, 0
	expected, longest := 0, 0
	var light, druid, serious, critical, heal int
	for _, c := range e.party {
		mins := 0
		if fixCaster(c) && c.Stats[statWis].Value > minFixWisdom {
			k := int(c.SpellSlots[slotIndex(spellClassCleric, 1)])
			n.light += k
			light = k * cureLightMinutes
			k = int(c.SpellSlots[slotIndex(spellClassDruid, 2)])
			n.light += k
			druid = k * druidCureMinutes
			k = int(c.SpellSlots[slotIndex(spellClassCleric, 4)])
			n.serious += k
			serious = k * cureSeriousMinutes
			k = int(c.SpellSlots[slotIndex(spellClassCleric, 5)])
			n.critical += k
			critical = k * cureCriticalMinutes
			k = int(c.SpellSlots[slotIndex(spellClassCleric, 6)])
			n.heal += k
			heal = k * healMinutes
		}
		if light > 0 {
			mins = 240
			expected += 27
		}
		if druid > 0 {
			mins = 240
			expected += 27
		}
		if serious+critical > 0 {
			mins = 360
			if critical > 0 {
				expected += 78
			} else {
				expected += 34
			}
		}
		if heal > 0 {
			mins = 480
			expected += 90
		}
		mins += light + druid + serious + critical + heal
		if longest < mins {
			longest = mins
		}
	}
	if w := e.woundsTotal(); w < expected && byte(expected/w) > 0 {
		longest /= int(byte(expected / w))
	}
	e.setRestMinutes(longest)
}

// fixNeeds = O0E1:2E41: who needs which mending spell.
func (e *Engine) fixNeeds() fixNeeds {
	var need fixNeeds
	for _, c := range e.party {
		if c.Status == statusDead && c.Race != raceHalfElf && c.Race != raceElf {
			need.raise++
		}
		if c.Status == statusStoned {
			need.stoneToFlesh++
		}
		if e.FindAffect(c, affectCursed) != nil {
			need.removeCurse++
		}
		if e.FindAffect(c, affectPoisoned) != nil {
			need.neutralize++
		}
		if e.FindAffect(c, affectFeebleminded) != nil {
			need.heal++
		} else {
			if e.FindAffect(c, affectBlind) != nil {
				need.cureBlind++
			}
			if e.FindAffect(c, affectDiseased) != nil {
				need.cureDisease++
			}
		}
	}
	return need
}

// Minutes of study for the mending spells.
const (
	healStudyMinutes      = 90
	neutralizeMinutes     = 60
	raiseMinutes          = 75
	thirdLevelStudyMinute = 45
)

// mendRestTime = O0E1:25EC: the casters' slots add to the mending spells,
// and the rest lasts as long as studying the ones needed takes.
func (e *Engine) mendRestTime(n *fixCounts, need fixNeeds) {
	for _, c := range e.party {
		if !fixCaster(c) {
			continue
		}
		third := int(c.SpellSlots[slotIndex(spellClassCleric, 3)])
		n.heals += int(c.SpellSlots[slotIndex(spellClassCleric, 6)])
		n.stoneToFlesh += int(c.SpellSlots[slotIndex(spellClassMage, 6)])
		n.raise += int(c.SpellSlots[slotIndex(spellClassCleric, 5)])
		n.neutralize += int(c.SpellSlots[slotIndex(spellClassCleric, 4)])
		n.removeCurse += int(c.SpellSlots[slotIndex(spellClassMage, 4)]) + third
		n.cureBlind += third
		n.cureDisease += third
	}
	mins := min(n.heals, need.heal) * healStudyMinutes
	mins += min(n.stoneToFlesh, need.stoneToFlesh) * healStudyMinutes
	mins += min(n.raise, need.raise) * raiseMinutes
	mins += min(n.neutralize, need.neutralize) * neutralizeMinutes
	mins += min(n.cureDisease, need.cureDisease) * thirdLevelStudyMinute
	mins += min(n.cureBlind, need.cureBlind) * thirdLevelStudyMinute
	mins += min(n.removeCurse, need.removeCurse) * thirdLevelStudyMinute
	e.setRestMinutes(int(uint16(mins)))
}

// Constitution above which raising costs hit points per class level.
const (
	raiseConBase   = 14
	raiseConDouble = 15
	raiseConFull   = 17
)

// fixCure = O0E1:28FE: the mending spells go to who needs them.
func (e *Engine) fixCure(n *fixCounts) {
	for _, c := range e.party {
		healed := false
		if n.heals > 0 && e.FindAffect(c, affectFeebleminded) != nil {
			e.RemoveAffect(c, affectFeebleminded, nil)
			if e.FindAffect(c, affectBlind) != nil {
				e.RemoveAffect(c, affectBlind, nil)
			}
			for _, d := range diseaseAffects[1:] {
				if e.FindAffect(c, d) != nil {
					e.RemoveAffect(c, d, nil)
				}
			}
			healed = true
			e.combatName(c, "is healed", 10, true)
			n.heals--
		}
		if n.stoneToFlesh > 0 && c.Status == statusStoned {
			c.Status, c.Okay, c.HP = statusOkay, 1, 1
			e.combatName(c, "is changed back to flesh", 10, true)
			n.stoneToFlesh--
		}
		if n.neutralize > 0 && e.FindAffect(c, affectPoisoned) != nil {
			if c.HP == 0 {
				c.HP = 1
			}
			e.rv.cure = 1
			e.RemoveAffect(c, affectPoisoned, nil)
			e.RemoveAffect(c, affectSlowPoison, nil)
			e.RemoveAffect(c, affectPoisonDamage, nil)
			e.rv.cure = 0
			c.Okay, c.Status = 1, statusOkay
			e.combatName(c, "is detoxified", 10, true)
			n.neutralize--
		}
		if n.raise > 0 && c.Status == statusDead && c.Race != raceHalfElf && c.Race != raceElf {
			e.raiseFromDead(c)
			n.raise--
		}
		if n.removeCurse > 0 && e.FindAffect(c, affectCursed) != nil {
			e.setTgt(1, c)
			e.spellRemoveCurse()
			e.combatName(c, "is relieved of a curse", 10, true)
			n.removeCurse--
		}
		if !healed && n.cureDisease > 0 {
			for _, d := range diseaseAffects[1:] {
				if e.FindAffect(c, d) != nil {
					e.rv.cure = 1
					e.RemoveAffect(c, d, nil)
					e.rv.cure = 0
					healed = true
					e.statAffects(c, statStr)
				}
			}
			if healed {
				e.combatName(c, "is cured of disease", 10, true)
			}
		}
		// The original cures the selected member's blindness, whoever is blind.
		if !healed && n.cureBlind > 0 && e.FindAffect(c, affectBlind) != nil {
			e.RemoveAffect(e.selected(), affectBlind, nil)
			e.combatName(e.selected(), "regains sight", 10, true)
		}
	}
}

// raiseFromDead: Fix's raise dead (in O0E1:28FE): a point of
// constitution goes, and with a constitution of 14 or more some of the
// hit points above the rolled ones.
func (e *Engine) raiseFromDead(c *Character) {
	e.rv.cure = 1
	e.RemoveAffect(c, affectPoisoned, nil)
	e.rv.cure = 0
	c.HP, c.Status, c.Okay = 1, statusOkay, 1
	con := &c.Stats[statCon].Value
	if *con > 0 {
		*con--
	}
	var lost byte
	if c.MaxHP > c.HPRoll {
		lost = c.MaxHP - c.HPRoll
	}
	if *con >= raiseConBase {
		var per byte
		for cls := clsCleric; cls < classCount; cls++ {
			lvl := c.Levels[cls]
			if int8(lvl) <= 0 {
				continue
			}
			switch {
			case cls == clsFighter || cls == clsPaladin || cls == clsRanger:
				per += (*con - raiseConBase) * lvl
			case *con > raiseConDouble:
				per += lvl * 2
			default:
				per += lvl
			}
		}
		if per > 0 {
			lost /= per
		}
		if *con < raiseConFull || int8(c.Levels[clsFighter]) > 0 || int8(c.Levels[clsFighter]) > int8(c.MulticlassLevel) {
			c.MaxHP -= lost
		}
	}
	e.combatName(c, "has been raised from the dead", 10, true)
}

// applyHealing = O0E1:27C2 (var healing): the points go to the wounded in
// party order.
func (e *Engine) applyHealing(healing *int) {
	for _, c := range e.party {
		if c.MaxHP <= c.HP {
			continue
		}
		need := int(c.MaxHP - c.HP)
		if need > *healing {
			need = *healing
		}
		if need < 1 {
			need = 0
		}
		if need > 0 && e.healPlayer(c, byte(need), false) && need <= *healing {
			*healing -= need
		}
	}
}

// campMenu = O0E1:30DF (var interrupted).
func (e *Engine) campMenu() bool {
	saved := e.gameState
	e.gameState = stateCamp
	e.camp.healSteps = 0
	e.camp.rest = [7]uint16{}
	picName, picBlock := e.pic.name, e.mapMode
	e.loadPicture()
	e.ClearCells(1, 17, 38, 22)
	e.displayStringBg(1, 18, colourGreen, colourBlack, "The party makes camp...")
	e.stopStudy()
	interrupted := false
	for key := byte(' '); !interrupted && key != 0 && key != 'E'; {
		var extra bool
		key, extra = e.DisplayInput("", "Save View Magic Rest Alter Fix Exit", colourLightMagenta, colourGreen, colourWhite, true, true)
		sel := e.menuSel
		if extra {
			e.selectStep(key)
		} else {
			switch key {
			case 'S':
				// Every 30th save the original asks for a word from the rule
				// book (O0E1:039D); the port has no copy protection.
				e.SaveGame()
				if e.YesNo(colourYellow, colourGreen, colourWhite, "Quit TO DOS ") == 'Y' {
					e.Exit()
				}
			case 'V':
				e.viewPlayer()
			case 'M':
				e.menuSel = 1
				e.magicMenu(&interrupted)
			case 'R':
				e.menuSel = 1
				interrupted = e.restMenu()
			case 'F':
				interrupted = e.fixMenu()
			case 'A':
				e.menuSel = 1
				e.alterMenu()
			}
		}
		e.menuSel = sel
	}
	if len(picName) >= 3 && picName[:3] == "PIC" {
		e.loadViewPicture(picName, picBlock, 0)
	}
	e.stopStudy()
	e.spellCaster = nil
	e.gameState = saved
	e.statusLine()
	e.clearCombatText()
	e.ClearRow24()
	return interrupted
}

// ---- O10C: resting ----

// subtractRest = O10C:03E0 (count, unit): take count units off the rest
// time, borrowing from the larger units; all of it when that runs out.
func (e *Engine) subtractRest(count byte, unit int) {
	r := &e.camp.rest
	if r[restDays] == 0 && r[restHours] == 0 && r[restTens] == 0 && r[restMinutes] == 0 {
		return
	}
	for uint16(count) > r[unit] {
		k := unit + 1
		for r[k] == 0 && k < restMonths {
			k++
		}
		if k == restMonths {
			e.camp.rest = [7]uint16{}
			count = 0
			continue
		}
		for j := k; j >= unit+1; j-- {
			r[j]--
			r[j-1] += clockLimits[j-1]
		}
	}
	r[unit] -= uint16(count)
	e.normalizeRest()
}

// carryClock = O10C:0240 (time): each unit at its limit carries one into
// the next; a year passing ages the party.
func (e *Engine) carryClock(t *[7]uint16) {
	for i := 0; i <= 6; i++ {
		if t[i] < clockLimits[i] {
			continue
		}
		if i == 6 {
			for _, c := range e.party {
				c.Age++
			}
			t[i] -= clockLimits[i]
			e.ageCheckDue = true
			continue
		}
		t[i+1]++
		t[i] -= clockLimits[i]
	}
}

// normalizeRest = O10C:0307: carries, months fold into days, at most 99.
func (e *Engine) normalizeRest() {
	r := &e.camp.rest
	e.carryClock(r)
	if r[restMonths] > 0 {
		r[restDays] += clockLimits[restDays] * r[restMonths]
		r[restMonths] = 0
		if r[restDays] > maxRestDays {
			r[restDays] = maxRestDays
		}
	}
}

// twoDigits = O10C:04C8.
func twoDigits(n uint16) string {
	s := itoa(int(n))
	if n < 10 {
		s = "0" + s
	}
	return s[:2]
}

// showRest = O10C:0553 (unit): "Rest Time:" days:hours:minutes, the unit
// being changed in white.
func (e *Engine) showRest(unit int) {
	var col [7]int
	for i := range col {
		col[i] = colourGreen
	}
	col[unit] = colourWhite
	r := &e.camp.rest
	e.displayStringBg(1, 17, colourGreen, colourBlack, "Rest Time:")
	x := 11
	for i := restDays; i >= restHours; i-- {
		e.displayStringBg(x+1, 17, col[i], colourBlack, twoDigits(r[i]))
		e.displayStringBg(x+3, 17, colourGreen, colourBlack, ":")
		x += 3
	}
	e.displayStringBg(x+1, 17, col[restTens], colourBlack, twoDigits(10*r[restTens]+r[restMinutes]))
}

// restTimeMenu = O10C:0691: set the rest time; true to rest.
func (e *Engine) restTimeMenu() bool {
	unit := restMinutes + 1
	for {
		e.showRest(unit)
		key, extra := e.DisplayInput("", "Rest Days Hours Mins Add Subtract Exit", colourLightMagenta, colourGreen, colourWhite, true, false)
		if extra {
			switch key {
			case scanUp:
				key = 'A'
			case scanDown:
				key = 'S'
			case scanLeft:
				unit++
				if unit > restDays {
					unit = restTens
				}
				key = 'X'
			case scanRight:
				unit--
				if unit < restTens {
					unit = restDays
				}
				key = 'X'
			default:
				key = 'X'
			}
		}
		if key == keyEnter {
			key = 'R'
		}
		switch key {
		case 'R':
			return true
		case 'D':
			unit = restDays
		case 'H':
			unit = restHours
		case 'M':
			unit = restTens
		case 'A':
			if unit == restTens {
				e.camp.rest[restMinutes] += 5
			} else {
				e.camp.rest[unit]++
			}
			e.normalizeRest()
		case 'S':
			if unit == restTens {
				e.subtractRest(5, restMinutes)
			} else {
				e.subtractRest(1, unit)
			}
			e.normalizeRest()
		}
		if key == 0 || key == 'E' {
			return false
		}
	}
}

// Five-minute steps to a day of rest (when everyone heals 1 point).
const stepsPerDay = 288

// healTick = O10C:07E1 (showing): every day of rest everyone heals 1 point.
func (e *Engine) healTick(showing bool) {
	e.camp.healSteps++
	if e.camp.healSteps < stepsPerDay {
		return
	}
	healed := false
	for _, c := range e.party {
		if e.healPlayer(c, 1, false) {
			healed = true
		}
	}
	if showing {
		e.showRest(0)
	}
	e.displayStringBg(1, 19, colourGreen, colourBlack, "The Whole Party Is Healed")
	if healed {
		e.playerSummary(e.selected())
	}
	e.pauseForMessage()
	e.clearCombatText()
	e.camp.healSteps = 0
}

// memorizeOne = O10C:089C (c, var learned): with learned
// clear the first waiting spell is learned; set, the level of the next.
func (e *Engine) memorizeOne(c *Character, learned *bool) byte {
	for i := 0; i < len(c.Memorized); i++ {
		if c.Memorized[i] > 127 {
			if *learned {
				return spellLevel(c.Memorized[i])
			}
			c.Memorized[i] -= 128
			e.showRest(0)
			e.spellMessage(c, "has memorized", c.Memorized[i])
			*learned = true
		}
	}
	return 0
}

// scribeOne = O10C:095E (c, var learned): memorizeOne for scroll spells,
// a bag's scrolls included.
func (e *Engine) scribeOne(c *Character, learned *bool) byte {
	for _, it := range append([]*Item(nil), c.Items...) {
		if !e.isScroll(it) {
			continue
		}
		scrolls := []*Item{it}
		if it.Type == itemTypeBag {
			scrolls = append([]*Item(nil), it.Contents...)
		}
		for _, sc := range scrolls {
			for k := 0; k < 3; k++ {
				if sc.Spells[k] <= 128 {
					continue
				}
				if *learned {
					return spellLevel(sc.Spells[k])
				}
				sp := sc.Spells[k] & 127
				c.SpellBook[sp-1] = 1
				e.scrollUsed(c, it, sp)
				e.showRest(0)
				e.spellMessage(c, "has scribed", sp)
				*learned = true
			}
		}
	}
	return 0
}

// spellMessage = O113:6628 cast_a_spell (c, text, spell): who did what
// with which spell.
func (e *Engine) spellMessage(c *Character, text string, sp byte) {
	if e.gameState == stateCombat {
		e.combatName(c, "Casts a Spell", 10, true)
		e.ClearCells(0, 23, 39, 23)
		e.displayStringBg(0, 23, colourGreen, colourBlack, "Spell:"+spellNames[sp])
		return
	}
	e.ClearCells(1, 18, 38, 22)
	e.displayName(c, 1, 19, false)
	e.displayStringBg(int(c.PName[0])+2, 19, colourGreen, colourBlack, text)
	e.displayStringBg(1, 20, colourGreen, colourBlack, spellNames[sp])
	e.pauseForMessage()
	e.clearCombatText()
}

// studyTick = O10C:0B8E: every five minutes the members not waiting for
// their study hours learn what is due.
func (e *Engine) studyTick() {
	for i, c := range e.party {
		if e.camp.study[i+1] > 0 {
			e.camp.study[i+1]--
		}
		if e.camp.study[i+1] == 0 && c.SpellsToLearn == 0 {
			learned := false
			l := e.scribeOne(c, &learned)
			if l == 0 {
				l = e.memorizeOne(c, &learned)
			}
			e.camp.study[i+1] = l * 3
		}
	}
}

// hourTick = O10C:0C3F (var count): every hour the study hours run down.
func (e *Engine) hourTick(count *byte) {
	*count++
	if *count < 12 {
		return
	}
	*count = 0
	for i, c := range e.party {
		if int8(c.SpellsToLearn) > 0 {
			c.SpellsToLearn--
			if c.SpellsToLearn == 0 {
				learned := true
				l := e.scribeOne(c, &learned)
				if l == 0 {
					l = e.memorizeOne(c, &learned)
				}
				e.camp.study[i+1] = l * 3
			}
		}
	}
}

// Area2 words of the encounters during rest.
const (
	gameWordRestInterval = 722 // steps between encounter rolls
	gameWordRestChance   = 723 // percent chance of an encounter
)

// restFor = O10C:0D25 (showing): rest in five-minute steps until
// the time is up, a key stops it or an encounter interrupts (true).
func (e *Engine) restFor(showing bool) bool {
	interrupted := false
	e.camp.study = [maxPartyMembers + 1]uint8{}
	var hours byte
	steps := 0
	if showing {
		e.ClearCells(1, 17, 38, 22)
		e.showRest(0)
	}
	e.resting = 1
	stop := false
	if showing {
		stop = !e.restTimeMenu()
	}
	r := &e.camp.rest
	for !stop && (r[restDays] > 0 || r[restHours] > 0 || r[restTens] > 0 || r[restMinutes] > 0) {
		if showing && e.KeyPressed() {
			e.showRest(0)
			if e.yesNoFrom(answerYes, colourLightMagenta, colourGreen, colourWhite, "Stop Resting? ") == 'Y' {
				stop = true
			} else {
				e.ClearRow24()
			}
		}
		if stop {
			break
		}
		e.subtractRest(5, restMinutes)
		steps++
		if showing && steps >= 5 {
			e.showRest(0)
			steps = 0
		}
		e.AdvanceClock(5, clockMinute)
		e.healTick(showing)
		e.studyTick()
		e.hourTick(&hours)
		if wordAt(e.game, 2*gameWordRestInterval) > 0 {
			e.camp.encounters++
			if e.camp.encounters >= wordAt(e.game, 2*gameWordRestInterval) {
				e.camp.encounters = 0
				if uint16(e.Dice(1, 100)) <= wordAt(e.game, 2*gameWordRestChance) {
					e.clearCombatText()
					e.showRest(0)
					e.displayStringBg(1, 19, colourWhite, colourBlack, "Your repose is suddenly interrupted!")
					stop = true
					interrupted = true
					e.pauseForMessage()
				}
			}
		}
	}
	e.ClearCells(1, 17, 38, 22)
	e.resting = 0
	return interrupted
}
