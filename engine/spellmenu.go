package engine

// The spell lists: memorised spells, the grimoire, scrolls, spells to
// learn (unit O0C2 / O0AF of the reconstruction).

// Spell list kinds (buildSpellList, spellMenu2).
const (
	listMemory    = 0
	listGrimoire  = 1
	listScroll    = 2 // the scroll in scrollItem
	listScrolls   = 3 // every scroll carried
	listToChoose  = 4 // spells the character may learn
	listMemorise  = 5 // spells still to memorise
	listToScribe  = 6
	maxListSpells = 84
	maxScrolls    = 49
)

// Spell menu modes (spellMenu).
const (
	menuCast     = 1
	menuMemorize = 2
	menuScribe   = 3
	menuLearn    = 4
)

// Item classes of scrolls (the first byte of a base item record).
const (
	itemClassFirstScroll  = 11
	itemClassClericScroll = 12
	itemClassLastScroll   = 13
	affectReadMagic       = 16
)

// spellList is the list a spell menu shows: menu lines (level headings
// and spells), the spell number of each spell line in order, how many
// copies of each are memorised, and for scrolls the item of each line.
type spellList struct {
	lines      []*ListNode
	ids        [maxListSpells]byte // DS 19380
	counts     [maxListSpells]byte // DS 19296
	scrolls    [maxScrolls]*Item   // DS 19464
	nScrolls   int                 // DS 19660
	scrollItem *Item               // DS 29366: the scroll being read
}

// isScroll = O0C2:63B8 item_is_scroll.
func (e *Engine) isScroll(it *Item) bool {
	if it == nil {
		return false
	}
	class := e.baseItem(it.Type, 0)
	return class > itemClassFirstScroll-1 && class < itemClassLastScroll+1
}

// spellLineText is "  name" (" *name" for a spell still to memorise).
func spellLineText(spell byte) string {
	mark := "  "
	if spell >= 128 {
		mark = " *"
	}
	return mark + spellNames[spell&127]
}

// spellLines counts the spell lines (not headings) of the list.
func (l *spellList) spellLines() int {
	n := 0
	for _, ln := range l.lines {
		if ln.Flag == 0 {
			n++
		}
	}
	return n
}

// addSpellLine = O0C2:03ED (spell): append the spell at the end of the
// list, with a level heading first when its level differs from the last.
func (e *Engine) addSpellLine(spell byte) {
	l := &e.spells
	lastLevel := byte(0)
	n := l.spellLines()
	if n > 0 {
		lastLevel = spellTable[l.ids[n-1]].level()
	}
	if lvl := spellTable[spell&127].level(); lvl != lastLevel {
		l.lines = append(l.lines, &ListNode{Flag: 1, Text: spellLevelHeadings[lvl]})
	}
	l.lines = append(l.lines, &ListNode{Text: spellLineText(spell)})
	l.ids[n] = spell & 127
}

// addSpellSorted = O0C2:05B9 (spell): insert the spell in level order; a
// spell already listed counts up "(n)" instead.
func (e *Engine) addSpellSorted(spell byte) {
	l := &e.spells
	id := spell & 127
	level := spellTable[id].level()
	var at int // index into l.lines of the spell's line
	slot := 0  // its index into ids/counts
	if len(l.lines) == 0 {
		l.counts = [maxListSpells]byte{}
		l.lines = []*ListNode{{}}
		at = 0
		l.counts[0] = 1
	} else {
		at = 0
		found := false
		for at < len(l.lines) && !found {
			if l.lines[at].Flag != 0 {
				at++
				continue
			}
			if spellTable[l.ids[slot]].level() <= level && l.ids[slot] != id {
				slot++
				at++
			} else {
				found = true
			}
		}
		if at >= len(l.lines) || l.ids[slot] != id {
			l.lines = append(l.lines, nil)
			copy(l.lines[at+1:], l.lines[at:])
			l.lines[at] = &ListNode{}
			carry := byte(1)
			for k := slot; k < maxListSpells; k++ {
				carry, l.counts[k] = l.counts[k], carry
			}
		} else {
			l.counts[slot]++
		}
	}
	text := spellLineText(spell)
	if l.counts[slot] > 1 {
		text += " (" + itoa(int(l.counts[slot])) + ")"
	}
	l.lines[at].Text, l.lines[at].Flag = text, 0
	if l.ids[slot] != id {
		carry := l.ids[slot]
		l.ids[slot] = id
		for k := slot + 1; k < maxListSpells; k++ {
			carry, l.ids[k] = l.ids[k], carry
		}
	}
}

// addScrollSpells = O0C2:0912 (unknownOnly): list the spells on the scroll
// in scrollItem; with unknownOnly only those still to be scribed (+128).
// A scroll a caster can read is identified first.
func (e *Engine) addScrollSpells(unknownOnly bool) {
	l := &e.spells
	it := l.scrollItem
	c := e.selected()
	if e.FindAffect(c, affectReadMagic) != nil ||
		(int8(c.Levels[clsCleric]) > 0 || int8(c.OldLevels[clsCleric]) > int8(c.MulticlassLevel)) &&
			e.baseItem(it.Type, 0) == itemClassClericScroll {
		it.Hidden = 0
	}
	if it.Hidden != 0 {
		return
	}
	for i := range it.Spells {
		s := it.Spells[i]
		if unknownOnly && s > 128 || !unknownOnly && s > 0 {
			e.addSpellLine(s)
			l.scrolls[l.nScrolls] = it
			l.nScrolls++
		}
	}
}

// addAllScrolls = O0C2:09F4 (unknownOnly): the spells of every scroll carried.
func (e *Engine) addAllScrolls(unknownOnly bool) {
	l := &e.spells
	l.scrolls = [maxScrolls]*Item{}
	l.nScrolls = 0
	for _, it := range e.selected().Items {
		l.scrollItem = it
		if e.isScroll(it) {
			e.addScrollSpells(unknownOnly)
		}
	}
}

// buildSpellList = O0C2:0A74 (kind): fill the list; false when it is empty.
func (e *Engine) buildSpellList(kind int) bool {
	l := &e.spells
	l.lines = nil
	l.ids = [maxListSpells]byte{}
	c := e.selected()
	sorted := true
	switch kind {
	case listMemory:
		for _, s := range c.Memorized {
			if s != 0 && e.canUseSpell(c, s&127) && s < 128 {
				e.addSpellSorted(s)
			}
		}
	case listMemorise:
		for _, s := range c.Memorized {
			if s >= 128 && e.canUseSpell(c, s&127) {
				e.addSpellSorted(s)
			}
		}
	case listGrimoire:
		for n := 1; n < len(spellTable); n++ {
			if c.SpellBook[n-1] != 0 && e.canUseSpell(c, byte(n)) {
				e.addSpellSorted(byte(n))
			}
		}
	case listScroll:
		e.addScrollSpells(false)
		sorted = false
	case listScrolls:
		e.addAllScrolls(false)
		sorted = false
	case listToScribe:
		e.addAllScrolls(true)
		sorted = false
	case listToChoose:
		for n := 1; n < len(spellTable); n++ {
			s := spellTable[n]
			if c.SpellSlots[int(s.class())*5+int(s.level())-1] != 0 && e.canUseSpell(c, byte(n)) && c.SpellBook[n-1] == 0 {
				e.addSpellSorted(byte(n))
			}
		}
	}
	if len(l.lines) == 0 {
		return false
	}
	if sorted {
		e.addLevelHeadings()
	}
	return true
}

// addLevelHeadings is the tail of O0C2:0A74 for the sorted lists: a heading
// before the first spell and wherever the level rises.
func (e *Engine) addLevelHeadings() {
	l := &e.spells
	var out []*ListNode
	level := spellTable[l.ids[0]].level()
	out = append(out, &ListNode{Flag: 1, Text: spellLevelHeadings[level]})
	slot := 0
	for _, ln := range l.lines {
		if ln.Flag == 0 {
			if lv := spellTable[l.ids[slot]].level(); lv > level {
				level = lv
				out = append(out, &ListNode{Flag: 1, Text: spellLevelHeadings[level]})
			}
			slot++
		}
		out = append(out, ln)
	}
	l.lines = out
}

// canUseSpell = O0C2:001E (c, spell): whether the character's classes can
// cast the spell (wisdom or intelligence 9 and the class level for it).
func (e *Engine) canUseSpell(c *Character, spell byte) bool {
	old := oldClassUsable(c)
	switch spellTable[spell&127].class() {
	case spellClassCleric:
		return c.Stats[statWis].Value > 8 && (int8(c.Levels[clsCleric]) > 0 ||
			old && int8(c.OldLevels[clsCleric]) > 0 || int8(c.Levels[clsPaladin]) > 8 ||
			int8(c.OldLevels[clsPaladin]) > 8 && old)
	case spellClassDruid:
		return c.Stats[statWis].Value > 8 && (int8(c.Levels[clsRanger]) > 6 ||
			old && int8(c.OldLevels[clsRanger]) > 6)
	case spellClassMage:
		return c.Stats[statInt].Value > 8 && (c.Race != raceHuman || c.Slots[slotBodyArmour] == nil ||
			e.gameState != stateCombat || int8(c.Levels[clsRanger]) > 8 ||
			old && int8(c.OldLevels[clsRanger]) > 8 && (int8(c.Levels[clsMage]) > 0 || old && int8(c.OldLevels[clsMage]) > 0))
	}
	return false
}

// Readied slot of body armour (magic-users can't cast in it in combat).
const slotBodyArmour = 2

// spellMenu = O0C2:0211 (mode, var index): show the list built before and
// return the chosen spell (0 for Exit / Escape).
func (e *Engine) spellMenu(mode int, index *int) byte {
	l := &e.spells
	title := map[int]string{menuCast: "Cast", menuMemorize: "Memorize", menuScribe: "Scribe", menuLearn: "Learn"}[mode]
	prompt := ""
	if mode >= menuCast && mode <= menuLearn {
		prompt = "Choose Spell: "
	}
	bottom := 22
	if mode == menuMemorize {
		bottom = 15
	}
	exit := mode != menuLearn
	redraw := false
	if *index < 0 {
		redraw = true
		*index = 0
	}
	if mode == menuLearn || mode == menuCast {
		redraw = true
	}
	var sel *ListNode
	var k byte
	for {
		k = e.ListMenu(prompt, title, colourLightMagenta, colourGreen, colourWhite, 1, 5, 38, bottom, l.lines, exit, &redraw, index, &sel)
		switch k {
		case 0, 13, 'C', 'E', 'L', 'M', 'S':
		default:
			continue
		}
		break
	}
	slot := 0
	for _, ln := range l.lines {
		if ln == sel {
			break
		}
		if ln.Flag == 0 {
			slot++
		}
	}
	spell := byte(0)
	if k != 0 && k != 'E' {
		spell = l.ids[slot]
	}
	if mode == menuScribe {
		l.scrollItem = l.scrolls[slot]
	}
	l.lines = nil
	return spell
}

// spellMenu2 = O0AF:347C (kind, mode, var index, var shown): frame the
// screen, title the list after its kind and run the menu; shown tells
// whether there was anything to list.
func (e *Engine) spellMenu2(kind, mode int, index *int, shown *bool) byte {
	heading := map[int]string{listMemory: "in Memory", listGrimoire: "in Grimoire", listScroll: "on Scroll",
		listScrolls: "on Scrolls", listToChoose: "to Choose", listMemorise: "to Memorize", listToScribe: "to Scribe"}[kind]
	*shown = e.buildSpellList(kind)
	if !*shown {
		return 0
	}
	if *index < 0 || mode == menuCast {
		switch {
		case e.gameState == stateCombat:
			e.drawScreenFrame()
		case mode == menuMemorize:
			e.drawMemoriseFrame()
		default:
			e.drawListFrame()
		}
	}
	c := e.selected()
	e.displayName(c, 1, 1, true)
	e.displayStringBg(int(c.PName[0])+4, 1, colourGreen, colourBlack, "Spells "+heading)
	return e.spellMenu(mode, index)
}

// learnNewSpell is the training's spell choice: until a spell is picked or
// there is nothing to choose.
func (e *Engine) learnNewSpell(c *Character) {
	index := -1
	for {
		var shown bool
		spell := e.spellMenu2(listToChoose, menuLearn, &index, &shown)
		if spell != 0 {
			c.SpellBook[spell-1] = 1
			return
		}
		if !shown {
			return
		}
	}
}
