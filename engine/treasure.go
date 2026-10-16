package engine

// Treasure: the TREASURE command (O03C:1B9D load_item) and the random item
// generator (O0BA:0FCF, 1007).

// TREASURE's item operand: below 128 an ITEM<area>.DAX block, 255 none,
// otherwise 128 + the number of random items.
const (
	treasureRandom = 128
	treasureNone   = 255
	coinKinds      = 7
)

// cmdTreasure = O03C:1B9D (copper .. jewelry, items).
func (e *Engine) cmdTreasure() {
	e.readArgs(8)
	for k := 0; k < coinKinds; k++ {
		e.treasureCoins[k] = int32(e.arg(k + 1))
	}
	what := byte(e.arg(8))
	if what < treasureRandom {
		b := e.daxBlock("ITEM"+e.areaDigit(), int(what))
		if len(b) == 0 {
			e.displayAndWait(colourWhite, colourBlack, "Unable to find item file")
			e.Exit()
			return
		}
		for _, it := range readRecords[Item](b, itemRecSize) {
			e.treasureItems = append([]*Item{it}, e.treasureItems...)
		}
		return
	}
	if what == treasureNone {
		return
	}
	for n := byte(1); n <= what-treasureRandom; n++ {
		it := e.createItem(e.randomTreasureType())
		e.treasureItems = append([]*Item{it}, e.treasureItems...)
	}
	for _, it := range e.treasureItems {
		e.ItemName(e.selected(), it, 0, 0, false, false)
	}
}

// randomTreasureType: O03C:1B9D's table of what a random treasure item is.
func (e *Engine) randomTreasureType() byte {
	switch r := e.Dice(1, 100); {
	case r <= 60:
		switch r2 := e.Dice(1, 100); {
		case r2 <= 47 || r2 >= 50 && r2 <= 59:
			if r2 == 45 {
				return 59
			}
			return byte(r2)
		case r2 >= 60 && r2 <= 90:
			switch e.Dice(1, 10) {
			case 1, 2, 3, 4:
				return 36
			case 5, 6, 7:
				return 35
			case 8:
				return 34
			case 9:
				return 37
			default:
				return 38
			}
		case r2 <= 94 && r2 >= 91:
			return 73
		case r2 >= 95 && r2 <= 97:
			return 93
		case r2 >= 98:
			return 77
		default:
			return 59
		}
	case r <= 85:
		return 61
	case r <= 92: // 86..92 (the original's 91..98 branch is reached only by 93..98)
		return 62
	case r <= 98:
		switch e.Dice(1, 15) {
		case 1, 2, 3, 4, 5, 6, 7, 8, 9:
			return 71
		case 10:
			return 84
		default:
			return 79
		}
	}
	return 59
}

// itemBonus = O0BA:0FCF: +1 (70%) or +2.
func (e *Engine) itemBonus() byte {
	if e.Dice(1, 20) <= 14 {
		return 1
	}
	return 2
}

// Item name words and types the generator uses.
const (
	wordPlusFirst    = 161 // "+1" is word 162
	itemArrow        = 21
	itemQuarrel      = 28
	itemGem          = 61
	itemJewel        = 62
	wordGem          = 209
	wordJewelry      = 208
	defaultItemCount = 6
)

// createItem = O0BA:1007 (type): a new item of the base type with its
// bonus, name words, weight and price rolled the way treasure gets them.
func (e *Engine) createItem(typ byte) *Item {
	it := &Item{}
	it.Hidden = defaultItemCount
	it.Type = typ
	template := 0
	switch {
	case typ >= 1 && typ <= 59 || typ == 73 || typ == 77 || typ == 93:
		it.Plus = e.itemBonus()
		plus := int(int8(it.Plus))
		switch typ {
		case itemArrow:
			if e.Dice(1, 5) == 5 {
				template = 49
			}
			it.Words[2] = itemArrow
			it.Words[1] = byte(plus + wordPlusFirst)
		case itemQuarrel:
			it.Words[2] = itemQuarrel
			it.Words[1] = byte(plus + wordPlusFirst)
		case 50, 51:
			it.Words[2] = typ
			it.Words[1] = 49
			it.Words[0] = byte(plus + wordPlusFirst)
			it.Hidden = 4
		case 52:
			it.Words[2] = typ
			it.Words[1] = 50
			it.Words[0] = byte(plus + wordPlusFirst)
			it.Hidden = 4
		case 53, 54, 55, 56, 57, 58:
			it.Words[2] = typ
			it.Words[1] = 48
			it.Words[0] = byte(plus + wordPlusFirst)
			it.Hidden = 4
		case 73:
			it.Words[2] = 61
			it.Words[1] = byte(plus + wordPlusFirst)
		case 77:
			it.Words[2] = 79
			it.Words[1] = 167
			it.Plus = byte(plus*2 + 2)
			switch it.Plus {
			case 4:
				it.Words[0] = 221
			case 6:
				it.Words[0] = 222
			}
		case 93:
			it.Words[2] = 66
			it.Words[1] = 224
			it.Words[0] = byte(plus + wordPlusFirst)
		default:
			it.Words[2] = typ
			it.Words[1] = byte(plus + wordPlusFirst)
		}
		it.Plus2 = 0
		it.Count = 0
		it.Weight, it.Count = itemWeight(typ)
		plus = int(int8(it.Plus))
		switch typ {
		case 59:
			it.Price = uint16(plus * 2500)
		case 73, 28:
			it.Price = uint16(plus * 150)
		case 53, 54:
			it.Price = uint16(plus * 3000)
		case 55, 56:
			it.Price = uint16(plus * 3500)
		case 57:
			it.Price = uint16(plus * 4000)
		case 58:
			it.Price = uint16(plus * 5000)
		case 77:
			it.Price = uint16(plus * 3000)
		default:
			it.Price = uint16(plus * 2000)
		}
	case typ == itemGem || typ == itemJewel:
		n := byte(e.Dice(1, 3))
		if typ == itemGem {
			it.Words[2] = wordGem
		} else {
			it.Words[2] = wordJewelry
		}
		it.Words[1] = n + wordGem
		it.Words[0] = 0
		it.Plus = 1
		it.Weight = 25
		it.Count = 0
		it.Price = 0
		for k := byte(1); k <= n; k++ {
			r := byte(e.Dice(1, 5))
			var spell byte
			if typ == itemGem {
				switch r {
				case 1:
					spell = byte(e.Dice(1, 13)) + 8
				case 2:
					spell = byte(e.Dice(1, 7)) + 28
				case 3:
					spell = byte(e.Dice(1, 11)) + 44
				case 4:
					spell = byte(e.Dice(1, 9)) + 80
				case 5:
					spell = byte(e.Dice(1, 4)) + 90
				}
			} else {
				switch r {
				case 1:
					spell = byte(e.Dice(1, 8))
				case 2:
					spell = byte(e.Dice(1, 7)) + 21
				case 3:
					spell = byte(e.Dice(1, 8)) + 36
				case 4:
					spell = byte(e.Dice(1, 5)) + 65
				case 5:
					spell = byte(e.Dice(1, 6)) + 70
				}
			}
			it.Spells[k-1] = spell
			it.Price = uint16(r)*300 + it.Price
		}
	case typ == 63 || typ == 67:
		template = 41
	case typ == 78 || typ == 79:
		template = 33
	case typ == 84 || typ == 92:
		template = 9
	case typ == 71:
		if e.Dice(1, 8) <= 5 {
			template = 17
		} else {
			template = 1
		}
	}
	if template == 0 {
		return it
	}
	for k := 0; k <= 2; k++ {
		it.Words[k] = byte(itemTemplates[template+k])
	}
	it.Plus, it.Plus2 = 1, 1
	it.Weight = itemTemplates[template+3]
	it.Count = 0
	it.Price = itemTemplates[template+4]
	for k := 1; k <= 3; k++ {
		it.Spells[k-1] = byte(itemTemplates[template+4+k])
	}
	return it
}

// itemWeight: O0BA:1007's weights (and bundle counts) by base type.
func itemWeight(typ byte) (uint16, byte) {
	switch typ {
	case 1, 13, 14, 35:
		return 75, 0
	case 2, 20, 29, 31, 32, 33, 39, 42, 44, 46, 59:
		return 50, 0
	case 3, 24, 40:
		return 125, 0
	case 4, 15, 23, 34, 43, 45, 51:
		return 100, 0
	case 5, 12, 17, 19, 50:
		return 150, 0
	case 6:
		return 15, 0
	case 7:
		return 30, 0
	case 8, 77:
		return 10, 0
	case 9:
		return 25, 5
	case 10, 26, 36:
		return 60, 0
	case 11, 16, 25, 27, 41, 47:
		return 80, 0
	case 18:
		return 175, 0
	case 21:
		return 20, 0
	case 22, 30:
		return 40, 0
	case 37:
		return 35, 0
	case 38, 53:
		return 250, 0
	case 52:
		return 200, 0
	case 54, 56:
		return 400, 0
	case 55:
		return 300, 0
	case 57:
		return 350, 0
	case 58:
		return 450, 0
	case 93:
		return 1, 0
	}
	return 40, 10
}
