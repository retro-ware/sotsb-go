package engine

import (
	"fmt"
	"os"
)

// Script variables (O07D:0EC7 setVar, 107E getVar and the character values
// behind area2 addresses, 07BD / 0CAD).

// Area words the setter watches.
const (
	areaWordRedrawA   = 253 // writing either asks for a redraw of the view
	areaWordRedrawB   = 254
	areaWordInside    = 230 // a change puts the party back in the dungeon
	areaWordFacing    = 234 // reads give the facing
	charValueBase     = scriptArea2First
	encounterSpecialA = 990 // special writes kept for the encounter
	encounterSpecialB = 184
	encounterSpecialC = 185
	specialRandomA    = 251 // special reads: the last random values
	specialRandomB    = 252
	specialCombatWord = 177
	specialFacing     = 829
)

// Special addresses past specialReadBase / specialBase.
const (
	specialPosX     = 0
	specialPosY     = 1
	specialDir      = 2
	specialWall     = 3
	specialCell     = 4
	specialSetPosX  = 227
	specialSetPosY  = 228
	specialSetDir   = 229
	specialSetLook1 = 241
	specialSetLook2 = 247
)

// Character values behind area2 addresses (address - 31744).
const (
	cvName         = 0
	cvIntelligence = 21
	cvConstitution = 24
	cvCharismaNow  = 25
	cvMemoFirst    = 27
	cvMemoLast     = 101
	cvRace         = 114
	cvClass        = 115
	cvMaxHP        = 118
	cvSave2        = 155
	cvMaxLevel     = 160
	cvThiefFirst   = 165
	cvThiefLast    = 172
	cvControl      = 184
	cvCoinFirst    = 187 // 187, 189 .. 197: copper .. gems
	cvCoinLast     = 197
	cvMageLevel    = 201
	cvClericLevel  = 202
	cvThiefLevel   = 203
	cvFighterLevel = 204
	cvPaladinLevel = 207
	cvRangerLevel  = 208
	cvSex          = 214
	cvAlignment    = 216
	cvExpValue     = 247
	cvExpPerHP     = 249
	cvStatus       = 256
	cvSide         = 268
	cvDummy        = 269
	cvHP           = 281
	cvMove         = 283
	cvPartyPlace   = 689
	cvPartyPlace2  = 692
	cvCharisma     = 719
	cvGameArea     = 786
	cvWalls1       = 802
	cvWalls2       = 804
	cvWalls3       = 806
	cvPartyCount   = 830

	controlCharmedHigh = 178 // control values above it lose 50 when set
	controlCharmedDrop = 50
	scriptStatusOkay   = 1 // the status value that makes a character okay again
	autoFlag           = 128
	sideMonster        = 129
	wallsReload        = 128
)

// scriptStatus: the status numbers scripts use for each character status
// (statuses not listed read as 0).
var scriptStatus = map[byte]uint16{
	statusOkay:          scriptStatusOkay,
	statusRunning:       6,
	statusUnconscious:   5,
	statusDying:         4,
	statusDead:          3,
	statusStoned:        7,
	statusDisintegrated: 2,
}

// charismaPercent: value 719's table (charisma as a percent reaction
// bonus); scores outside 3..25 give 0.
func charismaPercent(cha byte) uint16 {
	switch {
	case cha == 3:
		return 0
	case cha >= 4 && cha <= 7:
		return uint16(cha-3) * 5
	case cha >= 8 && cha <= 12:
		return 25
	case cha >= 13 && cha <= 15:
		return uint16(cha-12)*5 + 25
	case cha == 16:
		return 50
	case cha == 17:
		return 55
	case cha >= 18 && cha <= 25:
		return 60
	}
	return 0
}

func signed(b byte) uint16 { return uint16(int16(int8(b))) }

// classLevel: a class's level, with the old one added for a dual-classed
// human who may use it.
func classLevel(c *Character, class int) uint16 {
	level := signed(c.Levels[class])
	if oldClassUsable(c) {
		level += signed(c.OldLevels[class])
	}
	return level
}

// charValue = O07D:07BD: a character value behind an area2 address of the
// selected character, and whether there is one.
func (e *Engine) charValue(a uint16) (uint16, bool) {
	c := e.selected()
	v := a - charValueBase
	switch {
	case v == cvIntelligence:
		return uint16(c.Stats[statInt].Value), true
	case v == cvConstitution:
		return uint16(c.Stats[statCon].Value), true
	case v == cvCharismaNow:
		return uint16(c.Stats[statCha].Value), true
	case v == cvRace:
		return signed(c.Race), true
	case v == cvClass:
		return signed(c.Class), true
	case v == cvMaxHP:
		return uint16(c.MaxHP), true
	case v == cvSave2:
		return uint16(c.Saves[1]), true
	case v == cvMaxLevel:
		return signed(c.MaxLevel), true
	case v >= cvThiefFirst && v <= cvThiefLast:
		return uint16(c.Thief[v-cvThiefFirst]), true
	case v == cvControl:
		return uint16(c.Control), true
	case v >= cvCoinFirst && v <= cvCoinLast && (v-cvCoinFirst)%2 == 0:
		return c.Coins[(v-cvCoinFirst)/2], true
	case v == cvMageLevel:
		return classLevel(c, clsMage), true
	case v == cvClericLevel:
		return classLevel(c, clsCleric), true
	case v == cvThiefLevel:
		return classLevel(c, clsThief), true
	case v == cvFighterLevel:
		return classLevel(c, clsFighter), true
	case v == cvPaladinLevel:
		return classLevel(c, clsPaladin), true
	case v == cvRangerLevel:
		return classLevel(c, clsRanger), true
	case v == cvSex:
		return signed(c.Sex), true
	case v == cvAlignment:
		return signed(c.Alignment), true
	case v == cvExpValue:
		return c.ExpValue, true
	case v == cvExpPerHP:
		return uint16(c.ExpPerHP), true
	case v == cvStatus:
		if e.sc.noMember {
			return 0, true
		}
		return scriptStatus[c.Status], true
	case v == cvSide:
		switch {
		case c.Side == 0 && c.Auto != 0:
			return autoFlag, true
		case c.Side == 1:
			return sideMonster, true
		}
		return 0, true
	case v == cvDummy:
		return 0, true // the original returns whatever was in the result
	case v == cvHP:
		return uint16(c.HP), true
	case v == cvMove:
		return uint16(c.Move), true
	case v == cvPartyPlace || v == cvPartyPlace2:
		return uint16(e.partyIndex(c)), true
	case v == cvCharisma:
		return charismaPercent(c.Stats[statCha].Value), true
	case v == cvGameArea:
		return uint16(e.gameArea), true
	case v == cvPartyCount:
		return wordAt(e.game, 2*cvPartyCount), true
	}
	return 0, false
}

// setCharValue = O07D:0CAD: after an area2 write, the character value it
// stands for.
func (e *Engine) setCharValue(a, val uint16) {
	c := e.selected()
	v := a - charValueBase
	switch {
	case v == cvName:
		if val == 0 {
			e.sc.nameCleared = true
		}
	case v >= cvMemoFirst && v <= cvMemoLast:
		c.Memorized[v-cvMemoFirst] = byte(val)
	case v == cvControl:
		if val > controlCharmedHigh {
			val -= controlCharmedDrop
		}
		c.Control = byte(val)
	case v >= cvCoinFirst && v <= cvCoinLast && (v-cvCoinFirst)%2 == 0:
		c.Coins[(v-cvCoinFirst)/2] = val
	case v == cvExpValue:
		c.ExpValue = val
	case v == cvExpPerHP:
		c.ExpPerHP = byte(val)
	case v == cvStatus:
		if val == scriptStatusOkay {
			c.Status, c.Okay = statusOkay, 1
		}
	case v == cvSide:
		switch val {
		case 0:
			c.Side, c.Auto = 0, 0
		case autoFlag:
			c.Side, c.Auto = 0, 1
		case sideMonster:
			c.Side, c.Auto = 1, 1
		}
	case v == cvHP:
		c.HP = byte(val)
	case v == cvGameArea:
		e.gameAreaBackup = e.gameArea // R640:0AB0
		e.gameArea = byte((val + 1) / 2)
	case v == cvWalls1 || v == cvWalls2 || v == cvWalls3:
		if val > wallsReload {
			e.LoadWalldef(int(val%wallsReload), int(v-cvWalls1)/2+1)
		}
	}
}

// getVar = O07D:107E.
func (e *Engine) getVar(a uint16) uint16 {
	s := &e.sc
	switch addrKindOf(a) {
	case addrArea:
		if a-scriptAreaFirst == areaWordFacing {
			return uint16(s.facing)
		}
		return wordAt(e.area, 2*int(a-scriptAreaFirst))
	case addrArea2:
		if v, ok := e.charValue(a); ok {
			return v
		}
		return wordAt(e.game, 2*int(a-scriptArea2First))
	case addrAux:
		return wordAt(e.aux, 2*int(a-scriptAuxFirst))
	case addrCode:
		return uint16(s.codeByte(a))
	}
	if a < specialReadBase {
		switch a {
		case specialCombatWord:
			return s.random3
		case specialRandomA:
			return s.random1
		case specialRandomB:
			return s.random2
		case specialFacing:
			return uint16(s.facing)
		}
		return 0
	}
	switch a - specialReadBase {
	case specialPosX:
		return signed(s.posX)
	case specialPosY:
		return signed(s.posY)
	case specialDir:
		if wordAt(e.area, 2*areaWordInside) == 0 {
			return uint16(s.facing)
		}
		return uint16(s.facing / 2)
	case specialWall:
		return uint16(s.wallHere)
	case specialCell:
		return uint16(s.cellHere)
	}
	return 0
}

// setVar = O07D:0EC7.
func (e *Engine) setVar(a, val uint16) {
	s := &e.sc
	if scriptTrace {
		fmt.Fprintf(os.Stderr, "    set %d = %d\n", a, val)
	}
	switch addrKindOf(a) {
	case addrArea:
		w := a - scriptAreaFirst
		switch {
		case w == areaWordRedrawA || w == areaWordRedrawB:
			s.viewChanged = true
		case w == areaWordInside && wordAt(e.area, 2*areaWordInside) != val:
			e.lastGameState = e.gameState
			e.gameState = stateDungeon
		}
		putWord(e.area, 2*int(w), val)
	case addrArea2:
		putWord(e.game, 2*int(a-scriptArea2First), val)
		e.setCharValue(a, val)
	case addrAux:
		putWord(e.aux, 2*int(a-scriptAuxFirst), val)
	case addrCode:
		s.setCodeByte(a, byte(val))
	default:
		if a < specialBase {
			switch a {
			case encounterSpecialA:
				s.encounterA = val
			case encounterSpecialB:
				s.encounterB = val
			case encounterSpecialC:
				s.encounterC = val
			}
			return
		}
		switch a - specialBase {
		case specialSetPosX:
			s.moved = true
			s.posX = byte(val)
		case specialSetPosY:
			s.posY = byte(val)
			s.moved = true
		case specialSetDir:
			s.facing = byte(val%4) * 2
			s.moved = true
		case specialSetLook1:
			s.look1 = byte(val)
			s.redraw = true
		case specialSetLook2:
			s.look2 = byte(val)
			s.redraw = true
		}
	}
}
