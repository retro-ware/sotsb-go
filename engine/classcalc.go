package engine

// Class-derived values of a character: THAC0, spell slots, saving throws,
// thief skills, and the dual-class helpers (unit O164 of the reconstruction).

// itemEffect is a magic item's special power (the last name word: 128 + power).
func itemEffect(it *Item) byte { return it.Spells[2] }

// Special item powers (128 + n in itemEffect).
const (
	effectWizardry   = 1 // ring of wizardry: doubles the fifth-level magic-user spells
	effectFreeAction = 2
	effectSaveBonus  = 6 // a constitution bonus to the first saving throw
	effectThievery   = 11
	itemPowerFlag    = 128
)

// isHuman = O164:0D49: only humans change class.
func isHuman(c *Character) bool { return c.Race == raceHuman }

// noClass: a dual-class helper found no class.
const noClass = 17

// lastSingleClass bounds the class searches of the dual-class helpers,
// which never look past the thief.
const lastSingleClass = clsThief

// activeClass = O164:0C4D: a human's old class (the first with old
// levels) if it has positive levels; noClass otherwise and for the other races.
func activeClass(c *Character) int {
	if !isHuman(c) {
		return noClass
	}
	k := 0
	for k < lastSingleClass && c.OldLevels[k] == 0 {
		k++
	}
	if int8(c.OldLevels[k]) > 0 {
		return k
	}
	return noClass
}

// currentClass = O164:0CA3: a human's class in progress (noClass for
// others and for a human with no levels).
func currentClass(c *Character) int {
	if !isHuman(c) {
		return noClass
	}
	k := 0
	for k < lastSingleClass && c.Levels[k] == 0 {
		k++
	}
	if int8(c.Levels[k]) > 0 {
		return k
	}
	return noClass
}

// firstClassLevel = O164:0CF9: a human's level in the first class with
// levels; 0 for the other races.
func firstClassLevel(c *Character) int8 {
	if !isHuman(c) {
		return 0
	}
	k := 0
	for k < lastSingleClass && c.Levels[k] == 0 {
		k++
	}
	return int8(c.Levels[k])
}

// oldClassUsable = O164:0D6B: a dual-classed human whose new class has
// passed the old one may use the old class's abilities again.
func oldClassUsable(c *Character) bool {
	return firstClassLevel(c) > int8(c.MulticlassLevel)
}

// Attacks per two rounds of fighters, paladins and rangers of high level.
const (
	extraAttacks = 3
	moreAttacks  = 4
)

// recalcClassBonuses = O164:0366: THAC0, highest level and attacks from the
// class levels, then spell slots, saves, thief skills and the class mask;
// readied items the classes may no longer use (and not cursed) are put away.
func (e *Engine) recalcClassBonuses(c *Character) {
	c.THAC0Base = 0
	for k := 0; k < classCount; k++ {
		lvl := int8(c.Levels[k])
		c.THAC0Base = max(c.THAC0Base, thac0At(k, lvl))
		if int8(c.MaxLevel) < lvl {
			c.MaxLevel = byte(lvl)
		}
		switch k {
		case clsFighter, clsPaladin:
			if lvl > 6 {
				c.BaseAttack[0] = extraAttacks
			}
			if lvl > 12 {
				c.BaseAttack[0] = moreAttacks
			}
		case clsRanger:
			if lvl > 7 {
				c.BaseAttack[0] = extraAttacks
			}
			if lvl > 14 {
				c.BaseAttack[0] = moreAttacks
			}
		}
	}
	e.calcSpellSlots(c)
	e.savingThrows(c)
	if int8(c.Levels[clsThief]) > 0 {
		e.thiefSkills(c)
	}
	c.ClassMask = 0
	for k := 0; k < classCount; k++ {
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
	if !oldClassUsable(c) {
		return
	}
	for k := 0; k < classCount; k++ {
		old := int8(c.OldLevels[k])
		switch k {
		case clsFighter, clsPaladin:
			if old >= 13 {
				c.BaseAttack[0] = moreAttacks
			} else if old >= 7 {
				c.BaseAttack[0] = extraAttacks
			}
		case clsRanger:
			if old >= 15 {
				c.BaseAttack[0] = moreAttacks
			} else if old >= 8 {
				c.BaseAttack[0] = extraAttacks
			}
		case clsThief:
			if old > 0 {
				e.thiefSkills(c)
			}
		}
		c.THAC0Base = max(c.THAC0Base, thac0At(k, old))
	}
}

// itemClassFlags: the byte of a base item record holding the classes that may use it.
const itemClassFlags = 13

// thac0At reads the THAC0 table like the original: the flat table indexed
// class*19 + level (levels past the row run into the next class's row).
func thac0At(class int, level int8) byte {
	width := len(thac0Table[0])
	i := class*width + int(level)
	return thac0Table[i/width][i%width]
}

// slotIndex is the SpellSlots entry of a spell class and level (1..7).
func slotIndex(class, level int) int { return class*spellLevels + level - 1 }

// slotsGained reads a per-level slot table: the spells of each spell level
// a class gains at a class level.
func slotsGained(table []byte, classLevel, spellLevel int) byte {
	return table[classLevel*spellLevels+spellLevel]
}

// Class levels from which paladins and rangers cast spells.
const (
	paladinFirstSpellLevel = 9
	rangerFirstSpellLevel  = 8
	rangerDruidLevels      = 3
	rangerMageLevels       = 2
)

// calcSpellSlots = O164:0000: the spells per day of each spell class, and
// the spells clerics, paladins and rangers know without learning them.
func (e *Engine) calcSpellSlots(c *Character) {
	c.SpellSlots = [len(c.SpellSlots)]byte{}
	for k := 0; k < classCount; k++ {
		lvl := int(int8(c.Levels[k]))
		if oldClassUsable(c) && c.OldLevels[k] != 0 {
			lvl = int(int8(c.OldLevels[k]))
		}
		if lvl <= 0 {
			continue
		}
		switch k {
		case clsCleric:
			e.clericSlots(c)
			e.knowSpells(c, func(class, level int) bool {
				return class == spellClassCleric && c.SpellSlots[slotIndex(spellClassCleric, level)] > 0
			})
		case clsPaladin:
			if lvl < paladinFirstSpellLevel {
				continue
			}
			for l := paladinFirstSpellLevel; l <= lvl; l++ {
				for s := 1; s <= spellLevels; s++ {
					c.SpellSlots[slotIndex(spellClassCleric, s)] += slotsGained(paladinSlotData[:], l, s)
				}
			}
			e.knowSpells(c, func(class, level int) bool {
				return class == spellClassCleric && c.SpellSlots[slotIndex(spellClassCleric, level)] > 0
			})
		case clsRanger:
			if lvl < rangerFirstSpellLevel {
				continue
			}
			for l := rangerFirstSpellLevel; l <= lvl; l++ {
				for s := 1; s <= rangerDruidLevels; s++ {
					c.SpellSlots[slotIndex(spellClassDruid, s)] += slotsGained(rangerSlotData[:], l, s)
				}
				for s := 1; s <= rangerMageLevels; s++ {
					c.SpellSlots[slotIndex(spellClassMage, s)] += slotsGained(rangerSlotData[:], l, rangerDruidLevels+s)
				}
			}
			e.knowSpells(c, func(class, level int) bool {
				return class == spellClassDruid && c.SpellSlots[slotIndex(spellClassDruid, level)] > 0 ||
					class == spellClassMage && c.SpellSlots[slotIndex(spellClassMage, level)] > 0
			})
		case clsMage:
			c.SpellSlots[slotIndex(spellClassMage, 1)]++
			for l := 2; l <= lvl; l++ {
				for s := 1; s <= spellLevels; s++ {
					c.SpellSlots[slotIndex(spellClassMage, s)] += slotsGained(mageSlotData[:], l, s)
				}
			}
			intelligenceSlots(c)
		}
	}
	for _, it := range c.Items {
		if itemEffect(it) == itemPowerFlag+effectWizardry && it.Readied != 0 {
			c.SpellSlots[slotIndex(spellClassMage, 5)] *= 2
		}
	}
}

// knowSpells marks every spell the test accepts (by spell class and level) as known.
func (e *Engine) knowSpells(c *Character, known func(class, level int) bool) {
	for n := 1; n < len(spellTable); n++ {
		s := spellTable[n]
		if level := int(int8(s.level())); level >= 1 && level <= spellLevels && known(int(s.class()), level) {
			c.SpellBook[n-1] = 1
		}
	}
}

// clericSlots = O164:062C: the cleric spells per day from the cleric level
// (with the old class's when usable) and bonus spells for wisdom 13..18;
// sixth- and seventh-level spells need wisdom 17 and 18.
func (e *Engine) clericSlots(c *Character) {
	lvl := int(int8(c.Levels[clsCleric]) + int8(c.OldLevels[clsCleric])*b2i8(oldClassUsable(c)))
	if lvl <= 0 {
		return
	}
	slot := func(level int) *byte { return &c.SpellSlots[slotIndex(spellClassCleric, level)] }
	for s := 2; s <= spellLevels; s++ {
		*slot(s) = 0
	}
	*slot(1) = 1
	for l := 2; l <= lvl; l++ {
		for s := 1; s <= spellLevels; s++ {
			*slot(s) += slotsGained(clericSlotData[:], l, s)
		}
	}
	wis := c.Stats[statWis].Value
	for _, b := range []struct {
		above byte
		level int
	}{{12, 1}, {13, 1}, {14, 2}, {15, 2}, {16, 3}, {17, 4}} {
		if wis > b.above && *slot(b.level) > 0 {
			*slot(b.level)++
		}
	}
	if wis < 17 {
		*slot(6) = 0
	}
	if wis < 18 {
		*slot(7) = 0
	}
}

// intelligenceSlots = O164:07BE: sixth- and seventh-level magic-user
// spells need intelligence 12 and 14.
func intelligenceSlots(c *Character) {
	if c.Stats[statInt].Value < 12 {
		c.SpellSlots[slotIndex(spellClassMage, 6)] = 0
	}
	if c.Stats[statInt].Value < 14 {
		c.SpellSlots[slotIndex(spellClassMage, 7)] = 0
	}
}

// Saving throw rows.
const (
	saveCount       = 5
	saveWorst       = 20
	saveClassLevels = 18
)

// saveTarget is the saving throw table entry of a class, level and save.
func saveTarget(class, level, save int) byte {
	return saveTableData[class*saveClassLevels*saveCount+level*saveCount+save]
}

// savingThrows = O164:07F9: each save is the best of the class tables;
// a readied item of the save-bonus power adds a constitution bonus to the
// first save, high constitution helps everybody.
func (e *Engine) savingThrows(c *Character) {
	conBonus := false
	for _, it := range c.Items {
		if conBonus {
			break
		}
		if itemEffect(it) > itemPowerFlag && it.Readied != 0 {
			conBonus = itemEffect(it)&(itemPowerFlag-1) == effectSaveBonus
		}
	}
	con := c.Stats[statCon].Value
	for s := 0; s < saveCount; s++ {
		c.Saves[s] = saveWorst
		for k := 0; k < classCount; k++ {
			if lvl := int8(c.Levels[k]); lvl > 0 {
				c.Saves[s] = min(c.Saves[s], saveTarget(k, int(lvl), s))
			}
		}
		// The original reads the class loop's variable after the loop: the thief.
		if old := int8(c.OldLevels[clsThief]); int8(c.Levels[clsThief]) > old {
			c.Saves[s] = min(c.Saves[s], saveTarget(clsThief, int(old), s))
		}
		if s != 0 {
			continue
		}
		if conBonus {
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

// Thief skills.
const (
	thiefSkillCount = 8
	dexteritySkills = 5 // the first five skills have a dexterity adjustment
)

// thiefSkills = O164:0AC6: each skill from the thief level (with the old
// class's when usable), race and dexterity tables; a skill a race penalty
// would push below zero is 0.
func (e *Engine) thiefSkills(c *Character) {
	lvl := int(int8(c.Levels[clsThief]) + int8(c.OldLevels[clsThief])*b2i8(oldClassUsable(c)))
	race, dex := int(int8(c.Race)), int(c.Stats[statDex].Value)
	for s := 1; s <= thiefSkillCount; s++ {
		raceAdj := int(thiefRaceData[race*thiefSkillCount+s])
		levelBase := int(thiefBaseData[lvl*thiefSkillCount+s])
		if raceAdj < 0 && levelBase < -raceAdj {
			c.Thief[s-1] = 0
			continue
		}
		v := levelBase + raceAdj
		if s <= dexteritySkills {
			v += int(thiefDexData[dex*dexteritySkills+s])
		}
		c.Thief[s-1] = byte(v)
	}
}

// Readied-item slots (13 pointers from +357). Slot numbers come from the
// base item record; 9 and 10 are the two rings, 11 arrows, 12 quarrels.
const (
	slotWeapon = 0
	slotArmour = 2
	slotRing1  = 9
	slotRing2  = 10
	slotArrows = 11
	slotBolts  = 12
)
