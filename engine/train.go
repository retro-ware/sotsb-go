package engine

import (
	"sotsb/dax"
	"sotsb/ega"
)

// ovr17 (second half): Modify, Train and Add on the party screen, the
// portrait and icon editors, and the hit point helpers they share.

// trainingClasses = es:[game+0x550] (game word 680): the classes this training hall teaches
// (a mask that is ANDed with DS:5DE; the party menu only offers Train
// when it is set).
func (e *Engine) trainingClasses() int { return int(wordAt(e.game, 2*gameWordTraining)) }

// Experience a character may have and still be modified: none, or what
// a character brought over from Pool of Radiance starts with.
var modifiableExp = [...]uint32{0, 66667, 100000, 200000}

// Modify lines: the six abilities, the name and the hit points.
const (
	modifyNameLine = 6
	modifyHPLine   = 7
	nameMaxLen     = 15
)

// modifyDraw = O0EF:24E5 (highlight) nested in modifyPlayer: the line,
// for the name with the cursor character ('%' on a space).
func (e *Engine) modifyDraw(line, cursor int, hi bool) {
	c := e.selected()
	switch {
	case line < modifyNameLine:
		e.showStat(line, hi)
	case line == modifyHPLine:
		e.displayHP(c, 31, 3, hi, false)
	case hi:
		n := int(c.PName[0])
		e.displayStringBg(n+1, 1, colourBlack, colourBlack, " ")
		e.displayStringBg(1, 1, colourLightMagenta, colourBlack, c.Name())
		ch := string(rune(c.PName[cursor]))
		if c.PName[cursor] == ' ' {
			ch = "%"
		}
		e.displayStringBg(cursor, 1, colourWhite, colourBlack, ch)
	default:
		e.displayStringBg(1, 1, colourGreen, colourBlack, c.Name())
	}
}

// ModifyCharacter = O0EF:2653 modifyPlayer: pick a character without
// experience and change the abilities, name and hit points. Up/Down (or
// Enter) pick the line, Subtract and Add lower and raise, on the name they
// move the cursor and other letters type over it; Keep (K) accepts, Exit
// or Esc puts the old values back.
func (e *Engine) ModifyCharacter() {
	if !e.pickCharacter() {
		return
	}
	c := e.selected()
	fresh := false
	for _, x := range modifiableExp {
		fresh = fresh || c.Exp == x
	}
	if !fresh || c.MulticlassLevel != 0 {
		e.statusMessage(c.Name()+" can't be modified.", colourYellow, colourBlack)
		return
	}
	e.CharacterSheet()
	var saved [6]byte
	for i := range saved {
		saved[i] = c.Stats[i].Value
	}
	savedStr00, savedMax, savedName := c.Str00.Value, c.MaxHP, c.PName
	race, class, sex := int(c.Race), int(c.Class), int(c.Sex)
	limits := raceStatLimits[race]
	restore := func() {
		for i := range saved {
			c.Stats[i].Value = saved[i]
		}
		c.Str00.Value = savedStr00
		c.MaxHP = savedMax
		c.HP = c.MaxHP
		c.PName = savedName
		e.ApplyItems(c)
	}
	cursor := 1
	line := modifyHPLine
	e.modifyDraw(line, cursor, false)
	line = 0
	e.modifyDraw(line, cursor, true)
	var key byte
	for key != 'K' {
		for extra := true; extra; {
			if line == modifyNameLine {
				extra = false
				key = e.ReadKeyByte()
				if key >= '1' && key <= '9' || key == '\\' { // the keypad
					if key == '\\' {
						key = '7'
					} else {
						key = keypadMap[key-'1']
					}
					extra = true
				}
				if key == 0 {
					key = e.ReadKeyByte()
					extra = true
				}
				if key == keyEsc {
					key = 0
				}
			} else {
				key, extra = e.DisplayInput("Modify: ", "Add Subtract Keep Exit", colourLightMagenta, colourGreen, colourWhite, true, false)
			}
			e.modifyDraw(line, cursor, false)
			if extra {
				switch key {
				case scanDelete:
					if n := int(c.PName[0]); line == modifyNameLine && n > 1 {
						if cursor == n {
							c.PName[0]--
							cursor = int(c.PName[0])
						} else {
							copy(c.PName[cursor:n], c.PName[cursor+1:n+1])
							c.PName[0]--
						}
					}
				case scanDown:
					if line++; line > modifyHPLine {
						line = 0
					}
				case scanUp:
					if line--; line < 0 {
						line = modifyHPLine
					}
				}
			} else {
				switch {
				case key == 'S':
					switch {
					case line < modifyNameLine:
						e.modifyLower(c, line, limits, class, sex)
					case line == modifyHPLine:
						c.MaxHP--
						if m := e.minHitPoints(c); m > c.MaxHP {
							c.MaxHP = m
						}
						c.HP = c.MaxHP
					case cursor == 1:
						cursor = int(c.PName[0])
					default:
						cursor--
					}
				case key == 'A':
					switch {
					case line < modifyNameLine:
						e.modifyRaise(c, line, limits, sex)
					case line == modifyHPLine:
						c.MaxHP++
						if m := e.maxHitPoints(c); m < c.MaxHP {
							c.MaxHP = m
						}
						c.HP = c.MaxHP
					case cursor == int(c.PName[0]):
						cursor = 1
					default:
						cursor++
					}
				case key == keyEnter:
					if line++; line > modifyHPLine {
						line = 0
					}
				case key == keyBackspace:
					if n := int(c.PName[0]); cursor > 1 && line == modifyNameLine {
						for k := cursor; k <= n; k++ {
							c.PName[k-1] = c.PName[k]
						}
						c.PName[0]--
						if cursor > int(c.PName[0]) {
							cursor = int(c.PName[0])
						}
					}
				case key >= ' ' && key <= 'z':
					if line == modifyNameLine {
						if cursor <= nameMaxLen {
							c.PName[cursor] = key
							if cursor++; cursor > nameMaxLen {
								cursor = nameMaxLen
							}
							if cursor > int(c.PName[0]) {
								c.PName[0]++
								c.PName[cursor] = ' '
							}
						}
					} else if key == 'E' {
						restore()
						return
					}
				case key == 0:
					restore()
					return
				}
			}
			e.ApplyItems(c)
			e.showCombatStats()
			e.modifyDraw(line, cursor, true)
		}
	}
	e.clericSlots(c)
	c.Share = 1
	e.setRolledHP(c)
	for i := range c.Stats {
		c.Stats[i].Base = c.Stats[i].Value
	}
	c.Str00.Base = c.Str00.Value
}

// setRolledHP = the tail of modifyPlayer: the rolled hit points are the
// maximum less the constitution bonus averaged over the classes.
func (e *Engine) setRolledHP(c *Character) {
	var bonus byte
	n := 0
	for k := 0; k < classCount; k++ {
		lv := int8(c.Levels[k])
		if lv <= 0 {
			continue
		}
		b := int(e.conBonusForClass(k))
		switch {
		case byte(lv) >= classMaxHitDice[k]:
			bonus += byte((int(classMaxHitDice[k]) - 1) * b)
		case k == clsRanger:
			bonus += byte((int(lv) + 1) * b)
		default:
			bonus += byte(int(lv) * b)
		}
		n++
	}
	if n == 0 {
		return // a divide error in the original
	}
	bonus = byte(int(bonus) / n)
	c.HPRoll = c.MaxHP - bonus
}

// modifyLower: left on ability line i, kept within the race and class
// minimum; strength first gives up exceptional strength.
func (e *Engine) modifyLower(c *Character, i int, limits [16]byte, class, sex int) {
	st := &c.Stats[i].Value
	*st--
	min := limits[statLimitMin+2*i]
	if i == statStr {
		if c.Str00.Value > 0 {
			c.Str00.Value--
			*st++
		} else if *st < limits[sex] {
			*st = limits[sex]
		}
		min = 0
	}
	if i != statStr && *st < min {
		*st = min
	}
	if cm := classStatMinimum[class][i]; *st < cm {
		*st = cm
	}
	switch i {
	case statWis:
		if c.SpellSlots[0] > 0 {
			c.SpellSlots[0] = 1
		}
	case statCon:
		if m := e.maxHitPoints(c); m < c.MaxHP {
			c.MaxHP = m
		}
		c.HP = c.MaxHP
		e.modifyDraw(modifyNameLine, 0, false) // the original's line 6, Curse's hit points
	}
}

// modifyRaise: right on ability line i, kept within the race maximum;
// strength of a fighter, paladin or ranger goes on into exceptional
// strength.
func (e *Engine) modifyRaise(c *Character, i int, limits [16]byte, sex int) {
	st := &c.Stats[i].Value
	*st++
	if i == statStr {
		if max := limits[2+sex]; *st > max {
			if *st > 18 && (int8(c.Levels[clsFighter]) > 0 || int8(c.Levels[clsPaladin]) > 0 ||
				int8(c.Levels[clsRanger]) > 0) && limits[4+sex] > c.Str00.Value {
				c.Str00.Value++
			}
			*st = max
		}
		return
	}
	if max := limits[statLimitMin+2*i+1]; *st > max {
		*st = max
	}
	switch i {
	case statWis:
		if c.SpellSlots[0] > 0 {
			c.SpellSlots[0] = 1
		}
	case statCon:
		if m := e.minHitPoints(c); m > c.MaxHP {
			c.MaxHP = m
		}
		c.HP = c.MaxHP
		e.modifyDraw(modifyNameLine, 0, false) // the original's line 6, Curse's hit points
	}
}

// statLimitMin: in a raceStatLimits row, the minimum of ability i (int ..
// cha) is at statLimitMin+2*i, its maximum one after.
const statLimitMin = 4

// classConBonus = O0EF:44FD (c, class): the constitution hit point bonus per
// die for one class; fighters, paladins and rangers get more above 16.
func classConBonus(c *Character, class int) int8 {
	con := c.Stats[statCon].Value
	bonus := conHitPointBonus[con]
	if class == clsFighter || class == clsRanger || class == clsPaladin {
		switch {
		case con == 17:
			bonus++
		case con == 18:
			bonus += 2
		case con == 19 || con == 20:
			bonus += 3
		case con >= 21 && con <= 23:
			bonus += 4
		case con == 24 || con == 25:
			bonus += 5
		}
	}
	return bonus
}

// minHitPoints = O0EF:457C (char): the least hit points the levels allow,
// one per level (a ranger one more) plus the constitution bonus, averaged
// over the classes.
func (e *Engine) minHitPoints(c *Character) byte {
	var sum byte
	var bonus int8
	count := 0
	for k := 0; k < classCount; k++ {
		lv := int8(c.Levels[k])
		if lv <= 0 {
			continue
		}
		sum += byte(lv)
		count++
		if k == clsRanger {
			sum++
		}
		bonus += classConBonus(c, k)
	}
	if count == 0 {
		return 0 // a divide error in the original
	}
	if bonus < 0 {
		if int(sum) > -int(bonus)+count {
			return byte((int(sum) + int(bonus)) / count)
		}
		return 1
	}
	return byte((int(sum) + int(bonus)) / count)
}

// conBonusForClass = O0EF:22CF (class): the hit point bonus per level for
// the selected character's constitution; above 16 only fighters,
// paladins and rangers get more than 2.
func (e *Engine) conBonusForClass(k int) int8 {
	con := e.selected().Stats[statCon].Value
	switch {
	case con == 3:
		return -2
	case con >= 4 && con <= 6:
		return -1
	case con >= 7 && con <= 14:
		return 0
	case con == 15:
		return 1
	case con == 16:
		return 2
	case con >= 17 && con <= 19:
		if k == clsFighter || k == clsPaladin || k == clsRanger {
			return int8(con - 14)
		}
		return 2
	}
	return 0 // the original leaves the byte unset
}

// maxHitDie: per class, the sides of its hit die.
var maxHitDie = [classCount]int{8, 8, 10, 10, 8, 4, 6}

// hitPointsAfterDice: per class, the hit points at the last rolled level
// plus per level beyond it; a druid's are fixed.
func hitPointsAfterDice(k, beyond int) byte {
	switch k {
	case clsCleric:
		return byte(beyond*2 + 72)
	case clsDruid:
		return 112
	case clsFighter, clsPaladin, clsRanger:
		return byte(beyond*2 + 88)
	case clsMage:
		return byte(beyond + 44)
	}
	return byte(beyond*2 + 60) // thief
}

// maxHitPoints = O0EF:4655 (char): the most hit points the levels allow,
// a full die per level plus the constitution bonus, averaged over the
// classes. Past the rolled levels the original assigns rather than adds.
func (e *Engine) maxHitPoints(c *Character) byte {
	var sum byte
	n := 0
	for k := 0; k < classCount; k++ {
		lv := int(int8(c.Levels[k]))
		if lv <= 0 {
			continue
		}
		n++
		if byte(lv) < classMaxHitDice[k] {
			per := int(e.conBonusForClass(k)) + maxHitDie[k]
			if k == clsRanger {
				lv++
			}
			sum += byte(per * lv)
		} else {
			sum = hitPointsAfterDice(k, lv-int(classMaxHitDice[k])+1)
		}
	}
	if n == 0 {
		return 0 // a divide error in the original
	}
	return byte(int(sum) / n)
}

// iconPane = O0EF:3AB2 (x, y) nested in the icon editor: the two poses
// of the icon in slot 0xC at icon square (x,y), on a cleared 12 cell
// panel (on EGA over the slot 0x19 background icons).
func (e *Engine) iconPane(x, y int) {
	if e.Cfg.Graphics != 'E' && e.Cfg.Graphics != 'T' { // DS:4A1E == 0 (1 EGA, 2 Tandy)
		e.FillRect(x*3, y*0x18, 0xC, 0x18, 3)
	} else {
		e.FillRect(x*3, y*0x18, 0xC, 0x18, 0)
		e.drawIcon(x, y, 0, 0, 0x19)
		e.drawIcon(x+3, y, 0, 0, 0x19) // SotSB draws both background halves as the first
	}
	e.drawIcon(x, y, 0, 0, 0xC)
	e.drawIcon(x+3, y, 0, 1, 0xC)
	e.Flip() // R2BC:1104
}

// iconCopy = O0EF:3B91 (src, dst, recolour) nested in the icon editor:
// copy both poses of icon slot src into slot dst (the original also
// copies the mask planes, which the port keeps in the image); with
// recolour the selected character's six colour pairs (+0xC1..) replace
// the DS:4CAF colours in the copy.
func (e *Engine) iconCopy(src, dst byte, recolour bool) {
	for p := 0; p < 2; p++ {
		if im := e.icons[src][p]; im != nil {
			e.icons[dst][p] = im.Clone()
		}
	}
	if !recolour {
		return
	}
	c := e.selected()
	var to [16]byte
	for i := range to {
		to[i] = byte(i)
	}
	for i := 1; i <= len(c.IconColour); i++ {
		v := c.IconColour[i-1]
		pos := defaultIconColours[i] // the icon part's colour index
		to[pos] = v & 15
		to[pos+8] = v >> 4
	}
	for p := 0; p < 2; p++ {
		recolourPicture(e.icons[dst][p], to)
	}
}

// recolourIcon = R2BC:1418 (identity, to, 0, 0, icon): colour i becomes
// to[i] for i = 0..15 in turn, so a pixel already moved to a later colour
// moves again (3 -> 15 -> to[15]). Mask pixels are left alone. Amiga
// pixels (extra) take the same end result through amigaIconColour.
func recolourPicture(im *dax.Image, to [16]byte) {
	if im == nil {
		return
	}
	var f, am [16]byte
	for c := range f {
		x := byte(c)
		for i := x; i < 16; i++ {
			if x == i {
				x = to[i]
			}
		}
		f[c] = x
		am[amigaIconColour[c]] = amigaIconColour[x]
	}
	for k, v := range im.Data {
		switch {
		case v < 16 && int(v) != im.Mask:
			im.Data[k] = f[v]
		case v >= ega.AmigaBase:
			im.Data[k] = v&^0xF | am[v&0xF]
		}
	}
}

// amigaIconColour (extra): the Amiga palette slot for each EGA colour
// number a character's icon colours hold. The Amiga icon editor (its slot
// table at 0x8B10E, against DS:3CF5) colours the parts in slots 9/1,
// 10/2, 11/3, 6/7, 5/8 and 15/14 where DOS uses 1/9 .. 7/15; the four
// left over pair by shade. So the DOS default colours give the Amiga
// default icon and each step shows another Amiga colour.
var amigaIconColour = [16]byte{0, 9, 10, 11, 6, 4, 5, 15, 12, 1, 2, 3, 7, 13, 8, 14}

// classBit: the training bit of a class (trainers list the bits they teach).
func classBit(class int) byte { return classTrainingBits[class] }

// expThreshold is the experience a class needs to reach the level after
// level (0 or less when there is none).
func expThreshold(class, level int) int32 {
	if level+1 >= len(expTable[class]) {
		return 0
	}
	return expTable[class][level+1]
}

// levelHitPoints = O0EF:4840 (c, mask, classes): hit points for the
// classes of the mask, averaged over the character's classes. A class still
// rolling dice rolls twice and keeps the better (level 1: its first hit
// dice, after that one die); a class past its dice sets (not adds) a fixed
// amount (original quirk).
func (e *Engine) levelHitPoints(c *Character, mask byte, classes int) {
	var rolled, bonus byte // the bonus sums signed values in a byte
	for k := 0; k < classCount; k++ {
		lvl := int8(c.Levels[k])
		if lvl <= 0 || classBit(k)&mask == 0 {
			continue
		}
		if byte(lvl) < classMaxHitDice[k] {
			count := classFirstHitDice[k]
			if lvl > 1 {
				count = 1
			}
			a := byte(e.Dice(int(count), int(classHitDiceSides[k])))
			b := byte(e.Dice(int(count), int(classHitDiceSides[k])))
			rolled += max(a, b)
			bonus += count * byte(classConBonus(c, k))
			continue
		}
		switch k {
		case clsFighter, clsPaladin:
			rolled = 3
		case clsCleric, clsRanger, clsThief:
			rolled = 2
		case clsMage:
			rolled = 1
		}
	}
	rolled = byte(int(rolled) / classes)
	if rolled < 1 {
		rolled = 1
	}
	c.HPRoll += rolled
	bonus /= byte(classes)
	wounds := c.MaxHP - c.HP
	c.MaxHP += rolled + bonus
	c.HP = c.MaxHP - wounds
}

// goldWorth = O100:3A8E getPlayerGold: the character's coins in gold
// pieces.
func goldWorth(c *Character) int32 {
	var copper int32
	for k := range coinValues {
		copper += coinValues[k] * int32(c.Coins[k])
	}
	return copper / coinValues[coinGold]
}

// Coin indexes.
const (
	coinCopper = iota
	coinSilver
	coinElectrum
	coinGold
	coinPlatinum
	coinGems
	coinJewelry
)

const trainingCost = 1000 // gold pieces

// Spells a magic-user knows once trained silently to a level (new
// characters start above level 1), by the magic-user level reached.
var silentMageSpells = map[byte][]int{
	6: {12, 15, 30, 3, 34, 47, 51, 54},
	7: {14, 16, 29, 35, 45, 48, 81},
	8: {10, 9, 32, 55, 85},
	9: {50, 52, 82, 88, 94},
}

// trainSilently = the creation's training loop: train without questions
// or messages until no class can advance any more.
func (e *Engine) trainSilently(c *Character) {
	undo := e.selectTemp(c)
	defer undo()
	savedMask := wordAt(e.game, trainingMaskByte)
	putWord(e.game, trainingMaskByte, 255)
	e.cannotTrain = false
	e.silentTraining = true
	for !e.cannotTrain {
		e.TrainCharacter()
	}
	e.silentTraining = false
	putWord(e.game, trainingMaskByte, savedMask)
}

// trainingMaskByte: the byte of the second area record (p67CA) holding the
// classes the local trainer teaches.
const trainingMaskByte = 1360

// TrainCharacter = O0EF:4A8E train_player: raise every class of the selected
// character whose experience allows it and the local trainer teaches.
func (e *Engine) TrainCharacter() {
	c := e.selected()
	if c.Status != 0 && !e.freeTraining {
		e.statusMessage("we only train conscious people", colourYellow, colourBlack)
		return
	}
	var ready, have byte // classes with enough experience, classes the character has
	var cap int32        // the experience to keep: just short of the second level up
	trainer := e.game[trainingMaskByte]
	lvl := 0 // the last class's level (the original reuses it below)
	for k := 0; k < classCount; k++ {
		if int8(c.Levels[k]) <= 0 {
			continue
		}
		have += classBit(k)
		lvl = int(c.Levels[k])
		if atRaceLimit(c, k, lvl) {
			continue
		}
		need := expThreshold(k, lvl)
		if (need <= int32(c.Exp) || e.freeTraining) && need > 0 {
			if e.freeTraining && need > int32(c.Exp) {
				c.Exp = uint32(need)
			}
			ready += classBit(k)
			if next := expThreshold(k, lvl+1); next > 0 && int32(c.Exp) >= next && next > cap {
				cap = next - 1
			}
		}
	}
	if !e.silentTraining {
		// The class with the highest threshold at the last level read joins.
		var best int32
		bestClass := 0
		for k := 0; k < classCount; k++ {
			if classBit(k)&ready != 0 && expThreshold(k, lvl) > best {
				best, bestClass = expThreshold(k, lvl), k
			}
		}
		if best > 0 {
			ready |= classBit(bestClass)
			if next := expThreshold(bestClass, lvl+1); next > 0 && int32(c.Exp) >= next && next > cap {
				cap = next - 1
			}
		}
	}
	if cap > 0 && !e.silentTraining {
		c.Exp = uint32(cap)
	}
	if !e.freeTraining {
		if have&trainer == 0 && !e.silentTraining {
			e.statusMessage("We don't train that class here", colourYellow, colourBlack)
			return
		}
		if ready&trainer == 0 {
			e.cannotTrain = true
			if !e.silentTraining {
				e.statusMessage("Not Enough Experience", colourYellow, colourBlack)
			}
			return
		}
	}
	training := ready
	if !e.freeTraining {
		training &= trainer
	}
	if !e.silentTraining {
		e.ClearCells(1, 1, 38, 22)
		row := 4
		e.displayName(c, 4, row, false)
		e.displayStringBg(len(c.Name())+4, row, colourGreen, colourBlack, " will become:")
		for k := 0; k < classCount; k++ {
			if int8(c.Levels[k]) > 0 && classBit(k)&training != 0 {
				row++
				lead := "and a level "
				if row == 5 {
					lead = "    a level "
				}
				e.displayStringBg(6, row, colourGreen, colourBlack, lead+itoa(int(c.Levels[k])+1)+" "+classNames[k])
			}
		}
		if e.yesNoFrom(answerYes, colourLightMagenta, colourGreen, colourWhite, "Do you wish to train? ") != 'Y' {
			return
		}
		e.printMessage("Congratulations...")
	}
	classes := 0
	oldMage := int8(c.Levels[clsMage])
	c.ClassMask = 0
	for k := 0; k < classCount; k++ {
		if int8(c.Levels[k]) <= 0 {
			continue
		}
		classes++
		if classBit(k)&training != 0 {
			c.Levels[k]++
			if c.Drained != 0 {
				c.DrainedHP -= c.DrainedHP / c.Drained
				c.Drained--
			}
		}
	}
	e.recalcClassBonuses(c)
	if !e.silentTraining {
		if int8(c.Levels[clsMage]) > oldMage || int8(c.Levels[clsRanger]) > rangerSpellLevel {
			e.learnNewSpell(c)
		}
		if oldClassUsable(c) {
			if int8(c.OldLevels[clsRanger]) > 0 && e.FindAffect(c, affectRangerVsGiants) == nil {
				e.AddAffect(c, affectRangerVsGiants, 0, 255, 0)
			}
			if int8(c.OldLevels[clsPaladin]) > 0 && e.FindAffect(c, affectProtFromEvil) == nil {
				e.AddAffect(c, affectProtFromEvil, 0, 255, 0)
			}
		}
	} else {
		for _, s := range silentMageSpells[c.Levels[clsMage]] {
			c.SpellBook[s-1] = 1
		}
	}
	if int8(c.MaxLevel) > int8(c.MulticlassLevel) {
		e.levelHitPoints(c, training, classes)
	}
}

// rangerSpellLevel: rangers above this level learn magic-user spells.
const rangerSpellLevel = 8

// atRaceLimit = the race checks in O0EF:4A8E: demi-humans stop at a
// maximum level per class, some only with a high strength or intelligence.
func atRaceLimit(c *Character, class, level int) bool {
	str, intl := c.Stats[statStr].Value, c.Stats[statInt].Value
	// at top always, one lower with the ability 17, two lower under 17
	limit := func(top int, ability byte) bool {
		return level == top || level == top-1 && ability == 17 || level == top-2 && ability < 17
	}
	switch int(c.Race) {
	case raceDwarf:
		return class == clsFighter && limit(9, str) || class == clsCleric && level >= 8
	case raceElf:
		return class == clsFighter && limit(7, str) || class == clsCleric && level >= 7 ||
			class == clsMage && (level == 11 || level == 9 && intl < 17 || level == 10 && intl == 17)
	case raceGnome:
		return class == clsFighter && (level == 6 || level == 5 && str < 18) || class == clsCleric && level >= 7
	case raceHalfElf:
		return class == clsCleric && level == 5 ||
			(class == clsFighter || class == clsRanger) && limit(8, str) || class == clsMage && limit(8, intl)
	case raceHalfling:
		return class == clsFighter && limit(6, str)
	}
	return false
}
