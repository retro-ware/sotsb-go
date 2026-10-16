package engine

import (
	"bytes"
	"encoding/binary"
	"strconv"
)

// Character is a party member or a monster: its record (the DOS file
// layout), its item and affect lists, its combat block and readied items.
type Character struct {
	charRecord
	Items   []*Item     // list at +0xC8 (0x3F-byte nodes)
	Affects []*Affect   // list at +0x7F (9-byte nodes)
	Combat  *CombatInfo // block at +0x108 while fighting
	Slots   [13]*Item   // readied items, +0xCC (set by ApplyItems)
}

// statPair is one ability score: the character's own (Base, +0) and the
// value the rules use (+1), which statAffects works out from it, the
// readied items and the affects.
type statPair struct {
	Base, Value byte
}

// str00Pair is exceptional strength: the value in use (+0) and the
// character's own (+1), the reverse of statPair.
type str00Pair struct {
	Value, Base byte
}

// charRecord is Curse of the Azure Bonds' 422-byte character record (the
// CHRDAT/SAVGAM layout and the MON*CHA blocks), field for field, so it loads
// with encoding/binary. Offsets are decimal. Bytes stay bytes: the rules rely
// on their wrap-around and signed readings.
type charRecord struct {
	PName           [16]byte    // 0 Pascal string, max 15
	Stats           [6]statPair // 16 str int wis dex con cha
	Str00           str00Pair   // 28 exceptional strength
	Memorized       [84]byte    // 30 spell ids, +128 = still to be memorised
	SpellsToLearn   byte        // 114
	THAC0Base       byte        // 115
	Race            byte        // 116 (7 = human)
	Class           byte        // 117
	Age             uint16      // 118
	MaxHP           byte        // 120
	SpellBook       [100]byte   // 121 [n-1]: spell n known
	FightLevel      byte        // 221 attack level (1 for non-fighters)
	Size            byte        // 222 low bits: icon size; 128 flag
	Saves           [5]byte     // 223 saving throw targets
	BaseMove        byte        // 228
	MaxLevel        byte        // 229 highest class level (hit dice)
	MulticlassLevel byte        // 230
	Drained         byte        // 231 levels lost to energy drain
	DrainedHP       byte        // 232 hit points lost with them
	Undead          byte        // 233 turn undead row
	Thief           [8]byte     // 234 thief skills
	_               [4]byte     // 242 affect list pointer
	Pad83           byte        // 246
	Control         byte        // 247 0 party, 179 charmed, above 127 hired NPC
	Share           byte        // 248 hired NPC's treasure shares (& 7)
	Pad86           [2]byte     // 249
	Coins           [7]uint16   // 251 copper .. jewelry: only read when joining the party
	Levels          [8]byte     // 265 cleric druid fighter paladin ranger mage thief monk
	OldLevels       [8]byte     // 273 a dual-classed human's levels in the class left behind
	Sex             byte        // 281
	Kind            byte        // 282 monster kind (bonuses against it)
	Alignment       byte        // 283
	BaseAttack      [2]byte     // 284 attacks per hand
	BaseDice        [2]byte     // 286 damage dice count per hand
	BaseSides       [2]byte     // 288 damage dice sides per hand
	BaseBonus       [2]byte     // 290 damage bonus per hand
	BaseAC          byte        // 292
	StrBonus        byte        // 293 non-zero: strength modifies hit and damage
	RandomID        byte        // 294
	Exp             uint32      // 295
	ClassMask       byte        // 299 items the classes may use
	HPRoll          byte        // 300 rolled hit points
	SpellSlots      [15]byte    // 301 spells per level: cleric 1..5, druid 1..5, magic-user 1..5
	ExpValue        uint16      // 316 a monster's worth
	ExpPerHP        byte        // 318
	PortHead        byte        // 319
	PortBody        byte        // 320
	IconHead        byte        // 321
	IconBody        byte        // 322
	Order           byte        // 323
	IconSize        byte        // 324
	IconColour      [6]byte     // 325 icon parts, low nibble and high nibble
	Pad331          byte        // 331
	ItemCount       byte        // 332
	_               [4]byte     // 333 item list pointer
	_               [52]byte    // 337 readied item pointers
	Hands           byte        // 389
	SaveBonus       byte        // 390
	Weight          uint16      // 391
	_               [8]byte     // 393 party link and combat block pointers
	PaladinCures    byte        // 401 cure disease uses left this week
	Pad402          [3]byte     // 402
	Status          byte        // 405 index into the status names
	Okay            byte        // 406 non-zero = active
	Side            byte        // 407 0 party, 1 monsters (1 also draws the name yellow)
	Auto            byte        // 408 the computer fights for it
	THAC0           byte        // 409
	AC              [2]byte     // 410 60 - armour class: front, rear
	Attacks         [2]byte     // 412 attacks per hand this round
	Dice            [2]byte     // 414 damage dice count per hand
	Sides           [2]byte     // 416 damage dice sides per hand
	Bonus           [2]byte     // 418 damage bonus per hand
	HP              byte        // 420
	Move            byte        // 421
}

const charRecSize = 422

// Stats indexes.
const (
	statStr = iota
	statInt
	statWis
	statDex
	statCon
	statCha
)

// Levels indexes (the class order of the DS:0D38 names).
const (
	clsCleric = iota
	clsDruid
	clsFighter
	clsPaladin
	clsRanger
	clsMage
	clsThief
	clsMonk
)

// setRecord reads a record in the DOS layout (short data leaves the rest zero).
func (c *Character) setRecord(b []byte) {
	buf := make([]byte, charRecSize)
	copy(buf, b)
	binary.Read(bytes.NewReader(buf), binary.LittleEndian, &c.charRecord)
}

// record is the record in the DOS layout (pointers zero).
func (c *Character) record() []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, &c.charRecord)
	return buf.Bytes()
}

func (c *Character) Name() string {
	n := min(int(c.PName[0]), 15)
	return string(c.PName[1 : 1+n])
}

// SetName stores a pascal string (max 15).
func (c *Character) SetName(s string) {
	if len(s) > 15 {
		s = s[:15]
	}
	c.PName = [16]byte{byte(len(s))}
	copy(c.PName[1:], s)
}

// IntToStr = O0ED:10CC.
func itoa(n int) string { return strconv.Itoa(n) }

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func init() {
	if n := binary.Size(charRecord{}); n != charRecSize {
		panic("charRecord is " + strconv.Itoa(n) + " bytes, want 422")
	}
}
