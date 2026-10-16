package engine

// The EGA text colours the game draws with.
const (
	colourBlack        = 0
	colourBlue         = 1
	colourGreen        = 10
	colourCyan         = 11
	colourLightRed     = 12
	colourLightMagenta = 13
	colourYellow       = 14
	colourWhite        = 15
)

// ticksPerSecond: the game's clock (time01) counts hundredths of a second.
const ticksPerSecond = 100

// titleSound is the tune started with the Forgotten Realms banner (DS 9672, a word).
const titleSound = 13

// Game states (game_state_4FD3).
const (
	stateStartMenu = iota
	stateShop
	stateCamp
	stateWilderness
	stateDungeon
	stateCombat
	stateAfterCombat
	stateEndGame
)

// Scan codes of the extended keys the menus read (displayInput's extra keys).
const (
	scanHome     = 'G'
	scanUp       = 'H'
	scanPageUp   = 'I'
	scanLeft     = 'K'
	scanRight    = 'M'
	scanEnd      = 'O'
	scanDown     = 'P'
	scanPageDown = 'Q'
	scanDelete   = 'S'
)
