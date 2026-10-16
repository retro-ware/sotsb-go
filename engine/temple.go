package engine

// The temple (unit O06C): COMBAT with the area2 temple flag set opens it.

// Silver Blades' temple heals for free: each service tells how it went
// (Curse asked for gold and offered to cast anyway).

const (
	statusStoned = 7
	statusGone   = 8 // left the party's world: "is not here"
)

// castCureAnyway = O113:673D (char, text): "<name> <text>" and whether to
// cast anyway, No preselected.
func (e *Engine) castCureAnyway(c *Character, text string) byte {
	e.combatName(c, text, 0, false)
	k := e.yesNoFrom(answerNo, colourLightMagenta, colourGreen, colourWhite, "cast cure anyway: ")
	e.clearCombatText()
	return k
}

// templeSay: "<name> <text>" under the list (O151:173F at row 0, or 10).
func (e *Engine) templeSay(c *Character, text string, row int) { e.combatName(c, text, row, true) }

// templeCureBlindness = O06C:0016.
func (e *Engine) templeCureBlindness() {
	c := e.selected()
	text := "is not blind"
	if e.FindAffect(c, affectBlind) != nil {
		e.RemoveAffect(c, affectBlind, nil)
		text = "can see!"
	}
	e.templeSay(c, text, 10)
}

// templeCureDisease = O06C:00A3.
func (e *Engine) templeCureDisease() {
	c := e.selected()
	ill := false
	for _, d := range diseaseAffects[1:] {
		if e.FindAffect(c, d) != nil {
			ill = true
		}
	}
	text := "is not Diseased"
	if ill {
		text = "is cured"
		e.curingDisease = true
		for _, d := range diseaseAffects[1:] {
			e.RemoveAffect(c, d, nil)
		}
		e.statAffects(c, statStr)
		e.curingDisease = false
	}
	e.templeSay(c, text, 0)
}

// templeCureWounds = O06C:0201 (kind): light, serious, critical or Heal;
// none for the dead, the stoned or the gone.
func (e *Engine) templeCureWounds(kind int) {
	c := e.selected()
	text := "is healed"
	switch c.Status {
	case statusDead:
		kind, text = 0, "must be raised from dead"
	case statusStoned:
		kind, text = 0, "can only be healed by a stonemason"
	case statusGone:
		kind, text = 0, "is not here"
	}
	full := c.HP == c.MaxHP
	if full {
		text = "has no need of healing"
	}
	var amount byte
	switch kind {
	case 1:
		amount = byte(e.Dice(1, 8))
	case 2:
		amount = byte(e.Dice(2, 8) + 1)
	case 3:
		amount = byte(e.Dice(3, 8) + 3)
	case 4:
		if !full {
			amount = c.MaxHP - c.HP - byte(e.Dice(1, 4))
		}
		cured := false
		for _, a := range append([]byte{affectBlind, affectFeebleminded}, diseaseAffects[1:]...) {
			if e.FindAffect(c, a) != nil {
				e.RemoveAffect(c, a, nil)
				cured = true
			}
		}
		if cured {
			e.templeSay(c, "is cured", 0)
			if !full {
				text = "is fully healed"
			}
		}
		e.statAffects(c, statInt)
		e.statAffects(c, statWis)
	}
	if !full {
		e.healPlayer(c, amount, false)
	}
	e.templeSay(c, text, 0)
}

// templeRaiseDead = O06C:04C7: the dead (not elves) back with 1 hit point
// and a point of constitution less; at 14 and above the constitution bonus
// to the hit points shrinks with it.
func (e *Engine) templeRaiseDead() {
	c := e.selected()
	text := "is not dead yet"
	switch {
	case c.Status != statusDead:
	case c.Race == raceElf:
		text = "cannot be raised"
	default:
		text = "is ressurected"
		e.raiseDead(c)
	}
	e.templeSay(c, text, 0)
}

// raiseDead: the raising itself (O06C:04C7 after its checks).
func (e *Engine) raiseDead(c *Character) {
	e.curingDisease = true
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
	if c.MaxHP > c.HPRoll {
		excess = c.MaxHP - c.HPRoll
	}
	if *con < 14 {
		return
	}
	var levels byte
	for i := 0; i <= classCount; i++ { // the eighth is the first of the old levels
		lvl := int8(c.OldLevels[0])
		if i < classCount {
			lvl = int8(c.Levels[i])
		}
		if lvl <= 0 {
			continue
		}
		switch {
		case i == clsFighter || i == clsPaladin || i == clsRanger:
			levels += byte(int(*con-14) * int(lvl))
		case *con > 15:
			levels += byte(lvl) * 2
		default:
			levels += byte(lvl)
		}
	}
	if levels != 0 {
		excess /= levels
	}
	if *con < 17 || int8(c.Levels[clsFighter]) > 0 || int8(c.Levels[clsFighter]) > int8(c.MulticlassLevel) {
		c.MaxHP -= excess
	}
	e.ApplyItems(c)
}

// templeNeutralizePoison = O06C:0732.
func (e *Engine) templeNeutralizePoison() {
	c := e.selected()
	text := "is not poisoned"
	if e.FindAffect(c, affectPoisoned) != nil {
		if c.HP == 0 {
			c.HP = 1
		}
		e.curingDisease = true
		e.RemoveAffect(c, affectPoisoned, nil)
		e.RemoveAffect(c, affectSlowPoison, nil)
		e.RemoveAffect(c, affectPoisonDamage, nil)
		e.curingDisease = false
		c.Okay = 1
		c.Status = statusOkay
		text = "is no longer poisoned"
	}
	e.templeSay(c, text, 0)
}

// templeRemoveCurse = O06C:0846: cursed items or the curse affect.
func (e *Engine) templeRemoveCurse() {
	c := e.selected()
	cursed := e.FindAffect(c, affectCursed) != nil
	for _, it := range c.Items {
		if it.Cursed != 0 {
			cursed = true
		}
	}
	text := "is not cursed"
	if cursed {
		e.spellTarget = c
		e.spellRemoveCurse()
		text = "rejoices.  The curse has been lifted!"
	}
	e.templeSay(c, text, 0)
}

// templeStoneToFlesh = O06C:092B.
func (e *Engine) templeStoneToFlesh() {
	c := e.selected()
	text := "is not petrified"
	if c.Status == statusStoned {
		c.Status = statusOkay
		c.Okay = 1
		c.HP = 1
		text = "is turned to flesh"
	}
	e.templeSay(c, text, 0)
}

// The temple's services (DS:00BD).
var templeServices = [10]string{"Cure Blindness", "Cure Disease", "Cure Light Wounds", "Cure Serious Wounds",
	"Cure Critical Wounds", "Heal", "Neutralize Poison", "Raise Dead", "Remove Curse", "Stone to Flesh"}

// templeHeal = O06C:09C5: the list of services.
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

// templeMenu = O06C:0C89 temple_shop.
func (e *Engine) templeMenu() {
	e.gameState = stateShop
	e.shopFrame = false
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
			} else {
				e.selectStep(key)
			}
		case 'V':
			e.viewPlayer()
		case 'T':
			e.takeMoney()
		case 'P':
			if !extra {
				e.poolMoney()
			} else {
				e.selectStep(key)
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
