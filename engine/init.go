package engine

import (
	"log"
	"os"
)

// Init = O060:0054: allocate the global records, set the start state and
// load the always-resident assets. Everything the DOS code zeroed with
// FillChar is zero-valued here.
func (e *Engine) Init() {
	e.game = make([]byte, 0x800) // DS:49D6
	e.area = make([]byte, 0x800) // DS:49D2
	e.aux = make([]byte, 0x400)  // DS:49DA
	e.ecl = make([]byte, 0x1E00) // DS:49DE, ECL code at pc-0x9900
	e.eclPC = 0x9900             // DS:49ED
	e.symDAX = "8X8D1"           // DS:5376 = 1
	e.viewMode, e.prevViewMode = 1, 1
	e.saved = false
	e.flag49F5, e.flag49F6 = 1, 1         // DS:49F5/49F6
	e.slowText = true                     // DS:4A00
	e.listTop = 0                         // w72AC
	e.mapPairA[1], e.mapPairA[2] = -1, -1 // DS:6AAE+4i, i = 2..3
	e.mapPairB[1], e.mapPairB[2] = -1, -1 // DS:6AB0+4i
	e.es.v6F69, e.es.v6F68 = 0xE8, 0xDB   // DS:6F69/6F68
	e.symPrev = e.symDAX                  // DS:5377
	e.newGameState()

	e.font = e.daxBlock(e.symDAX, 201) // R4CE:017C
	e.ClearRow24()
	e.DisplayString(0, 24, 0xA, "Loading...Please Wait")
	e.LoadSymbolSet(4, 202) // Pool ovr35:003B
	e.LoadSymbolSet(0, 203)
	e.sqrpaci[0] = e.LoadPic("SQRPACI", 1) // R226:0111 → DS:5E52
	e.sqrpaci[1] = e.LoadPic("SQRPACI", 2) // DS:5E56
	for i := 0; i <= 0xB; i++ {
		e.LoadIconPic("COMSPR", i, i+0xD) // O12A:01D0
	}
	e.LoadIconPic("COMSPR", 0x19, 0x19)
	e.picturesOn, e.loadingMessage = true, true // b4FD7, b4FD8 (O060:0054)
	if e.Cfg.Graphics == 'E' {
		e.videoMode = 1 // b5000
		e.loadSkyPictures()
	}
	e.items = e.readItems() // ITEMS file, 2-byte header then 0x810 bytes → DS:5582
	e.ClearRow24()
	e.Randomize()      // R811:24D1
	e.spellTableInit() // O0C2:658F
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
	e.font = e.daxBlock(e.symDAX, 201)
	e.LoadSymbolSet(4, 202)
	e.LoadSymbolSet(0, 203)
}

// newGameState = O060:06E5: the state of a fresh game (also after the
// party is lost).
func (e *Engine) newGameState() {
	s := &e.sc
	for i := range e.area {
		e.area[i] = 0
	}
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
