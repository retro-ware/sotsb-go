package engine

// The character sheet and the small displays it is made of (units O0AF
// and O0ED of the reconstruction).

// Rows and columns of the character sheet.
const (
	sheetStatsRow    = 9
	sheetLevelRow    = 7
	sheetWeaponRow   = 21
	sheetStatColumn  = 5
	sheetRightColumn = 20 // the right half: status, coins, hit points, ...
	acBase           = 60 // armour class is kept as 60 - AC
	wordPlural       = 93 // a third name word that is already plural
)

var statLabels = [6]string{"STR ", "INT ", "WIS ", "DEX ", "CON ", "CHA "}

// CharacterSheet = O100:0041 playerDisplayFull: the selected character's
// sheet. A human who changed class shows "old/new" class and levels.
func (e *Engine) CharacterSheet() {
	c := e.selected()
	e.drawShopFrame()
	e.displayName(c, 1, 1, false)
	e.displayStringBg(sheetRightColumn, 1, colourWhite, colourBlack, "Status:")
	e.displayStringBg(27, 1, colourGreen, colourBlack, statusNames[c.Status])
	e.displayStringBg(1, 3, colourWhite, colourBlack, genderNames[c.Sex])
	e.displayStringBg(8, 3, colourWhite, colourBlack, itoa(int(c.Age))+" years")
	e.displayStringBg(1, 4, colourWhite, colourBlack, alignmentNames[c.Alignment])
	e.displayStringBg(sheetRightColumn, 4, colourWhite, colourBlack, raceNames[c.Race])
	class := classNames[c.Class]
	if c.Race == raceHuman {
		if k, ok := formerClass(c); ok {
			class += "/" + classNames[k]
		}
	}
	e.displayStringBg(1, 5, colourWhite, colourBlack, class)
	e.displayStringBg(1, sheetLevelRow, colourWhite, colourBlack, "Level")
	e.displayStringBg(7, sheetLevelRow, colourWhite, colourBlack, levelsText(c))
	e.displayStringBg(24, 6, colourWhite, colourBlack, "Experience:")
	exp := itoa(int(int32(c.Exp)))
	e.displayStringBg((20-len(exp))/2+sheetRightColumn, sheetLevelRow, colourGreen, colourBlack, exp)
	for i, l := range statLabels {
		e.displayStringBg(1, sheetStatsRow+i, colourGreen, colourBlack, l)
		e.showStat(i, false)
	}
	e.showCoins()
	e.showCombatStats()
	if w := c.Slots[slotWeapon]; w != nil {
		e.ItemName(c, w, 1, sheetWeaponRow, false, true)
	}
	if a := c.Slots[slotBodyArmour]; a != nil {
		e.ItemName(c, a, 1, sheetWeaponRow+1, false, true)
	}
}

// formerClass: the first class a dual-classed character left behind
// below its new class's level, if any (O100:0041).
func formerClass(c *Character) (int, bool) {
	first := firstClassLevel(c)
	for k := 0; k < classCount; k++ {
		if old := int8(c.OldLevels[k]); old > 0 && old < first {
			return k, true
		}
	}
	return 0, false
}

// changedClass: whether any old class level is kept (O100:0041).
func changedClass(c *Character) bool {
	for k := 0; k < classCount; k++ {
		if int8(c.OldLevels[k]) > 0 {
			return true
		}
	}
	return false
}

// levelsText: the class levels "5" or "3/3"; a human who changed class
// shows the new class's level, then "/" and the old one while it is still
// the higher.
func levelsText(c *Character) string {
	levels := ""
	if c.Race == raceHuman && changedClass(c) {
		for k := 0; k < classCount; k++ {
			if int8(c.Levels[k]) > 0 {
				levels += itoa(int(c.Levels[k]))
				break
			}
		}
		if k, ok := formerClass(c); ok {
			levels += "/" + itoa(int(c.OldLevels[k]))
		}
		return levels
	}
	for k := 0; k < classCount; k++ {
		if int8(c.Levels[k]) > 0 {
			if levels != "" {
				levels += "/"
			}
			levels += itoa(int(byte(int8(c.Levels[k]) + int8(c.OldLevels[k]))))
		}
	}
	return levels
}

// showCoins = O100:063A: the coins carried, copper first, values
// right-aligned to the frame.
func (e *Engine) showCoins() {
	c := e.selected()
	e.ClearCells(sheetRightColumn, sheetStatsRow, screenLastCol-1, 15)
	row := sheetStatsRow
	for k := 0; k <= coinJewelry; k++ {
		if c.Coins[k] > 0 {
			v := itoa(int(c.Coins[k]))
			e.displayStringBg(sheetRightColumn, row, colourGreen, colourBlack, coinNames[k])
			e.displayStringBg(screenLastCol-1-len(v), row, colourGreen, colourBlack, v)
			row++
		}
	}
}

// padLeft = O100:0709: right-align s in width columns.
func padLeft(s string, width int) string {
	for len(s) < width {
		s = " " + s
	}
	return s
}

// showCombatStats = O100:078E display_player_stats01: hit points, armour
// class, encumbrance, THAC0, movement and damage.
func (e *Engine) showCombatStats() {
	c := e.selected()
	e.ApplyItems(c)
	e.displayStringBg(sheetRightColumn, 3, colourWhite, colourBlack, "Hit Points ")
	e.ClearCells(31, 3, 37, 3)
	e.displayHP(c, 31, 3, false, true)
	e.displayStringBg(1, 17, colourWhite, colourBlack, "Armor Class")
	acX := 15 // three characters: 10 and better, -10 and worse
	switch {
	case c.AC[0] >= 1 && c.AC[0] <= 50, c.AC[0] >= 61 && c.AC[0] <= 69:
		acX = 16
	case c.AC[0] >= 51 && c.AC[0] <= 60:
		acX = 17
	}
	e.ClearCells(15, 17, 17, 17)
	e.displayAC(c, acX, 17)
	e.displayStringBg(sheetRightColumn, 17, colourWhite, colourBlack, "Encumbrance")
	e.displayStringBg(33, 17, colourGreen, colourBlack, padLeft(itoa(int(c.Weight)), 5))
	e.displayStringBg(1, 18, colourWhite, colourBlack, "THAC0   ")
	e.displayStringBg(16, 18, colourGreen, colourBlack, padLeft(itoa(int(byte(acBase-int(c.THAC0)))), 2))
	e.displayStringBg(sheetRightColumn, 18, colourWhite, colourBlack, "Movement")
	e.moveRate(c)
	e.displayStringBg(35, 18, colourGreen, colourBlack, padLeft(itoa(int(c.Move)), 3))
	dmg := itoa(int(c.Dice[0])) + "d" + itoa(int(c.Sides[0]))
	if b := int8(c.Bonus[0]); b > 0 {
		dmg += "+" + itoa(int(b))
	} else if b < 0 {
		dmg += "-" + itoa(int(-b))
	}
	e.displayStringBg(1, 19, colourWhite, colourBlack, "Damage")
	e.displayStringBg(11, 19, colourGreen, colourBlack, padLeft(dmg, 7))
}

// showStat = O100:0AE2 display_stat (stat, highlight): one ability score
// (and exceptional strength) in its sheet row; "*" when it differs from
// the character's own.
func (e *Engine) showStat(stat int, hi bool) {
	c := e.selected()
	col := colourGreen
	if hi {
		col = colourLightMagenta
	}
	row := stat + sheetStatsRow
	x := sheetStatColumn
	e.ClearCells(x, row, 11, row)
	if c.Stats[stat].Value < 10 {
		x++
	}
	e.displayStringBg(x, row, col, colourBlack, itoa(int(c.Stats[stat].Value)))
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
	if c.Stats[stat].Value != c.Stats[stat].Base || stat == statStr && c.Str00.Base != c.Str00.Value {
		e.displayStringBg(12, row, col, colourBlack, "*")
	}
}

// displayAC = O151:0B1C minus (c, x, y): the armour class.
func (e *Engine) displayAC(c *Character, x, y int) {
	s := itoa(abs(int(c.AC[0]) - acBase))
	if c.AC[0] > acBase {
		s = "-" + s
	}
	e.displayStringBg(x, y, colourGreen, colourBlack, s)
}

// displayHP = O151:0BA6 (c, x, y, highlight, showMax): the hit points,
// yellow when wounded, then "/max" when asked.
func (e *Engine) displayHP(c *Character, x, y int, hi, showMax bool) {
	col := colourGreen
	if c.HP < c.MaxHP {
		col = colourYellow
	}
	if hi {
		col = colourLightMagenta
	}
	hp := itoa(int(c.HP))
	e.displayStringBg(x, y, col, colourBlack, hp)
	if showMax {
		e.displayStringBg(x+len(hp), y, colourGreen, colourBlack, "/"+itoa(int(c.MaxHP)))
	}
}

// displayName = O151:185C (c, x, y, possessive): light red when out of
// action, yellow for monsters in combat, cyan otherwise.
func (e *Engine) displayName(c *Character, x, y int, possessive bool) {
	col := colourLightRed
	if c.Okay != 0 {
		col = colourCyan
		if c.Side == 1 && e.gameState == stateCombat {
			col = colourYellow
		}
	}
	e.displayStringBg(x, y, col, colourBlack, c.Name())
	if possessive {
		e.displayStringBg(x+int(c.PName[0]), y, col, colourBlack, "'s")
	}
}

// ItemName = O151:0438 (c, item, x, y, readied, display): build the item's
// shown name into its name field: the readied column, "*" for magic when
// someone detects it, the count, the known name words (plural for counts);
// a bag shows the number of items it holds in place of its second word.
// Draw it when display is set.
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
		if e.FindAffect(p, affectDetectMagic) != nil {
			detect = true
			break
		}
	}
	if detect && (int8(it.Plus) > 0 || int8(it.Plus2) > 0 || it.Cursed != 0) {
		name += "*" + " "
	}
	bag := it.Type == itemTypeBag
	if it.Count > 0 && !bag {
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
		if bag && i == 2 {
			name += itoa(int(it.Count))
		} else {
			name += itemWords[it.Words[i-1]]
		}
		if it.Count < 2 || plural {
			name += " "
			continue
		}
		missile := it.Type == itemTypeArrows || it.Type == itemTypeDart || it.Type == itemTypeBolts
		if 1<<(i-1) == shown || i == 1 && shown > 4 || i == 1 && bag || i == 2 && shown%2 == 0 ||
			i == 3 && missile && it.Words[2] != wordPlural {
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
