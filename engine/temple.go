package engine

// The temple (unit O042): COMBAT with the area2 temple flag set opens it.

// castCureAnyway = O042:0045 (text): "<name> <text>" and whether to pay
// anyway.
func (e *Engine) castCureAnyway(text string) byte {
	e.combatName(e.selected(), text, 0, false)
	k := e.YesNo(colourLightMagenta, colourGreen, colourWhite, "cast cure anyway: ")
	e.clearCombatText()
	return k
}

// payForCure = O042:00F1 (cure, price): the price from the member's coins
// or else the pool; 'Y' when paid.
func (e *Engine) payForCure(cure string, price uint16) byte {
	e.msgCol, e.msgRow = textLeft, textTop
	e.DisplayTextBlock(textLeft, textTop, textRight, textBottom, colourGreen, true,
		cure+" will only cost "+itoa(int(price))+" gold pieces.")
	k := e.YesNo(colourLightMagenta, colourGreen, colourWhite, "pay for cure ")
	if k == 'Y' {
		if gold := uint16(goldWorth(e.selected())); price <= gold {
			e.setGold(gold - price)
		} else if pool := poolGold(e.treasureCoins); int32(price) <= pool {
			e.setPoolGold(uint16(pool - int32(price)))
		} else {
			e.printMessage("Not enough money.")
			k = 'N'
		}
	}
	if k == 'Y' {
		e.clearCombatText()
		e.combatName(e.selected(), "is cured.", 0, true)
	}
	return k
}

// templeCure: "not ..." and the question when not needed, then the price.
func (e *Engine) templeCure(needed bool, notMsg, cure string, price uint16) bool {
	k := byte('Y')
	if !needed {
		k = e.castCureAnyway(notMsg)
	}
	return k == 'Y' && e.payForCure(cure, price) == 'Y'
}

// Affects the temple removes.
const (
	affectBlind     = 33
	affectPoison2   = 22
	affectPoison3   = 15
	affectCursed    = 36
	affectDead      = 32
	affectEnfeebled = 68
	statusStoned    = 7
)

// templeCureBlindness = O042:027E.
func (e *Engine) templeCureBlindness() {
	c := e.selected()
	if e.templeCure(e.FindAffect(c, affectBlind) != nil, "is not blind.", "Cure Blindness", 1000) {
		e.RemoveAffect(c, affectBlind, nil)
	}
}

// templeCureDisease = O042:0322.
func (e *Engine) templeCureDisease() {
	c := e.selected()
	ill := false
	for _, d := range diseaseAffects {
		if e.FindAffect(c, d) != nil {
			ill = true
		}
	}
	if e.templeCure(ill, "is not Diseased.", "Cure Disease", 1000) {
		e.curingDisease = true
		for _, d := range diseaseAffects {
			e.RemoveAffect(c, d, nil)
		}
		e.curingDisease = false
	}
}

// templeCureWounds = O042:043B (kind): light, serious, critical or Heal.
func (e *Engine) templeCureWounds(kind int) {
	c := e.selected()
	switch kind {
	case 1:
		if e.payForCure("Cure Light Wounds", 100) == 'Y' {
			e.healPlayer(c, byte(e.Dice(1, 8)), false)
		}
	case 2:
		if e.payForCure("Cure Serious Wounds", 350) == 'Y' {
			e.healPlayer(c, byte(e.Dice(2, 8)+1), false)
		}
	case 3:
		if e.payForCure("Cure Critical Wounds", 600) == 'Y' {
			e.healPlayer(c, byte(e.Dice(3, 8)+3), false)
		}
	case 4:
		if e.payForCure("Heal", 5000) != 'Y' {
			return
		}
		e.healPlayer(c, c.MaxHP-c.HP-byte(e.Dice(1, 4)), false)
		e.RemoveAffect(c, affectBlind, nil)
		for _, d := range diseaseAffects {
			e.RemoveAffect(c, d, nil)
		}
		e.RemoveAffect(c, affectEnfeebled, nil)
		e.statAffects(c, statInt)
		e.statAffects(c, statWis)
	}
}

// templeRaiseDead = O042:0636: back with 1 hit point and a point of
// constitution less; above 13 the constitution bonus to the hit points
// shrinks with it.
func (e *Engine) templeRaiseDead() {
	c := e.selected()
	dead := c.Status == statusDead || c.Status == statusAnimated
	k := byte('Y')
	if !dead {
		k = e.castCureAnyway("is not dead.")
	}
	if k != 'Y' || e.payForCure("Raise Dead", 5500) != 'Y' || !dead {
		return
	}
	e.curingDisease = true
	e.RemoveAffect(c, affectDead, nil)
	e.RemoveAffect(c, affectPoisoned, nil)
	e.curingDisease = false
	c.HP = 1
	c.Status = statusOkay
	c.Okay = 1
	con := &c.Stats[statCon].Value
	if *con > 0 {
		*con--
	}
	var excess byte
	if c.HPRoll < c.MaxHP {
		excess = c.MaxHP - c.HPRoll
	}
	if *con <= 13 {
		return
	}
	var levels byte
	for i := 0; i <= 7; i++ {
		lvl := int8(c.Levels[i])
		if lvl <= 0 {
			continue
		}
		switch {
		case i == clsFighter:
			levels += byte(int(*con-14) * int(int8(c.Levels[clsFighter])))
		case *con < 16:
			levels += byte(lvl)
		default:
			levels += byte(lvl) * 2
		}
	}
	if levels != 0 {
		excess = byte(uint32(excess) / uint32(levels))
	}
	if *con < 17 || int8(c.Levels[clsFighter]) > 0 || int8(c.MulticlassLevel) < int8(c.Levels[clsFighter]) {
		c.MaxHP -= excess
	}
}

// templeNeutralizePoison = O042:088B.
func (e *Engine) templeNeutralizePoison() {
	c := e.selected()
	if e.templeCure(e.FindAffect(c, affectPoisoned) != nil, "is not poisoned.", "Neutralize Poison", 1000) {
		e.curingDisease = true
		e.RemoveAffect(c, affectPoisoned, nil)
		e.RemoveAffect(c, affectPoison2, nil)
		e.RemoveAffect(c, affectPoison3, nil)
		e.curingDisease = false
	}
}

// templeRemoveCurse = O042:0974: cursed items or the curse affect.
func (e *Engine) templeRemoveCurse() {
	c := e.selected()
	cursed := false
	for _, it := range c.Items {
		if it.Cursed != 0 {
			cursed = true
			break
		}
	}
	k := byte('Y')
	if !cursed && e.FindAffect(c, affectCursed) == nil {
		k = e.castCureAnyway("is not cursed.")
	}
	if k == 'Y' && e.payForCure("Remove Curse", 3500) == 'Y' {
		e.spellTarget = c
		e.spellRemoveCurse()
	}
}

// templeStoneToFlesh = O042:0A62.
func (e *Engine) templeStoneToFlesh() {
	c := e.selected()
	if e.templeCure(c.Status == statusStoned, "is not stoned.", "Stone to Flesh", 2000) && c.Status == statusStoned {
		c.Status = statusOkay
		c.Okay = 1
		c.HP = 1
	}
}

// The temple's services (DS:00BD).
var templeServices = [10]string{"Cure Blindness", "Cure Disease", "Cure Light Wounds", "Cure Serious Wounds",
	"Cure Critical Wounds", "Heal", "Neutralize Poison", "Raise Dead", "Remove Curse", "Stone to Flesh"}

// templeHeal = O042:0B11: the list of services.
func (e *Engine) templeHeal() {
	list := make([]*ListNode, len(templeServices))
	for i, s := range templeServices {
		list[i] = &ListNode{Text: s}
	}
	e.ClearCells(0, statusRow, screenLastCol, statusRow)
	redraw := true
	e.drawWildernessFrame()
	index := 0
	for done := false; !done; {
		e.displayStringBg(1, 1, colourWhite, colourBlack, cut(e.selected().Name()+", how can we help you?", 40))
		var sel *ListNode
		key := e.ListMenu("", "Heal Exit", colourLightMagenta, colourGreen, colourWhite, 2, 4, 38, 15, list, false, &redraw, &index, &sel)
		switch key {
		case 'H', keyEnter:
			switch index {
			case 0:
				e.templeCureBlindness()
			case 1:
				e.templeCureDisease()
			case 2, 3, 4, 5:
				e.templeCureWounds(index - 1)
			case 6:
				e.templeNeutralizePoison()
			case 7:
				e.templeRaiseDead()
			case 8:
				e.templeRemoveCurse()
			case 9:
				e.templeStoneToFlesh()
			case 10:
				done = true
			}
		case 0:
			done = true
		}
	}
	e.loadPicture()
	e.playerSummary(e.selected())
}

// templeMenu = O042:0DD9 temple_shop.
func (e *Engine) templeMenu() {
	e.gameState = stateShop
	e.shopFrame = wordAt(e.area, 2*areaWordFreeSteps) == 0
	e.loadPicture()
	e.shopFrame = true
	e.playerSummary(e.selected())
	e.treasureCoins = [coinKinds]int32{}
	e.poolHasMoney = false
	for leave := false; !leave; {
		money, _ := e.treasureLeft()
		menu := "Heal View Pool Appraise Exit"
		if money {
			menu = "Heal View Take Pool Share Appraise Exit"
		}
		key, extra := e.DisplayInput("", menu, colourLightMagenta, colourGreen, colourWhite, true, false)
		appraised := false
		switch key {
		case 'H':
			if !extra {
				e.templeHeal()
			}
		case 'V':
			e.viewPlayer()
		case 'T':
			e.takeMoney()
		case 'P':
			if !extra {
				e.poolMoney()
			}
		case 'S':
			e.shareMoney()
		case 'A':
			appraised = e.appraise()
		case 'E':
			if money, _ := e.treasureLeft(); money {
				e.msgCol, e.msgRow = textLeft, textTop
				e.DisplayTextBlock(textLeft, textTop, textRight, textBottom, colourGreen, true,
					`As you leave a priest says, "Excuse me but you have left some money here" `)
				e.DisplayTextBlock(textLeft, textTop, textRight, textBottom, colourGreen, true,
					"Do you want to go back and retrieve your money?")
				if e.scriptMenu("", "~Yes ~No", colourLightMagenta, colourGreen, colourWhite, false, false) == 1 {
					leave = true
				} else {
					e.ClearCells(textLeft, textTop, textRight, textBottom)
				}
			} else {
				leave = true
			}
		case scanHome, scanEnd:
			e.cycleSelected(key)
		}
		switch {
		case key == 'B' || key == 'T':
			e.loadPicture()
		case key == 'A' && appraised:
			e.loadPicture()
		}
		e.playerSummary(e.selected())
	}
}
