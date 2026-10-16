package engine

// Treasure: the TREASURE command (O050:1E3D load_item) and the random item
// generator (O13C:10B0, 10F2).

// TREASURE's item operand: below 128 an ITEM<area>.DAX block, 255 none,
// otherwise 128 + the number of random items.
const (
	treasureRandom = 128
	treasureNone   = 255
	coinKinds      = 7
)

// cmdTreasure = O050:1E3D (copper .. jewelry, items).
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
		for _, it := range readItems(b, itemRecSize) {
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

// randomTreasureType: O050:1E3D's table of what a random treasure item is.
func (e *Engine) randomTreasureType() byte {
	switch r := e.Dice(1, 100); {
	case r <= 31:
		switch r2 := e.Dice(1, 100); {
		case r2 <= 20:
			switch e.Dice(1, 10) {
			case 1, 2:
				return 6
			case 3, 4, 5, 6:
				return 8
			case 7, 8:
				return 70
			case 9:
				return 15
			default:
				return 29
			}
		case r2 <= 44:
			return [...]byte{5, 12, 30}[e.Dice(1, 3)-1]
		case r2 <= 60:
			switch k := e.Dice(1, 20); {
			case k <= 3:
				return 7
			case k <= 6:
				return 24
			case k <= 9:
				return 22
			case k <= 12:
				return 25
			case k <= 14:
				return 23
			case k <= 17:
				return 28
			default:
				return 15
			}
		default:
			if e.Dice(1, 4) < 4 {
				switch k := e.Dice(1, 20); {
				case k <= 8:
					return 18
				case k <= 13:
					return 17
				case k == 14:
					return 13
				case k == 15:
					return 16
				case k <= 18:
					return 19
				default:
					return 20
				}
			}
			return [...]byte{1, 4, 4, 9, 9, 10, 14, 71}[e.Dice(1, 8)-1]
		}
	case r <= 62:
		return 39
	case r <= 70:
		return 40
	case r <= 80:
		return 37
	case r <= 85:
		return 59
	case r <= 95:
		switch k := e.Dice(1, 20); {
		case k <= 3:
			return 31
		case k <= 11:
			return 35
		case k <= 16:
			return 36
		case k == 17:
			return 32
		case k == 18:
			return 33
		default:
			return 34
		}
	default:
		return [...]byte{50, 50, 47, 47, 52, 53}[e.Dice(1, 6)-1]
	}
}

// itemBonus = O13C:10B0: +1 (60%), +2 (30%) or +3.
func (e *Engine) itemBonus() byte {
	switch r := e.Dice(1, 10); {
	case r <= 6:
		return 1
	case r <= 9:
		return 2
	}
	return 3
}

// Item name words and types the generator uses.
const (
	wordPlusZero     = 110 // the word for "+n" is wordPlusZero+n
	itemScrollMage   = 39  // scroll types: magic-user and cleric spells
	itemScrollCleric = 40
	wordScrollCount  = 99 // + 1..3: "1 spell" ..
	wordScroll       = 39
	wordMageScroll   = 40
	wordClericScroll = 38
	scrollSpellPrice = 150
	defaultItemCount = 6
	unidentified     = 4
)

// Scroll spells by level (O13C:10F2's tables at DS 580C .. 589E): cleric
// levels 1..6, then magic-user levels 1..7. A level's spell is entry 1d(n-1)
// of its n entries; magic-user level 3 rolls one past its table into the
// next one's first byte (spell 47), kept as its last entry.
var (
	clericScrollSpells = [6][]byte{
		{7, 1, 2, 3, 4, 5, 6, 7, 8},
		{0, 22, 23, 24, 25, 26, 27, 28},
		{0, 37, 38, 39, 40, 46, 42, 43, 44},
		{0, 58, 66, 67, 68, 69, 70},
		{0, 71, 72, 73, 74, 75, 76},
		{0, 36, 56},
	}
	mageScrollSpells = [7][]byte{
		{0, 9, 10, 13, 16, 17, 19, 20, 14, 15, 12},
		{0, 29, 30, 31, 32, 33, 34, 35},
		{0, 45, 46, 48, 49, 50, 51, 52, 53, 54, 55, 47},
		{47, 81, 82, 83, 84, 85, 86, 87, 88, 89, 100},
		{0, 91, 92, 93, 94},
		{0, 111, 112, 113, 114, 110},
		{0, 115, 116, 117},
	}
)

// createItem = O13C:10F2 (type): a new item of the base type with its
// bonus, name words, weight and price rolled the way treasure gets them.
func (e *Engine) createItem(typ byte) *Item {
	it := &Item{}
	it.Hidden = defaultItemCount
	it.Type = typ
	template := 0
	switch {
	case typ >= 1 && typ <= 37 || typ == 50 || typ == 59 || typ == 70 || typ == 71:
		it.Plus = e.itemBonus()
		plusWord := byte(int(int8(it.Plus)) + wordPlusZero)
		switch typ {
		case 29:
			it.Words = [3]byte{plusWord, 28, 29}
			it.Hidden = unidentified
		case 7:
			if e.Dice(1, 5) == 5 {
				template = 49
			}
			it.Words[2], it.Words[1] = 7, plusWord
		case 31:
			it.Words = [3]byte{plusWord, 48, 31}
			it.Hidden = unidentified
		case 32, 33, 34, 35, 36:
			it.Words = [3]byte{plusWord, 47, typ}
			it.Hidden = unidentified
		case 50:
			it.Plus = 4
			it.Words = [3]byte{107, 90, 50}
		case 59:
			it.Plus, it.Plus2 = 1, 1
			it.Words = [3]byte{wordPlusZero + 1, 49, 46}
		default:
			it.Words[2], it.Words[1] = typ, plusWord
		}
		it.Weight, it.Count = itemWeight(typ)
		plus := uint16(int8(it.Plus))
		switch typ {
		case 12, 5, 30:
			it.Price = plus * 50
		case 37:
			it.Price = plus * 1250
		case 32, 33, 50:
			it.Price = plus * 1500
		case 34:
			it.Price = plus * 1750
		case 35:
			it.Price = plus * 2000
		case 36:
			it.Price = plus * 2500
		default:
			it.Price = plus * 1000
		}
	case typ == itemScrollMage || typ == itemScrollCleric:
		n := byte(e.Dice(1, 3))
		it.Words[0] = n + wordScrollCount
		it.Words[1] = wordScroll
		if typ == itemScrollCleric {
			it.Words[2] = wordClericScroll
		} else {
			it.Words[2] = wordMageScroll
		}
		it.Plus = 1
		it.Weight = 25
		for k := byte(0); k < n; k++ {
			var level int
			var spells []byte
			if typ == itemScrollMage {
				level = e.Dice(1, len(mageScrollSpells))
				it.Hidden = unidentified
				spells = mageScrollSpells[level-1]
			} else {
				level = e.Dice(1, len(clericScrollSpells))
				it.Hidden = 0
				spells = clericScrollSpells[level-1]
			}
			it.Spells[k] = spells[e.Dice(1, len(spells)-1)]
			it.Price = uint16(level)*scrollSpellPrice + it.Price
		}
	case typ == 52:
		template = 33
	case typ == 53:
		template = 9
	case typ == 47:
		template = [4]int{1, 17, 25, 41}[(e.Dice(1, 8)-1)/2]
	}
	if template == 0 {
		return it
	}
	for k := 0; k <= 2; k++ {
		it.Words[k] = byte(itemTemplates[template+k])
	}
	it.Plus, it.Plus2 = 1, 0
	it.Weight = itemTemplates[template+3]
	it.Count = 0
	it.Price = itemTemplates[template+4]
	for k := 1; k <= 3; k++ {
		it.Spells[k-1] = byte(itemTemplates[template+4+k])
	}
	return it
}

// itemWeight: O13C:10F2's weights (and bundle counts) by base type.
func itemWeight(typ byte) (uint16, byte) {
	switch typ {
	case 1, 17:
		return 75, 0
	case 6, 14, 15, 23, 25, 37, 29:
		return 50, 0
	case 9:
		return 125, 0
	case 8, 16, 24:
		return 100, 0
	case 70, 31:
		return 150, 0
	case 71:
		return 175, 0
	case 4, 50:
		return 10, 0
	case 5:
		return 5, 20
	case 10, 18:
		return 60, 0
	case 22:
		return 80, 0
	case 7:
		return 20, 0
	case 13:
		return 40, 0
	case 19:
		return 35, 0
	case 20, 32:
		return 250, 0
	case 33:
		return 400, 0
	case 34:
		return 300, 0
	case 35:
		return 350, 0
	case 36:
		return 450, 0
	case 28, 59:
		return 1, 0
	}
	return 4, 20
}
