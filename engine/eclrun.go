package engine

import (
	"fmt"
	"os"
	"strconv"
)

// scriptTrace (SOTSB_ECLTRACE=1) prints each command run, the way the
// original does with its debug flag (b4FDA).
var scriptTrace = os.Getenv("SOTSB_ECLTRACE") != ""

// Running the area script: init (O07D:01B9), loading (O07D:032C), the
// event runner (O050:3BF2), area changes (O050:3C5C) and the adventuring
// loop (O050:3DB9).

// Area and area2 words the loop keeps.
const (
	areaWordFreeSteps  = 230 // 1: looking ahead is limited by walls (dungeons)
	areaWordLastECL    = 242 // the ECL block to come back to
	areaWordOldX       = 240
	areaWordOldY       = 241
	areaWordNoMagic    = 229 // spells and magic items cannot be used here; cleared with a fresh area
	areaWordFlagsFirst = 256 // .. 287: cleared with a fresh area
	areaWordFlagsLast  = 287
	gameWordFlagsFirst = 889 // .. 898
	gameWordFlagsLast  = 898
	gameWordDistance   = 705 // the distance to the encounter
	gameWordLook       = 714 // above 1: the party searches / looks
	gameWordTime1      = 722
	gameWordTime2      = 723
	gameWordFace       = 737 // 255 or the portrait the pictures use
	gameWordSearchFlag = 725 // the event ran from a step
	gameWordEncDist    = 704
	firstMonsterIcon   = 8
	textLookColumn     = 65
	textLookRow        = 9
)

// initScript = O07D:01B9: reset the machine and read the five entry
// points at the start of the code.
func (e *Engine) initScript() {
	s := &e.sc
	s.pictureOn, s.clearPicture = false, false
	s.activeCleared, s.nameCleared = false, false
	s.look1, s.look2 = textLookColumn, textLookRow
	s.redraw = true
	s.pictureShown, s.pictureBig = false, false
	s.iconSlot = firstMonsterIcon
	s.pc = scriptCodeFirst
	s.monsterCount = 0
	s.calls = nil
	s.flags = [6]bool{}
	putWord(e.game, 2*gameWordFace, 255)
	putWord(e.game, 2*gameWordTime1, 0)
	putWord(e.game, 2*gameWordTime2, 0)
	putWord(e.area, 2*areaWordNoMagic, 0)
	for k := range s.events {
		e.readArgs(1)
		s.events[k] = varAddr(s.argLo[1], s.argHi[1])
	}
	if !e.loadedGame {
		for w := areaWordFlagsFirst; w <= areaWordFlagsLast; w++ {
			putWord(e.area, 2*w, 0)
		}
		for w := gameWordFlagsFirst; w <= gameWordFlagsLast; w++ {
			putWord(e.game, 2*w, 0)
		}
	}
}

// loadScript = O07D:032C load_ecl_dax (block): ECL<area>.DAX block into
// the code (its first word is the size).
func (e *Engine) loadScript(block byte) {
	e.sc.code = [scriptCodeSize]byte{}
	e.ClearRow24()
	e.displayStringBg(0, statusRow, colourGreen, colourBlack, "Loading...Please Wait")
	b := e.daxBlock("ECL"+strconv.Itoa(int(e.gameArea)), int(block))
	if len(b) >= 2 {
		copy(e.sc.code[:], b[2:])
	}
	e.ClearRow24()
}

// runEvent = O050:3BF2 (entry): run commands from entry until one ends the
// event or the game is over.
func (e *Engine) runEvent(entry uint16) {
	s := &e.sc
	s.pc = entry
	s.stop = false
	for !s.stop && !e.partyDead {
		s.lastCommand = s.command
		s.command = s.codeByte(s.pc)
		if scriptTrace {
			fmt.Fprintf(os.Stderr, "ecl %5d cmd %2d\n", s.pc, s.command)
		}
		e.commandTable()
	}
	s.stop = false
}

// skipCommand = O07D:1F3D: step over the command at pc.
func (e *Engine) skipCommand() {
	s := &e.sc
	s.command = s.codeByte(s.pc)
	switch s.command {
	case 1, 2, 10, 14, 17, 18, 29, 32, 45, 50, 52, 54, 56, 57, 60, 63, 64:
		e.readArgs(1)
	case 3, 8, 9, 15, 16, 31, 34, cmdGainExp:
		e.readArgs(2)
	case 4, 5, 6, 7, 11, 12, 33, 40, 42, 47, 48, 53, 55, 59:
		e.readArgs(3)
	case 20, 35:
		e.readArgs(4)
	case 46:
		e.readArgs(5)
	case 30, 44:
		e.readArgs(6)
	case 39:
		e.readArgs(8)
	case 41:
		e.readArgs(14)
	default:
		s.pc++
	}
}

// scriptRestart = O050:3C5C: after an event loaded another script, run the
// new one's start events until none loads yet another.
func (e *Engine) scriptRestart() {
	s := &e.sc
	for {
		e.restoreBackground()
		e.picFrame = 0
		e.spriteFrame = noPicture
		s.restart = false
		s.cellHere = e.MapCell(int(s.posX), int(s.posY))
		s.wallHere = e.WallAt(int(s.posX), int(s.posY), int(s.facing))
		s.moved = true
		putWord(e.game, 2*gameWordSearchFlag, 0)
		s.savedSelected = e.selected()
		e.runEvent(s.events[4])
		if !s.restart {
			putWord(e.area, 2*areaWordLastECL, uint16(s.block))
		}
		if !s.restart {
			if (e.lastGameState != stateDungeon || e.gameState == stateDungeon) && s.keepScript ||
				e.lastGameState == stateDungeon && e.gameState == stateDungeon {
				e.drawView()
			}
			s.restart = false
			e.runEvent(s.events[0])
			if !s.restart {
				e.runEvent(s.events[1])
				if !s.restart {
					e.selectMember(s.savedSelected)
					e.playerSummary(e.selected())
				}
			}
		}
		if !s.restart {
			break
		}
	}
	e.lastGameState = e.gameState
}

// Adventure = O050:3DB9: load the area's script, run its start events,
// then take commands until the game ends.
func (e *Engine) Adventure() {
	s := &e.sc
	e.partyDead = e.partyAllDown()
	s.savedSelected = e.selected()
	e.viewReset = true
	s.clearPicture = false
	s.wallsLoaded, s.mapLoaded = false, false
	s.restoreSelected = false
	s.keepScript = false
	s.firstLoad = true
	e.gameState = stateDungeon
	s.restart = false
	if last := wordAt(e.area, 2*areaWordLastECL); last == 0 {
		s.firstLoad = false
		if e.demo {
			e.drawScreenFrame()
			s.block = demoBlock
			e.gameArea = 1
		} else {
			e.gameArea = 2
			s.block = firstBlock
			e.playerSummary(e.selected())
		}
	} else {
		s.block = byte(last)
	}
	if wordAt(e.area, 2*areaWordFreeSteps) == 0 {
		e.gameState = stateWilderness
	}
	e.loadScript(s.block)
	e.initScript()
	if e.demo {
		e.drawAdventureFrame()
		e.playerSummary(e.selected())
	}
	e.runEvent(s.events[4])
	if e.demo {
		e.releaseParty(true)
		return
	}
	if !s.restart {
		putWord(e.area, 2*areaWordLastECL, uint16(s.block))
	} else {
		e.scriptRestart()
	}
	if e.gameState != stateWilderness && e.loadedGame {
		if s.firstLoad {
			e.loadPicture()
		}
		e.viewReset = true
		e.drawView()
	}
	e.loadedGame = false
	s.moveMode = false
	for !e.partyDead {
		k := e.adventureInput()
		s.savedSelected = e.selected()
		if !s.restart {
			putWord(e.area, 2*areaWordLastECL, uint16(s.block))
		}
		for (wordAt(e.game, 2*gameWordLook) > 1 || upper(k) == 'E') && !e.partyDead {
			if upper(k) == 'E' {
				e.campEvents()
			} else {
				s.lookSave = byte(wordAt(e.game, 2*gameWordLook) & 1)
				putWord(e.game, 2*gameWordLook, 1)
				e.viewReset = true
				e.drawView()
				e.runEvent(s.events[1])
				if s.restart {
					e.scriptRestart()
				}
				putWord(e.game, 2*gameWordLook, uint16(s.lookSave))
			}
			if !e.partyDead {
				k = e.adventureInput()
				s.savedSelected = e.selected()
			}
		}
		if !e.partyDead {
			e.runEvent(s.events[0])
		}
		if s.restart {
			e.scriptRestart()
		} else if !e.partyDead {
			putWord(e.area, 2*areaWordOldX, signed(s.posX))
			putWord(e.area, 2*areaWordOldY, signed(s.posY))
			e.lockedDoor()
			e.drawView()
			if signed(s.posX) != wordAt(e.area, 2*areaWordOldX) || signed(s.posY) != wordAt(e.area, 2*areaWordOldY) {
				e.PlaySound(sndStep)
			}
			s.pictureOn = false
			s.viewDrawn = true
			e.runEvent(s.events[1])
			if s.restart {
				e.scriptRestart()
			}
		}
	}
	e.partyDead = false
	e.releaseParty(false)
}

// partyAllDown = O050:3D56: outside the demonstration, whether no party
// member is still up (an empty party counts as down).
func (e *Engine) partyAllDown() bool {
	if e.demo {
		return false
	}
	for _, c := range e.party {
		if c.Okay != 0 {
			return false
		}
	}
	return true
}

// releaseParty: the loop of O050:3DB9 that frees every member in turn
// (O0EF:38A4); the demonstration keeps the party count.
func (e *Engine) releaseParty(keepCount bool) {
	for len(e.party) > 0 {
		e.partySel = 0
		e.freePlayers(keepCount, true)
	}
}

// The scripts a game starts with: the demonstration's (in ECL1) and a new
// party's (in ECL2).
const (
	demoBlock  = 3
	firstBlock = 17
)
