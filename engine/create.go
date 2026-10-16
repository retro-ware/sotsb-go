package engine

// Character creation and dropping, ported from ovr17.

// charMgmtState holds ovr17 state (filled in by the ovr17 port).
type charMgmtState struct{}

// freeCharacter = O0A7:003C (var c): disposes the combat block, the item
// and affect lists and the record, then nils the pointer. Go's collector
// does the freeing.
func (e *Engine) freeCharacter(c **Character) {
	if *c != nil {
		(*c).Combat, (*c).Items, (*c).Affects = nil, nil, nil
	}
	*c = nil
}

// selectTemp makes c the selected character (DS:5D92 = c) although it is not
// in the party; the returned func restores the old selection.
func (e *Engine) selectTemp(c *Character) func() {
	sel, n := e.partySel, len(e.party)
	e.party = append(e.party, c)
	e.partySel = n
	e.tempSel = c
	return func() {
		e.party = e.party[:n]
		e.partySel = sel
		e.tempSel = nil
	}
}

// Races.
const (
	raceMonster = iota
	raceDwarf
	raceElf
	raceGnome
	raceHalfElf
	raceHalfling
	raceHalfOrc
	_ // raceHuman (classchange.go)
)

// Classes beyond the eight single classes: the multi-class combinations.
const (
	classClericFighter = iota + 8
	classClericFighterMage
	classClericRanger
	classClericMage
	classClericThief
	classFighterMage
	classFighterThief
	classFighterMageThief
	classMageThief
)

// Racial abilities given as permanent affects at creation.
const (
	affectProtectionFromEvil = 8
	affectGnomeVsManSized    = 18
	affectDwarfVsOrc         = 26
	affectVsGiants           = 47
	affectGnomeExtra         = 48
	affectConSavingBonus     = 97
	affectElfResistSleep     = 107
	affectHalfElfResistance  = 124
	affectRangerVsGiants     = 134
	permanent                = 255
)

// A new character's fixed starting values.
const (
	startBaseAC        = 50
	startTHAC0         = 40
	startIconID        = 10
	startExperience    = 25000 // one class
	startExperience2   = 12500 // two classes
	startExperience3   = 8333  // three classes
	startPlatinum      = 300
	startBaseMove      = 12
	exceptionalStrDice = 100
	strengthMaximum    = 18
	multiClericWisdom  = 13
	maxNameLength      = 15
)

// classLevelsOf: the classes a class choice starts with at level 1.
var classLevelsOf = map[int][]int{
	classClericFighter:     {clsCleric, clsFighter},
	classClericFighterMage: {clsCleric, clsFighter, clsMage},
	classClericRanger:      {clsCleric, clsRanger},
	classClericMage:        {clsCleric, clsMage},
	classClericThief:       {clsCleric, clsThief},
	classFighterMage:       {clsFighter, clsMage},
	classFighterThief:      {clsFighter, clsThief},
	classFighterMageThief:  {clsFighter, clsMage, clsThief},
	classMageThief:         {clsMage, clsThief},
}

// CreateCharacter = O0A7:0752 createPlayer: pick race, gender, class and
// alignment, roll the abilities (best of six 3d6 each, then the age, race
// and class limits), hit points and starting spells, train to the starting
// level; keep it, name it, pick the pictures, save it.
func (e *Engine) CreateCharacter() {
	c := &Character{}
	for i, t := range defaultIconColours {
		c.IconColour[i] = (t+8)<<4 + t
	}
	c.BaseAC = startBaseAC
	c.THAC0Base = startTHAC0
	c.Status = 0
	c.Okay = 1
	c.Size = 1
	c.RandomID = byte(e.Random(256))
	c.Order = startIconID

	// pick runs one of the four selection menus; false = left with Esc/Exit,
	// which abandons the whole creation.
	pick := func(list []*ListNode) (int, bool) {
		index, redraw := 1, true
		var sel *ListNode
		for {
			k := e.ListMenu("", "Select", colourLightMagenta, colourGreen, colourWhite, 1, 2, 38, 22, list, true, &redraw, &index, &sel)
			if k == 0 {
				return 0, false
			}
			if k == 'S' || k == 13 {
				return index, true
			}
		}
	}

	// Race: the seven playable races (half-orcs are not offered).
	list := newList(7)
	list[0].Flag, list[0].Text = 1, "Pick Race"
	for i, r := range []int{raceDwarf, raceElf, raceGnome, raceHalfElf, raceHalfling, raceHuman} {
		list[i+1].Text = "  " + raceNames[r]
	}
	idx, ok := pick(list)
	if !ok {
		return
	}
	if idx == 6 {
		idx = raceHuman
	}
	c.Race = byte(idx)
	switch int(c.Race) {
	case raceHalfling:
		c.IconSize = 1
		e.AddAffect(c, affectConSavingBonus, 0, permanent, 0)
	case raceDwarf:
		c.IconSize = 1
		for _, t := range []byte{affectConSavingBonus, affectDwarfVsOrc, affectVsGiants} {
			e.AddAffect(c, t, 0, permanent, 0)
		}
	case raceGnome:
		c.IconSize = 1
		for _, t := range []byte{affectConSavingBonus, affectGnomeVsManSized, affectVsGiants, affectGnomeExtra} {
			e.AddAffect(c, t, 0, permanent, 0)
		}
	case raceElf:
		c.IconSize = 2
		e.AddAffect(c, affectElfResistSleep, 0, permanent, 0)
	case raceHalfElf:
		c.IconSize = 2
		e.AddAffect(c, affectHalfElfResistance, 0, permanent, 0)
	default:
		c.IconSize = 2
	}

	// Gender.
	list = newList(3)
	list[0].Flag, list[0].Text = 1, "Pick Gender"
	for i, g := range genderNames {
		list[i+1].Text = "  " + g
	}
	if idx, ok = pick(list); !ok {
		return
	}
	c.Sex = byte(idx - 1)

	// Class.
	race := int(c.Race)
	classes := raceClasses[race][1 : 1+int(raceClasses[race][0])]
	list = newList(len(classes) + 1)
	list[0].Flag, list[0].Text = 1, "Pick Class"
	for k, cl := range classes {
		list[k+1].Text = "  " + classNames[cl]
	}
	if idx, ok = pick(list); !ok {
		return
	}
	class := int(classes[idx-1])
	c.Class = byte(class)
	c.Exp = startExperience
	c.MaxLevel = 1
	switch {
	case class <= clsFighter || class >= clsMage && class <= clsMonk:
		c.Levels[class] = 1
	case class == clsPaladin:
		c.PaladinCures = 1
		c.Levels[clsPaladin] = 1
		e.AddAffect(c, affectProtectionFromEvil, 0, permanent, 0)
	case class == clsRanger:
		c.Levels[clsRanger] = 1
		e.AddAffect(c, affectRangerVsGiants, 0, permanent, 0)
	default:
		levels := classLevelsOf[class]
		for _, k := range levels {
			c.Levels[k] = 1
		}
		if class == classClericRanger {
			e.AddAffect(c, affectRangerVsGiants, 0, permanent, 0)
		}
		c.Exp = startExperience2
		if len(levels) == 3 || class == classMageThief {
			c.Exp = startExperience3
		}
	}
	if int8(c.Levels[clsThief]) > 0 {
		e.thiefSkills(c)
	}
	c.ClassMask = 0
	c.THAC0Base = 0
	for k := 0; k <= clsMonk; k++ {
		if lvl := int8(c.Levels[k]); lvl > 0 {
			if t := thac0Table[k][lvl]; t > c.THAC0Base {
				c.THAC0Base = t
			}
			c.ClassMask += classItemFlags[k]
		}
	}
	e.savingThrows(c)

	// Alignment.
	aligns := classAlignments[class][1 : 1+int(classAlignments[class][0])]
	list = newList(len(aligns) + 1)
	list[0].Flag, list[0].Text = 1, "Pick Alignment"
	for k, al := range aligns {
		list[k+1].Text = "  " + alignmentNames[al]
	}
	if idx, ok = pick(list); !ok {
		return
	}
	c.Alignment = aligns[idx-1]

	// Starting age: single classes roll their dice, multi-classes take the
	// maximum of the cleric's, fighter's or magic-user's dice.
	ageEntry := func(k int) (base, count, sides int) {
		row := raceAges[race][k*4:]
		return int(row[0]) | int(row[1])<<8, int(row[2]), int(row[3])
	}
	if class <= clsMonk {
		base, count, sides := ageEntry(class)
		c.Age = uint16(base + int(byte(e.Dice(count, sides))))
	} else {
		k := clsCleric
		switch class {
		case classFighterMage, classFighterMageThief, classMageThief:
			k = clsMage
		case classFighterThief:
			k = clsFighter
		}
		base, count, sides := ageEntry(k)
		c.Age = uint16(base + count*sides)
	}

	undo := e.selectTemp(c)
	e.CharacterSheet()
	for {
		e.rollAbilities(c, class)
		e.rollHitPoints(c)
		e.showLevels(c)
		e.showCombatStats()
		e.showCoins()
		if e.YesNo(colourLightMagenta, colourGreen, colourWhite, "Reroll stats? ") == 'N' {
			break
		}
	}
	e.CharacterSheet()
	undo()

	for c.PName[0] == 0 {
		c.SetName(e.InputString("Character name: ", colourLightMagenta, maxNameLength))
	}
	undo = e.selectTemp(c)
	e.iconBuilder()
	for i := range c.Stats {
		c.Stats[i].Base = c.Stats[i].Value
	}
	c.Str00.Base = c.Str00.Value
	if e.YesNo(colourLightMagenta, colourGreen, colourWhite, "Save "+c.Name()+"? ") == 'Y' {
		e.SaveCharacter(c, "")
	}
	undo()
	e.freeCharacter(&c)
}

// rollAbilities rolls the six abilities into their full values: the best of
// six 3d6+1 each, then the age steps, the race limits, the class minimum and
// exceptional strength (fighters, paladins, rangers with 18).
func (e *Engine) rollAbilities(c *Character, class int) {
	race, gender := int(c.Race), int(c.Sex)
	for k := 0; k <= clsMonk; k++ {
		if int8(c.Levels[k]) > 0 {
			c.Levels[k] = 1
		}
	}
	c.Str00.Value = 0
	limits := raceStatLimits[race]
	older := func(step int) bool { return ageSteps[race][step] < c.Age }
	for i := range c.Stats {
		st := &c.Stats[i].Value
		*st = 0
		for range 6 {
			if roll := byte(e.Dice(3, 6) + 1); *st < roll {
				*st = roll
			}
		}
		switch i {
		case statStr:
			if *st != 0 && older(0) {
				*st++
			}
			if *st != 0 && older(1) {
				*st--
			}
			if *st != 0 && older(2) {
				*st -= 2
			}
			if *st != 0 && older(3) {
				*st--
			}
		case statInt:
			if older(1) {
				*st++
			}
			if older(3) {
				*st++
			}
		case statWis:
			if !older(0) {
				*st--
			}
			if older(1) {
				*st++
			}
			if older(2) {
				*st++
			}
			if older(3) {
				*st++
			}
		case statDex:
			if older(2) {
				*st -= 2
			}
			if older(3) {
				*st--
			}
		case statCon:
			if older(1) {
				*st--
			}
			if older(2) {
				*st--
			}
			if older(3) {
				*st--
			}
		}
		lo, hi := limits[2*i+4], limits[2*i+5] // int .. cha: min/max pairs after the strength block
		if i == statStr {
			lo, hi = limits[gender], limits[2+gender]
		}
		*st = max(*st, lo)
		*st = min(*st, hi)
		*st = max(*st, classStatMinimum[class][i])
		if i == statStr && *st == strengthMaximum &&
			(int8(c.Levels[clsFighter]) > 0 || int8(c.Levels[clsRanger]) > 0 || int8(c.Levels[clsPaladin]) > 0) {
			c.Str00.Value = byte(e.Random(exceptionalStrDice) + 1)
			c.Str00.Value = min(c.Str00.Value, limits[4+gender])
		}
		if i == statWis && *st < multiClericWisdom && class >= classClericFighter && class <= classClericThief {
			*st = multiClericWisdom
		}
		e.showStat(i, false)
	}
}

// rollHitPoints sets the starting attacks, spells and hit points and trains
// the character up to its starting experience.
func (e *Engine) rollHitPoints(c *Character) {
	c.HP = c.MaxHP
	c.BaseAttack[0], c.BaseDice[0], c.BaseSides[0], c.StrBonus, c.BaseMove = 2, 1, 2, 1, startBaseMove
	c.SpellSlots = [15]byte{}
	count := 0
	for k := 0; k <= clsMonk; k++ {
		if int8(c.Levels[k]) <= 0 {
			continue
		}
		switch k {
		case clsCleric:
			c.SpellSlots[slotsCleric] = 1
		case clsMage:
			c.SpellSlots[slotsMage] = 1
		}
		e.Dice(int(classHitDice[k][0]), int(classHitDice[k][1])) // rolled and not used (sic): keeps the random sequence
		switch k {
		case clsCleric:
			e.wisdomSlots(c, false)
			for s := 1; s < len(spellTable); s++ {
				if spellTable[s].class() == spellClassCleric && spellTable[s].level() == 1 {
					c.SpellBook[s-1] = 1
				}
			}
		case clsMage:
			for _, s := range []int{spellDetectMagicMU, spellReadMagic, spellEnlarge, spellSleep} {
				c.SpellBook[s-1] = 1
			}
		}
		count++
	}
	c.Coins[coinPlatinum] = startPlatinum
	c.HPRoll = byte(e.rollHitPoints2(c, 0xFF))
	c.MaxHP = c.HPRoll
	adj := int(int8(e.conHitPointBonusOf(c)))
	if adj < 0 && int(c.MaxHP) <= -adj+count {
		c.MaxHP = 1
	} else {
		c.MaxHP = byte((int(c.MaxHP) + adj) / count)
	}
	c.HP = c.MaxHP
	c.HPRoll = byte(int(c.HPRoll) / count)
	e.trainSilently(c)
}

// DropCharacter = O0A7:25DA: asks twice, then the selected member leaves
// the party (a hired NPC bids farewell, anyone else is dumped).
func (e *Engine) DropCharacter() {
	if len(e.party) == 0 {
		return
	}
	c := e.selected()
	if e.YesNo(0xE, 0xA, 0xF, "Drop "+c.Name()+" forever? ") == 'Y' &&
		e.YesNo(0xE, 0xA, 0xF, "Are you sure? ") == 'Y' {
		if c.Okay == 0 {
			e.printMessage("You dump " + c.Name() + " out back.")
		} else {
			e.printMessage(c.Name() + " bids you farewell.")
		}
		e.freePlayers(false, true) // O0A7:3A25
		return
	}
	e.printMessage(c.Name() + " breathes a sigh of relief.")
}
