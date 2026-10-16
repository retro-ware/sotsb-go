package engine

// What a character's items make of it: readied slots, weight, attack,
// damage, armour class, movement (unit O0ED of the reconstruction).

// Fields of a base item record (the ITEMS table, 16 bytes per type).
const (
	itemSlot        = 0 // which readied slot the item takes (9 rings)
	itemHands       = 1 // hands it needs
	itemDiceLarge   = 2 // damage against large creatures
	itemSidesLarge  = 3
	itemBonusLarge  = 4
	itemAttacks     = 5
	itemACBonus     = 6 // above 127: armour class bonus (low bits: 0 shield-like, else armour base)
	itemDiceNormal  = 9
	itemSidesNormal = 10
	itemBonusNormal = 11
	itemRange       = 12
	itemFlags       = 14 // 1 uses arrows, 2 dexterity to hit, 4 strength to hit and damage, 128 uses bolts
)

// Item slot classes (itemSlot).
const (
	slotClassArmour = 2
	slotClassRing   = 9
	slotClassShield = 1
	slotClassHelm   = 9 + 0 // (rings share 9)
)

// Item types with special treatment.
const (
	itemTypeArrows  = 73
	itemTypeBolts   = 28
	itemTypeElfBow1 = 37
	itemTypeElfBow2 = 36
)

// baseItem returns byte k of the base record of an item type.
func (e *Engine) baseItem(typ byte, k int) byte {
	if o := int(typ)*16 + k; o < len(e.items) {
		return e.items[o]
	}
	return 0
}

// readiedWeight = DS 19664: the weight of the readied items (ApplyItems).
var readiedWeight uint16

// Movement limits by weight carried over the strength allowance.
const (
	moveLight  = 9
	moveMedium = 6
	moveHeavy  = 3
	coinWeight = 1 // every coin weighs one unit
)

// ApplyItems = O0ED:0C20: everything that follows from the items carried
// and readied: slots, weight, hands, attack and damage, armour class,
// movement and the fighter's attack level.
func (e *Engine) ApplyItems(c *Character) {
	c.Slots = [13]*Item{}
	c.ItemCount = 0
	c.Hands = 0
	c.Weight = 0
	readiedWeight = 0
	for _, it := range c.Items {
		c.ItemCount++
		w := it.Weight
		if it.Count > 0 {
			w *= uint16(it.Count)
		}
		c.Weight += w
		if it.Readied == 0 {
			continue
		}
		readiedWeight += w
		switch slot := e.baseItem(it.Type, itemSlot); {
		case slot < slotClassRing:
			c.Slots[slot] = it
		case slot == slotClassRing:
			if c.Slots[slotRing1] == nil {
				c.Slots[slotRing1] = it
			} else if c.Slots[slotRing2] == nil {
				c.Slots[slotRing2] = it
			}
		}
		if it.Type == itemTypeArrows {
			c.Slots[slotArrows] = it
		}
		if it.Type == itemTypeBolts {
			c.Slots[slotBolts] = it
		}
		c.Hands += e.baseItem(it.Type, itemHands)
	}
	for k := 0; k < len(c.Coins); k++ {
		c.Weight += c.Coins[k] * coinWeight
	}
	for h := 0; h < 2; h++ { // attacks per round are not reset here
		c.Dice[h] = c.BaseDice[h]
		c.Sides[h] = c.BaseSides[h]
		c.Bonus[h] = c.BaseBonus[h]
	}
	var ac [5]int8 // dexterity, shield, armour, helm-like, base
	cursedAC := false
	c.SaveBonus = 0
	c.AC[0] = c.BaseAC
	c.Move = c.BaseMove
	c.THAC0 = c.THAC0Base
	ac[0] = dexAC(c)
	if c.Slots[slotWeapon] == nil {
		c.THAC0 += byte(strHit(c))
		c.Bonus[0] = byte(int8(c.Bonus[0]) + strDamage(c))
	}
	e.weaponStats(c)
	for _, it := range c.Items {
		if it.Readied != 0 {
			e.armourMove(c, it)
			e.itemAC(c, it, &ac, &cursedAC)
		}
	}
	if cursedAC {
		ac[3] = 0
	}
	e.encumbrance(c)
	if int(ac[4]) < int(c.AC[0]) {
		ac[4] = int8(c.AC[0])
	}
	c.AC[0] = 0
	for k := range ac {
		c.AC[0] += byte(ac[k])
	}
	c.AC[1] = byte(ac[4] + ac[2] + ac[3] - 2)
	fighter := int8(c.Levels[clsFighter])
	old := int8(c.OldLevels[clsFighter])
	if fighter < 1 && (old < 1 || !oldClassUsable(c)) || int8(c.Race) < 1 {
		c.FightLevel = 1
	} else {
		c.FightLevel = byte(fighter + old*b2i8(oldClassUsable(c)))
	}
}

func b2i8(b bool) int8 {
	if b {
		return 1
	}
	return 0
}

// weaponStats = O0ED:0023: to-hit and damage of the readied weapon:
// dexterity for missiles, strength for melee, the ammunition's bonus, elves
// with their bows.
func (e *Engine) weaponStats(c *Character) {
	w := c.Slots[slotWeapon]
	if w == nil {
		return
	}
	t := w.Type
	flags := e.baseItem(t, itemFlags)
	c.THAC0 = c.THAC0Base
	if flags&2 != 0 {
		c.THAC0 += byte(dexMissile(c))
	}
	c.Bonus[0] = e.baseItem(t, itemBonusNormal)
	if flags&4 != 0 {
		c.THAC0 += byte(strHit(c))
		c.Bonus[0] = byte(int8(c.Bonus[0]) + strDamage(c))
	}
	plus := int8(w.Plus)
	if flags > 127 && c.Slots[slotBolts] != nil {
		plus += int8(c.Slots[slotBolts].Plus)
	}
	if flags&1 != 0 && c.Slots[slotArrows] != nil {
		plus += int8(c.Slots[slotArrows].Plus)
	}
	c.Bonus[0] = byte(int8(c.Bonus[0]) + plus)
	if c.Race == raceElf && (t > 40 && t < 45 || t == itemTypeElfBow1 || t == itemTypeElfBow2) {
		plus++
	}
	c.THAC0 += byte(plus)
	c.Dice[0] = e.baseItem(t, itemDiceNormal)
	c.Sides[0] = e.baseItem(t, itemSidesNormal)
}

// armourMove = O0ED:021E (c, item): body armour sets the movement by its weight.
func (e *Engine) armourMove(c *Character, it *Item) {
	if e.baseItem(it.Type, itemSlot) != slotClassArmour {
		return
	}
	switch {
	case it.Weight <= 150:
		c.Move = c.BaseMove
	case it.Weight <= 399:
		c.Move = moveLight
	default:
		c.Move = moveMedium
	}
	if it.Plus != 0 && c.Move <= moveLight {
		c.Move += 3
	}
}

// itemAC = O0ED:02A6 (c, item, ac, cursed): an armour class item adds to
// its part of ac (shields and rings add up, cloaks keep the best, armour
// sets the base); its second plus is a saving throw bonus.
func (e *Engine) itemAC(c *Character, it *Item, ac *[5]int8, cursed *bool) {
	bonus := e.baseItem(it.Type, itemACBonus)
	if bonus <= 127 {
		return
	}
	bonus &= 127
	slot := e.baseItem(it.Type, itemSlot)
	if slot == slotClassShield {
		ac[1] = int8(bonus) + int8(it.Plus)
		return
	}
	if bonus == 0 {
		if slot == slotClassRing {
			if int8(it.Plus) > ac[3] {
				ac[3] = int8(it.Plus)
			}
		} else {
			ac[2] += int8(it.Plus)
		}
		c.SaveBonus = byte(int8(c.SaveBonus) + int8(it.Plus2))
		return
	}
	if int8(bonus)+int8(it.Plus) > ac[4] {
		ac[4] = int8(bonus) + int8(it.Plus)
		if int8(it.Plus) > 0 && slot == slotClassArmour {
			*cursed = true
		}
	}
}

// encumbrance = O0ED:03C4: movement by the weight over the strength allowance.
func (e *Engine) encumbrance(c *Character) {
	over := int(c.Weight) - int(strCarry(c))
	if over < 0 {
		over = 0
	}
	m := c.Move
	switch {
	case over <= 512:
	case over <= 768:
		m = moveLight
	case over <= 1024:
		m = moveMedium
	default:
		m = moveHeavy
	}
	if m < c.Move {
		c.Move = m
	}
}

// dexAC = O0ED:117B stat_bonus: armour class from dexterity (full value).
func dexAC(c *Character) int8 {
	d := c.Stats[statDex].Value
	switch {
	case d >= 1 && d <= 3:
		return -4
	case d >= 4 && d <= 6:
		return int8(d) - 7
	case d >= 15 && d <= 18:
		return int8(d) - 14
	case d == 19 || d == 20:
		return 4
	case d >= 21 && d <= 23:
		return 5
	case d == 24 || d == 25:
		return 6
	}
	return 0
}

// dexMissile = O0ED:11FD stat_bonus02: to-hit with missiles from dexterity.
func dexMissile(c *Character) int8 {
	d := c.Stats[statDex].Value
	switch {
	case d <= 2:
		return -4
	case d >= 3 && d <= 5:
		return int8(d) - 6
	case d >= 16 && d <= 18:
		return int8(d) - 15
	case d == 19 || d == 20:
		return 3
	case d >= 21 && d <= 23:
		return 4
	case d == 24 || d == 25:
		return 5
	}
	return 0
}

// strIndex = O0ED:127F: strength as one scale (18/xx = 18..23, 19+ = 24+).
func strIndex(c *Character) byte {
	s := c.Stats[statStr].Value
	switch {
	case s <= 17:
		return s
	case s == 18:
		switch x := c.Str00.Value; {
		case x == 0:
			return 18
		case x <= 50:
			return 19
		case x <= 75:
			return 20
		case x <= 90:
			return 21
		case x <= 99:
			return 22
		case x == 100:
			return 23
		}
		return 0 // the original leaves the result undefined above 100 (sic)
	case s <= 25:
		return s + 5
	}
	return 0
}

// strHit = O0ED:1317: to-hit from strength (only for StrBonus characters).
func strHit(c *Character) int8 {
	if c.StrBonus == 0 {
		return 0
	}
	switch v := strIndex(c); {
	case v >= 1 && v <= 3:
		return -3
	case v == 4 || v == 5:
		return -2
	case v == 6 || v == 7:
		return -1
	case v >= 17 && v <= 19:
		return 1
	case v >= 20 && v <= 22:
		return 2
	case v >= 23 && v <= 25:
		return 3
	case v == 26 || v == 27:
		return 4
	case v >= 28 && v <= 30:
		return int8(v) - 23
	}
	return 0
}

// strDamage = O0ED:13BA: damage from strength (only for StrBonus characters).
func strDamage(c *Character) int8 {
	if c.StrBonus == 0 {
		return 0
	}
	switch v := strIndex(c); {
	case v == 1 || v == 2:
		return -2
	case v >= 3 && v <= 5:
		return -1
	case v == 16:
		return 1
	case v >= 17 && v <= 19:
		return int8(v) - 16
	case v >= 20 && v <= 29:
		return int8(v) - 17
	case v == 30:
		return 14
	}
	return 0
}

// strCarry = O0ED:1440: the weight allowance change from strength.
func strCarry(c *Character) int16 {
	switch v := int16(strIndex(c)); {
	case v >= 1 && v <= 3:
		return -350
	case v == 4 || v == 5:
		return -250
	case v == 6 || v == 7:
		return -150
	case v == 12 || v == 13:
		return 100
	case v == 14 || v == 15:
		return 200
	case v == 16:
		return 350
	case v >= 17 && v <= 21:
		return (v-17)*250 + 500
	case v >= 22 && v <= 26:
		return (v-22)*1000 + 2000
	case v == 27:
		return 7500
	case v >= 28 && v <= 30:
		return (v-28)*3000 + 9000
	}
	return 0
}

// isHelpless = O0ED:0BDB: the character has one of the four helpless affects.
func (e *Engine) isHelpless(c *Character) bool {
	found := false
	for _, a := range helplessAffectData[1:] {
		if e.FindAffect(c, a) != nil {
			found = true
		}
	}
	return found
}

// atRaceLimitAfterRestore = O0ED:3138 (char, class): the demihuman level limit is
// reached (fighter limits depend on strength 17/18; race 4 clerics stop at 5).
func atRaceLimitAfterRestore(c *Character, class int) bool {
	lvl, str := int8(c.Levels[class]), c.Stats[statStr].Value
	if lvl <= 0 {
		return false
	}
	fighter := class == 2
	switch c.Race {
	case 1:
		return fighter && (lvl == 8 && str == 0x11 || lvl == 7 && str < 0x11)
	case 2:
		return fighter && (lvl == 7 || lvl == 6 && str == 0x11 || lvl == 5 && str < 0x11)
	case 3:
		return fighter && (lvl == 6 || lvl == 5 && str < 0x12)
	case 4:
		if class == 0 {
			return lvl == 5
		}
		return fighter && (lvl == 8 || lvl == 7 && str == 0x11 || lvl == 6 && str < 0x11)
	case 5:
		return fighter && (lvl == 6 || lvl == 5 && str == 0x11 || lvl == 4 && str < 0x11)
	}
	return false
}
