package engine

import (
	"bytes"
	"encoding/binary"
	"os"
	"sort"
	"strings"
)

// Characters brought over from Curse of the Azure Bonds (unit O0F8): its
// .guy records are read as they are and turned into Silver Blades ones.

// curseRecord is Curse of the Azure Bonds' 422-byte character record, in
// its file layout (encoding/binary).
type curseRecord struct {
	PName           [16]byte
	Stats           [6]statPair
	Str00           str00Pair
	Memorized       [84]byte
	SpellsToLearn   byte
	THAC0Base       byte
	Race            byte // curseDwarf ...
	Class           byte
	Age             uint16
	MaxHP           byte
	SpellBook       [100]byte
	FightLevel      byte
	Size            byte
	Saves           [5]byte
	BaseMove        byte
	MaxLevel        byte
	MulticlassLevel byte
	Drained         byte
	DrainedHP       byte
	Undead          byte
	Thief           [8]byte
	_               [4]byte
	_               byte
	Control         byte
	Share           byte
	OldClass        byte
	Pad250          byte
	Coins           [7]uint16
	Levels          [8]byte // Silver Blades has no monks: the last is dropped
	OldLevels       [8]byte
	Sex             byte
	Kind            byte
	Alignment       byte
	BaseAttack      [2]byte
	BaseDice        [2]byte
	BaseSides       [2]byte
	BaseBonus       [2]byte
	BaseAC          byte
	StrBonus        byte
	RandomID        byte
	Exp             uint32
	ClassMask       byte
	HPRoll          byte
	SpellSlots      [3][5]byte // cleric, druid, magic-user levels 1..5
	ExpValue        uint16
	ExpPerHP        byte
	PortHead        byte
	PortBody        byte
	IconHead        byte
	IconBody        byte
	Order           byte
	IconSize        byte
	IconColour      [6]byte
	_               byte
	ItemCount       byte
	_               [4]byte
	_               [52]byte
	Hands           byte
	SaveBonus       byte
	Weight          uint16
	_               [8]byte
	PaladinCures    byte
	_               [3]byte
	Status          byte
	Okay            byte
	Side            byte
	Auto            byte
	THAC0           byte
	AC              [2]byte
	Attacks         [2]byte
	Dice            [2]byte
	Sides           [2]byte
	Bonus           [2]byte
	HP              byte
	Move            byte
}

const curseRecSize = 422

func readCurseRecord(b []byte) *curseRecord {
	buf := make([]byte, curseRecSize)
	copy(buf, b)
	r := &curseRecord{}
	binary.Read(bytes.NewReader(buf), binary.LittleEndian, r)
	return r
}

// Spells of Curse of the Azure Bonds a character does not keep.
var curseSpellsDropped = map[int]bool{36: true, 56: true, 63: true, 90: true, 96: true, 98: true, 100: true}

// The spell slot groups of a Silver Blades record (seven levels each).
const (
	slotGroupCleric    = 0
	slotGroupDruid     = 1
	slotGroupMagicUser = 3
	slotLevels         = 7
)

// fromCurse = O0F8:13C3: the Curse record r as this character: the name
// without surrounding blanks, the race in Silver Blades' numbering, the
// known spells Silver Blades also has, and the strength within the race's
// limit. What is not copied (memorised spells, coins, the turn undead row,
// the order and the item count) starts at zero.
func (c *Character) fromCurse(r *curseRecord) {
	c.charRecord = charRecord{}
	n := min(int(r.PName[0]), 15)
	c.SetName(strings.Trim(string(r.PName[1:1+n]), " "))
	c.Stats = r.Stats
	c.Str00 = r.Str00
	c.THAC0Base = r.THAC0Base
	c.Race = raceFromCurse(r.Race)
	c.Class = r.Class
	c.Age = r.Age
	c.MaxHP = r.MaxHP
	for spell := 1; spell <= len(r.SpellBook); spell++ {
		if !curseSpellsDropped[spell] {
			c.SpellBook[spell-1] = r.SpellBook[spell-1]
		}
	}
	c.FightLevel = r.FightLevel
	c.Size = r.Size
	c.Saves = r.Saves
	c.BaseMove = r.BaseMove
	c.MaxLevel = r.MaxLevel
	c.MulticlassLevel = r.MulticlassLevel
	c.Drained = r.Drained
	c.DrainedHP = r.DrainedHP
	c.Thief = r.Thief
	c.Control = r.Control
	c.Share = r.Share
	c.OldClass = r.OldClass
	c.Pad258 = r.Pad250
	copy(c.Levels[:], r.Levels[:classCount])
	copy(c.OldLevels[:], r.OldLevels[:classCount])
	c.Sex = r.Sex
	c.Alignment = r.Alignment
	c.BaseAttack = r.BaseAttack
	c.BaseDice = r.BaseDice
	c.BaseSides = r.BaseSides
	c.BaseBonus = r.BaseBonus
	c.BaseAC = r.BaseAC
	c.StrBonus = r.StrBonus
	c.RandomID = r.RandomID
	c.Exp = r.Exp
	c.ClassMask = r.ClassMask
	c.HPRoll = r.HPRoll
	for level := 0; level < len(r.SpellSlots[0]); level++ {
		c.SpellSlots[slotGroupCleric*slotLevels+level] = r.SpellSlots[0][level]
		c.SpellSlots[slotGroupDruid*slotLevels+level] = r.SpellSlots[1][level]
		c.SpellSlots[slotGroupMagicUser*slotLevels+level] = r.SpellSlots[2][level]
	}
	c.ExpValue = r.ExpValue
	c.ExpPerHP = r.ExpPerHP
	c.PortHead = r.PortHead
	c.PortBody = r.PortBody
	c.IconHead = r.IconHead
	c.IconBody = r.IconBody
	c.IconSize = r.IconSize
	c.IconColour = r.IconColour
	c.Hands = r.Hands
	c.SaveBonus = r.SaveBonus
	c.Status = r.Status
	c.Okay = r.Okay
	c.Side = r.Side
	c.THAC0 = r.THAC0
	c.AC = r.AC
	c.Attacks = r.Attacks
	c.Dice = r.Dice
	c.Sides = r.Sides
	c.Bonus = r.Bonus
	c.HP = r.HP
	if limit := raceAbilityLimits[c.Race][strMaximumBySex+c.Sex]; c.Stats[statStr].Base > limit {
		c.Stats[statStr].Base = limit
	}
	c.Move = r.Move
}

// Columns of raceAbilityLimits: by sex, then the minimum and maximum of
// intelligence, wisdom, dexterity, constitution and charisma.
const (
	strMinimumBySex   = 0
	strMaximumBySex   = 2
	str00MaximumBySex = 4
	abilityLimitsRest = 6
)

// Curse of the Azure Bonds' race numbers.
const (
	curseMonster = iota
	curseDwarf
	curseElf
	curseGnome
	curseHalfElf
	curseHalfling
	curseHalfOrc
	curseHuman
)

// raceFromCurse: a Curse of the Azure Bonds race as a Silver Blades one
// (half-orcs become monsters).
func raceFromCurse(race byte) byte {
	switch race {
	case curseDwarf:
		return raceDwarf
	case curseElf:
		return raceElf
	case curseGnome:
		return raceGnome
	case curseHalfElf:
		return raceHalfElf
	case curseHalfling:
		return raceHalfling
	case curseHuman:
		return raceHuman
	}
	return raceMonster
}

// The two kinds of roster file (DS:09CE, 09D0, 09D4): where the hired-NPC
// byte is and how long the record is.
type rosterKind struct {
	pattern string
	npcByte int
	size    int
}

var rosterKinds = [2]rosterKind{
	fromCurse:        {".guy", 132, curseRecSize},
	fromSilverBlades: {".who", 255, charRecSize},
}

// A control byte above this marks a hired NPC.
const npcFlag = 127

// Where "from saved game" files keep their game letter.
const savedGameLetter = 7

// rosterFiles = O0F8:0397 (nested 0016): the source's files of the right
// size whose character is not a hired NPC and not already in the party,
// sorted as the directory lists them.
func (e *Engine) rosterFiles() []rosterEntry {
	if !e.saveDirReady() {
		return nil
	}
	kind := rosterKinds[e.addSource]
	ents, _ := os.ReadDir(e.Cfg.SaveDir)
	var names []string
	for _, en := range ents {
		if !en.IsDir() && strings.EqualFold(filepathExt(en.Name()), kind.pattern) {
			names = append(names, en.Name())
		}
	}
	sort.Strings(names)
	var out []rosterEntry
	for _, f := range names {
		b, err := os.ReadFile(e.savePath(f))
		if err != nil || len(b) != kind.size {
			continue
		}
		n := min(int(b[0]), 15)
		name := string(b[1 : 1+n])
		text := name + strings.Repeat(" ", 15-len(name))
		if len(f) >= 12 && strings.EqualFold(f[8:12], extGameCharacter) {
			text += "from saved game " + string(f[savedGameLetter-1])
		}
		inParty := false
		for _, p := range e.party {
			if p.Name()+strings.Repeat(" ", 15-len(p.Name())) == text[:15] {
				inParty = true
			}
		}
		if b[kind.npcByte] > npcFlag || inParty {
			continue
		}
		out = append(out, rosterEntry{text: text, file: f})
	}
	return out
}
