package engine

import (
	"log"
	"os"
)

// Init = O060:0054: allocate the global records, set the start state and
// load the always-resident assets. Everything the DOS code zeroed with
// FillChar is zero-valued here.
func (e *Engine) Init() {
	e.Randomize()                // O0A1:0027 seeds before anything else
	e.game = make([]byte, 0x800) // DS:49D6
	e.area = make([]byte, 0x800) // DS:49D2
	e.aux = make([]byte, 0x400)  // DS:49DA
	e.ecl = make([]byte, 0x1E00) // DS:49DE, ECL code at pc-0x9900
	e.eclPC = 0x9900             // DS:49ED
	e.gameArea = 5               // b7420 (O0A1:0397): the symbols of 8X8D1, the font of 8X8D5
	e.viewMode, e.prevViewMode = 1, 1
	e.saved = false
	e.flag49F5, e.flag49F6 = 1, 1         // DS:49F5/49F6
	e.slowText = true                     // DS:4A00
	e.listTop = 0                         // w72AC
	e.mapPairA[1], e.mapPairA[2] = -1, -1 // DS:6AAE+4i, i = 2..3
	e.mapPairB[1], e.mapPairB[2] = -1, -1 // DS:6AB0+4i
	e.es.v6F69, e.es.v6F68 = 232, 219     // b8F4D/b8F4C
	e.newGameState()

	e.loadGameFont()
	e.ClearRow24()
	e.DisplayString(0, 24, 0xA, "Loading...Please Wait")
	e.LoadSymbolSet(4, 202) // O0A1:0489
	e.LoadSymbolSet(0, 203)
	for i := 0; i <= 0xB; i++ {
		e.LoadIconPic("COMSPR", i, i+0xD) // O195:01CF
	}
	e.LoadIconPic("COMSPR", 0x19, 0x19)
	e.picturesOn, e.loadingMessage = true, true // b4FD7, b4FD8 (O060:0054)
	if e.Cfg.Graphics == 'E' {
		e.videoMode = 1 // b5000
	}
	if e.colourCard() {
		e.loadSkyPictures()
	}
	e.items = e.readItems() // ITEMS file, 2-byte header then 0x810 bytes → DS:5582
	e.ClearRow24()
	e.spellTableInit() // O113:67A1
	setupAffectHandlers()
}

// readItems reads the base item table: skips the 2-byte count and takes up
// to 0x810 bytes of 16-byte records.
func (e *Engine) readItems() []byte {
	p := e.L.Path("ITEMS")
	b, err := os.ReadFile(p)
	if err != nil || len(b) < 2 {
		// extra: without the table no class may use any item and nothing has stats
		log.Println("ITEMS (the base item table) is missing from the game data: import the game again")
		return make([]byte, 0x810)
	}
	out := make([]byte, 0x810)
	copy(out, b[2:])
	return out
}

// InitGraphics loads the font and symbol sets 4 and 0 the way Init does
// (for tools that skip Init).
func (e *Engine) InitGraphics() {
	e.loadGameFont()
	e.LoadSymbolSet(4, 202)
	e.LoadSymbolSet(0, 203)
}

// newGameState = O0A1:07C2: the state of a fresh game (also after the
// party is lost).
func (e *Engine) newGameState() {
	s := &e.sc
	for i := range e.area {
		e.area[i] = 0
	}
	putWord(e.area, 2*areaWordDifficulty, 2)
	putWord(e.area, 2*areaWordFreeSteps, 1)
	putWord(e.area, 2*areaWordLastECL, 0)
	for i := range e.game {
		e.game[i] = 0
	}
	for i := range e.aux {
		e.aux[i] = 0
	}
	s.code = [scriptCodeSize]byte{}
	s.posX, s.posY, s.facing = startX, startY, dirEast
	s.wallHere, s.cellHere = 0, 0
	e.canBash, e.canPick, e.canKnock = true, true, true
	for k := 2; k <= 3; k++ {
		e.mapPairA[k-1], e.mapPairB[k-1] = -1, -1
	}
	e.debug = false
	e.slowText = true
	e.loadedGame = false
	e.party, e.partySel = nil, 0
	s.pc = scriptCodeFirst
	e.gameSpeed = defaultGameSpeed
	e.gameArea, e.gameAreaBackup = 1, 1
	e.gs.overhead = false
	putWord(e.game, 2*cvPartyCount, 0)
	e.listTop = 0
	s.lookSave = 0
	s.inSpecial = false
	e.textNewLine = false
	s.pictureOn = false
	e.partyDead = false
	s.spriteOn = false
	e.spriteShown = false
	e.picFrame = 0
	e.mapMode = noPicture
	e.spriteFrame = noPicture
	e.textWaiting = false
	s.look2, s.look1 = lookStart2, lookStart1
	s.inEncounter = false
	e.areaKind = noPicture
	e.silentTraining = false
	e.ClearRow24()
	e.menuSel = 1
	e.gameState = stateDungeon
	e.lastGameState = stateStartMenu
	e.gameWon = false
}

// Where a new party wakes up, and the start's look values.
const (
	startX           = 7
	startY           = 13
	defaultGameSpeed = 4
	lookStart1       = 219
	lookStart2       = 232
)

// The game's font: block 201 of 8X8D5.DAX, 8 bytes per glyph.
const (
	gameFontDAX   = "8X8D5"
	gameFontBlock = 201
)

// loadGameFont = R540:0184.
func (e *Engine) loadGameFont() {
	e.font = e.daxBlock(gameFontDAX, gameFontBlock)
}
