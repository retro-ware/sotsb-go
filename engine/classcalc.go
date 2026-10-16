package engine

// Class-derived values of a character: THAC0, spell slots, saving throws,
// thief skills (unit O0FE of the reconstruction).

// itemEffect is a magic item's special power (the last name word: 128 + power).
func itemEffect(it *Item) byte { return it.Spells[2] }

// Special item powers (128 + n in itemEffect).
const (
	effectFreeAction = 2  // gauntlets: thief skills as level 4 and +10
	effectSaveBonus  = 6  // saving throw bonus as a dwarf or halfling
	effectThievery   = 11 // thief skills as at least level 5 / 7
	effectWizardry   = 1  // ring of wizardry: doubles the first magic-user slots
	itemPowerFlag    = 128
)

// readiedEffect reports whether a readied item has the given power.
func readiedEffect(c *Character, power byte) bool {
	for _, it := range c.Items {
		if it.Readied != 0 && itemEffect(it) > itemPowerFlag && itemEffect(it)&(itemPowerFlag-1) == power {
			return true
		}
	}
	return false
}

// firstClassLevel = O0FE:1362: a human's level in the first class with
// levels; 0 for the other races.
func firstClassLevel(c *Character) int8 {
	if c.Race != raceHuman {
		return 0
	}
	k := 0
	for k < clsMonk && c.Levels[k] == 0 {
		k++
	}
	return int8(c.Levels[k])
}

// currentClass = O0FE:1306: a human's class in progress (noClass for
// others and for a human with no levels).
func currentClass(c *Character) int {
	if c.Race != raceHuman {
		return noClass
	}
	k := 0
	for k < clsMonk && c.Levels[k] == 0 {
		k++
	}
	if int8(c.Levels[k]) < 1 {
		return noClass
	}
	return k
}

// oldClassUsable = O0FE:13D1: a dual-classed human whose new class has
// passed the old one may use the old class's abilities again.
func oldClassUsable(c *Character) bool {
	return int8(c.MulticlassLevel) < firstClassLevel(c)
}

// Fighters, paladins (above 6) and rangers (above 7) attack 3 times per 2 rounds.
const extraAttacks = 3

// recalcClassBonuses = O0FE:03C6: THAC0, highest level and attacks from the
// class levels, then spell slots, saves, thief skills and the class mask;
// readied items the classes may no longer use (and not cursed) are put away.
func (e *Engine) recalcClassBonuses(c *Character) {
	c.THAC0Base = 0
	for k := 0; k <= clsMonk; k++ {
		lvl := int8(c.Levels[k])
		c.THAC0Base = max(c.THAC0Base, thac0At(k, lvl))
		if int8(c.MaxLevel) < lvl {
			c.MaxLevel = byte(lvl)
		}
		switch {
		case k == clsFighter && lvl > 6, k == clsRanger && lvl > 7, k == clsPaladin && lvl > 6:
			c.BaseAttack[0] = extraAttacks
		}
	}
	e.calcSpellSlots(c)
	e.savingThrows(c)
	if int8(c.Levels[clsThief]) > 0 {
		e.thiefSkills(c)
	}
	c.ClassMask = 0
	for k := 0; k <= clsMonk; k++ {
		old := int8(c.OldLevels[k])
		if int8(c.Levels[k]) > 0 || old > 0 && old < int8(c.MaxLevel) {
			c.ClassMask += classItemFlags[k]
		}
	}
	for _, it := range c.Slots {
		if it != nil && e.baseItem(it.Type, itemClassFlags)&c.ClassMask == 0 && it.Cursed == 0 {
			it.Readied = 0
		}
	}
	if oldClassUsable(c) {
		for k := 0; k <= clsMonk; k++ {
			old := int8(c.OldLevels[k])
			switch {
			case (k == clsFighter || k == clsPaladin) && old > 6, k == clsRanger && old > 7:
				c.BaseAttack[0] = extraAttacks
			}
			c.THAC0Base = max(c.THAC0Base, thac0At(k, old))
		}
		if int8(c.OldLevels[clsFighter]) > 6 || int8(c.OldLevels[clsRanger]) > 7 || int8(c.OldLevels[clsPaladin]) > 6 {
			c.BaseAttack[0] = extraAttacks
		}
		if int8(c.OldLevels[clsThief]) > 0 {
			e.thiefSkills(c)
		}
	}
}

// itemClassFlags: the byte of a base item record holding the classes that may use it.
const itemClassFlags = 13

// thac0At reads the THAC0 table like the original: the flat table indexed
// class*13 + level (levels past the row run into the next class's row).
func thac0At(class int, level int8) byte {
	i := class*len(thac0Table[0]) + int(level)
	return thac0Table[i/len(thac0Table[0])][i%len(thac0Table[0])]
}

// Spell slot indexes: cleric levels 1..5, druid 1..5, magic-user 1..5.
const (
	slotsCleric = 0
	slotsDruid  = 5
	slotsMage   = 10
)

// calcSpellSlots = O0FE:000F: the spells per day of each spell class, and
// the spells clerics and rangers know without learning them.
func (e *Engine) calcSpellSlots(c *Character) {
	c.SpellSlots = [15]byte{}
	for k := 0; k <= clsMonk; k++ {
		lvl := int8(c.Levels[k])
		if oldClassUsable(c) && c.OldLevels[k] != 0 {
			lvl = int8(c.OldLevels[k])
		}
		if lvl <= 0 {
			continue
		}
		switch k {
		case clsCleric:
			c.SpellSlots[slotsCleric]++
			for l := int8(2); l <= lvl; l++ {
				for s := 1; s <= 5; s++ {
					c.SpellSlots[slotsCleric+s-1] += clericSlotData[int(l)*5+s]
				}
			}
			e.wisdomSlots(c, false)
			e.knowClassSpells(c, spellClassCleric, true)
		case clsPaladin:
			if lvl > 8 {
				for l := int8(9); l <= lvl; l++ {
					for s := 1; s <= 5; s++ {
						c.SpellSlots[slotsCleric+s-1] += paladinSlotData[int(l)*5+s]
					}
				}
				e.knowClassSpells(c, spellClassCleric, false)
			}
		case clsRanger:
			if lvl > 7 {
				for l := int8(8); l <= lvl; l++ {
					for s := 1; s <= 3; s++ {
						c.SpellSlots[slotsDruid+s-1] += rangerSlotData[int(l)*5+s]
					}
					for s := 4; s <= 5; s++ {
						c.SpellSlots[slotsMage+s-4] += rangerSlotData[int(l)*5+s]
					}
				}
				for n := 1; n < len(spellTable); n++ {
					if spellTable[n].class() == spellClassDruid {
						c.SpellBook[n-1] = 1
					}
				}
			}
		case clsMage:
			c.SpellSlots[slotsMage]++
			for l := int8(2); l <= lvl; l++ {
				for s := 1; s <= 5; s++ {
					c.SpellSlots[slotsMage+s-1] += mageSlotData[int(l)*5+s]
				}
			}
		}
	}
	for _, it := range c.Items {
		if itemEffect(it) == itemPowerFlag+effectWizardry && it.Readied != 0 {
			for s := 0; s < 3; s++ {
				c.SpellSlots[slotsMage+s] *= 2
			}
		}
	}
}

// knowClassSpells marks every cleric spell of a level the character has
// slots for as known (clerics leave out animate dead).
func (e *Engine) knowClassSpells(c *Character, class byte, skipAnimateDead bool) {
	for n := 1; n < len(spellTable); n++ {
		s := spellTable[n]
		if s.class() != class || c.SpellSlots[slotsCleric+int(s.level())-1] == 0 {
			continue
		}
		if skipAnimateDead && n == spellAnimateDead {
			continue
		}
		c.SpellBook[n-1] = 1
	}
}

// wisdomSlots = O0FE:0686 (reset, c): bonus cleric spells for wisdom 13..18.
// With reset the slots are first rebuilt from the cleric level alone.
func (e *Engine) wisdomSlots(c *Character, reset bool) {
	lvl := int8(c.Levels[clsCleric])
	if oldClassUsable(c) {
		lvl += int8(c.OldLevels[clsCleric])
	}
	if lvl <= 0 {
		return
	}
	if reset {
		for s := 1; s < 5; s++ {
			c.SpellSlots[slotsCleric+s] = 0
		}
		c.SpellSlots[slotsCleric] = 1
		for l := int8(2); l <= lvl; l++ {
			for s := 1; s <= 5; s++ {
				c.SpellSlots[slotsCleric+s-1] += clericSlotData[int(l)*5+s]
			}
		}
	}
	wis := c.Stats[statWis].Value
	for _, b := range []struct {
		above byte
		slot  int
	}{{12, 0}, {13, 0}, {14, 1}, {15, 1}, {16, 2}, {17, 3}} {
		if wis > b.above && c.SpellSlots[slotsCleric+b.slot] != 0 {
			c.SpellSlots[slotsCleric+b.slot]++
		}
	}
}

// Saving throw rows.
const (
	saveCount       = 5
	saveWorst       = 20
	saveClassStride = 60
)

// savingThrows = O0FE:07FB: each save is the best of the class tables;
// dwarves and halflings (or a readied item of the power) add a constitution
// bonus to the first save, high constitution helps everybody.
func (e *Engine) savingThrows(c *Character) {
	conBonus := readiedEffect(c, effectSaveBonus)
	con := c.Stats[statCon].Value
	for s := 0; s < saveCount; s++ {
		c.Saves[s] = saveWorst
		for k := 0; k <= clsMonk; k++ {
			if lvl := int8(c.Levels[k]); lvl > 0 {
				c.Saves[s] = min(c.Saves[s], saveTableData[k*saveClassStride+int(lvl)*saveCount+s])
			}
		}
		if int8(c.OldLevels[clsMonk]) < int8(c.Levels[clsMonk]) {
			c.Saves[s] = min(c.Saves[s], oldClassSaveData[int(int8(c.OldLevels[clsMonk]))*saveCount+s])
		}
		if s != 0 {
			continue
		}
		if c.Race == raceDwarf || c.Race == raceHalfling || conBonus {
			switch {
			case con >= 4 && con <= 6:
				c.Saves[0]++
			case con >= 7 && con <= 10:
				c.Saves[0] += 2
			case con >= 11 && con <= 13:
				c.Saves[0] += 3
			case con >= 14 && con <= 17:
				c.Saves[0] += 4
			case con == 18:
				c.Saves[0] += 5
			}
		}
		switch con {
		case 19, 20:
			c.Saves[0]++
		case 21, 22:
			c.Saves[0] += 2
		case 23, 24:
			c.Saves[0] += 3
		case 25:
			c.Saves[0] += 4
		}
	}
}

// Thief skill count and the levels items lift them to.
const (
	thiefSkillCount     = 8
	thieveryFirstLevel  = 5
	thieverySecondLevel = 7
	thieveryBonus       = 5
	freeActionLevel     = 4
	freeActionBonus     = 10
)

// thiefSkills = O0FE:0AEA: each skill from the thief level, race and
// dexterity tables; items of thievery and free action raise them.
func (e *Engine) thiefSkills(c *Character) {
	thievery, freeAction := false, false
	for _, it := range c.Items {
		if thievery || freeAction {
			break
		}
		if itemEffect(it) > itemPowerFlag && it.Readied != 0 {
			p := itemEffect(it) & (itemPowerFlag - 1)
			thievery, freeAction = p == effectThievery, p == effectFreeAction
		}
	}
	base := int8(c.Levels[clsThief])
	if oldClassUsable(c) {
		base += int8(c.OldLevels[clsThief])
	}
	if base < freeActionLevel && freeAction {
		base, freeAction = freeActionLevel, false
	}
	race, dex := int(c.Race), int(c.Stats[statDex].Value)
	for s := 1; s <= thiefSkillCount; s++ {
		lvl := base
		bonus := int8(0) // fix: the original leaves this uninitialised without an item of thievery
		if thievery {
			switch s {
			case 1:
				if lvl < thieveryFirstLevel {
					lvl = thieveryFirstLevel
				} else {
					bonus = thieveryBonus
				}
			case 2:
				if lvl < thieverySecondLevel {
					lvl = thieverySecondLevel
				} else {
					bonus = thieveryBonus
				}
			}
		}
		raceAdj := thiefRaceData[race*thiefSkillCount+s]
		levelBase := int(thiefBaseData[int(lvl)*thiefSkillCount+s])
		if raceAdj < 0 && levelBase < int(-raceAdj)+int(bonus) {
			c.Thief[s-1] = 0
		} else {
			v := bonus + int8(levelBase) + raceAdj
			if s < 6 {
				v += thiefDexData[dex*5+s]
			}
			c.Thief[s-1] = byte(v)
		}
		if freeAction {
			c.Thief[s-1] += freeActionBonus
		}
	}
}

// Readied-item slots at char+0xCC (13 far pointers). Slot numbers come
// from the base item record; 9 and 10 are the two rings, 11 arrows
// (type 0x49), 12 quarrels (type 0x1C).
const (
	slotWeapon = 0
	slotArmour = 2 // +345
	slotRing1  = 9
	slotRing2  = 10
	slotArrows = 11
	slotBolts  = 12
)
