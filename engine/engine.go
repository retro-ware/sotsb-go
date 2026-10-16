// Package engine is the Secret of the Silver Blades game engine, ported
// routine by routine from the Turbo Pascal reconstruction of START.EXE and
// GAME.OVR. It reads every asset from the original data directory at run time.
package engine

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sotsb/dax"
	"sotsb/ega"
)

// Config mirrors BLADES.CFG (read by R210's initialisation): the graphics
// letter (C, E or T), the sound letter (A AdLib, T Tandy, P PC speaker, S
// silent), the alternate input (J joystick, M mouse, N neither) and the save
// directory. The joystick calibration lines are not used: the port has no
// joystick.
type Config struct {
	Graphics, Sound, Input byte
	GameDir, SaveDir       string
	Param                  string // ParamStr(1): "Hoop" skips the title
	Param2                 string // ParamStr(2): the code word that skips the code wheel
}

// Engine holds the global state the DOS program kept in its data segment.
type Engine struct {
	textWindowReady  bool // b67C5: the game screen is up; messages go to its text window
	inTitle          bool // b741E: the title runs (Ctrl-S does not toggle the sound)
	mouseShown       bool // R73D: the cursor is shown
	soundDevice      byte // b741A: the sound device chosen in BLADES.CFG (soundDeviceNone: none)
	digitizedSample  int  // O04B: the digitized sound playing
	digitizedPlaying bool
	savedSoundDevice byte // b741B: the device before a Ctrl-S toggle
	music            musicState

	L         *dax.Loader
	S         *ega.Screen
	Cfg       Config
	tempSel   *Character             // selectTemp: selected but not in the party
	page      [pageStride * 168]byte // DS:4A20 drawing page contents (pageSet)
	pageDraw  bool                   // drawing the 3D view: the original draws into page DS:4A20, one cell up and left of the screen
	flips     int                    // traced Flips (PAGEDUMP debug)
	RandSeed  uint32                 // DS:4428, Turbo Pascal RandSeed
	FixedSeed *uint32                // if set, Randomize uses it instead of the clock (trace runs)
	Trace     TraceFunc              // nil in normal play; see trace.go
	tr        *traceState            // trace mode clock and key script

	font   []byte        // DS:7DC8 -> 8X8D5 block 201, 8 bytes per glyph
	sym    [5]*dax.Image // DS:5E3A symbol sets
	symDAX string        // "8X8D" + DS:5376 digit
	start  time.Time

	pendingScan byte // ReadKey: scan code waiting after a 0 byte
	draining    bool // ReadKeyByte is dropping typed-ahead keys
	menuSel     int  // DS:6B2B, DisplayInput selection
	menuExtra   bool // DS:6B2C

	inputTimeout           int64 // l72B0: DisplayInput returns inputTimeoutKey after this many ticks (0: never)
	inputTimeoutKey        byte  // b72B4
	demo                   bool  // inDemo_4FD5: the game plays its demonstration
	gameArea               byte  // game_area_5C08: 1 in the demo, 2 for the real game
	gameAreaBackup         byte  // game_area_backup_5C09
	lastGameState          byte  // DS:4FD4
	sc                     script
	partyDead              bool                            // b4FE0: everyone is down; back to the party menu
	viewReset              bool                            // b759A: the view must be drawn from scratch
	videoMode              byte                            // b5000: 0 EGA
	mapCursorX, mapCursorY int                             // DS:8C8C, 8C8E
	mapCursorUnder         [mapCursorPx * mapCursorPx]byte // DS:65E4
	canBash                bool                            // b75A2: the locked door may still be bashed
	canPick                bool                            // b75A3
	canKnock               bool                            // b75A4
	cheatArg               bool                            // ParamStr(2) gave the map cheat
	loadingMessage         bool                            // b4FD8: show "Loading...Please Wait" (Alter Pics: animation)
	picturesOn             bool                            // b4FD7: Alter Pics
	spriteShown            bool                            // b7292
	bigPic                 *dax.Image                      // p72A6
	skySun                 *dax.Image                      // l723A
	skyMoon                *dax.Image                      // l723E
	skyFloor               *dax.Image                      // l7242
	shopFrame              bool                            // b8B6E
	shopHead               byte                            // b4FDE: the shop keeper's portrait
	shopBody               byte                            // b4FDF
	treasureCoins          [7]int32                        // a6F8A: the treasure's money
	treasureItems          []*Item                         // p6FA6: the treasure's items
	poolHasMoney           bool                            // b757E: money lies in the pool
	viewStart              *Character                      // p4B10: who the View started with (Trade's first choice)
	curingDisease          bool                            // b6FB6: affects being removed by a cure
	textNewLine            bool                            // b8B7B
	picFrame               byte                            // b729B
	spriteFrame            byte                            // b72A5
	textWaiting            bool                            // b5D0B
	turnOrder              []*Character                    // DS:8E2A the order of this round's turns
	turnIndex              int                             // b8F4A the turn being taken
	ageCheckDue            bool                            // b7D44 someone aged: see who dies of old age
	duelling               bool                            // bA4BA: the duel instead of a fight; SotSB never sets it
	forcePicture           bool                            // b4FD9: the encounter picture shows even during ENCOUNTER
	gameSpeed              byte                            // game_speed_var_4FC2
	freeTraining           bool                            // b75A1: the 'J' cheat of the party menu
	silentTraining         bool                            // b4FE1: training without questions (character creation)
	cannotTrain            bool                            // b75A0: the last training found nothing to raise
	gameWon                bool                            // b8B89: the ending was reached; training is free
	gameState              byte                            // game_state_4FD3: what the party is doing (stateCombat, ...)
	spells                 spellList
	addSource              byte // b759E: where AddPlayer loads characters from (fromCurse ...)
	areaKind               byte // b72AA: 'y' on the wilderness map (its cursor blinks)
	mapMode                byte // b72A4: 'P' while a picture hides the map

	party      []*Character // DS:5D96 list head ...
	partySel   int          // DS:5D92 selected character
	rosterHi   *Character   // SelectCharacter: O0ED:07A2 highlights this one (nil: partySel)
	rosterNone bool         // ... or no one (who = nil)
	loadedGame bool         // DS:49F8
	saved      bool         // b7523 nothing has happened since the game was saved
	debug      bool         // DS:470C

	game, area, aux, ecl []byte // DS:49D6, 49D2, 49DA, 49DE records
	eclPC                int    // DS:49ED
	viewMode             int    // DS:49FA (1 = 3D view)
	prevViewMode         int    // DS:49F9
	sqrpaci              [2]*dax.Image
	items                []byte // DS:5582: 16-byte base item records
	pos3, pos4           byte   // DS:6AB0, 6AB1
	flag49F5, flag49F6   byte
	mapPairA, mapPairB   [3]int // DS:6AAE+4i / 6AB0+4i
	loadingMonster       bool   // DS:557C

	cm             charMgmtState       // ovr17 create/modify/train state
	es             eclState            // ovr08/ovr04 interpreter state
	symPrev        string              // DS:5377
	msgRow, msgCol int                 // DS:65BB / 65BA text-block cursor
	slowText       bool                // DS:4A00: DisplayTextBlock types out
	listTop        int                 // DS:6B29 ovr26 list menu: first visible row
	gs             geoState            // ovr30 map, wall sets and view
	run            eclRun              // ovr04 globals
	pic            animPic             // DS:6ABE view-window picture
	port           portrait            // DS:6B00.. HEAD/BODY portrait
	eclFile        string              // DS:547C
	ext            extras              // convenience features (extras.go)
	ac             afterCombat         // ovr06 globals
	camp           campState           // ovr16/ovr21 globals
	animInput      bool                // animate the view picture while waiting for a key
	resting        byte                // DS:6DDA (ovr21 rest): messages skip the box border
	view           viewState           // ovr20 View screen
	itemsRedraw    bool                // DS:6821
	usingItem      bool                // DS:6DBF
	spellMenuKey   byte                // (extra) the key that closed the last spell menu
	rv             ruleVars            // DS:6814..682A rule globals
	near           nearState           // DS:6716 combatants near a square
	nearList       nearState           // DS:671B copy kept by checkAffect
	combatScreenX  [0x48]int8          // DS:604A
	combatScreenY  [0x48]int8          // DS:6092
	combatants     []*Character        // DS:65BD, combatant i (1-based) at [i-1]; filled by combat setup
	targets        [0x48]byte          // DS:6DE4, near list indexes kept by nearEnemies
	cmap           *combatMap          // DS:6716
	cpos           [0x49]combatPos     // DS:5F27, combatant i at [i]
	ncomb          int                 // DS:5F2A combatants on the map list
	occ            [25][50]byte        // DS:60DB combatant index per square
	bodies         [10]body            // DS:66D6, 1-based, count DS:6715
	fullPicture    bool                // DS:67EC a full-screen picture holds the screen
	summaryBox     textBox             // DS:8A83..8A86 the party summary rows the mouse picks from
	nbodies        int                 // DS:6715
	v70C7          byte                // DS:70C7
	icons          [0x48][2]*dax.Image // DS:5E5A, ready and action pose per slot
	spr            *dax.Image          // DS:6E36, 4 frame missile/hit sprite
	sprUnder       []byte              // DS:4A24, background saved by a sprite draw
	place          placement           // DS:4440..4658 combat start formations
	stinkingClouds []*cloudRecord      // DS:7575
	poisonClouds   []*cloudRecord      // DS:7579
	delayedBlasts  []*cloudRecord      // DS:8D8D delayed blast fireballs waiting
	computerMagic  bool                // b8E14 the computer casts for party members too
	thrownItem     *Item               // p6528: the thrown item that ends the treasure monsters left
	v49F7          byte                // DS:49F7 debug display (never set)

	// combat overlays (ovr10/13/14/23)
	spellTarget            *Character              // DS:6C95 target slot 1
	spellList              []*ListNode             // DS:4766 spell menu lines (O0C2)
	spellIDs               []byte                  // DS:476A spell of each non-header line
	spellItems             [0x31]*Item             // DS:477F scroll of each line
	spellItemCount         int                     // DS:4843
	spellTargets           [0x49]*Character        // DS:6C91+4i targets 2..; slot 1 is spellTarget
	nSpellTargets          int                     // DS:6C94
	spellCaster            *Character              // DS:6DB5
	curSpell               byte                    // DS:681B spell being cast
	tgtX, tgtY             int                     // DS:6DB9/6DBA target square
	combatTargets          bool                    // DS:72BA points at O0C9:1FA1 (set by Combat), else O0C2:114D
	whoVar                 bool                    // SelectCharacter's who is DS:5D92 itself (opWho)
	spellFx                [len(spellNames)]func() // DS:6B84 spell effect table (slot 0 unused)
	v4844, v4845           byte                    // DS:4844/4845 spell scratch
	v681D                  byte                    // DS:681D the speed affects are working out the moves
	panelDirty             bool                    // DS:75FF the side panel is to be redrawn
	showMoves              bool                    // DS:7600 the acting combatant's moves are drawn
	itemCall               bool                    // DS:759C the next affect call is a readied item's
	callItem               *Item                   // the item it is for
	combatRound            byte                    // DS:75A7 rounds since the fight (or the last wait) began
	roundLimit             byte                    // DS:75A8 the fight ends at this round (15 after the last attack)
	shots6E2C              [3]byte                 // DS:6E2C+i, i = 1..2 attack slots
	lastStep               int                     // b59C6 the computer's last step direction
	stuckCount, stuckSteps byte                    // b59C7, b59C8 its steps that went back
	partyMorale            byte                    // b8E13 the party's morale this round
	aiEngaged              bool                    // DS:75FE the computer's combatant has a target in reach

	// H
	areaIconTick byte  // DS:6DDC area map party icon pose counter
	snd          sound // PC speaker driver (08AE)

	DevAction func() // run once at the next adventuring command prompt (cmd/snap)
}

// Dice = O145:11C6 (n, sides): sum of n rolls of 1..sides, kept in a
// byte like the original (wraps at 256).
func (e *Engine) Dice(n, sides int) int {
	t := 0
	for i := 0; i < n; i++ {
		t = (t + e.Random(sides) + 1) & 0xFF
	}
	return t
}

// New opens the data directory and reads POOL.CFG.
func New(dir string, scr *ega.Screen) (*Engine, error) {
	l, err := dax.NewLoader(dir)
	if err != nil {
		return nil, err
	}
	e := &Engine{L: l, S: scr, symDAX: "8X8D1", start: time.Now()}
	e.Cfg = Config{Graphics: 'E', Sound: 'A', Input: 'M', GameDir: dir, SaveDir: filepath.Join(dir, "SAVE")}
	if p := l.Path("BLADES.CFG"); p != "" {
		f, err := os.Open(p)
		if err == nil {
			defer f.Close()
			var lines []string
			sc := bufio.NewScanner(f)
			for sc.Scan() {
				lines = append(lines, strings.TrimSpace(sc.Text()))
			}
			if len(lines) >= 3 && lines[0] != "" && lines[1] != "" && lines[2] != "" {
				e.Cfg.Graphics, e.Cfg.Sound, e.Cfg.Input = upper(lines[0][0]), upper(lines[1][0]), upper(lines[2][0])
				// The save directory (line 4, e.g. C:\SAVE\) is kept as a SAVE folder next to the data.
			}
		}
	}
	e.soundDevice = soundDeviceFor(e.Cfg.Sound)
	if !l.Exists("TITLE.DAX") || !l.Exists("8X8D1.DAX") {
		return nil, fmt.Errorf("%s: not a Secret of the Silver Blades directory (TITLE.DAX / 8X8D1.DAX missing)", dir)
	}
	e.pic.block = -1
	e.port = portrait{headID: 0xFF, bodyID: 0xFF}
	e.loadFont()
	e.loadNotes()
	return e, nil
}

func upper(c byte) byte {
	if c >= 'a' && c <= 'z' {
		return c - 32
	}
	return c
}

// Run is the main program (START.PAS). It runs on its own goroutine and
// blocks on keyboard reads like the DOS program did.
func (e *Engine) Run() {
	e.Init()
	e.musicInstall()
	if e.Cfg.Param != "Hoop" { // ParamStr(1) = 'Hoop' skips the title
		e.Title()
	} else {
		e.textWindowReady = true
	}
	e.gameArea = 2
	e.DisplayString(0, 22, colourGreen, "      Secret of the Silver Blades      ")
	e.DisplayString(0, 23, colourGreen, "   Version 1.30           May 18,1990  ")
	e.demo = e.playOrDemo("              ", demoTimeoutFirst)
	for {
		if e.demo {
			e.gameArea, e.gameSpeed = 2, demoSpeed
		} else {
			e.gameArea = 1
		}
		e.musicRestart()
		if !e.demo {
			e.PartyMenu() // O0EF:01A1
			// The code wheel check (O069:0112) is left out.
			if e.gameState == stateWilderness && !e.wildernessOverview() {
				e.drawTextWindowFrame()
			} else {
				e.drawAdventureFrame()
			}
			e.playerSummary(e.selected())
		}
		e.musicRestart()
		e.SetSkyColours(e.gs.sky[0], e.gs.sky[1], e.gs.sky[2], e.gs.sky[3])
		e.musicRestart()
		e.Adventure()
		e.musicRestart()
		e.newGameState()
		if e.demo {
			e.musicCommand(musicShutdown)
			e.Title()
			e.musicInstall()
			e.ClearKeyboard()
			e.demo = e.playOrDemo("Secret of the Silver Blades ", demoTimeoutAgain)
			e.musicCommand(musicSilence)
		}
	}
}

// The "Play / Demo" prompt after the title: with no answer the demo starts.
const (
	demoTimeoutFirst = 3000 // ticks (30 seconds)
	demoTimeoutAgain = 1000
	demoSpeed        = 9
)

// playOrDemo asks "Play Demo" with a timeout that picks Demo; it tells
// whether the demo was chosen.
func (e *Engine) playOrDemo(prompt string, timeout int64) bool {
	e.inputTimeout, e.inputTimeoutKey = timeout, 'D'
	k, _ := e.DisplayInput(prompt, "Play Demo", colourLightMagenta, colourGreen, colourWhite, false, false)
	e.inputTimeout, e.inputTimeoutKey = 0, 0
	return k == 'D'
}

// Exit = R670:0000 print_and_exit: the sound card back from a Ctrl-S,
// the driver stopped, back to DOS.
func (e *Engine) Exit() {
	e.snd.typ = e.snd.saved
	e.PlaySound(sndStop)
	if e.tr != nil {
		panic(TraceDone{"exit"})
	}
	os.Exit(0)
}

// Randomize = R811:24D1: RandSeed = DOS time (CX = hour:min low word,
// DX = sec:hundredths high word).
func (e *Engine) Randomize() {
	if e.Trace != nil {
		e.Trace("R811:24D1", "Randomize")
	}
	if e.FixedSeed != nil {
		e.RandSeed = *e.FixedSeed
		return
	}
	t := time.Now()
	e.RandSeed = uint32(t.Hour())<<8 | uint32(t.Minute()) | (uint32(t.Second())<<8|uint32(t.Nanosecond()/1e7))<<16
}

// Random = Turbo Pascal Random(n) (R811:244A): RandSeed = RandSeed*0x08088405+1,
// then the high word mod n. The seed advances even for n = 0, which returns 0.
func (e *Engine) Random(n int) int {
	e.RandSeed = e.RandSeed*0x08088405 + 1
	if uint16(n) == 0 {
		return 0
	}
	return int(uint16(e.RandSeed>>16) % uint16(n))
}

// Ticks = R540:0A6B, time in hundredths of a second.
func (e *Engine) Ticks() int64 {
	if e.tr != nil { // virtual clock: 10 ms per call (emu prims.py _ticks)
		e.Trace("R540:0A6B", "Ticks")
		t := e.tr.ms / 10
		e.tr.ms += 10
		return t
	}
	return int64(time.Since(e.start) / (10 * time.Millisecond))
}

// Delay = Crt:029C (Turbo Pascal Delay, milliseconds).
func (e *Engine) Delay(ms int) {
	if e.tr != nil {
		e.Trace("Crt:029C", "Delay", ms)
		e.tr.ms += int64(ms)
		return
	}
	e.S.SetState(ega.Pausing)
	ega.Delay(ms)
	e.S.SetState(ega.Working)
}

// KeyPressed / ReadKey / ClearKeyboard = R811:05DB, BIOS read, R670:0252.
func (e *Engine) KeyPressed() bool {
	if e.tr != nil {
		return e.tr.keyPressed(e)
	}
	e.walkFeed()
	for len(e.ext.ahead) == 0 && e.S.KeyPressed() { // nav.go: translate as the key is seen
		e.feed(e.S.ReadKey())
	}
	e.release()
	return len(e.ext.ahead) > 0
}

func (e *Engine) ReadKey() uint16 {
	if scriptTrace {
		fmt.Fprintf(os.Stderr, "key wait pos %d,%d dir %d wall %d cell %d seed %08x\n", e.sc.posX, e.sc.posY, e.sc.facing, e.sc.wallHere, e.sc.cellHere, e.RandSeed)
	}
	if e.tr != nil {
		return e.tr.readKey(e)
	}
	e.ext.skip = false
	for e.release(); len(e.ext.ahead) == 0; e.release() {
		if e.ext.menu == 0 {
			e.showHint()
		}
		e.feed(e.S.ReadKey())
	}
	k := e.ext.ahead[0]
	e.ext.ahead = e.ext.ahead[1:]
	return k
}

// feed translates a key from the window (nav.go) into the keys the game
// will read.
func (e *Engine) feed(raw uint16) {
	if k := e.translate(raw); k != 0 {
		e.ext.ahead = append(e.ext.ahead, k)
	}
}

// release hands the game the next of the keys a translated key left to
// follow it, once the keys before are read. Not while ReadKeyByte drains
// typed-ahead keys, which would keep only the last.
func (e *Engine) release() {
	if len(e.ext.ahead) == 0 && len(e.ext.later) > 0 && !e.draining {
		e.ext.ahead = append(e.ext.ahead, e.ext.later[0])
		e.ext.later = e.ext.later[1:]
	}
}

// The original's drain loop is dead code: R670:0252 jumps straight to its exit.
func (e *Engine) ClearKeyboard() {}

// idle yields the CPU inside the original's busy-wait loops; it is not a
// call of the original, so it is untraced and costs no virtual time.
func (e *Engine) idle() {
	e.ext.skip = false
	if e.tr == nil {
		e.S.SetState(ega.Waiting)
		ega.Delay(1)
	}
}

// Play is the part of the main loop after the title (dev tools start here).
func (e *Engine) Play() {
	e.gameArea = 2
	e.PartyMenu()
	e.Adventure()
}

// wildernessOverview is the game word the main loop reads before the first
// frame (p67C6^[312]): set while the wilderness shows its overview map.
func (e *Engine) wildernessOverview() bool {
	const overviewWord = 624
	return wordAt(e.game, overviewWord) != 0
}

// textBox is a rectangle of text cells, edges included.
type textBox struct {
	left, top, right, bottom int
}
