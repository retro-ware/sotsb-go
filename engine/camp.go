package engine

// Encamp (unit O09D) and resting (unit O0B6): the camp menu, Magic (cast,
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

// studyMinutes = O09D:0032 (c): minutes of study for the spells waiting to
// be memorized (+128) or scribed; the base hours (4, or 6 for spells above
// 2nd level) go to the record's SpellsToLearn.
func (e *Engine) studyMinutes(c *Character) uint16 {
	var maxMem, maxScr, sumMem, sumScr byte
	for i := 0; i < len(c.Memorized); i++ {
		if s := c.Memorized[i]; s > 127 {
			l := spellLevel(s)
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
		for k := 0; k < 3; k++ {
			if s := it.Spells[k]; s > 127 {
				l := spellLevel(s)
				if l > maxScr {
					maxScr = l
				}
				sumScr += l
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
	return uint16(hours)*60 + uint16(sumMem)*15 + uint16(sumScr)*15
}

// forgetPending = O09D:018B (c): the spells still to be memorized go.
func forgetPending(c *Character) {
	for i := range c.Memorized {
		if c.Memorized[i] > 127 {
			c.Memorized[i] = 0
		}
	}
	c.SpellsToLearn = 0
}

// stopScribing = O09D:01CE (c): no scroll spell is being scribed.
func (e *Engine) stopScribing(c *Character) {
	for _, it := range c.Items {
		if e.isScroll(it) {
			for k := range it.Spells {
				it.Spells[k] &= 127
			}
		}
	}
}

// stopStudy = O09D:0246: nobody studies any more.
func (e *Engine) stopStudy() {
	for _, c := range e.party {
		forgetPending(c)
		e.stopScribing(c)
	}
}

// slotsLeft = O09D:028E (level, class): spells of the class and level the
// selected member may still memorize.
func (e *Engine) slotsLeft(level, class byte) byte {
	c := e.selected()
	var used byte
	for i := 0; i < len(c.Memorized); i++ {
		if s := c.Memorized[i]; s > 0 && spellLevel(s) == level && spellClass(s) == class {
			used++
		}
	}
	return c.SpellSlots[int(class)*5+int(level)-1] - used
}

// Why a member cannot study (canStudy's mode).
const (
	studyCast     = 1
	studyMemorize = 2
	studyScribe   = 3
)

// canStudy = O09D:03A0 (mode): not where spells are blocked, and not when
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

// castMenu = O09D:04D5 cast_spell: cast memorized spells until none is
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

// The spell class headings of "can memorize:".
var memorizeHeadings = [3]string{"    Cleric Spells:", "     Druid Spells:", "Magic-User Spells:"}

// memorizeInfo = O09D:05D4: "can memorize:" with the free slots per class
// and level; false when there are none.
func (e *Engine) memorizeInfo() bool {
	c := e.selected()
	any := false
	var counts [5][3]string
	var has [3]bool
	for class := 0; class <= 2; class++ {
		for level := 1; level <= 5; level++ {
			counts[level-1][class] = itoa(int(e.slotsLeft(byte(level), byte(class))))
			if c.SpellSlots[class*5+level-1] == 0 {
				counts[level-1][class] = " "
			} else {
				has[class] = true
				any = true
			}
		}
	}
	if any {
		e.combatName(c, "can memorize:", 10, false)
		row := 3
		for class := 0; class <= 2; class++ {
			if !has[class] {
				continue
			}
			e.displayStringBg(1, row+17, colourGreen, colourBlack, memorizeHeadings[class])
			x := 19
			for level := 1; level <= 5; level++ {
				e.displayStringBg(x+1, row+17, colourGreen, colourBlack, counts[level-1][class])
				x += 3
			}
			row++
		}
	}
	return any
}

// restMenu = O09D:07C2 (var interrupted): rest as long as the longest
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

// sortSpells = O09D:086F: the selected member's memorized spells in spell
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

// memorizeMenu = O09D:0955 memorize_spell.
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
		if e.YesNo(colourYellow, colourGreen, colourWhite, "Memorize These Spells? ") == 'N' {
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
		if shown && e.YesNo(colourYellow, colourGreen, colourWhite, "Memorize these spells? ") == 'N' {
			forgetPending(e.selected())
		}
	}
	if redraw {
		e.loadPicture()
	}
}

// scribeMenu = O09D:0BAB scribe_spell.
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
		if e.YesNo(colourYellow, colourGreen, colourWhite, "Scribe These Spells? ") == 'N' {
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
			if e.isScroll(it) {
				for k := 0; k < 3; k++ {
					if it.Spells[k] > 127 && it.Spells[k]&127 == sp {
						e.printMessage("You are already scibing that spell")
						busy = true
					}
				}
			}
		}
		if busy {
			continue
		}
		if c.SpellSlots[int(spellClass(sp))*5+int(spellLevel(sp))-1] == 0 {
			e.printMessage("You can not scribe that spell.")
			continue
		}
		marked := false
		for _, it := range c.Items {
			if marked {
				break
			}
			for k := 0; k < 3 && !marked; k++ {
				if it.Spells[k] == sp {
					it.Spells[k] |= 128
					marked = true
				}
			}
		}
	}
	if index != -1 {
		index = -1
		e.spellMenu2(6, 0, &index, &shown)
		if shown && e.YesNo(colourYellow, colourGreen, colourWhite, "Scribe these spells? ") == 'N' {
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

// displayEffects = O09D:103A display_magic_effects: everyone's spell
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

// magicMenu = O09D:14D5 (var interrupted).
func (e *Engine) magicMenu(interrupted *bool) {
	key := byte(' ')
	for !*interrupted && key != 0 && key != 'E' {
		var extra bool
		key, extra = e.DisplayInput("", "Cast Memorize Scribe Display Rest Exit", colourLightMagenta, colourGreen, colourWhite, true, true)
		if extra {
			e.cycleSelected(key)
			e.playerSummary(e.selected())
			continue
		}
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
}

// movePrev = O09D:158D: the selected member swaps places with the one
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

// moveNext = O09D:16E5: swaps with the one after; the last goes first.
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

// orderMenu = O09D:1864 reorder_party: Select a member, then Place it with
// Home / End.
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
				e.cycleSelected(key)
				e.playerSummary(e.selected())
			} else {
				switch key {
				case scanHome:
					e.movePrev()
				case scanEnd:
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

// dropMember = O09D:19C9 drop_player: the selected member leaves the
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
	e.combatName(c, "Breathes A sigh of relief", 10, true)
}

// Game speed limits.
const maxGameSpeed = 9

// campSpeedMenu = O09D:1B7B game_speed: Faster / Slower (the arrow keys too).
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

// alterMenu = O09D:1D38.
func (e *Engine) alterMenu() {
	key := byte(' ')
	for key != 0 && key != 'E' {
		var extra bool
		key, extra = e.DisplayInput("Alter: ", "Order Drop Speed Icon Pics Exit", colourLightMagenta, colourGreen, colourWhite, true, true)
		if extra {
			e.cycleSelected(key)
			e.playerSummary(e.selected())
			continue
		}
		switch key {
		case 'O':
			e.orderMenu()
		case 'D':
			e.dropMember()
		case 'S':
			e.campSpeedMenu()
		case 'I':
			e.iconBuilder()
			e.loadPicture()
		case 'P':
			for {
				menu := "Pics off  "
				if e.picturesOn {
					menu = "Pics on  "
					if e.loadingMessage {
						menu += "Animation on  "
					} else {
						menu += "Animation off  "
					}
				}
				menu += "Exit"
				k, _ := e.DisplayInput("", menu, colourLightMagenta, colourGreen, colourWhite, false, true)
				switch k {
				case 'P':
					e.picturesOn = !e.picturesOn
				case 'A':
					e.loadingMessage = !e.loadingMessage
				}
				if k == 0 || k == 'E' {
					break
				}
			}
		}
	}
}

// Cure spells Fix casts (memorized slots), and the hit dice they heal.
const (
	spellCureLight    = 3
	spellCureSerious  = 58
	spellCureCritical = 71
)

// woundsTotal = O09D:208F: the party's hit points short of the maximum.
func (e *Engine) woundsTotal() uint16 {
	var w uint16
	for _, c := range e.party {
		w += uint16(c.MaxHP - c.HP)
	}
	return w
}

// fixMenu = O09D:2338 (var interrupted): the party's clerics cast their
// cure spells, rest for them to be memorized again and cast them again.
func (e *Engine) fixMenu() bool {
	if e.woundsTotal() == 0 {
		return false
	}
	var healing uint16
	healing += e.castCures()
	if e.woundsTotal() == 0 {
		e.playerSummary(e.selected())
		e.statusLine()
		return false
	}
	saved := e.camp.rest
	counts := e.fixRestTime(healing)
	interrupted := e.restFor(false)
	if !interrupted {
		healing += e.rollCures(counts)
		e.applyHealing(&healing)
		e.playerSummary(e.selected())
		e.statusLine()
		e.camp.rest = saved
	}
	return interrupted
}

// castCures = O09D:1F22: every okay member casts its memorized cures; the
// hit points rolled.
func (e *Engine) castCures() uint16 {
	var total uint16
	for _, c := range e.party {
		if c.Status != statusOkay {
			continue
		}
		for i := 0; i < len(c.Memorized); i++ {
			switch c.Memorized[i] {
			case spellCureLight:
				total += uint16(e.Dice(1, 8))
			case spellCureSerious:
				total += uint16(e.Dice(2, 8) + 1)
			case spellCureCritical:
				total += uint16(e.Dice(3, 8) + 3)
			}
		}
	}
	return total
}

// fixCounts: the cure spells the party can cast per round of Fix.
type fixCounts struct{ light, serious, critical uint16 }

// fixRestTime = O09D:20ED (healed): how long the clerics study their cures
// again, shortened when fewer rounds are needed.
func (e *Engine) fixRestTime(healed uint16) fixCounts {
	var n fixCounts
	var longest uint16
	for _, c := range e.party {
		var mins, light, serious, critical uint16
		if c.Status == statusOkay {
			n.light += uint16(c.SpellSlots[0])
			light = uint16(c.SpellSlots[0]) * 15
			n.serious += uint16(c.SpellSlots[3])
			serious = uint16(c.SpellSlots[3]) * 60
			n.critical += uint16(c.SpellSlots[4])
			critical = uint16(c.SpellSlots[4]) * 75
		}
		if light > 0 {
			mins = 240
			mins += 27
		}
		if serious+critical > 0 {
			mins = 360
			if critical > 0 {
				mins += 78
			} else {
				mins += 34
			}
		}
		mins += light + serious + critical
		if longest < mins {
			longest = mins
		}
	}
	if w := e.woundsTotal(); w < healed && w > 0 {
		longest /= uint16(byte(healed / w))
	}
	r := &e.camp.rest
	r[restHours] = longest / 60
	r[restTens] = (longest - 60*r[restHours]) / 10
	r[restMinutes] = longest % 10
	return n
}

// rollCures = O09D:1FDD: the cures cast after the rest.
func (e *Engine) rollCures(n fixCounts) uint16 {
	var total uint16
	for k := uint16(1); k <= n.light; k++ {
		total += uint16(e.Dice(1, 8))
	}
	for k := uint16(1); k <= n.serious; k++ {
		total += uint16(e.Dice(2, 8) + 1)
	}
	for k := uint16(1); k <= n.critical; k++ {
		total += uint16(e.Dice(3, 8) + 3)
	}
	return total
}

// applyHealing = O09D:2280 (var healing): the points go to the wounded in
// party order.
func (e *Engine) applyHealing(healing *uint16) {
	for _, c := range e.party {
		if c.MaxHP <= c.HP {
			continue
		}
		need := uint16(c.MaxHP - c.HP)
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

// campMenu = O09D:242B make_camp (var interrupted).
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
		key, extra = e.DisplayInput("Camp:", "Save View Magic Rest Alter Fix Exit", colourLightMagenta, colourGreen, colourWhite, true, true)
		if extra {
			e.cycleSelected(key)
			e.playerSummary(e.selected())
			continue
		}
		switch key {
		case 'S':
			e.SaveGame()
			if e.YesNo(colourYellow, colourGreen, colourWhite, "Quit TO DOS ") == 'Y' {
				e.Exit()
			}
		case 'V':
			e.menuSel = 1
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

// ---- O0B6: resting ----

// subtractRest = O0B6:049F (count, unit): take count units off the rest
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

// carryClock = O0B6:0317 (time): each unit at its limit carries one into
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
			continue
		}
		t[i+1]++
		t[i] -= clockLimits[i]
	}
}

// normalizeRest = O0B6:03C8: carries, months fold into days, at most 99.
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

// twoDigits = O0B6:058A.
func twoDigits(n uint16) string {
	s := itoa(int(n))
	if n < 10 {
		s = "0" + s
	}
	return s[:2]
}

// showRest = O0B6:0615 (unit): "Rest Time:" days:hours:minutes, the unit
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

// restTimeMenu = O0B6:0751: set the rest time; true to rest.
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

// healTick = O0B6:08A1 (showing): every day of rest everyone heals 1 point.
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
	e.gameDelay()
	e.clearCombatText()
	e.camp.healSteps = 0
}

// memorizeOne = O0B6:0960 rest_memorize (c, var learned): with learned
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

// scribeOne = O0B6:0A22 reset_scribe: memorizeOne for scroll spells.
func (e *Engine) scribeOne(c *Character, learned *bool) byte {
	for _, it := range append([]*Item(nil), c.Items...) {
		if !e.isScroll(it) {
			continue
		}
		for k := 0; k < 3; k++ {
			if it.Spells[k] > 128 {
				if *learned {
					return spellLevel(it.Spells[k])
				}
				sp := it.Spells[k] & 127
				c.SpellBook[sp-1] = 1
				e.useScrollSpell(c, it, sp)
				e.showRest(0)
				e.spellMessage(c, "has scribed", sp)
				*learned = true
				return 0
			}
		}
	}
	return 0
}

// useScrollSpell = O0C2:63FF (c, scroll, spell): the spell leaves the
// scroll; an empty scroll goes.
func (e *Engine) useScrollSpell(c *Character, it *Item, sp byte) {
	k := -1
	for i := 0; i < 3; i++ {
		if it.Spells[i]&127 == sp {
			k = i
		}
	}
	if k < 0 {
		return
	}
	it.Spells[k] = 0
	it.Words[1]--
	if it.Words[1] < wordScrollSpells {
		e.loseItem(c, it)
	}
}

// wordScrollSpells: "With 1 Spell" is word 210; below it the scroll is empty.
const wordScrollSpells = 210

// spellMessage = O0C2:64A1 cast_a_spell (c, text, spell): who did what
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
	e.gameDelay()
	e.clearCombatText()
}

// studyTick = O0B6:0B4D: every five minutes the members not waiting for
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

// hourTick = O0B6:0C03 (var count): every hour the study hours run down.
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

// restFor = O0B6:0CEF reseting (showing): rest in five-minute steps until
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
			if e.YesNo(colourLightMagenta, colourGreen, colourWhite, "Stop Resting? ") == 'Y' {
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
					e.gameDelay()
				}
			}
		}
	}
	e.ClearCells(1, 17, 38, 22)
	e.resting = 0
	return interrupted
}
