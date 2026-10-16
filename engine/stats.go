package engine

// The ability scores in use (O0E3:16B0..1FB9): a character's own scores
// changed by its readied magic items and its affects.

// Magic item properties: a readied item whose Spells[2] has the top bit set
// changes a score; Spells[1] is the property's value.
const (
	itemPropertyFlag       = 128
	itemDexterityUp        = 2  // dexterity +4 / +2 / +1 for dexterity 0..6 / 7..13 / more
	itemGiantStrength      = 5  // value 0..6: strength 18/00, 19 .. 24
	itemConstitutionUp     = 6  // constitution +1, charisma -1
	itemScoreUp            = 8  // value: the score raised by 1 (if under 18)
	itemDexterityDown      = 10 // dexterity -2
	itemLowIntelligence    = 12 // intelligence 7
	itemLowStrengthAndMind = 13 // strength and intelligence 3
)

// Score limits of 18D9.
const (
	strengthMaximum18   = 18
	exceptionalMaximum  = 100
	exceptionalStep     = 10 // 18/xx gained per point of strength above 18
	hitPointBonusScore  = 15
	regenerationScore   = 20 // constitution from which the body heals itself
	regenerationTime    = 60
	lowScore            = 3 // a score item 13 or feeblemindedness leaves
	noFixedScore        = 255
	giantStrengthBase   = 18 // value 0 is 18/00; value n >= 1 is 18 + n
	strengthEncodedBase = 100
	giantStrength21     = 21 // the Strength spell of O0C2:3FB7
)

// strengthCode = O0E3:16B0 (str, str00): one byte for a strength: 18/xx as
// xx+1 (18/00 is 101), others +100.
func strengthCode(str, str00 byte) byte {
	if str == strengthMaximum18 {
		return str00 + 1
	}
	return str + strengthEncodedBase
}

// strengthDecode = O0E3:16D9 (affect, var str, var str00): the inverse,
// from an affect's data.
func strengthDecode(a *Affect) (str, str00 byte) {
	v := a.Data & 0x7F
	if v <= strengthEncodedBase+1 {
		return strengthMaximum18, v - 1
	}
	return v - strengthEncodedBase, 0
}

// strongerThan = O0E3:1728 (c, str, str00, var code): the strength beats
// the character's own; code is its strengthCode.
func strongerThan(c *Character, str, str00 byte) (byte, bool) {
	if str > c.Stats[statStr].Base || str == strengthMaximum18 && str00 > c.Str00.Base {
		return strengthCode(str, str00), true
	}
	return 0, false
}

// statAffects = O0E3:18D9 (c, stat): the score in use from the
// character's own, its readied magic items and its affects. Constitution
// also works out the maximum hit points again.
func (e *Engine) statAffects(c *Character, stat int) {
	value := c.Stats[stat].Base
	value00 := c.Str00.Base
	var better, better00 byte
	fixed := byte(noFixedScore)
	takeBetter := func() { // O0E3:1771
		if better > value || better == strengthMaximum18 && better00 > value00 {
			value, value00 = better, better00
		}
	}
	for _, it := range c.Items {
		if it.Spells[2] <= itemPropertyFlag || it.Readied == 0 {
			continue
		}
		prop, arg := it.Spells[2]&0x7F, it.Spells[1]
		switch stat {
		case statStr:
			switch prop {
			case itemGiantStrength:
				switch arg {
				case 0:
					better, better00 = giantStrengthBase, exceptionalMaximum
				case 1, 2, 3, 4, 5, 6:
					better = giantStrengthBase + arg
				}
			case itemScoreUp:
				if c.Stats[statStr].Base < strengthMaximum18 && arg == statStr {
					better, better00 = c.Stats[statStr].Base+1, 0
				}
			case itemLowStrengthAndMind:
				fixed = lowScore
			}
			takeBetter()
		case statCon:
			if prop == itemConstitutionUp {
				value++
			} else if prop == itemScoreUp && c.Stats[statCon].Base < strengthMaximum18 && arg == statCon {
				value++
			}
		case statInt:
			switch prop {
			case itemScoreUp:
				if c.Stats[statInt].Base < strengthMaximum18 && arg == statInt {
					value++
				}
			case itemLowIntelligence:
				fixed = feebleScore
			case itemLowStrengthAndMind:
				fixed = lowScore
			}
		case statWis:
			if prop == itemScoreUp && arg == statWis && c.Stats[statWis].Base < strengthMaximum18 {
				value++
			}
		case statDex:
			switch prop {
			case itemDexterityUp:
				switch dex := c.Stats[statDex].Base; {
				case dex <= 6:
					value += 4
				case dex <= 13:
					value += 2
				default:
					value++
				}
			case itemScoreUp:
				if c.Stats[statDex].Base < strengthMaximum18 && arg == statDex {
					value++
				}
			case itemDexterityDown:
				value -= 2
			}
		case statCha:
			switch prop {
			case itemConstitutionUp:
				value--
			case itemScoreUp:
				if c.Stats[statCha].Base < strengthMaximum18 && arg == statCha {
					value++
				}
			}
		}
	}
	switch stat {
	case statStr:
		if a := e.FindAffect(c, affectStrengthSpell); a != nil {
			better, better00 = strengthDecode(a)
			if value <= strengthMaximum18 && value00 < exceptionalMaximum {
				better += value
				if better > strengthMaximum18 {
					if fighterClass(c) {
						better00 = byte(int(better-strengthMaximum18)*exceptionalStep) + c.Str00.Value
						if better00 > exceptionalMaximum {
							better00 = exceptionalMaximum
						}
					}
					better = strengthMaximum18
				}
			}
			takeBetter()
		}
		if a := e.FindAffect(c, affectGiantStrength); a != nil {
			better, better00 = strengthDecode(a)
			takeBetter()
		}
		if a := e.FindAffect(c, affectEnlarge); a != nil {
			better, better00 = strengthDecode(a)
			takeBetter()
		}
		if fixed != noFixedScore {
			c.Stats[statStr].Value = fixed
			c.Str00.Value = 0
		} else {
			c.Stats[statStr].Value = value
			c.Str00.Value = value00
		}
	case statCon:
		e.conHitPoints(c, value)
		c.Stats[statCon].Value = value
		if c.Stats[statCon].Value >= regenerationScore {
			if e.FindAffect(c, affectHealingOverTime) == nil {
				e.AddAffect(c, affectHealingOverTime, regenerationTime, 0xFF, 1)
			} else {
				e.RemoveAffect(c, affectHealingOverTime, nil)
			}
		}
	case statInt, statWis:
		if e.FindAffect(c, affectFeebleminded) != nil && fixed > feebleScore {
			fixed = lowScore
		}
		if fixed != noFixedScore {
			c.Stats[stat].Value = fixed
		} else {
			c.Stats[stat].Value = value
		}
	case statDex:
		if fixed != noFixedScore {
			c.Stats[statDex].Value = fixed
		} else {
			c.Stats[statDex].Value = value
		}
	case statCha:
		if a := e.FindAffect(c, affectCharisma); a != nil {
			value += a.Data
		}
		c.Stats[statCha].Value = value
	}
}

// fighterClass: a level (or an old level) as fighter, paladin or ranger.
func fighterClass(c *Character) bool {
	for _, k := range []int{clsFighter, clsPaladin, clsRanger} {
		if int8(c.Levels[k]) > 0 || int8(c.OldLevels[k]) > 0 {
			return true
		}
	}
	return false
}

// conHitPoints is 18D9's constitution part: the maximum hit points are the
// rolled ones plus the constitution bonus of each class level (fighters
// more), shared out over the classes; the hit points follow the change.
func (e *Engine) conHitPoints(c *Character, con byte) {
	classes := 0
	oldMax := c.MaxHP
	c.MaxHP = c.HPRoll
	var bonus byte
	addLevels := func(k int, lvl byte) { // O0E3:17BE
		if classMaxHitDice[k] <= lvl {
			lvl = classMaxHitDice[k] - 1
		}
		if k == clsRanger && c.MulticlassLevel != 0 && c.OldLevels[clsRanger] == c.MulticlassLevel {
			lvl++
		}
		if k == clsFighter || k == clsPaladin || k == clsRanger {
			switch {
			case con >= 15 && con <= 19:
				bonus += lvl * (con - 14)
			case con == 20:
				bonus += lvl * 5
			case con >= 21 && con <= 23:
				bonus += lvl * 6
			case con >= 24 && con <= 25:
				bonus += lvl * 7
			}
			return
		}
		if con > hitPointBonusScore {
			bonus += lvl * 2
		} else if con == hitPointBonusScore {
			bonus += lvl
		}
	}
	for k := 0; k < len(c.Levels); k++ {
		if lvl := c.OldLevels[k]; lvl > 0 {
			addLevels(k, lvl)
		}
		lvl := c.Levels[k]
		if lvl > 0 {
			classes++
		}
		if classMaxHitDice[k] < lvl {
			lvl = classMaxHitDice[k]
		}
		if lvl > c.MulticlassLevel {
			addLevels(k, lvl-c.MulticlassLevel)
		}
	}
	if classes > 0 { // always: a character has a class (DOS would stop on a divide error)
		bonus /= byte(classes)
	}
	c.MaxHP += bonus
	switch {
	case c.MaxHP > oldMax:
		c.HP += c.MaxHP - oldMax
	case c.MaxHP < oldMax:
		if lost := oldMax - c.MaxHP; c.HP > lost {
			c.HP -= lost
		} else {
			c.HP = 0
		}
	}
}
