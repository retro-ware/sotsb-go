package engine

// The View screen (unit O0AF): the character sheet and its menu, coins
// traded and dropped, and a paladin's healing and curing.

// viewPlayer = O0AF:0B64 (var done): the selected member's sheet with
// Items, Spells, Trade, Drop, Heal and Cure; done is set when an item was
// used in combat.
func (e *Engine) viewPlayer() bool {
	if e.gameState == stateCombat {
		e.combatViewEnter()
	}
	key := byte(' ')
	done := false
	e.viewStart = e.selected()
	e.CharacterSheet()
	for key != 0 && key != 'E' && !done {
		c := e.selected()
		spells, money := false, false
		for i := 0; i < len(c.Memorized); i++ {
			if c.Memorized[i] > 0 {
				spells = true
			}
		}
		for k := 0; k < coinKinds; k++ {
			if c.Coins[k] > 0 {
				money = true
			}
		}
		menu := ""
		if len(c.Items) > 0 {
			menu += "Items "
		}
		if spells {
			menu += "Spells "
		}
		if (c.Control < npcControl || c.Okay == 0 || c.Status == statusAnimated) && money && e.gameState != stateCombat {
			menu += "Trade "
		}
		if money {
			menu += "Drop "
		}
		if e.canLayOnHands(c) {
			menu += "Heal "
		}
		if e.canCureDisease(c) {
			menu += "Cure "
		}
		menu += "Exit"
		key, _ = e.DisplayInput("", menu, colourLightMagenta, colourGreen, colourWhite, false, false)
		spellIndex := -1
		switch key {
		case 'I':
			done = e.ItemsMenu()
		case 'S':
			e.spellMenu2(0, 0, &spellIndex, &spells)
		case 'T':
			e.tradeCoins()
		case 'D':
			e.dropCoinsMenu()
			e.showCoins()
		case 'H':
			e.layOnHands(c)
		case 'C':
			e.paladinCure(c)
		}
		if !done && (key == 'I' || key == 'S' || key == 'T') {
			e.CharacterSheet()
		}
	}
	if e.gameState == stateCombat {
		e.combatViewLeave()
	}
	e.loadPicture()
	return done
}

// coinLines: the coin list of Trade and Drop, Jewelry first, names
// right-aligned to 8 characters.
func coinLines(c *Character) []*ListNode {
	var list []*ListNode
	for k := 0; k < coinKinds; k++ {
		if c.Coins[k] != 0 {
			line := spaces(8-len(coinNames[k])) + coinNames[k] + " " + itoa(int(c.Coins[k]))
			list = append([]*ListNode{{Text: line}}, list...)
		}
	}
	return list
}

// tradeCoins = O0AF:2C3F tradeCoin: hand coins to another member.
func (e *Engine) tradeCoins() {
	for {
		e.loadPicture()
		to := e.selectMemberMenu("Trade to?", true, e.viewStart)
		if to == nil {
			return
		}
		e.CharacterSheet()
		for {
			e.showCoins()
			e.viewStart = to
			list := coinLines(e.selected())
			index := 0
			redraw := true
			var sel *ListNode
			e.ListMenu("Select type of coin ", " Select", colourLightMagenta, colourGreen, colourWhite, 12, 7, 25, 13,
				list, true, &redraw, &index, &sel)
			if sel == nil {
				break
			}
			kind, name := coinKindOf(sel.Text)
			amount := e.askAmount(e.selected().Coins[kind], "How much "+name+"will you trade? ", colourGreen)
			e.moveCoins(e.selected(), to, amount, kind)
			left := false
			for k := 0; k < coinKinds; k++ {
				if e.selected().Coins[k] > 0 {
					left = true
				}
			}
			if !left {
				return
			}
		}
	}
}

// dropCoinsMenu = O0AF:2F97 drop_coin: coins are dropped (into the pool
// in a shop or after a fight).
func (e *Engine) dropCoinsMenu() {
	for {
		e.showCoins()
		list := coinLines(e.selected())
		index := 0
		redraw := true
		var sel *ListNode
		e.ListMenu("Select type of coin ", " Select", colourLightMagenta, colourGreen, colourWhite, 12, 7, 25, 13,
			list, true, &redraw, &index, &sel)
		if sel == nil {
			return
		}
		kind, name := coinKindOf(sel.Text)
		amount := e.askAmount(e.selected().Coins[kind], "How much "+name+"will you drop? ", colourGreen)
		e.dropCoins(e.selected(), amount, kind)
		left := false
		for k := 0; k < coinKinds; k++ {
			if e.selected().Coins[k] > 0 {
				left = true
			}
		}
		if !left {
			return
		}
	}
}

// Affects that mark a paladin's healing and curing as used.
const (
	affectLaidOnHands = 140
	affectCuredToday  = 141
	layOnHandsTime    = 1440
	cureDiseaseTime   = 10080
	clsPaladinIndex   = 3
)

// paladinLevel: the paladin levels that count (old ones when usable).
func paladinLevel(c *Character) int {
	old := 0
	if oldClassUsable(c) {
		old = 1
	}
	return int(int8(c.Levels[clsPaladin])) + int(int8(c.OldLevels[clsPaladin]))*old
}

func isPaladin(c *Character) bool {
	return c.Class == clsPaladinIndex || int8(c.OldLevels[clsPaladin]) > 0 && oldClassUsable(c)
}

// canLayOnHands = O0AF:35F0 (c): a paladin, not fighting, okay, who has
// not healed today.
func (e *Engine) canLayOnHands(c *Character) bool {
	return isPaladin(c) && e.gameState != stateCombat && c.Status == statusOkay && e.FindAffect(c, affectLaidOnHands) == nil
}

// canCureDisease = O0AF:3655 (c): a paladin with cures left this week.
func (e *Engine) canCureDisease(c *Character) bool {
	return isPaladin(c) && e.gameState != stateCombat && c.Status == statusOkay && c.PaladinCures > 0
}

// layOnHands = O0AF:36D7 (paladin): heal two points per level once a day.
func (e *Engine) layOnHands(p *Character) {
	e.loadPicture()
	who := e.selectMemberMenu("Heal whom? ", true, e.party[0])
	if who == nil {
		e.CharacterSheet()
		return
	}
	if e.healPlayer(who, byte(paladinLevel(p)*2), false) {
		e.printMessage(who.Name() + " feels better")
	} else {
		e.printMessage(who.Name() + " is unaffected")
	}
	e.AddAffect(p, affectLaidOnHands, layOnHandsTime, 0, 0)
	e.CharacterSheet()
}

// paladinCure = O0AF:37EC (paladin): remove the diseases (asking first
// when there are none), once per cure left.
func (e *Engine) paladinCure(p *Character) {
	e.loadPicture()
	who := e.selectMemberMenu("Cure whom? ", true, e.party[0])
	if who == nil {
		e.CharacterSheet()
		return
	}
	sick := false
	for _, a := range diseaseAffects {
		if e.FindAffect(who, a) != nil {
			sick = true
		}
	}
	answer := byte('Y')
	if !sick {
		e.combatName(who, "is not diseased", 0, false)
		answer = e.YesNo(colourLightMagenta, colourGreen, colourWhite, "cure anyway: ")
		e.clearCombatText()
	}
	if answer == 'Y' {
		e.curingDisease = true
		for _, a := range diseaseAffects {
			e.RemoveAffect(who, a, nil)
		}
		e.curingDisease = false
		if p.PaladinCures > 0 {
			p.PaladinCures--
		}
		if e.FindAffect(p, affectCuredToday) == nil {
			e.AddAffect(p, affectCuredToday, cureDiseaseTime, 0, 1)
		}
		e.printMessage(who.Name() + " is cured")
	}
	e.CharacterSheet()
}

// healPlayer = O0E3:2432 heal_player (c, points, onlyWounded): hit points
// up to the maximum for the living, the dying and the unconscious; one who
// was down wakes up (out of combat).
func (e *Engine) healPlayer(c *Character, points byte, onlyWounded bool) bool {
	switch c.Status {
	case statusOkay, statusAnimated, statusUnconscious, statusDying:
	default:
		return false
	}
	if onlyWounded && c.HP >= c.MaxHP {
		return false
	}
	c.HP += points
	if c.HP > c.MaxHP {
		c.HP = c.MaxHP
	}
	if c.Okay == 0 {
		if c.Status == statusDying {
			c.Status = statusUnconscious
		}
		if c.Status == statusUnconscious && e.gameState != stateCombat {
			e.callAffect(affectWakeUp, c, nil, 1)
		}
	}
	return true
}

const affectWakeUp = 78
