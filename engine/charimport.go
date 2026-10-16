package engine

import (
	"encoding/binary"
	"os"
	"sort"
	"strings"
)

// Characters brought over from Pool of Radiance (.CHA, and the party of a
// saved game, .SAV) and from Hillsfar (.HIL): unit O0A1's roster (0465) and
// import_char01 (22A0) with its nested load_player (14B3), the Hillsfar merge
// (1F35) and the name search (13AE).

// The record sizes of the three games (DS:0511) and where each keeps the
// name (DS:0508) and the hired-NPC byte (DS:050B).
const (
	poolRecSize     = 285
	hillsfarRecSize = 188
	poolNPCByte     = 132
	curseNPCByte    = 247
	hillsfarName    = 4 // a C string
	npcFlag         = 127
)

// Extensions of the other games' files.
const (
	extPool      = ".cha"
	extPoolSave  = ".sav"
	extHillsfar  = ".hil"
	extPoolSpecs = ".spc" // Pool's racial affects
	poolSaveSlot = 7      // the saved game letter in CHRDATx.SAV
)

// The Hillsfar record (188 bytes).
const (
	hilStr         = 20
	hilStr00       = 21
	hilAlignment   = 28
	hilWandCharges = 29
	hilAge         = 30 // word
	hilHP          = 32
	hilMaxHP       = 33
	hilRingCharges = 35
	hilPaladin     = 38 // non-zero: a paladin's cure disease
	hilGold        = 40 // longint
	hilSex         = 44
	hilRace        = 45 // 0-based, 5 = human
	hilExp         = 46 // longint
	hilClass       = 53
	hilScrollsA    = 134
	hilScrollsB    = 135
	hilCleric      = 183 // non-zero: has the class
	hilMage        = 184
	hilFighter     = 185
	hilThief       = 186
)

// fieldCopy is one Move of load_player: n bytes from the Pool record to the
// Curse record.
type fieldCopy struct{ to, from, n int }

// poolCopies = load_player's moves after the name and abilities, in order.
// The 52 bytes of readied item pointers (Pool +204 to +337) are pointers and
// not copied.
var poolCopies = [...]fieldCopy{
	{115, 45, 1},  // THAC0 base
	{117, 47, 1},  // class
	{118, 48, 2},  // age
	{120, 50, 1},  // max HP
	{121, 51, 56}, // spells known 1..56
	{221, 107, 1}, // fight level
	{222, 108, 1}, // size
	{223, 109, 5}, // saving throws
	{228, 114, 1}, // base move
	{229, 115, 1}, // highest level
	{231, 116, 1}, // drained levels
	{232, 117, 1}, // drained HP
	{233, 118, 1}, // turn undead
	{234, 119, 8}, // thief skills
	{246, 131, 1},
	{247, 132, 1}, // control
	{248, 133, 1}, // share
	{249, 134, 1},
	{250, 135, 1},
	{265, 150, 8}, // levels
	{282, 159, 1}, // kind
	{283, 160, 1}, // alignment
	{284, 161, 2}, // attacks
	{286, 163, 2}, // dice
	{288, 165, 2}, // sides
	{290, 167, 2}, // bonus
	{292, 169, 1}, // base AC
	{293, 170, 1}, // strength bonus
	{294, 171, 1}, // random id
	{295, 172, 4}, // experience
	{299, 176, 1}, // class mask
	{300, 177, 1}, // HP roll
	{301, 178, 3}, // cleric slots 1..3
	{311, 181, 3}, // magic-user slots 1..3
	{316, 184, 2}, // exp value
	{318, 186, 1},
	{319, 187, 1}, // portrait
	{320, 188, 1},
	{321, 189, 1}, // icon
	{322, 190, 1},
	{324, 192, 1}, // icon size
	{325, 193, 6}, // icon colours
	{332, 199, 1}, // item count
	{389, 256, 1}, // hands
	{390, 257, 1}, // save bonus
	{391, 258, 2}, // weight
	{405, 268, 1}, // status
	{406, 269, 1}, // okay
	{407, 270, 1}, // side
	{409, 272, 1}, // THAC0
	{410, 273, 2}, // AC
	{412, 275, 2}, // attacks this round
	{414, 277, 2},
	{416, 279, 2},
	{418, 281, 2},
	{420, 283, 1}, // HP
	{421, 284, 1}, // move
}

// Pool record bytes load_player reads on their own.
const (
	poolStats        = 16
	poolStr00        = 22
	poolRace         = 46
	poolSex          = 158
	importPlatinum   = 300 // Coins[platinum] of an imported Pool character
	spellBookUnknown = 35  // SpellBook entry load_player clears (spell 36)
)

// poolKeptAffects: the Pool .SPC affects a character keeps (racial ones).
var poolKeptAffects = map[byte]bool{
	affectGnomeVsManSized: true, affectDwarfVsOrc: true, affectVsGiants: true, affectGnomeExtra: true,
	affectConSavingBonus: true, affectElfResistSleep: true, affectHalfElfResistance: true,
}

// hillsfarClasses = DS:051A: Hillsfar's class (& 15) to Curse's class byte.
var hillsfarClasses = [16]byte{17, 6, 2, 14, 5, 16, 13, 15, 0, 12, 8, 17, 11, 17, 9, 17}

// cString = O0A1:001E: the zero-terminated name at b.
func cString(b []byte) string {
	n := 0
	for n < len(b) && b[n] != 0 {
		n++
	}
	if n > 255 {
		n = 255
	}
	return string(b[:n])
}

// recordName: the name a roster file holds (first 16 bytes: a Pascal string,
// or Hillsfar's C string at +4).
func (e *Engine) recordName(b []byte) string {
	if e.addSource == fromHillsfar {
		return cString(b[hillsfarName : hillsfarName+16])
	}
	n := min(int(b[0]), 15)
	return string(b[1 : 1+n])
}

// rosterFiles = O0A1:0465 (nested 008B): the source's files of the right
// size whose character is not a hired NPC and not already in the party,
// sorted as the directory lists them. Pool's saved games add "from saved
// game X".
func (e *Engine) rosterFiles() []rosterEntry {
	if !e.saveDirReady() {
		return nil
	}
	var out []rosterEntry
	scan := func(ext string, size, npcAt int) {
		ents, _ := os.ReadDir(e.Cfg.SaveDir)
		var names []string
		for _, en := range ents {
			if !en.IsDir() && strings.EqualFold(filepathExt(en.Name()), ext) {
				names = append(names, en.Name())
			}
		}
		sort.Strings(names)
		for _, f := range names {
			b, err := os.ReadFile(e.savePath(f))
			if err != nil || len(b) != size {
				continue
			}
			name := e.recordName(b)
			npc := e.addSource != fromHillsfar && b[npcAt] > npcFlag
			inParty := false
			for _, p := range e.party {
				if p.Name() == cut(name, 15) {
					inParty = true
				}
			}
			if npc || inParty {
				continue
			}
			text := cut(name, 15) + strings.Repeat(" ", max(0, 15-len(name)))
			if len(f) >= 12 && strings.EqualFold(f[8:12], extPoolSave) {
				text += "from saved game " + string(f[poolSaveSlot-1])
			}
			out = append(out, rosterEntry{text: text, file: f})
		}
	}
	switch e.addSource {
	case fromCurse:
		scan(extCharacter, charRecSize, curseNPCByte)
	case fromPool:
		scan(extPool, poolRecSize, poolNPCByte)
		scan(extPoolSave, poolRecSize, poolNPCByte)
	case fromHillsfar:
		scan(extHillsfar, hillsfarRecSize, 0)
	}
	return out
}

// loadPoolRecord = O0A1:14B3 load_player: a Pool of Radiance record into rec
// (a Curse record), the abilities kept within the race's limits.
func loadPoolRecord(rec, pool []byte) {
	race, sex := pool[poolRace], pool[poolSex]
	rec[offRace] = race
	rec[offSex] = sex
	n := min(int(pool[0]), 15)
	rec[0] = byte(n)
	copy(rec[1:16], pool[1:1+n])
	limits := raceStatLimits[race&7]
	for i := 0; i < 6; i++ {
		v := pool[poolStats+i]
		lo, hi := limits[statLimitMin+2*i], limits[statLimitMin+2*i+1]
		if i == statStr {
			lo, hi = limits[sex&1], limits[2+sex&1]
		}
		if v < lo {
			v = lo
		}
		if v > hi {
			v = hi
		}
		rec[offStats+2*i] = v
		rec[offStats+2*i+1] = v
	}
	rec[offStr00] = pool[poolStr00]
	// The original compares the exceptional strength maximum with the
	// strength itself and raises the strength to it; see CHECKLIST.
	if max00 := limits[4+sex&1]; rec[offStr00] > max00 {
		rec[offStr00] = max00
	}
	rec[offStr00+1] = pool[poolStr00]
	for _, f := range poolCopies {
		copy(rec[f.to:f.to+f.n], pool[f.from:f.from+f.n])
	}
	rec[offSpellBook+spellBookUnknown] = 0
	rec[offMulticlassLevel] = rec[offMaxLevel]
	binary.LittleEndian.PutUint16(rec[offCoins+2*coinPlatinum:], importPlatinum)
}

// Curse record offsets the importer writes.
const (
	offStats           = 16
	offStr00           = 28
	offRace            = 116
	offSpellBook       = 121
	offMaxLevel        = 229
	offMulticlassLevel = 230
	offCoins           = 251
	offSex             = 281
)

// hilWord / hilLong read Hillsfar's little-endian numbers.
func hilWord(h []byte, at int) uint16 { return binary.LittleEndian.Uint16(h[at:]) }
func hilLong(h []byte, at int) uint32 { return binary.LittleEndian.Uint32(h[at:]) }

// startLevel is Hillsfar's class byte as a starting level.
func startLevel(b byte) byte {
	if b > 0 {
		return 1
	}
	return 0
}

// hillsfarTrain: game word 680 to every class, then train_player without
// questions until nothing more can be raised (b4FE1, b75A0).
func (e *Engine) hillsfarTrain() {
	putWord(e.game, 2*gameWordTraining, 255)
	e.cannotTrain = false
	e.silentTraining = true
	for !e.cannotTrain {
		e.TrainCharacter()
	}
	e.silentTraining = false
}

// hillsfarFinish: the hit points from Hillsfar.
func (e *Engine) hillsfarFinish(c *Character, h []byte) {
	c.MaxHP = h[hilMaxHP]
	c.HPRoll = c.MaxHP - byte(e.conHitPointBonusOf(c))
	c.HP = h[hilHP]
}

// hillsfarMerge = O0A1:1F35: a Hillsfar hero who is also a Curse or Pool
// character keeps the better abilities, experience, money and age; the
// classes start again at level 1 and are trained up.
func (e *Engine) hillsfarMerge(c *Character, h []byte) {
	if c.Stats[statStr].Base < h[hilStr] {
		c.Stats[statStr] = statPair{h[hilStr], h[hilStr]}
	}
	if c.Str00.Value < h[hilStr00] {
		c.Str00 = str00Pair{h[hilStr00], h[hilStr00]}
	}
	for i := 1; i <= statCha; i++ {
		if v := h[hilStr00+i]; c.Stats[i].Base < v {
			c.Stats[i] = statPair{v, v}
		}
	}
	if int32(c.Exp) < int32(hilLong(h, hilExp)) {
		c.Exp = hilLong(h, hilExp)
	}
	if goldWorth(c) < int32(hilLong(h, hilGold)) {
		for k := 0; k <= coinPlatinum; k++ {
			e.dropCoins(c, c.Coins[k], k)
		}
		e.addPlatinum(uint16(int32(hilLong(h, hilGold)) / platinumGold))
	}
	if c.Age < hilWord(h, hilAge) {
		c.Age = hilWord(h, hilAge)
	}
	c.setHillsfarClasses(h)
	c.MaxLevel = 1
	if h[hilPaladin] != 0 {
		c.Pad402[0] = 1 // +402
	}
	e.hillsfarTrain()
	e.hillsfarFinish(c, h)
}

func (c *Character) setHillsfarClasses(h []byte) {
	c.Levels[clsCleric] = startLevel(h[hilCleric])
	c.Levels[clsMage] = startLevel(h[hilMage])
	c.Levels[clsFighter] = startLevel(h[hilFighter])
	c.Levels[clsThief] = startLevel(h[hilThief])
}

// hillsfarItems: the charged wands and scrolls a merged Hillsfar hero brings
// (O0ED:327F new items, O0ED:1690 added).
func (c *Character) hillsfarItems(h []byte) {
	add := func(typ, w0, w1, w2, plus byte, weight uint16, price uint16, charges, spell byte) {
		it := &Item{Type: typ, Words: [3]byte{w0, w1, w2}, Plus: plus, Weight: weight, Price: price}
		it.Spells = [3]byte{charges, spell, 0}
		c.Items = append(c.Items, it)
	}
	if n := h[hilWandCharges]; n > 0 {
		add(70, 168, 167, 87, 0, 0, uint16(n)*200, n, 31)
	}
	if n := h[hilRingCharges]; n > 0 {
		add(79, 206, 167, 69, 1, 1, uint16(n)*350, n, 65)
	}
	if n := h[hilScrollsA]; n > 0 {
		add(69, 168, 167, 66, 0, 0, uint16(n)*200, n, 31)
	}
	if n := h[hilScrollsB]; n > 0 {
		add(70, 185, 167, 64, 0, uint16(n)*10, uint16(n)*400, n, 62)
	}
}

// nameInFiles = O0A1:13AE (ext, name): a file of the extension holding the
// character.
func (e *Engine) nameInFiles(ext, name string) bool {
	ents, _ := os.ReadDir(e.Cfg.SaveDir)
	for _, en := range ents {
		if en.IsDir() || !strings.EqualFold(filepathExt(en.Name()), ext) {
			continue
		}
		b, err := os.ReadFile(e.savePath(en.Name()))
		if err != nil || len(b) < 16 {
			continue
		}
		if n := min(int(b[0]), 15); string(b[1:1+n]) == name {
			return true
		}
	}
	return false
}

// newHillsfarCharacter: a Hillsfar hero with no earlier self, built from
// the defaults and Hillsfar's record.
func (e *Engine) newHillsfarCharacter(c *Character, h []byte) {
	for i, t := range defaultIconColours {
		c.IconColour[i] = (t+8)<<4 + t
	}
	c.BaseAC = newCharBaseAC
	c.THAC0Base = newCharTHAC0
	c.Status = statusOkay
	c.Okay = 1
	c.PortHead, c.PortBody = 1, 1
	c.Size = 1
	c.RandomID = byte(e.Random(256))
	c.Order = 10
	c.BaseAttack[0], c.BaseDice[0], c.BaseSides[0] = 2, 1, 2
	c.StrBonus = 1
	c.BaseMove = newCharMove
	c.SetName(cString(h[hillsfarName:]))
	c.Stats[statStr] = statPair{h[hilStr], h[hilStr]}
	c.Str00 = str00Pair{h[hilStr00], h[hilStr00]}
	for i := 1; i <= statCha; i++ {
		c.Stats[i] = statPair{h[hilStr00+i], h[hilStr00+i]}
	}
	c.Race = h[hilRace] + 1
	if c.Race == raceHalfOrc {
		c.Race = raceHuman
	}
	small := func(affects ...byte) {
		c.IconSize = 1
		for _, a := range affects {
			e.AddAffect(c, a, 0, 255, 0)
		}
	}
	switch c.Race {
	case raceHalfling:
		small(affectConSavingBonus)
	case raceDwarf:
		small(affectConSavingBonus, affectDwarfVsOrc, affectVsGiants)
	case raceGnome:
		small(affectConSavingBonus, affectGnomeVsManSized, affectVsGiants, affectGnomeExtra)
	case raceElf:
		c.IconSize = 2
		e.AddAffect(c, affectElfResistSleep, 0, 255, 0)
	case raceHalfElf:
		c.IconSize = 2
		e.AddAffect(c, affectHalfElfResistance, 0, 255, 0)
	default:
		c.IconSize = 2
	}
	c.Class = hillsfarClasses[h[hilClass]&15]
	c.Age = hilWord(h, hilAge)
	c.setHillsfarClasses(h)
	c.MaxLevel = 1
	c.Sex = h[hilSex]
	c.Alignment = h[hilAlignment]
	c.Exp = hilLong(h, hilExp)
	if int8(c.Levels[clsMage]) > 0 {
		for _, sp := range hillsfarMageSpells {
			c.SpellBook[sp-1] = 1
		}
	}
	e.hillsfarTrain()
	e.addPlatinum(importPlatinum)
	e.hillsfarFinish(c, h)
}

// hillsfarMageSpells: what a Hillsfar magic-user knows (SpellBook +10, +17,
// +18, +20).
var hillsfarMageSpells = [...]int{11, 18, 19, 21}

// A Hillsfar hero's new record.
const (
	newCharBaseAC = 50
	newCharTHAC0  = 40
	newCharMove   = 12
)

// importCharacter = O0A1:22A0 import_char01 for Pool and Hillsfar.
func (e *Engine) importCharacter(file string, c *Character) bool {
	e.DisplayString(0, statusRow, colourGreen, "Loading...Please Wait")
	b, err := os.ReadFile(e.savePath(file))
	if err != nil {
		e.fileError("read", e.savePath(file))
		return false
	}
	base := strings.TrimSuffix(file, filepathExt(file))
	rec := make([]byte, charRecSize)
	switch e.addSource {
	case fromPool:
		loadPoolRecord(rec, b)
		c.setRecord(rec)
	case fromHillsfar:
		h := b
		name := cut(cString(h[hillsfarName:]), 15)
		switch {
		case e.nameInFiles(extCharacter, name):
			g, _ := os.ReadFile(e.savePath(base + extCharacter))
			c.setRecord(g)
			undo := e.selectTemp(c)
			e.hillsfarMerge(c, h)
			undo()
			c.hillsfarItems(h)
		case e.nameInFiles(extPool, name):
			p, _ := os.ReadFile(e.savePath(base + extPool))
			if len(p) < poolRecSize {
				p = append(p, make([]byte, poolRecSize-len(p))...)
			}
			loadPoolRecord(rec, p)
			c.setRecord(rec)
			undo := e.selectTemp(c)
			e.hillsfarMerge(c, h)
			undo()
		default:
			undo := e.selectTemp(c)
			e.newHillsfarCharacter(c, h)
			undo()
		}
	}
	files := makeFileName(c.Name())
	if b, err := os.ReadFile(e.savePath(files + extItems)); err == nil {
		c.Items = append(c.Items, readRecords[Item](b, itemRecSize)...)
	}
	if b, err := os.ReadFile(e.savePath(files + extAffects)); err == nil {
		c.Affects = append(c.Affects, readAffects(b)...)
	}
	if e.addSource == fromPool {
		if b, err := os.ReadFile(e.savePath(files + extPoolSpecs)); err == nil {
			for _, a := range readAffects(b) {
				if poolKeptAffects[a.Type] {
					c.Affects = append(c.Affects, a)
				}
			}
		}
	}
	e.ClearKeyboard()
	e.ApplyItems(c)
	e.recalcClassBonuses(c)
	return true
}
