package engine

import (
	"fmt"
	"os"
)

// Script variables (O04C:0D52 setVar, 0F16 getVar and the character values
// behind area2 addresses, 07E4 / 0B4F).

// Area words the setter watches.
const (
	areaWordRedrawA   = 253 // writing either asks for a redraw of the view
	areaWordRedrawB   = 254
	areaWordInside    = 230 // 0 outdoors (wilderness map), else in a dungeon
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
	cvRace         = 114
	cvClass        = 115
	cvSave2        = 155
	cvMaxLevel     = 160
	cvThiefFirst   = 165
	cvThiefLast    = 172
	cvMemoFirst    = 32
	cvMemoLast     = 112
	cvControl      = 184
	cvCoinFirst    = 187 // 187, 189 .. 195: copper .. platinum
	cvCoinLast     = 195
	cvMageLevel    = 201
	cvSex          = 214
	cvAlignment    = 216
	cvPaladinFlag  = 228
	cvExpValue     = 247
	cvExpPerHP     = 249
	cvActive       = 256
	cvSide         = 268
	cvDummy        = 269
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
	statusGone         = 7
	inactiveFlag       = 128
	sideMonster        = 129
	wallsReload        = 128
)

// charismaPercent: O04C:07E4's table for value 719 (charisma as a percent
// reaction bonus).
func charismaPercent(cha byte) uint16 {
	switch {
	case cha <= 3:
		return 0
	case cha <= 7:
		return uint16(cha-3) * 5
	case cha <= 12:
		return 25
	case cha <= 15:
		return uint16(cha-12)*5 + 25
	case cha == 16:
		return 50
	case cha == 17:
		return 55
	case cha <= 25:
		return 60
	}
	return 0
}

func signed(b byte) uint16 { return uint16(int16(int8(b))) }

// charValue = get_player_values (O04C:07E4): a character value behind an area2
// address of the selected character, and whether there is one.
func (e *Engine) charValue(a uint16) (uint16, bool) {
	c := e.selected()
	v := a - charValueBase
	switch {
	case v == cvIntelligence:
		return uint16(c.Stats[statInt].Value), true
	case v == cvConstitution:
		return uint16(c.Stats[statCon].Value), true
	case v == cvRace:
		return signed(c.Race), true
	case v == cvClass:
		return signed(c.Class), true
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
		old := uint16(0)
		if oldClassUsable(c) {
			old = 1
		}
		return signed(c.Levels[clsMage]) + signed(c.OldLevels[clsMage])*old, true
	case v == cvSex:
		return signed(c.Sex), true
	case v == cvAlignment:
		return signed(c.Alignment), true
	case v == cvPaladinFlag:
		return uint16(c.Pad402[0] & 1), true
	case v == cvExpValue:
		return c.ExpValue, true
	case v == cvExpPerHP:
		return uint16(c.ExpPerHP), true
	case v == cvActive:
		r := uint16(inactiveFlag)
		if c.Okay != 0 {
			r = 1
		}
		if e.sc.justJoined {
			r = 0
		}
		e.sc.justJoined = false
		return r, true
	case v == cvSide:
		switch {
		case c.Side == 0 && c.Auto != 0:
			return inactiveFlag, true
		case c.Side == 1:
			return sideMonster, true
		}
		return 0, true
	case v == cvDummy:
		return 0, true // the original returns whatever was in the result
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

// setCharValue = O04C:0B4F: after an area2 write, the character value it
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
		c.Memorized[v-cvMemoFirst+1] = byte(val) // offset 30 + (v-31)
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
	case v == cvActive:
		if val >= inactiveFlag {
			c.Okay = 0
			if val == inactiveFlag+statusGone {
				c.Status = statusGone
			}
		}
		if val == 0 {
			e.sc.activeCleared = true
		}
	case v == cvSide:
		switch val {
		case 0:
			c.Side, c.Auto = 0, 0
		case inactiveFlag:
			c.Side, c.Auto = 0, 1
		case sideMonster:
			c.Side, c.Auto = 1, 1
		}
	case v == cvGameArea:
		e.gameAreaBackup = e.gameArea // R5BD:0B0C
		e.gameArea = byte(val)
	case v == cvWalls1 || v == cvWalls2 || v == cvWalls3:
		if val > wallsReload {
			e.LoadWalldef(int(val%wallsReload), int(v-cvWalls1)/2+1)
		}
	}
}

// getVar = O04C:0F16.
func (e *Engine) getVar(a uint16) uint16 {
	s := &e.sc
	switch addrKindOf(a) {
	case addrArea:
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
		return uint16(s.facing / 2)
	case specialWall:
		return uint16(s.wallHere)
	case specialCell:
		return uint16(s.cellHere)
	}
	return 0
}

// setVar = O04C:0D52.
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
			if val == 0 {
				e.gameState = stateWilderness
			} else {
				e.gameState = stateDungeon
			}
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
