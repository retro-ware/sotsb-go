package engine

// Human dual-classing (unit O0FE of the reconstruction).

// raceHuman is the Race value of humans.
const raceHuman = 7

// noClass: activeClass found no class in progress.
const noClass = 17

// activeClass = O0FE:12AA: for a human, the first class with levels,
// if that class is still active (positive level); noClass otherwise and
// for every other race.
func activeClass(c *Character) int {
	if c.Race != raceHuman {
		return noClass
	}
	k := 0
	for k < clsMonk && c.OldLevels[k] == 0 {
		k++
	}
	if int8(c.OldLevels[k]) > 0 {
		return k
	}
	return noClass
}

// mayChangeClass tells whether the party menu offers "Human Change Classes"
// for c: a human who has not changed class yet (O0FE:13B2 and 12AA).
func (e *Engine) mayChangeClass(c *Character) bool {
	return c != nil && c.Race == raceHuman && activeClass(c) == noClass
}

// NPC tells whether c is a non-player character (control value above 127).
func (c *Character) NPC() bool { return c.Control >= 128 }

// gameFinished: the end of the game was reached (area word 1018, set to 255
// by the ending): the party may not begin adventuring again.
func (e *Engine) gameFinished() bool { return wordAt(e.area, gameFinishedWord) != 0 }

const gameFinishedWord = 1018

// Ability scores a human needs to change class: every prime requisite
// (class minimum above 8) of the old class at least 15, of the new class at least 17.
const (
	primeRequisite = 9
	oldClassPrime  = 15
	newClassPrime  = 17
)

// mayTakeClass = O0FE:0D3E (nested in multiclass): may c change to class?
func mayTakeClass(c *Character, class int) bool {
	cur := currentClass(c)
	for k := 0; k < len(c.Stats); k++ {
		if classStatMinimum[cur][k] >= primeRequisite && c.Stats[k].Base < oldClassPrime {
			return false
		}
	}
	if class == cur {
		return false
	}
	for k := 0; k < len(c.Stats); k++ {
		if classStatMinimum[class][k] >= primeRequisite && c.Stats[k].Base < newClassPrime {
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

// ChangeClass = O0FE:0EED multiclass: a human picks a new class; the old
// class's levels are kept apart and the character starts again at level 1.
func (e *Engine) ChangeClass(c *Character) {
	race := int(c.Race)
	list := []*ListNode{{Flag: 1, Text: "Pick New Class"}}
	for _, cl := range raceClasses[race][1 : 1+int(raceClasses[race][0])] {
		if mayTakeClass(c, int(cl)) {
			list = append(list, &ListNode{Text: classNames[cl]})
		}
	}
	if len(list) == 1 {
		e.statusMessage(c.Name()+" doesn't qualify.", colourWhite, colourBlue)
		return
	}
	index, redraw := 1, true
	var sel *ListNode
	for {
		k := e.ListMenu("", "Select", colourLightMagenta, colourGreen, colourWhite, 1, 2, 38, 22, list, true, &redraw, &index, &sel)
		if k == 0 {
			return
		}
		if k == 'S' {
			break
		}
	}
	c.Exp = 0
	c.BaseAttack[0] = 2
	newClass := 0
	for newClass < clsMonk+1 && classNames[newClass] != sel.Text {
		newClass++
	}
	old := currentClass(c)
	c.OldLevels[old] = byte(firstClassLevel(c))
	c.MulticlassLevel = c.MaxLevel
	c.MaxLevel = 1
	c.Levels[currentClass(c)] = 0
	c.Levels[newClass] = 1
	c.SpellSlots = [15]byte{}
	switch newClass {
	case clsCleric:
		c.SpellSlots[slotsCleric] = 1
	case clsMage:
		c.SpellSlots[slotsMage] = 1
		for _, s := range []int{spellDetectMagicMU, spellReadMagic, spellEnlarge, spellSleep} {
			c.SpellBook[s-1] = 1
		}
	}
	c.Class = byte(newClass)
	e.statusMessage(c.Name()+" is now a 1st level "+classNames[newClass]+".", colourGreen, colourBlack)
	c.Memorized = [len(c.Memorized)]byte{}
	e.recalcClassBonuses(c)
	e.wisdomSlots(c, true)
	e.savingThrows(c)
	e.thiefSkills(c)
	for _, it := range c.Items {
		if e.baseItem(it.Type, itemClassFlags)&c.ClassMask == 0 && it.Cursed == 0 {
			it.Readied = 0
		}
	}
}
