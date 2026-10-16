package engine

// The character sheet and the small displays it is made of (units O0AF
// and O0ED of the reconstruction).

// Rows and columns of the character sheet.
const (
	sheetStatsRow          = 7
	sheetLevelRow          = 15
	sheetCombatRow         = 17
	sheetWeaponRow         = 20
	sheetStatColumn        = 5
	acBase                 = 60 // armour class is kept as 60 - AC
	affectDetectMagicItems = 5
	itemTypeWand           = 86
	wordQuarrel            = 135 // name words that are already plural
	wordArrows             = 177
)

var statLabels = [6]string{"STR ", "INT ", "WIS ", "DEX ", "CON ", "CHA "}

// CharacterSheet = O0AF:0088 playerDisplayFull: the selected character's
// sheet.
func (e *Engine) CharacterSheet() {
	c := e.selected()
	e.drawScreenFrame()
	e.displayName(c, 1, 1, false)
	if c.NPC() {
		e.displayStringBg(int(c.PName[0])+3, 1, colourGreen, colourBlack, "(NPC)")
	}
	x := 1
	for _, s := range []string{genderNames[c.Sex], raceNames[c.Race]} {
		e.displayStringBg(x, 3, colourWhite, colourBlack, s)
		x += len(s) + 1
	}
	e.displayStringBg(x, 3, colourWhite, colourBlack, "Age "+itoa(int(c.Age)))
	e.displayStringBg(1, 4, colourWhite, colourBlack, alignmentNames[c.Alignment])
	e.displayStringBg(1, 5, colourWhite, colourBlack, classNames[c.Class])
	for i, l := range statLabels {
		e.displayStringBg(1, sheetStatsRow+i, colourGreen, colourBlack, l)
		e.showStat(i, false)
	}
	e.showCoins()
	e.displayStringBg(1, sheetLevelRow, colourWhite, colourBlack, "Level")
	e.showLevels(c)
	e.displayStringBg(17, sheetLevelRow, colourWhite, colourBlack, "Exp "+itoa(int(int32(c.Exp))))
	e.showCombatStats()
	row := sheetWeaponRow
	if w := c.Slots[slotWeapon]; w != nil {
		e.displayStringBg(1, row, colourWhite, colourBlack, "Weapon")
		e.ItemName(c, w, 8, row, false, true)
	}
	row++
	if a := c.Slots[slotBodyArmour]; a != nil {
		e.displayStringBg(2, row, colourWhite, colourBlack, "Armor")
		e.ItemName(c, a, 8, row, false, true)
	}
	row++
	e.displayStringBg(1, row, colourWhite, colourBlack, "Status")
	e.displayStringBg(8, row, colourGreen, colourBlack, statusNames[c.Status])
}

// showLevels: the class levels "5" or "3/3" in the sheet's level row (part
// of playerDisplayFull, also drawn by the creation's reroll loop).
func (e *Engine) showLevels(c *Character) {
	levels := ""
	first := firstClassLevel(c)
	for k := 0; k <= clsMonk; k++ {
		old := int8(c.OldLevels[k])
		if int8(c.Levels[k]) > 0 || old < first && old > 0 {
			if levels != "" {
				levels += "/"
			}
			levels += itoa(int(byte(int8(c.Levels[k]) + old)))
		}
	}
	e.displayStringBg(7, sheetLevelRow, colourWhite, colourBlack, levels)
}

// showCoins = O0AF:0596: the coins carried, jewelry first, right-aligned names.
func (e *Engine) showCoins() {
	c := e.selected()
	e.ClearCells(12, sheetStatsRow, 26, 14)
	row := sheetStatsRow
	for k := coinJewelry; k >= 0; k-- {
		if c.Coins[k] > 0 {
			e.displayStringBg(20-len(coinNames[k]), row, colourGreen, colourBlack, coinNames[k])
			e.displayStringBg(21, row, colourGreen, colourBlack, itoa(int(c.Coins[k])))
			row++
		}
	}
}

// showCombatStats = O0AF:0697 display_player_stats01: AC, hit points,
// THAC0, damage, encumbrance and movement.
func (e *Engine) showCombatStats() {
	c := e.selected()
	e.ApplyItems(c)
	y := sheetCombatRow
	e.displayStringBg(1, y, colourWhite, colourBlack, "AC    ")
	e.displayAC(c, 4, y)
	e.displayStringBg(1, y+1, colourWhite, colourBlack, "HP    ")
	e.displayHP(c, 4, y+1, false)
	x := 8
	e.displayStringBg(x+1, y, colourWhite, colourBlack, "THAC0   ")
	e.displayStringBg(x+7, y, colourGreen, colourBlack, itoa(int(byte(acBase-int(c.THAC0)))))
	dmg := itoa(int(c.Dice[0])) + "d" + itoa(int(c.Sides[0]))
	if b := int8(c.Bonus[0]); b > 0 {
		dmg += "+" + itoa(int(c.Bonus[0]))
	} else if b < 0 {
		dmg += "-" + itoa(int(-b))
	}
	e.displayStringBg(x, y+1, colourWhite, colourBlack, "Damage        ")
	e.displayStringBg(x+7, y+1, colourGreen, colourBlack, dmg)
	x = 22
	e.displayStringBg(x, y, colourWhite, colourBlack, "Encumbrance      ")
	e.displayStringBg(x+12, y, colourGreen, colourBlack, itoa(int(c.Weight)))
	move := c.Move
	if e.FindAffect(c, affectHaste) != nil {
		move *= 2
	}
	if e.FindAffect(c, affectSlow) != nil {
		move >>= 1
	}
	e.displayStringBg(x+3, y+1, colourWhite, colourBlack, "Movement   ")
	e.displayStringBg(x+12, y+1, colourGreen, colourBlack, itoa(int(move)))
}

// showStat = O0AF:09AE display_stat (stat, highlight): one ability score
// (and exceptional strength) in its sheet row.
func (e *Engine) showStat(stat int, hi bool) {
	c := e.selected()
	col := colourGreen
	if hi {
		col = colourLightMagenta
	}
	x := sheetStatColumn
	e.ClearCells(x, stat+sheetStatsRow, 11, stat+sheetStatsRow)
	if c.Stats[stat].Value < 10 {
		x++
	}
	e.displayStringBg(x, stat+sheetStatsRow, col, colourBlack, itoa(int(c.Stats[stat].Value)))
	if stat == statStr && c.Stats[statStr].Value == strengthMaximum && c.Str00.Value > 0 {
		s := itoa(int(c.Str00.Value))
		if c.Str00.Value < 10 {
			s = "0" + s
		}
		if c.Str00.Value == 100 {
			s = "00"
		}
		e.displayStringBg(7, sheetStatsRow, col, colourBlack, "("+s+")")
	}
}

// displayAC = O0ED:0933 minus (c, x, y): the armour class.
func (e *Engine) displayAC(c *Character, x, y int) {
	s := itoa(abs(int(c.AC[0]) - acBase))
	if c.AC[0] > acBase {
		s = "-" + s
	}
	e.displayStringBg(x, y, colourGreen, colourBlack, s)
}

// displayHP = O0ED:09BA display_hp (c, x, y, highlight): yellow when wounded.
func (e *Engine) displayHP(c *Character, x, y int, hi bool) {
	col := colourGreen
	if c.HP < c.MaxHP {
		col = colourYellow
	}
	if hi {
		col = colourLightMagenta
	}
	e.displayStringBg(x, y, col, colourBlack, itoa(int(c.HP)))
}

// displayName = O0ED:18A2 (c, x, y, possessive): light red when out of
// action, yellow for monsters, cyan otherwise.
func (e *Engine) displayName(c *Character, x, y int, possessive bool) {
	col := colourLightRed
	if c.Okay != 0 {
		col = colourCyan
		if c.Side == 1 {
			col = colourYellow
		}
	}
	e.displayStringBg(x, y, col, colourBlack, c.Name())
	if possessive {
		e.displayStringBg(x+int(c.PName[0]), y, col, colourBlack, "'s")
	}
}

// ItemName = O0ED:0466 (c, item, x, y, readied, display): build the item's
// shown name into its name field: the readied column, "*" for magic when
// someone detects it, the count, the known name words (plural for counts);
// draw it when display is set.
func (e *Engine) ItemName(c *Character, it *Item, x, y int, readied, display bool) {
	name := ""
	if readied {
		if it.Readied != 0 {
			name = " Yes  "
		} else {
			name = " No   "
		}
	}
	detect := false
	for _, p := range e.party {
		if e.FindAffect(p, affectDetectMagicItems) != nil {
			detect = true
			break
		}
	}
	if detect && (int8(it.Plus) > 0 || int8(it.Plus2) > 0 || it.Cursed != 0) {
		name += "*" + " "
	}
	if it.Count > 0 {
		name += itoa(int(it.Count)) + " "
	}
	shown := 0 // bit i-1: word i is known and present
	for i := 1; i <= 3; i++ {
		if it.Words[i-1] != 0 && it.Hidden>>(3-i)&1 == 0 {
			shown += 1 << (i - 1)
		}
	}
	plural := false
	for i := 3; i >= 1; i-- {
		if shown>>(i-1)&1 == 0 {
			continue
		}
		name += itemWords[it.Words[i-1]]
		if it.Count < 2 || plural {
			name += " "
			continue
		}
		if 1<<(i-1) == shown || i == 1 && shown > 4 && it.Type != itemTypeWand || i == 2 && shown%2 == 0 ||
			i == 3 && it.Type == itemTypeWand ||
			it.Words[2] != wordQuarrel && (it.Type == itemTypeArrows || it.Type == itemTypeBolts) && it.Words[2] != wordArrows {
			name += "s "
			plural = true
		} else {
			name += " "
		}
	}
	it.setName(name)
	if display {
		e.displayStringBg(x, y, colourGreen, colourBlack, name)
	}
}
