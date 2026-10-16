package engine

// Character creation and dropping (unit O0EF).

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

// Races (the order of the race names); Silver Blades has no half-orcs.
const (
	raceTribble = iota // the first name slot, never a character's race
	raceElf
	raceHalfElf
	raceDwarf
	raceGnome
	raceHalfling
	raceHuman
	raceMonster
)

// Classes beyond the seven single classes: the multi-class combinations.
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

// An affect time that never runs out.
const permanent = 255

// A new character's fixed starting values.
const (
	startBaseAC          = 50
	startTHAC0           = 40
	startIconID          = 10
	startExperience      = 200000 // one class
	startExperience2     = 100000 // two classes
	startExperience3     = 66667  // three classes
	startClericExpLawful = 2000   // a cleric still without experience (never happens)
	startClericExpOther  = 1500
	startBaseMove        = 12
	startClericSlots     = 2
	exceptionalStrDice   = 100
	strengthMaximum      = 18
	multiClericWisdom    = 13
	maxNameLength        = 15
	abilityRolls         = 6
	allClasses           = 255 // a class mask with every training bit
)

// classLevelsOf: the classes a multi-class choice starts with at level 1,
// and the experience it starts with.
var classLevelsOf = map[int]struct {
	classes    []int
	experience uint32
}{
	classClericFighter:     {[]int{clsCleric, clsFighter}, startExperience2},
	classClericFighterMage: {[]int{clsCleric, clsFighter, clsMage}, startExperience3},
	classClericRanger:      {[]int{clsCleric, clsRanger}, startExperience2},
	classClericMage:        {[]int{clsCleric, clsMage}, startExperience2},
	classClericThief:       {[]int{clsCleric, clsThief}, startExperience2},
	classFighterMage:       {[]int{clsFighter, clsMage}, startExperience2},
	classFighterThief:      {[]int{clsFighter, clsThief}, startExperience2},
	classFighterMageThief:  {[]int{clsFighter, clsMage, clsThief}, startExperience3},
	classMageThief:         {[]int{clsMage, clsThief}, startExperience3},
}

// The spells a new magic-user knows.
var mageStartSpells = []int{spellDetectMagicMU, spellReadMagic, spellShield, spellSleep}

// abilityMinimum and abilityMaximum: the race's limits of ability i for the
// character's sex (raceAbilityLimits: strength by sex, then pairs).
func abilityMinimum(c *Character, i int) byte {
	if i == statStr {
		return raceAbilityLimits[c.Race][strMinimumBySex+c.Sex]
	}
	return raceAbilityLimits[c.Race][abilityLimitsRest+2*(i-1)]
}

func abilityMaximum(c *Character, i int) byte {
	if i == statStr {
		return raceAbilityLimits[c.Race][strMaximumBySex+c.Sex]
	}
	return raceAbilityLimits[c.Race][abilityLimitsRest+2*(i-1)+1]
}

// exceptionalStrengthMaximum: the race's limit of exceptional strength.
func exceptionalStrengthMaximum(c *Character) byte {
	return raceAbilityLimits[c.Race][str00MaximumBySex+c.Sex]
}

// fighterKind: fighters, paladins and rangers get exceptional strength.
func fighterKind(c *Character) bool {
	return int8(c.Levels[clsFighter]) > 0 || int8(c.Levels[clsPaladin]) > 0 || int8(c.Levels[clsRanger]) > 0
}

// pickFromList runs one of creation's selection menus; false = left with
// Exit, which abandons the whole creation. Only Select picks.
func (e *Engine) pickFromList(list []*ListNode) (int, bool) {
	index, redraw := 1, true
	var sel *ListNode
	for {
		k := e.ListMenu("", "Select", colourLightMagenta, colourGreen, colourWhite, 1, 2, 38, 22, list, true, &redraw, &index, &sel)
		if k == 0 {
			return 0, false
		}
		if k == 'S' {
			return index, true
		}
	}
}

// CreateCharacter = O0EF:07C6: pick race, gender, class and alignment,
// roll the abilities (best of six each, then the age, race and class
// limits), hit points and starting spells, train to the starting
// experience; keep it, name it, pick the pictures, save it.
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

	// Race: elf, half-elf, dwarf, gnome, halfling, human.
	list := newList(raceHuman + 1)
	list[0].Flag, list[0].Text = 1, "Pick Race"
	for r := raceElf; r <= raceHuman; r++ {
		list[r].Text = "  " + raceNames[r]
	}
	race, ok := e.pickFromList(list)
	if !ok {
		return
	}
	c.Race = byte(race)
	small := func(affects ...byte) {
		c.IconSize = 1
		for _, a := range affects {
			e.AddAffect(c, a, 0, permanent, 0)
		}
	}
	switch race {
	case raceHalfling:
		small(affectConSavingBonus)
	case raceDwarf:
		small(affectDwarfDodges, affectDwarfVsFoes, affectConSavingBonus)
	case raceGnome:
		small(affectGnomeDodges, affectGnomeVsFoes, affectConSavingBonus)
	case raceElf:
		c.IconSize = 2
		e.AddAffect(c, affectElfResistance, 0, permanent, 0)
	case raceHalfElf:
		c.IconSize = 2
		e.AddAffect(c, affectHalfElfResistance, 0, permanent, 0)
	default:
		c.IconSize = 2
	}

	// Gender.
	list = newList(len(genderNames) + 1)
	list[0].Flag, list[0].Text = 1, "Pick Gender"
	for i, g := range genderNames {
		list[i+1].Text = "  " + g
	}
	gender, ok := e.pickFromList(list)
	if !ok {
		return
	}
	c.Sex = byte(gender - 1)

	// Class.
	classes := raceClasses[race][1 : 1+int(raceClasses[race][0])]
	list = newList(len(classes) + 1)
	list[0].Flag, list[0].Text = 1, "Pick Class"
	for k, cl := range classes {
		list[k+1].Text = "  " + classNames[cl]
	}
	choice, ok := e.pickFromList(list)
	if !ok {
		return
	}
	class := int(classes[choice-1])
	c.Class = byte(class)
	c.Exp = startExperience
	c.MaxLevel = 1
	switch class {
	case clsPaladin:
		c.PaladinCures = 1
		c.Levels[clsPaladin] = 1
		e.AddAffect(c, affectProtFromEvil, 0, permanent, 0)
	case clsRanger:
		c.Levels[clsRanger] = 1
		e.AddAffect(c, affectRangerVsGiants, 0, permanent, 0)
	case clsCleric, clsDruid, clsFighter, clsMage, clsThief:
		c.Levels[class] = 1
	default:
		multi := classLevelsOf[class]
		for _, k := range multi.classes {
			c.Levels[k] = 1
		}
		if class == classClericRanger {
			e.AddAffect(c, affectRangerVsGiants, 0, permanent, 0)
		}
		c.Exp = multi.experience
	}
	if int8(c.Levels[clsThief]) > 0 {
		e.thiefSkills(c)
	}
	c.ClassMask = 0
	c.THAC0Base = 0
	for k := 0; k < classCount; k++ {
		if level := int8(c.Levels[k]); level > 0 {
			c.THAC0Base = max(c.THAC0Base, thac0Table[k][level])
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
	choice, ok = e.pickFromList(list)
	if !ok {
		return
	}
	c.Alignment = aligns[choice-1]
	if c.Exp == 0 && int8(c.Levels[clsCleric]) > 0 {
		switch c.Alignment % 3 {
		case 0:
			c.Exp = startClericExpLawful
		case 1:
			c.Exp = startClericExpOther
		}
	}
	c.Age = startingAge(c, func(count, sides int) int { return e.Dice(count, sides) })

	undo := e.selectTemp(c)
	defer undo()
	e.CharacterSheet()
	for {
		e.rollAbilities(c, class)
		e.startingHitPoints(c)
		e.DisplayString(7, 7, colourWhite, levelsText(c))
		e.showCombatStats()
		e.showCoins()
		if e.YesNo(colourLightMagenta, colourGreen, colourWhite, "Reroll stats? ") == 'N' {
			break
		}
	}
	e.CharacterSheet()
	for c.PName[0] == 0 {
		c.SetName(e.InputString("Character name: ", colourLightMagenta, maxNameLength))
	}
	defaultIconParts(c)
	e.iconBuilder()
	for i := range c.Stats {
		c.Stats[i].Base = c.Stats[i].Value
	}
	c.Str00.Base = c.Str00.Value
	if e.YesNo(colourLightMagenta, colourGreen, colourWhite, "Save "+c.Name()+"? ") == 'N' {
		return
	}
	e.SaveCharacter(c, "")
}

// startingAge: a single class rolls the race's dice for it; a multi-class
// takes the most the cleric's, the magic-user's or the fighter's dice can
// give. A cleric/thief has no entry and stays 0 years old.
func startingAge(c *Character, dice func(count, sides int) int) uint16 {
	entry := func(k int) (base uint16, count, sides int) {
		row := raceAges[c.Race][k*4:]
		return uint16(row[0]) | uint16(row[1])<<8, int(row[2]), int(row[3])
	}
	class := int(c.Class)
	if class < classCount {
		base, count, sides := entry(class)
		return uint16(dice(count, sides)) + base
	}
	column := clsCleric
	switch class {
	case classFighterMage, classFighterMageThief, classMageThief:
		column = clsMage
	case classFighterThief:
		column = clsFighter
	case classClericThief:
		return 0
	}
	base, count, sides := entry(column)
	return base + uint16(count*sides)
}

// Age steps (ageSteps per race): past each one abilities change.
const (
	ageYoungAdult = iota
	ageMature
	ageMiddleAged
	ageOld
)

// rollAbilities rolls the six abilities: the best of six rolls each, then
// the age steps, the race limits, the class minimum and exceptional
// strength (fighters, paladins, rangers with 18). The original also has
// race modifiers per roll, but they test the race menu's loop counter,
// which is always 6 by then, so they never apply.
func (e *Engine) rollAbilities(c *Character, class int) {
	for k := 0; k < classCount; k++ {
		if int8(c.Levels[k]) > 0 {
			c.Levels[k] = 1
		}
	}
	c.Str00.Value = 0
	past := func(step int) bool { return ageSteps[c.Race][step] < c.Age }
	for i := range c.Stats {
		ability := &c.Stats[i].Value
		*ability = 0
		for range abilityRolls {
			*ability = max(*ability, byte(e.Dice(3, 6)+1))
		}
		switch i {
		case statStr:
			if *ability > 0 && past(ageYoungAdult) {
				*ability++
			}
			if *ability > 0 && past(ageMature) {
				*ability--
			}
			if *ability > 0 && past(ageMiddleAged) {
				*ability -= 2
			}
			if *ability > 0 && past(ageOld) {
				*ability--
			}
		case statInt:
			if past(ageMature) {
				*ability++
			}
			if past(ageOld) {
				*ability++
			}
		case statWis:
			if !past(ageYoungAdult) {
				*ability--
			}
			for _, step := range []int{ageMature, ageMiddleAged, ageOld} {
				if past(step) {
					*ability++
				}
			}
		case statDex:
			if past(ageMiddleAged) {
				*ability -= 2
			}
			if past(ageOld) {
				*ability--
			}
		case statCon:
			for _, step := range []int{ageMature, ageMiddleAged, ageOld} {
				if past(step) {
					*ability--
				}
			}
		}
		*ability = max(*ability, abilityMinimum(c, i))
		*ability = min(*ability, abilityMaximum(c, i))
		*ability = max(*ability, classStatMinimum[class][i])
		if i == statStr && *ability == strengthMaximum && fighterKind(c) {
			c.Str00.Value = byte(e.Random(exceptionalStrDice) + 1)
			c.Str00.Value = min(c.Str00.Value, exceptionalStrengthMaximum(c))
		}
		if i == statWis && *ability < multiClericWisdom && class >= classClericFighter && class <= classClericThief {
			*ability = multiClericWisdom
		}
		c.Stats[i].Base = *ability
		if i == statStr {
			c.Str00.Base = c.Str00.Value
		}
		e.showStat(i, false)
	}
}

// startingHitPoints sets the starting attacks, spells, gold and hit
// points, then trains the character silently up to its experience.
func (e *Engine) startingHitPoints(c *Character) {
	c.HP = c.MaxHP
	c.BaseAttack[0], c.BaseDice[0], c.BaseSides[0], c.StrBonus, c.BaseMove = 2, 1, 2, 1, startBaseMove
	classes := 0
	var gold byte
	c.SpellSlots = [len(c.SpellSlots)]byte{}
	for k := 0; k < classCount; k++ {
		if int8(c.Levels[k]) <= 0 {
			continue
		}
		switch k {
		case clsCleric:
			c.SpellSlots[slotIndex(spellClassCleric, 1)] = startClericSlots
			e.clericSlots(c)
			for _, s := range clericStartSpells[1:] {
				if s == 0 {
					break
				}
				c.SpellBook[s-1] = 1
			}
		case clsMage:
			c.SpellSlots[slotIndex(spellClassMage, 1)] = 1
			for _, s := range mageStartSpells {
				c.SpellBook[s-1] = 1
			}
		case clsThief:
			gold += byte(e.Dice(2, 6))
		}
		classes++
	}
	c.Coins[coinGold] = uint16(gold)
	c.MaxHP, c.HPRoll, c.HP = 0, 0, 0
	e.levelHitPoints(c, allClasses, classes)
	e.trainSilently(c)
}

// defaultIconParts = O0EF:39E7: the icon head by sex and size, the body
// (weapon pose) by class.
func defaultIconParts(c *Character) {
	const (
		headFemaleLarge, headFemaleSmall = 9, 7
		headMaleLarge, headMaleSmall     = 5, 0
		bodyCleric, bodyRanger           = 23, 1
		bodyFighter, bodyMage, bodyThief = 24, 29, 5
	)
	large := c.IconSize == 2
	switch {
	case c.Sex == 1 && large:
		c.IconHead = headFemaleLarge
	case c.Sex == 1:
		c.IconHead = headFemaleSmall
	case large:
		c.IconHead = headMaleLarge
	default:
		c.IconHead = headMaleSmall
	}
	switch {
	case int8(c.Levels[clsCleric]) > 0:
		c.IconBody = bodyCleric
	case int8(c.Levels[clsRanger]) > 0:
		c.IconBody = bodyRanger
	case int8(c.Levels[clsPaladin]) > 0 || int8(c.Levels[clsFighter]) > 0:
		c.IconBody = bodyFighter
	case int8(c.Levels[clsMage]) > 0:
		c.IconBody = bodyMage
	default:
		c.IconBody = bodyThief
	}
}

// DropCharacter = O0EF:23BB: pick a member and ask twice; then it leaves
// the party for good (its roster files go too), a hired NPC bidding
// farewell, anyone else being dumped.
func (e *Engine) DropCharacter() {
	if !e.pickCharacter() {
		return
	}
	if c := e.selected(); c != nil {
		if e.YesNo(colourYellow, colourGreen, colourWhite, "Drop "+c.Name()+" forever? ") == 'Y' &&
			e.YesNo(colourYellow, colourGreen, colourWhite, "Are you sure? ") == 'Y' {
			if c.Okay == 0 {
				e.printMessage("You dump " + c.Name() + " out back.")
			} else {
				e.printMessage(c.Name() + " bids you farewell.")
			}
			e.removeCharacterFiles(c)
			e.freePlayers(false, true)
		} else {
			e.printMessage(c.Name() + " breathes a sigh of relief.")
		}
	}
	e.DisplayPartyList()
}

// pickCharacter = O100:40C8: Up and Down move the selection through the
// party until Select (true) or Exit.
func (e *Engine) pickCharacter() bool {
	for {
		key, extra := e.DisplayInput("Pick Character ", "Select Exit", colourLightMagenta, colourGreen, colourWhite, true, false)
		if extra {
			e.stepSelection(key)
			e.DisplayPartyList()
			continue
		}
		switch key {
		case 'S':
			return true
		case 0, 'E':
			return false
		}
	}
}

// stepSelection = O100:39AE (key): Up selects the previous member (from
// the first, the last), Down the next (staying on the last); any other
// key the first.
func (e *Engine) stepSelection(key byte) {
	if len(e.party) == 0 {
		return
	}
	switch key {
	case scanUp:
		if e.partySel == 0 {
			e.partySel = len(e.party) - 1
		} else {
			e.partySel--
		}
	case scanDown:
		if e.partySel+1 < len(e.party) {
			e.partySel++
		}
	default:
		e.partySel = 0
	}
}
