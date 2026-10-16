package engine

// Human dual-classing (unit O164 of the reconstruction).

// mayChangeClass tells whether the party menu offers "Human Change Classes"
// for c: a human who has not changed class yet (O164:0D49 and 0C4D).
func (e *Engine) mayChangeClass(c *Character) bool {
	return c != nil && isHuman(c) && activeClass(c) == noClass
}

// NPC tells whether c is a non-player character (control value above 127).
func (c *Character) NPC() bool { return c.Control >= 128 }

// gameFinished: the end of the game was reached (area word 1018, set to 255
// by the ending): the party may not begin adventuring again.
func (e *Engine) gameFinished() bool { return wordAt(e.area, gameFinishedWord) != 0 }

const gameFinishedWord = 1018

// Ability scores a human needs to change class: every prime requisite
// (class minimum above 8) of the old class above 14, of the new class above 16.
const (
	primeRequisite = 9
	oldClassPrime  = 14
	newClassPrime  = 16
)

// mayTakeClass = O164:0D97 (nested in O164:0F49): may c change to class?
func mayTakeClass(c *Character, class int) bool {
	cur := currentClass(c)
	if class == cur {
		return false
	}
	for k := 0; k < len(c.Stats); k++ {
		if classStatMinimum[cur][k] >= primeRequisite && c.Stats[k].Base <= oldClassPrime {
			return false
		}
		if classStatMinimum[class][k] >= primeRequisite && c.Stats[k].Base <= newClassPrime {
			return false
		}
	}
	aligns := classAlignments[class][1 : 1+int(classAlignments[class][0])]
	for _, a := range aligns {
		if a == c.Alignment {
			return true
		}
	}
	return false
}

// ChangeClass = O164:0F49: a human picks a new class; the old class's
// levels are kept apart and the character starts again at level 1. The
// selected character must be a human who has not changed class before.
// Returns the new class, noClass when there is none.
func (e *Engine) ChangeClass(c *Character) int {
	doesNotQualify := func() { e.statusMessage(c.Name()+" doesn't qualify.", colourRed, colourWhite) }
	if sel := e.selected(); !isHuman(sel) || activeClass(sel) != noClass {
		doesNotQualify()
		return noClass
	}
	race := int(c.Race)
	list := []*ListNode{{Flag: 1, Text: "Pick New Class"}}
	for _, cl := range raceClasses[race][1 : 1+int(raceClasses[race][0])] {
		if mayTakeClass(c, int(cl)) {
			list = append(list, &ListNode{Text: classNames[cl]})
		}
	}
	if len(list) == 1 {
		doesNotQualify()
		return noClass
	}
	index, redraw := 1, true
	var sel *ListNode
	for {
		k := e.ListMenu("", "Select", colourLightMagenta, colourGreen, colourWhite, 1, 2, 38, 22, list, true, &redraw, &index, &sel)
		if k == 0 {
			return noClass
		}
		if k == 'S' {
			break
		}
	}
	c.Exp = 0
	c.BaseAttack[0] = 2
	newClass := 0
	for newClass < classCount && classNames[newClass] != sel.Text {
		newClass++
	}
	c.OldLevels[currentClass(c)] = byte(firstClassLevel(c))
	c.MulticlassLevel = c.MaxLevel
	c.MaxLevel = 1
	c.Levels[currentClass(c)] = 0
	c.Levels[newClass] = 1
	c.SpellSlots = [len(c.SpellSlots)]byte{}
	switch newClass {
	case clsCleric:
		c.SpellSlots[slotIndex(spellClassCleric, 1)] = 1
	case clsMage:
		c.SpellSlots[slotIndex(spellClassMage, 1)] = 1
		for _, s := range []int{spellDetectMagicMU, spellReadMagic, spellEnlarge, spellSleep} {
			c.SpellBook[s-1] = 1
		}
	}
	c.Class = byte(newClass)
	e.statusMessage(c.Name()+" is now a 1st level "+classNames[newClass]+".", colourGreen, colourBlack)
	c.Memorized = [len(c.Memorized)]byte{}
	e.recalcClassBonuses(c)
	e.clericSlots(c)
	intelligenceSlots(c)
	e.savingThrows(c)
	e.thiefSkills(c)
	for _, it := range c.Items {
		if e.baseItem(it.Type, itemClassFlags)&c.ClassMask == 0 && it.Cursed == 0 {
			it.Readied = 0
		}
	}
	return newClass
}
