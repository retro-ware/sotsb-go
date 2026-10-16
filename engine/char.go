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

// charRecord is Silver Blades' 439-byte character record (the CHRDATA/SAVGAM
// layout and the MON*CHA blocks), field for field, so it loads with
// encoding/binary. Offsets are decimal. Bytes stay bytes: the rules rely on
// their wrap-around and signed readings. O0F8:13C3 converts Curse of the
// Azure Bonds' 422-byte record into this one.
type charRecord struct {
	PName           [16]byte    // 0 Pascal string, max 15
	Stats           [6]statPair // 16 str int wis dex con cha
	Str00           str00Pair   // 28 exceptional strength
	Memorized       [75]byte    // 30 spell ids, +128 = still to be memorised
	SpellsToLearn   byte        // 105
	THAC0Base       byte        // 106
	Race            byte        // 107 (raceHuman ...)
	Class           byte        // 108
	PaladinCures    byte        // 109 cure disease uses left this week
	Age             uint16      // 110
	MaxHP           byte        // 112
	SpellBook       [117]byte   // 113 [n-1]: spell n known
	FightLevel      byte        // 230 attack level (1 for non-fighters)
	Size            byte        // 231 low bits: icon size; 128 flag
	Saves           [5]byte     // 232 saving throw targets
	BaseMove        byte        // 237
	MaxLevel        byte        // 238 highest class level (hit dice)
	MulticlassLevel byte        // 239
	Drained         byte        // 240 levels lost to energy drain
	DrainedHP       byte        // 241 hit points lost with them
	Undead          byte        // 242 turn undead row
	Thief           [8]byte     // 243 thief skills
	_               [4]byte     // 251 affect list pointer
	Control         byte        // 255 0 party, 179 charmed, above 127 hired NPC
	Share           byte        // 256 hired NPC's treasure shares (& 7)
	OldClass        byte        // 257 the class a dual-classed human left (clsCleric ...)
	Pad258          byte        // 258
	Coins           [7]uint16   // 259 copper .. jewelry: only read when joining the party
	Levels          [7]byte     // 273 cleric druid fighter paladin ranger mage thief
	OldLevels       [7]byte     // 280 a dual-classed human's levels in the class left behind
	Sex             byte        // 287
	Alignment       byte        // 288
	BaseAttack      [2]byte     // 289 attacks per hand
	BaseDice        [2]byte     // 291 damage dice count per hand
	BaseSides       [2]byte     // 293 damage dice sides per hand
	BaseBonus       [2]byte     // 295 damage bonus per hand
	BaseAC          byte        // 297
	StrBonus        byte        // 298 non-zero: strength modifies hit and damage
	RandomID        byte        // 299
	Exp             uint32      // 300
	ClassMask       byte        // 304 items the classes may use
	HPRoll          byte        // 305 rolled hit points
	SpellSlots      [28]byte    // 306 spells per level, seven levels each: cleric, druid, paladin/ranger, magic-user
	ExpValue        uint16      // 334 a monster's worth
	ExpPerHP        byte        // 336
	PortHead        byte        // 337
	PortBody        byte        // 338
	IconHead        byte        // 339
	IconBody        byte        // 340
	Order           byte        // 341
	IconSize        byte        // 342
	IconColour      [6]byte     // 343 icon parts, low nibble and high nibble
	CreatureFlags   byte        // 349 creature kind bits (creatureEnchanted ...)
	MonsterFlags    byte        // 350 more kind bits
	Pad351          byte        // 351
	ItemCount       byte        // 352
	_               [4]byte     // 353 item list pointer
	_               [52]byte    // 357 readied item pointers
	Hands           byte        // 409
	SaveBonus       byte        // 410
	Weight          uint16      // 411
	_               [8]byte     // 413 party link and combat block pointers
	MagicResistance byte        // 421 percent, before the caster level
	Status          byte        // 422 index into the status names
	Okay            byte        // 423 non-zero = active
	Side            byte        // 424 0 party, 1 monsters (1 also draws the name yellow)
	Auto            byte        // 425 the computer fights for it
	THAC0           byte        // 426
	AC              [2]byte     // 427 60 - armour class: front, rear
	Attacks         [2]byte     // 429 attacks per hand this round
	Dice            [2]byte     // 431 damage dice count per hand
	Sides           [2]byte     // 433 damage dice sides per hand
	Bonus           [2]byte     // 435 damage bonus per hand
	HP              byte        // 437
	Move            byte        // 438
}

const charRecSize = 439

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
	classCount // seven classes; Silver Blades has no monks
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
		panic("charRecord is " + strconv.Itoa(n) + " bytes, want 439")
	}
}
