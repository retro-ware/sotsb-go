package engine

// O0AF: the View screen (character sheet, items, money, spell lists).

// ClearScreen = R4CE:0A02.
func (e *Engine) ClearScreen() {
	if e.Trace != nil {
		e.Trace("R4CE:0A02", "ClearScreen")
	}
	e.fillCells(0, 0, 0x27, 0x18)
}

// viewChar = DS:4716, the character on the sheet.
func (e *Engine) viewChar() *Character { return e.view.char }

type viewState struct {
	char  *Character // DS:4716
	start *Character // DS:471A
}

// canLoseItem = O0AF:0EC1 (item): readied items must be unreadied first;
// a scroll someone is about to scribe from asks before it goes.
func (e *Engine) canLoseItem(it *Item) bool {
	ok := false
	switch {
	case it.Readied != 0:
		e.printMessage("Must be unreadied")
	case !e.isScroll(it):
		ok = true
	case it.Spells[0] > 0x7F || it.Spells[1] > 0x7F || it.Spells[2] > 0x7F:
		c := e.selected()
		e.displayName(c, 1, 0x15, false)
		e.msgCol, e.msgRow = len(c.Name())+2, 0x15
		e.DisplayTextBlock(1, 0x15, 0x26, 0x16, 0xE, false, " was going to scribe from that scroll")
		ok = e.YesNo(0xD, 0xA, 0xF, "is it Okay to lose it? ") == 'Y'
	default:
		ok = true
	}
	e.ClearCells(1, 0x15, 0x26, 0x16)
	return ok
}

// ItemsMenu = O0AF:1583 use_item (var done): the character's items with
// Ready, Use, Trade, Drop, Halve, Join, Sell and Id. Returns done (an item
// was used in combat).
func (e *Engine) ItemsMenu() bool {
	c := e.selected()
	key := byte(' ')
	index := 0
	redraw, full := true, true
	done := false
	for key != 0 && key != 'E' && !done && c.ItemCount > 0 {
		count := c.ItemCount
		if len(c.Items) > 0 {
			menu := "Ready"
			if c.Okay != 0 && wordAt(e.area, 2*areaWordNoMagic) == 0 &&
				(e.gameState == stateCamp || e.gameState == stateWilderness || e.gameState == stateDungeon ||
					e.gameState == stateCombat && c.Combat != nil && c.Combat.CanUse != 0) {
				menu += " Use"
			}
			hired := c.Control >= npcControl && c.Okay != 0 && c.Status != statusAnimated
			if !hired && e.gameState != stateCombat {
				menu += " Trade"
			}
			menu += " Drop"
			if c.ItemCount < maxItems {
				menu += " Halve"
			}
			menu += " Join"
			if !hired && e.gameState == stateShop {
				menu += " Sell"
			}
			if e.gameState == stateShop {
				menu += " Id"
			}
			list := make([]*ListNode, len(c.Items))
			for i, x := range c.Items {
				e.ItemName(c, x, 0, 0, true, false)
				list[i] = &ListNode{Text: x.Name()}
			}
			if full || e.itemsRedraw {
				e.drawListFrame()
				e.displayName(c, 1, 1, true)
				e.displayStringBg(int(c.PName[0])+4, 1, colourGreen, colourBlack, "Items")
				e.displayStringBg(1, 3, colourWhite, colourBlack, "Ready Item")
				redraw, full, e.itemsRedraw = true, false, false
			}
			var sel *ListNode
			key = e.ListMenu("", menu, colourLightMagenta, colourGreen, colourWhite, 1, 5, 38, 22, list, true, &redraw, &index, &sel)
			var it *Item
			for i, n := range list {
				if n == sel {
					it = c.Items[i]
				}
			}
			if it != nil {
				switch key {
				case 'V':
					e.itemDebug(it)
					redraw, full = true, true
				case 'R':
					e.readyItem(it)
				case 'U':
					if it.Readied == 0 {
						e.printMessage("Must be Readied")
						key = ' '
					} else if e.isScroll(it) || it.Spells[1] > 0 && it.Spells[2] < 0x80 {
						done = e.useItem(it)
						if e.gameState != stateCombat {
							done = false
						}
						if !done {
							full = true
						}
					}
				case 'T':
					if e.canLoseItem(it) {
						e.tradeItem(it)
					} else {
						key = ' '
					}
					full = true
				case 'D':
					if e.canLoseItem(it) {
						e.ItemName(c, it, 0, 0, false, false)
						e.msgCol, e.msgRow = 1, 21
						e.DisplayTextBlock(1, 21, 38, 22, colourYellow, true, "Your "+it.Name()+"will be gone forever")
						if e.YesNo(colourLightMagenta, colourGreen, colourWhite, "Drop It? ") == 'Y' {
							e.loseItem(e.selected(), it)
							redraw = true
						}
						e.ClearCells(1, 21, 38, 22)
					} else {
						key = ' '
					}
				case 'H':
					e.halveItem(it)
				case 'J':
					e.joinItems(it)
				case 'S':
					if e.canLoseItem(it) {
						e.sellItem(it)
					} else {
						key = ' '
					}
				case 'I':
					if e.identifyItem(it) {
						redraw = true
					}
				}
			}
			e.ApplyItems(c)
		}
		if c.ItemCount != count {
			redraw = true
		}
	}
	return done
}

// itemDebug = O0AF:10A6: the item record's fields (a debugging page).
func (e *Engine) itemDebug(it *Item) {
	e.drawScreenFrame()
	labels := []string{"itemptr:      ", "namenum(1):   ", "namenum(2):   ", "namenum(3):   ", "plus:         ",
		"plussave:     ", "ready:        ", "identified:   ", "cursed:       ", "value:        ", "special(1):   ",
		"special(2):   ", "special(3):   ", "dice large:    ", "sides large:   "}
	for i, l := range labels {
		e.displayStringBg(1, i+1, colourGreen, colourBlack, l)
	}
	values := []int{int(it.Type), int(it.Words[0]), int(it.Words[1]), int(it.Words[2]), int(it.Plus), int(it.Plus2),
		int(it.Readied), int(it.Hidden), int(it.Cursed), int(it.Price), int(it.Spells[0]), int(it.Spells[1]), int(it.Spells[2]),
		int(e.baseItem(it.Type, baseDiceLarge)), int(e.baseItem(it.Type, baseSidesLarge))}
	for i, v := range values {
		e.displayStringBg(20, i+1, colourGreen, colourBlack, itoa(v))
	}
	e.displayAndWait(colourGreen, colourBlack, "press a key")
}

// Base item record fields itemDebug shows.
const (
	baseDiceLarge  = 2
	baseSidesLarge = 3
)

// readyItem = O0AF:1ED6 ready_Item (item): ready or unready, with the reasons it
// cannot be readied and the item's own effect (+0x3E > 0x7F).
func (e *Engine) readyItem(it *Item) {
	effect := it.Spells[2] > 0x7F
	c := e.selected()
	if it.Readied != 0 {
		if it.Cursed != 0 {
			e.printMessage("It's Cursed")
			return
		}
		it.Readied = 0
		if effect {
			e.itemReadyEffect(it, false)
		}
		return
	}
	why := 0
	typ := it.Type
	if int(e.baseItem(typ, 1))+int(c.Hands) > 2 {
		why = 3
	}
	slot := int(e.baseItem(typ, 0))
	switch {
	case slot <= 8:
		if c.Slots[slot] != nil {
			why = 2
		}
	case slot == 9:
		if c.Slots[slotRing2] != nil {
			why = 2
		}
	}
	if typ == 0x49 && c.Slots[slotArrows] != nil {
		why, slot = 2, slotArrows
	}
	if typ == 0x1C && c.Slots[slotBolts] != nil {
		why, slot = 2, slotBolts
	}
	if e.baseItem(typ, 0xD)&c.ClassMask == 0 {
		why = 1
	}
	switch why {
	case 0:
		it.Readied = 1
		if effect {
			e.itemReadyEffect(it, true)
		}
	case 1:
		e.printMessage("Wrong Class")
	case 2:
		// slot 9 (ring 2 taken) names Slots[9], which may be empty: the
		// original then reads through a nil pointer.
		name := ""
		if o := c.Slots[slot]; o != nil {
			e.ItemName(c, o, 0, 0, false, false)
			name = o.Name()
		}
		e.printMessage("already using " + name)
	case 3:
		if !(e.gameState == 5 && c.Auto != 0) {
			e.printMessage("Your hands are full!")
		}
	}
}

// Item powers itemReadyEffect knows beyond classcalc.go's.
const (
	effectGrantsAffect   = 0 // the affect Spells[1] names
	effectAlignmentBurn  = 4 // Spells[1]: alignment (low nibble), damage (high)
	effectStrength       = 5
	effectConstitution   = 6
	effectScore          = 8 // Spells[1]: the score
	effectHammer         = 9 // unready: the spiritual hammer goes
	effectDexterity      = 10
	effectIntelligence   = 12
	effectStrAndInt      = 13
	alignmentMask        = 15
	alignmentDamageShift = 4
	wizardrySlots        = 3 // the first mage slot levels the ring doubles
	mageSlotLevels       = 5
)

// itemReadyEffect = O0AF:1B04 (item, ready): what a magic item does as it
// is readied (ready) or put away.
func (e *Engine) itemReadyEffect(it *Item, ready bool) {
	c := e.selected()
	switch itemEffect(it) & (itemPowerFlag - 1) {
	case effectGrantsAffect:
		e.itemCall = true
		e.callItem = it
		var flag byte
		if !ready {
			flag = 1
		}
		e.callAffect(itemEffect(it), c, nil, flag)
	case effectWizardry:
		if ready {
			for s := 0; s < wizardrySlots; s++ {
				c.SpellSlots[slotsMage+s] *= 2
			}
			break
		}
		lvl := int(int8(c.Levels[clsMage]))
		if oldClassUsable(c) {
			lvl += int(int8(c.OldLevels[clsMage]))
		}
		for s := 0; s < mageSlotLevels; s++ {
			c.SpellSlots[slotsMage+s] = 0
		}
		c.SpellSlots[slotsMage] = 1
		for l := 2; l <= lvl; l++ {
			for s := 1; s <= mageSlotLevels; s++ {
				c.SpellSlots[slotsMage+s-1] += mageSlotData[l*mageSlotLevels+s]
			}
		}
		// memorized spells beyond the slots are forgotten
		var used [mageSlotLevels + 1]int
		for i, sp := range c.Memorized {
			if sp == 0 || spellClass(sp) != spellClassMage {
				continue
			}
			l := int(spellLevel(sp))
			used[l]++
			if used[l] > int(c.SpellSlots[slotsMage+l-1]) {
				c.Memorized[i] = 0
			}
		}
	case effectFreeAction:
		e.statAffects(c, statDex)
		e.thiefSkills(c)
	case effectAlignmentBurn:
		if it.Spells[1]&alignmentMask != c.Alignment {
			it.Readied = 0
			e.rv.dmgType = dmgMagic
			if e.gameState == stateCombat {
				e.combatUseHook()
			}
			e.damagePerson(c, int(it.Spells[1]>>alignmentDamageShift), 0, false)
			e.itemsRedraw = true
		}
	case effectStrength:
		e.statAffects(c, statStr)
	case effectConstitution:
		e.statAffects(c, statCon)
		e.statAffects(c, statCha)
	case effectScore:
		e.statAffects(c, int(it.Spells[1]))
	case effectHammer:
		if !ready {
			e.RemoveAffect(c, affectSpiritualHammer, nil)
		}
	case effectDexterity:
		e.statAffects(c, statDex)
	case effectThievery:
		e.thiefSkills(c)
	case effectIntelligence:
		e.statAffects(c, statInt)
	case effectStrAndInt:
		e.statAffects(c, statStr)
		e.statAffects(c, statInt)
	}
}

// tradeItem = O0AF:211C (item).
func (e *Engine) tradeItem(it *Item) {
	to := e.view.start
	e.DrawMainScreen()
	to = e.SelectCharacter("Trade with Whom?", true, to)
	if to == nil {
		return
	}
	e.view.start = to
	if e.overloadedBy(to, it) {
		e.printMessage("Overloaded")
		return
	}
	e.AddItemCopy(to, it)
	e.loseItem(e.selected(), it)
	e.ApplyItems(to)
}

// halveItem = O0AF:21D5 (item): half the stack becomes a new unreadied
// item right after it.
func (e *Engine) halveItem(it *Item) {
	half := it.Count >> 1
	if half == 0 {
		e.printMessage("Can't halve that")
		return
	}
	n := *it
	n.Count = half
	n.Readied = 0
	it.Count -= half
	c := e.selected()
	for i, x := range c.Items {
		if x == it {
			c.Items = append(c.Items[:i+1], append([]*Item{&n}, c.Items[i+1:]...)...)
			return
		}
	}
}

// joinItems = O0AF:2285 (item): every matching stack joins it, up to 255
// a stack. (The original compares +0x3C..+0x3E of each item with itself.)
func (e *Engine) joinItems(it *Item) {
	total := int(it.Count)
	c := e.selected()
	for _, p := range append([]*Item(nil), c.Items...) {
		if p == it || p.Count == 0 {
			continue
		}
		same := p.Words == it.Words && p.Type == it.Type && p.Plus == it.Plus && p.Plus2 == it.Plus2 && p.Cursed == it.Cursed
		if !same || p.Weight != it.Weight || p.Spells[0] >= 2 {
			continue
		}
		if int(p.Count)+total <= 0xFF {
			total += int(p.Count)
			e.loseItem(c, p)
		} else {
			it.Count = 0xFF
			p.Count -= byte(0xFF - total)
			total = int(p.Count)
			it = p
		}
	}
	it.Count = byte(total)
}

// Who can read a scroll (O0AF:2478): magic-users and clerics always, a
// high-level thief now and then.
const (
	scrollThiefLevel  = 9  // above: a thief may try
	scrollThiefChance = 75 // percent
)

// canReadScroll is the class test of O0AF:2478.
func (e *Engine) canReadScroll(c *Character) bool {
	if activeClass(c) == clsMage || currentClass(c) == clsMage || activeClass(c) == clsCleric ||
		currentClass(c) == clsCleric || int8(c.Levels[clsMage]) > 0 || int8(c.Levels[clsCleric]) > 0 {
		return true
	}
	return int8(c.Levels[clsThief]) > scrollThiefLevel && e.Dice(1, 100) <= scrollThiefChance
}

// useItem = O0AF:2478 (item, var done): cast the scroll's spell or the
// item's own (+0x3D & 127, charges +0x3C). Returns done: the spell was cast
// and, in combat, the turn is over.
func (e *Engine) useItem(it *Item) bool {
	idx := -1
	e.usingItem = false
	var spell byte
	done := false
	if e.isScroll(it) {
		e.usedScroll = it
		var flag bool
		spell = e.SpellList(2, 1, &idx, &flag)
	} else if it.Spells[1] > 0 && it.Spells[2] < itemPowerFlag {
		e.usingItem = true
		spell = it.Spells[1] & (itemPowerFlag - 1)
	}
	if spell == 0 {
		return false
	}
	c := e.selected()
	if e.gameState == stateCombat && c.Auto == 0 {
		e.combatUseHook()
	}
	if e.usingItem {
		e.combatName(c, "uses an item", combatMsgRow, false)
		if e.gameState == stateCombat {
			e.displayStringBg(0, 23, colourGreen, colourBlack, "Item:")
			e.ItemName(c, it, 5, 23, false, true)
		} else {
			e.ItemName(c, it, 1, 22, false, true)
		}
		e.gameDelay()
		e.clearCombatText()
	}
	e.usingItem = true
	if !e.isScroll(it) || e.canReadScroll(c) {
		e.CastSpell(spell, c.Auto, 0, &done)
	} else {
		e.combatName(c, "oops!", e.msgRow+1, true)
	}
	e.usingItem = false
	if e.gameState == stateCombat && e.spellRec(spell, spellEndsTurn) != 0 {
		done = e.endTurn(c)
	}
	if !done {
		return false
	}
	if e.isScroll(it) {
		e.scrollUsed(c, it, spell)
	} else if it.Spells[0] > 0 {
		if it.Count > 1 {
			it.Count--
		} else {
			it.Spells[0]--
			if it.Spells[0] == 0 {
				e.loseItem(c, it)
			}
		}
	}
	return true
}

// spellRec = DS:37DC + 16*spell.
func (e *Engine) spellRec(spell byte, k int) byte {
	return spellTable[spell][k]
}

// Selling and identifying (O0AF:27C2, 2A29).
const (
	stackSaleDivisor = 20  // a stack sells for count × half price / 20
	identifyCost     = 200 // gold pieces
)

// sellItem = O0AF:27C2 (item): half the price (a twentieth of that per
// piece of a stack), paid in platinum and gold; what the seller cannot
// carry goes into the treasure pool.
func (e *Engine) sellItem(it *Item) {
	var v uint16
	if it.Price > 0 {
		v = it.Price >> 1
	}
	if it.Count > 1 {
		// (the arrows/quarrels branch of the original can never be taken)
		v = uint16(uint32(it.Count)*uint32(v)) / stackSaleDivisor
	}
	e.ItemName(e.viewChar(), it, 0, 0, false, false)
	e.DisplayTextBlock(1, 0x15, 0x26, 0x16, 0xE, true,
		"I'll give you "+itoa(int(v))+" gold pieces for your "+it.Name())
	if e.YesNo(0xD, 0xA, 0xF, "Is It a Deal? ") == 'Y' {
		e.printMessage("Sold!")
		c := e.selected()
		e.loseItem(c, it)
		platinum, gold := v/platinumGold, v%platinumGold
		if over, room := willOverload(c, gold+platinum); over {
			e.printMessage("Overloaded.  Money will be put in pool.")
			if room > platinum {
				c.Coins[coinPlatinum] += platinum
			} else {
				c.Coins[coinPlatinum] += room
				e.treasureCoins[coinPlatinum] += int32(platinum - room)
			}
		} else {
			c.Coins[coinPlatinum] += platinum
		}
		c.Coins[coinGold] += gold
	}
	e.ClearCells(1, 0x15, 0x26, 0x16)
}

// identifyItem = O0AF:2A29 (item, redraw): 200 gp, from the character or
// else the treasure pool, shows the hidden words.
func (e *Engine) identifyItem(it *Item) (redraw bool) {
	paid := false
	e.ItemName(e.viewChar(), it, 0, 0, false, false)
	e.DisplayTextBlock(1, 0x15, 0x26, 0x16, 0xE, true, "For 200 gold pieces I'll identify your "+it.Name())
	if e.YesNo(0xD, 0xA, 0xF, "Is It a Deal? ") == 'Y' {
		if g := goldWorth(e.selected()); g >= identifyCost {
			paid = true
			e.setGold(uint16(g - identifyCost))
		} else if g := poolGold(e.treasureCoins); g >= identifyCost {
			paid = true
			e.setPoolGold(uint16(g - identifyCost))
		} else {
			e.printMessage("Not Enough Money")
		}
	}
	if paid {
		if it.Hidden == 0 {
			e.DisplayTextBlock(1, 0x15, 0x26, 0x16, 0xE, true, "I can't tell anything new about your "+it.Name())
		} else {
			it.Hidden = 0
			e.ItemName(e.viewChar(), it, 0, 0, false, false)
			e.DisplayTextBlock(1, 0x15, 0x26, 0x16, 0xE, true, "It looks like some sort of "+it.Name())
			redraw = true
		}
		e.gameDelay()
	}
	e.ClearCells(1, 0x15, 0x26, 0x16)
	return
}

// SpellList = O0AF:347C spell_menu2 (kind, mode, index, any): the list of spells of
// one kind (0 memory, 1 book, 2 scroll, 3 scrolls, 4 choose, 5 to be
// memorised, 6 to be scribed); returns the chosen spell or 0.
func (e *Engine) SpellList(kind, mode int, index *int, any *bool) byte {
	titles := []string{"in Memory", "in Spell Book", "on Scroll", "on Scrolls", "to choose from", "to be memorized", "to be scribed"}
	title := ""
	if kind >= 0 && kind < len(titles) {
		title = titles[kind]
	}
	*any = e.buildSpellList(kind)
	if !*any {
		return 0
	}
	if *index < 0 {
		e.DrawBoxTitled(1, 1, 0x26, 1, 0, 0, "")
		c := e.selected()
		e.displayName(c, 1, 1, true)
		e.DisplayString(len(c.Name())+4, 1, 0xA, "Spells "+title)
	}
	return e.spellMenu(mode, index)
}
