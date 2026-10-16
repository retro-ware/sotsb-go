package engine

// Script commands that show things: pictures, monsters approaching, the
// area's map and walls, and CALL's special routines (O03C).

// cmdSetupMonster = O03C:03C9 (sprite, distance, picture): the encounter's
// sprite at the distance (no farther than the walls allow).
func (e *Engine) cmdSetupMonster() {
	s := &e.sc
	e.readArgs(3)
	s.sprite = byte(e.arg(1))
	putWord(e.game, 2*gameWordEncDist, e.arg(2))
	s.closeup = byte(e.arg(3))
	e.setEncounterDistance()
	e.showEncounter(s.sprite, s.closeup, byte(wordAt(e.game, 2*gameWordDistance)))
}

// setEncounterDistance: the distance is the free steps ahead, at most the
// one the script asked for.
func (e *Engine) setEncounterDistance() {
	s := &e.sc
	d := uint16(e.freeSteps(s.posX, s.posY, s.facing))
	if limit := wordAt(e.game, 2*gameWordEncDist); limit < d {
		d = limit
	}
	putWord(e.game, 2*gameWordDistance, d)
}

// cmdApproach = O03C:0835: the monsters come one step closer.
func (e *Engine) cmdApproach() {
	s := &e.sc
	if d := wordAt(e.game, 2*gameWordDistance); d > 0 {
		putWord(e.game, 2*gameWordDistance, d-1)
		e.showEncounter(s.sprite, s.closeup, byte(d-1))
	}
	s.pc++
}

// freeSteps = O04C:04B4 (x, y, dir): open squares ahead, 0..2; 2 without
// walls to stop the view (outdoors).
func (e *Engine) freeSteps(x, y, dir byte) byte {
	if wordAt(e.area, 2*areaWordFreeSteps) == 0 {
		putWord(e.game, 2*gameWordDistance, maxDistance)
		return maxDistance
	}
	var n byte
	for n < maxDistance {
		if e.WallAt(int(int8(x)), int(int8(y)), int(dir)) != 0 {
			break
		}
		n++
		switch dir {
		case dirNorth:
			y--
		case dirEast:
			x++
		case dirSouth:
			y++
		case dirWest:
			x--
		}
	}
	return n
}

const maxDistance = 2

// showEncounter = O04C:0580 (sprite, picture, distance): the approaching
// monster's sprite for the distance; at distance 0 its picture (or the
// portrait the script set).
func (e *Engine) showEncounter(sprite, pic, dist byte) {
	s := &e.sc
	if !s.pictureBig {
		if !s.pictureShown {
			if e.gs.overhead {
				e.gs.overhead = false
				e.viewReset = true
				e.drawView()
			}
			if wordAt(e.area, 2*areaWordFreeSteps) != 0 {
				e.loadViewPicture("SPRIT", sprite, 1)
				s.pictureShown = true
				s.spriteOn = true
			}
		} else {
			e.viewReset = true
			e.drawView()
		}
		if e.gameState == stateDungeon {
			e.showSprite(int(dist) + 1)
		}
	}
	face := wordAt(e.game, 2*gameWordFace)
	if (!s.pictureBig || uint16(s.faceShown) != face) && dist == 0 && e.gameState == stateDungeon &&
		(!s.inEncounter || e.forcePicture) {
		s.faceShown = byte(face)
		s.pictureOn = true
		if face == noPicture {
			e.loadViewPicture("PIC", pic, 0)
			s.pictureBig = true
			e.drawViewPicture(viewPicX, viewPicY, true, 0)
			return
		}
		e.showPortrait(byte(face), pic)
		s.pictureBig = true
		s.viewDrawn = false
	}
}

// showPortrait = O04C:0543 (head, body).
func (e *Engine) showPortrait(head, body byte) {
	e.sc.viewDrawn = false
	e.shopHead, e.shopBody = head, body
	e.loadPortrait(head, body)
	e.drawPortrait(viewPicX, viewPicY, true)
}

// cmdPicture = O03C:0873 (picture): show a picture (a big one from 120
// on), or with 255 put the view back.
func (e *Engine) cmdPicture() {
	s := &e.sc
	e.readArgs(1)
	n := byte(e.arg(1))
	if n != noPicture {
		s.pictureBig = true
		s.pictureOn = true
		if face := wordAt(e.game, 2*gameWordFace); face == noPicture {
			s.viewDrawn = true
			if n >= firstBigPicture {
				e.bigPicture(n)
				e.wildernessPicture()
				e.viewReset = false
			} else {
				e.loadViewPicture("PIC", n, 0)
				e.drawViewPicture(viewPicX, viewPicY, true, 0)
			}
		} else {
			e.showPortrait(byte(face), n)
		}
		return
	}
	if (e.lastGameState != stateDungeon || e.gameState == stateDungeon) && (s.pictureOn || s.spriteOn) {
		e.viewReset = true
		e.drawView()
		s.pictureOn = false
		s.spriteOn = false
		s.viewDrawn = true
	}
	s.pictureShown = false
	s.pictureBig = false
}

const firstBigPicture = 120

// Area words LOAD FILES reads.
const (
	areaWordGeo       = 197 // the GEO block loaded
	areaWordWalls2Set = 231 // both set: only slots 1 and 3 are loaded
	areaWordWalls3Set = 232
	noBlock           = 255
	defaultWalls      = 127
	gameWordViewSet   = 713
)

// cmdLoadFiles = O03C:0C41 (LOAD FILES 33: walls, -, map; LOAD PIECES 55:
// wall sets 1..3).
func (e *Engine) cmdLoadFiles() {
	s := &e.sc
	e.readArgs(3)
	s.keepScript = true
	var a [4]byte
	for k := 1; k <= 3; k++ {
		a[k] = byte(e.arg(k))
	}
	if s.command == cmdLoadFiles {
		s.mapLoaded = true
		if a[1] != noBlock && a[1] != defaultWalls && wordAt(e.area, 2*areaWordFreeSteps) != 0 {
			putWord(e.area, 2*areaWordGeo, uint16(a[1]))
			e.loadGeo(a[1])
			putWord(e.game, 2*gameWordViewSet, 0)
		}
		if a[3] != noBlock && wordAt(e.area, 2*areaWordFreeSteps) == 0 && e.mapMode != mapPicture {
			e.bigPicture(bigPicWild)
		}
	} else {
		s.wallsLoaded = true
		switch {
		case a[1] == defaultWalls:
			e.LoadWalldef(0, 1)
		case wordAt(e.area, 2*areaWordWalls2Set) != 0 && wordAt(e.area, 2*areaWordWalls3Set) != 0:
			if a[1] != noBlock {
				e.LoadWalldef(int(a[1]), 1)
			}
			if a[3] != noBlock {
				e.LoadWalldef(int(a[3]), 3)
			}
		default:
			for k := 1; k <= 3; k++ {
				if a[k] != noBlock {
					e.LoadWalldef(int(a[k]), k)
				} else {
					e.mapPairA[k-1], e.mapPairB[k-1] = -1, -1
				}
			}
		}
	}
	if s.wallsLoaded && s.mapLoaded && e.lastGameState == stateWilderness {
		if e.gameState != stateWilderness && s.firstLoad {
			e.drawAdventureFrame()
			e.playerSummary(e.selected())
			e.statusLine()
		}
		s.firstLoad = false
	}
}

// loadGeo = O116:130D Load3DMap (block): GEO<area>.DAX block into the map.
func (e *Engine) loadGeo(block byte) { e.LoadGeo(int(block)) }

// CALL addresses (the operand's address, not its value).
const (
	callRedrawView  = 11792
	callDuelWin     = 32768
	callDuelLose    = 32769
	callSound       = 45568
	callStepForward = 49182
	callWallAhead   = 49176
	callAnimate     = 26627
	soundDuelA      = 8
	soundDuelB      = 10
)

// cmdCall = O03C:2F23: a routine the engine provides, chosen by address.
func (e *Engine) cmdCall() {
	s := &e.sc
	e.readArgs(1)
	switch varAddr(s.argLo[1], s.argHi[1]) {
	case callRedrawView:
		s.cellHere = e.MapCell(int(s.posX), int(s.posY))
		if s.keepScript && (s.pictureOn || s.spriteOn || s.redraw || s.moved || s.viewChanged) {
			e.viewReset = true
			e.drawView()
			e.statusLine()
			s.viewChanged, s.redraw, s.moved = false, false, false
			s.pictureOn, s.spriteOn = false, false
			s.wallHere = e.WallAt(int(s.posX), int(s.posY), int(s.facing))
		}
	case callDuelWin:
		e.scriptDuel(true)
	case callDuelLose:
		e.scriptDuel(false)
	case callSound:
		switch s.encounterA {
		case soundDuelA:
			e.PlaySound(sndStep)
		case soundDuelB:
			e.PlaySound(sndSpell47)
		default:
			e.PlaySound(sndStep)
		}
	case callStepForward:
		e.stepForward()
	case callWallAhead:
		if wordAt(e.area, 2*areaWordFreeSteps) == 0 {
			s.wallHere = e.WallAt(int(s.posX), int(s.posY), int(s.facing))
		}
	case callAnimate:
		a := &e.pic
		e.drawViewPicture(viewPicX, viewPicY, true, a.cur-1)
		a.cur++
		if a.cur > len(a.frames) {
			a.cur = 1
		}
		e.gameDelay()
	}
}
