// Package tablespec lists the data tables cmd/tablegen copies out of
// START.EXE: name, place in the data segment (decimal offsets, as in the
// reconstruction's GAMEDATA unit), length and kind. Only the generator uses it.
package tablespec

// Kind is the element type of a table.
type Kind int

const (
	Bytes Kind = iota
	Signed
	Words
	SignedWords
	Longs // signed 32-bit
	// Strings of a fixed slot size: String(n) holds a length byte and n-1 characters.
	stringBase
)

// String is a table of Pascal strings in slots of size bytes.
func String(size int) Kind { return stringBase + Kind(size) }

// Size is the bytes per element.
func (k Kind) Size() int {
	switch k {
	case Bytes, Signed:
		return 1
	case Words, SignedWords:
		return 2
	case Longs:
		return 4
	}
	return int(k - stringBase)
}

// Table is one generated Go value.
type Table struct {
	Name string
	At   int // offset in the data segment
	Len  int // elements (rows when Cols is set)
	Kind Kind
	Doc  string
	Cols int // set for a two-dimensional table: Len rows of Cols elements
	// Curse marks a table still read from Curse of the Azure Bonds' START.EXE:
	// its users are not yet re-ported to Silver Blades.
	Curse bool
	// Stride is the bytes from one row to the next when the rows carry
	// padding (0: Cols elements exactly).
	Stride int
}

// Tables are generated in this order.
var Tables = []Table{
	// Screen frames drawn from 8x8 symbols (R133 draw8x8_01..): symbol = entry + 286.

	// Icon recolouring on EGA and Tandy (R2BC:1418's DS:19A4 from / DS:19B4 to tables).
	{"recolourFrom", 6564, 16, Bytes, "The colours recolouring maps from (identity).", 0, false, 0},
	{"recolourIcons", 6580, 16, Bytes, "Combat icons recoloured to (colour 8 becomes 0).", 0, false, 0},

	// The wilderness map (O109): where each place's cursor goes, by area word 417.
	{"wildMapCursorX", 2634, 32, Bytes, "Wilderness map cursor columns (8-pixel cells), by place.", 0, true, 0},
	{"wildMapCursorY", 2666, 32, Bytes, "Wilderness map cursor rows (8-pixel cells), by place.", 0, true, 0},

	// Names.
	{"raceNames", 3441, 8, String(9), "Race names (raceTribble, raceElf ...).", 0, false, 0},
	{"genderNames", 3666, 2, String(7), "Gender names.", 0, false, 0},
	{"classNames", 2955, 17, String(27), "Class names; 8.. are the multi-class combinations.", 0, false, 0},
	{"alignmentNames", 3513, 9, String(17), "Alignment names.", 0, false, 0},
	{"coinNames", 3680, 7, String(11), "Coin names: copper .. jewelry.", 0, false, 0},
	{"statusNames", 3757, 9, String(13), "Character status names (Status).", 0, false, 0},
	{"itemWords", 3853, 126, String(20), "The words item names are made of (entry 0 unused); an item type is also the number of its word.", 0, false, 0},

	// Character creation (O0A7 createPlayer).
	{"defaultIconColours", 19631, 7, Bytes, "Entries 1..6: combat icon colours of a new character, one per icon part.", 0, false, 0},
	{"raceClasses", 19967, 8, Bytes, "Per race: the number of classes it may take, then those classes.", 14, false, 0},
	{"classAlignments", 20395, 17, Bytes, "Per class: the number of alignments it allows, then those alignments.", 10, false, 0},
	{"thac0Table", 19468, 7, Bytes, "Per class and level (0..18): the THAC0 base value.", 19, false, 0},
	{"classItemFlags", 19602, 7, Bytes, "Per class: the item-use bits it adds to the class mask.", 0, false, 0},
	{"raceAges", 20037, 8, Bytes, "Per race, for the classes 0..6: starting age base (word), dice count, dice sides.", 28, false, 0},
	{"ageSteps", 20223, 8, Words, "Per race: the ages at which abilities change (four steps, one spare word).", 5, false, 0},
	{"raceAbilityLimits", 19869, 8, Bytes, "Per race: strength minimum and maximum by gender, exceptional strength maximum by gender, then minimum and maximum of int wis dex con cha.", 16, false, 0},
	{"clericStartSpells", 22540, 11, Bytes, "Entries 1..10: the spells a new cleric knows (0 ends the list).", 0, false, 0},
	{"raceStatLimits", 16264, 8, Bytes, "Per race: strength min/max by gender, exceptional strength max by gender, then min/max of int wis dex con cha.", 16, true, 0},
	{"classStatMinimum", 20293, 17, Bytes, "Per class: the minimum of each ability (str int wis dex con cha).", 6, false, 0},

	// Training (O0A7 train_player and its hit point rolls).
	{"coinValues", 2880, 5, Longs, "Worth of copper, silver, electrum, gold and platinum, in copper.", 0, false, 0},
	{"classTrainingBits", 19609, 7, Bytes, "Per class: the training bit of the class (trainers list the bits they train; clerics and druids share one).", 0, false, 0},
	{"classMaxHitDice", 19625, 7, Bytes, "Per class: the level up to which hit points are rolled.", 0, false, 0},
	{"classFirstHitDice", 2480, 7, Bytes, "Per class: hit dice rolled at level 1.", 0, false, 0},
	{"classHitDiceSides", 2487, 7, Bytes, "Per class: sides of the hit dice.", 0, false, 0},
	{"conHitPointBonus", 20562, 26, Signed, "Per constitution score: the hit point bonus per die.", 0, false, 0},
	{"expTable", 20580, 7, Longs, "Per class: the experience each level needs (level n: entry n).", 46, false, 187},
	{"modifyKeypad", 13342, 9, Bytes, "The keys 1..9 of the number pad as the scan codes Modify works with.", 0, false, 0},

	// Class calculations (O164).
	{"clericSlotData", 20641, 133, Bytes, "Cleric spells gained per level: entry level*7 + spell level (1..7).", 0, false, 0},
	{"paladinSlotData", 21202, 133, Bytes, "Paladin cleric spells gained per level: entry level*7 + spell level.", 0, false, 0},
	{"rangerSlotData", 21389, 133, Bytes, "Ranger spells gained per level: entry level*7 + k (k 1..3 druid, 4..5 magic-user).", 0, false, 0},
	{"mageSlotData", 21576, 133, Bytes, "Magic-user spells gained per level: entry level*7 + spell level.", 0, false, 0},
	{"saveTableData", 21906, 635, Bytes, "Saving throw targets: entry class*90 + level*5 + save (levels 1..18).", 0, false, 0},

	{"thiefBaseData", 19629, 145, Bytes, "Thief skills by level: entry level*8 + skill (1..8).", 0, false, 0},
	{"thiefRaceData", 19773, 65, Signed, "Thief skill adjustment by race: entry race*8 + skill.", 0, false, 0},
	{"thiefDexData", 19784, 131, Signed, "Thief skill adjustment by dexterity: entry dex*5 + skill (1..5).", 0, false, 0},

	// The adventuring screen.
	{"skyColours", 2728, 16, Bytes, "The sky colour by the area's sky word.", 0, false, 0},
	{"directionNames", 6458, 8, String(3), "Compass point names by facing 0..7.", 0, false, 0},
	{"stepX", 6800, 9, Signed, "Column step by facing 0..7 (8: none).", 0, false, 0},
	{"stepY", 6809, 9, Signed, "Row step by facing 0..7 (8: none).", 0, false, 0},

	// Combat.
	{"groundInfo", 6814, 29, Bytes, "Combat ground tiles: move cost (255 blocks), two flags and the tile's kind.", 4, false, 0},
	{"lineStepDirs", 6482, 3, Bytes, "Facing of a line step by row step + 1 and column step + 1 (8: no step).", 3, false, 0},
	{"footprints", 6930, 5, Signed, "Squares a combatant of size 0..4 covers: column and row offsets of its four parts (-1 none).", 8, false, 0},
	{"teamSides", 848, 4, Bytes, "Formation: the side each of the four teams of a facing looks to.", 4, false, 0},
	{"teamDirs", 864, 4, Bytes, "Formation: each team's direction (times 2) for the facing.", 4, false, 0},
	{"combatFacings", 880, 4, Bytes, "Combat facing for the party's map facing / 2.", 0, false, 0},
	{"formationStartX", 884, 2, Signed, "Formation start column: first team / other teams, by direction.", 4, false, 0},
	{"formationStartY", 892, 2, Signed, "Formation start row: first team / other teams, by direction.", 4, false, 0},
	{"formationRows", 900, 5, Signed, "Formation slots: per direction (4: the second team), rows 0..5 of first and last column.", 12, false, 0},
	{"itemTemplates", 2600, 57, Words, "Magic item templates read by the treasure generator: from entry n, name words n..n+2, weight n+3, price n+4, spells n+5..n+7 (low bytes).", 0, false, 0},
	{"diseaseAffects", 6973, 5, Bytes, "Entries 1..4: the affects a paladin's Cure Disease removes (diseases).", 0, false, 0},
	{"clockLimits", 15916, 7, Words, "Clock carries: minutes per ten, tens per hour, hours per day ... (area words 198..204).", 0, true, 0},

	// Affects.
	{"helplessAffectData", 6969, 5, Bytes, "Entries 1..4: the affects that leave a character helpless.", 0, false, 0},
	{"downAffectData", 2713, 15, Bytes, "Entries 1..14: the affects a character loses when it goes out of action (O145:13E9).", 0, false, 0},
	{"cloudSquares4", 13472, 4, Bytes, "The facings from a small cloud's centre to its 4 squares (stinking cloud).", 0, false, 0},
	{"dispelCloudSquares", 13472, 9, Bytes, "The squares Dispel Magic tries for a cloud: DS:27D9 read from entry 1,\nthe small cloud's table running on into the next, also for the large\ncloud (O0C2:31E4).", 0, false, 0},
	{"cloudSquares9", 13476, 9, Bytes, "The facings from a large cloud's centre to its 9 squares (cloudkill); 8 is the centre.", 0, false, 0},

	// Spells.
	{"spellNames", 13451, 118, String(35), "Spell names by spell number.", 0, false, 0},
	{"affectDisplayNames", 8743, 114, String(38), "Affect names the game shows, by affect number (empty: none).", 0, false, 0},
	{"spellLevelHeadings", 2167, 10, String(41), "Spell menu headings by spell level.", 0, true, 0},
	{"spellTableData", 17565, 118, Bytes, "The spell table: one 16-byte record per spell number (spellEntry).", 16, false, 0},
	{"turnUndeadRolls", 884, 10, Signed, "turnUndeadRolls = DS:0369 + 10*undead + column: the d20 a cleric needs to\nturn undead of each row (1..10), by cleric column (levels 1..8, 9..13,\n14 and up). 0: turned automatically, negative: destroyed (needing its\nabsolute value), 99: impossible.", 10, true, 0},
	{"wildTerrainFlags", 852, 132, Bytes, "wildTerrainFlags = DS:0354: the wilderness terrain flags (terrainSparse\n...) by area byte 0x342.", 0, true, 0},
}
