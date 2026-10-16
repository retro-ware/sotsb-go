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
}

// Tables are generated in this order.
var Tables = []Table{
	// Screen frames drawn from 8x8 symbols (R133 draw8x8_01..): symbol = entry + 286.
	{"frameTop", 2896, 40, Bytes, "Frame symbols along the top row (y 0), one per column.", 0},
	{"frameMiddleRow", 2936, 40, Bytes, "Frame symbols of the separator rows of the credits and the 3D view frame.", 0},
	{"frameBottom", 2976, 40, Bytes, "Frame symbols along row 23.", 0},
	{"frameLeft", 3042, 24, Bytes, "Frame symbols down the left column (x 0).", 0},
	{"frameRight", 3083, 24, Bytes, "Frame symbols down the right column (x 39).", 0},

	// Picture recolouring (R226:15F3 from/to tables).
	{"recolourFrom", 2848, 16, Bytes, "The colours recolouring maps from (identity).", 0},
	{"recolourCombat", 2864, 16, Bytes, "Combat pictures (COMSPR and the area's own icons) recoloured to.", 0},
	{"recolourIcon", 2880, 16, Bytes, "The ICON pictures recoloured to (colour 13 becomes 0).", 0},

	// The wilderness map (O109): where each place's cursor goes, by area word 417.
	{"wildMapCursorX", 2634, 32, Bytes, "Wilderness map cursor columns (8-pixel cells), by place.", 0},
	{"wildMapCursorY", 2666, 32, Bytes, "Wilderness map cursor rows (8-pixel cells), by place.", 0},

	// Names.
	{"raceNames", 3742, 8, String(10), "Race names: monster, dwarf, elf, gnome, half-elf, halfling, half-orc, human.", 0},
	{"genderNames", 3975, 2, String(7), "Gender names.", 0},
	{"classNames", 3256, 17, String(27), "Class names; 8.. are the multi-class combinations.", 0},
	{"alignmentNames", 3822, 9, String(17), "Alignment names.", 0},
	{"coinNames", 3989, 7, String(11), "Coin names: copper .. jewelry.", 0},
	{"statusNames", 4066, 9, String(13), "Character status names (Status).", 0},
	{"itemWords", 4162, 252, String(21), "The words item names are made of (entry 0 unused).", 0},

	// Character creation (O0A7 createPlayer).
	{"defaultIconColours", 16067, 6, Bytes, "Combat icon colours of a new character, one per icon part.", 0},
	{"raceClasses", 16378, 8, Bytes, "Per race: the number of classes it may take, then those classes.", 14},
	{"classAlignments", 16858, 17, Bytes, "Per class: the number of alignments it allows, then those alignments.", 10},
	{"thac0Table", 15930, 8, Bytes, "Per class and level (0..12): the THAC0 base value.", 13},
	{"classItemFlags", 16034, 8, Bytes, "Per class: the item-use bits it adds to the class mask.", 0},
	{"raceAges", 16462, 8, Bytes, "Per race, for the classes 0..6: starting age base (word), dice count, dice sides.", 28},
	{"ageSteps", 16676, 8, Words, "Per race: the ages at which abilities change (four steps, one spare word).", 5},
	{"raceStatLimits", 16264, 8, Bytes, "Per race: strength min/max by gender, exceptional strength max by gender, then min/max of int wis dex con cha.", 16},
	{"classStatMinimum", 16756, 17, Bytes, "Per class: the minimum of each ability (str int wis dex con cha).", 6},
	{"classHitDice", 17843, 8, Bytes, "Per class: dice count and sides of the starting hit points.", 2},

	// Training (O0A7 train_player and its hit point rolls).
	{"coinValues", 3236, 5, Longs, "Worth of copper, silver, electrum, gold and platinum, in copper.", 0},
	{"classExpMasks", 16042, 17, Bytes, "Per class: the training bit of the class (trainers list the bits they train).", 0},
	{"classMaxHitDice", 16059, 8, Bytes, "Per class: the level up to which hit points are rolled.", 0},
	{"classHitDiceFirst", 2074, 8, Bytes, "Per class: hit dice rolled at level 1.", 0},
	{"classHitDiceSides", 2082, 8, Bytes, "Per class: sides of the hit dice.", 0},
	{"conHitPointBonus", 17025, 26, Signed, "Per constitution score: the hit point bonus per die.", 0},
	{"expTableData", 17043, 8, Bytes, "Per class (99-byte rows): longint experience thresholds; level n needs the one at index n+1.", 99},

	// Class calculations (O0FE).
	{"clericSlotData", 17084, 297, Bytes, "Cleric spells gained per level: entry level*5 + spell level (1..5).", 0},
	{"paladinSlotData", 17381, 99, Bytes, "Paladin cleric spells gained per level: entry level*5 + spell level.", 0},
	{"rangerSlotData", 17480, 99, Bytes, "Ranger spells gained per level: entry level*5 + k (k 1..3 druid, 4..5 magic-user).", 0},
	{"mageSlotData", 17579, 264, Bytes, "Magic-user spells gained per level: entry level*5 + spell level.", 0},
	{"saveTableData", 17854, 480, Bytes, "Saving throw targets: entry class*60 + level*5 + save.", 0},
	{"oldClassSaveData", 18274, 130, Bytes, "Saving throws by the old-class level column: entry level*5 + save.", 0},
	{"thiefBaseData", 16064, 96, Bytes, "Thief skills by level: entry level*8 + skill (1..8).", 0},
	{"thiefRaceData", 16160, 64, Signed, "Thief skill adjustment by race: entry race*8 + skill.", 0},
	{"thiefDexData", 16179, 130, Signed, "Thief skill adjustment by dexterity: entry dex*5 + skill (1..5).", 0},

	// The adventuring screen.
	{"skyColours", 2698, 16, Bytes, "The sky colour by the area's sky word.", 0},
	{"viewFrameTop", 3014, 15, Bytes, "Symbols of the 3D view frame's top row, columns 2..14.", 0},
	{"viewFrameBottom", 3027, 15, Bytes, "Symbols of the 3D view frame's bottom row, columns 2..14.", 0},
	{"viewFrameLeft", 3105, 15, Bytes, "Symbols of the 3D view frame's left column, rows 2..14.", 0},
	{"viewFrameRight", 3118, 15, Bytes, "Symbols of the 3D view frame's right column, rows 2..14.", 0},
	{"viewDivider", 3066, 17, Bytes, "Symbols of the column-16 rule beside the view, rows 0..16.", 0},
	{"directionNames", 9538, 8, String(3), "Compass point names by facing 0..7.", 0},
	{"stepX", 9878, 9, Signed, "Column step by facing 0..7 (8: none).", 0},
	{"stepY", 9887, 9, Signed, "Row step by facing 0..7 (8: none).", 0},

	// Combat.
	{"combatFrameLeft", 3133, 23, Bytes, "Combat screen frame symbols down column 0 (rows 0..22).", 0},
	{"combatFrameMiddle", 3156, 23, Bytes, "Combat screen frame symbols down column 22, between the map and the panel.", 0},
	{"combatFrameRight", 3179, 23, Bytes, "Combat screen frame symbols down column 39.", 0},
	{"groundInfo", 9892, 64, Bytes, "Combat ground tiles: move cost (255 blocks), two flags and the tile's kind.", 4},
	{"lineStepDirs", 9562, 3, Bytes, "Facing of a line step by row step + 1 and column step + 1 (8: no step).", 3},
	{"footprints", 10148, 5, Signed, "Squares a combatant of size 0..4 covers: column and row offsets of its four parts (-1 none).", 8},
	{"teamSides", 732, 4, Bytes, "Formation: the side each of the four teams of a facing looks to.", 4},
	{"teamDirs", 748, 4, Bytes, "Formation: each team's direction (times 2) for the facing.", 4},
	{"combatFacings", 764, 4, Bytes, "Combat facing for the party's map facing / 2.", 0},
	{"formationStartX", 768, 2, Signed, "Formation start column: first team / other teams, by direction.", 4},
	{"formationStartY", 776, 2, Signed, "Formation start row: first team / other teams, by direction.", 4},
	{"formationRows", 784, 5, Signed, "Formation slots: per direction (4: the second team), rows 0..5 of first and last column.", 12},
	{"itemTemplates", 2094, 57, Words, "Magic item templates read by the treasure generator: from entry n, name words n..n+2, weight n+3, price n+4, spells n+5..n+7 (low bytes).", 0},
	{"diseaseAffects", 2090, 6, Bytes, "The affects a paladin's Cure removes (diseases).", 0},
	{"clockLimits", 15916, 7, Words, "Clock carries: minutes per ten, tens per hour, hours per day ... (area words 198..204).", 0},

	// Affects.
	{"helplessAffectData", 10187, 5, Bytes, "Entries 1..4: the affects that leave a character helpless.", 0},
	{"downAffectData", 2609, 20, Bytes, "Entries 1..19: the affects a character loses when it goes out of action (O0E3:15AB).", 0},
	{"cloudSquares4", 10202, 4, Bytes, "The facings from a small cloud's centre to its 4 squares (stinking cloud).", 0},
	{"dispelCloudSquares", 10202, 9, Bytes, "The squares Dispel Magic tries for a cloud: DS:27D9 read from entry 1,\nthe small cloud's table running on into the next, also for the large\ncloud (O0C2:31E4).", 0},
	{"cloudSquares9", 10206, 9, Bytes, "The facings from a large cloud's centre to its 9 squares (cloudkill); 8 is the centre.", 0},
	{"fightAffectData", 2629, 5, Bytes, "Entries 1..4: the affects that end with the fight (O0E3:160D).", 0},

	// Spells.
	{"spellNames", 10175, 101, String(41), "Spell names by spell number.", 0},
	{"spellLevelHeadings", 2167, 10, String(41), "Spell menu headings by spell level.", 0},
	{"spellTableData", 14300, 101, Bytes, "The spell table: one 16-byte record per spell number (spellEntry).", 16},
	{"turnUndeadRolls", 884, 10, Signed, "turnUndeadRolls = DS:0369 + 10*undead + column: the d20 a cleric needs to\nturn undead of each row (1..10), by cleric column (levels 1..8, 9..13,\n14 and up). 0: turned automatically, negative: destroyed (needing its\nabsolute value), 99: impossible.", 10},
	{"wildTerrainFlags", 852, 132, Bytes, "wildTerrainFlags = DS:0354: the wilderness terrain flags (terrainSparse\n...) by area byte 0x342.", 0},
}
