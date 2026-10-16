package engine

import (
	"fmt"
	"os"
)

// The ECL script machine: operands, variables, strings, comparisons, calls
// (unit O04C of the reconstruction). Script addresses are 16-bit:
//
//	19200..20223  area record words      (p4FB2)
//	31232..31743  aux record words       (p4FBA)
//	31744..32767  area2 record words     (p4FB6; some are character values)
//	32768..40447  the script code, bytes (p4FBE)
//	others        special values (direction, position, ...)
const (
	scriptAreaFirst  = 19200
	scriptAreaLast   = 20223
	scriptAuxFirst   = 31232
	scriptAuxLast    = 31743
	scriptArea2First = 31744
	scriptArea2Last  = 32767
	scriptCodeFirst  = 32768
	scriptCodeLast   = 40447
	scriptCodeSize   = scriptCodeLast - scriptCodeFirst + 1
	specialBase      = 49000 // writes above it: position and direction
	specialReadBase  = 49227 // reads above it: position, direction, ...
)

// Address kinds (O04C:0723).
const (
	addrArea = iota
	addrArea2
	addrAux
	addrCode
	addrSpecial
)

// Operand types (the byte before each operand).
const (
	argImmediate = 0
	argVarByte   = 1 // 1..3: a variable address (two bytes)
	argAddress   = 2
	argVarWord   = 3
	argText      = 128 // packed text follows
	argTextVar   = 129 // text read from a variable address
	maxArgs      = 64
)

// script is the interpreter state kept in the data segment.
type script struct {
	code    [scriptCodeSize]byte // p4FBE
	pc      uint16               // w4FCD
	argType [maxArgs]byte        // a769F
	argLo   [maxArgs]byte        // a771F
	argHi   [maxArgs]byte        // a76DF
	text    [maxArgs]string      // a7662: the text operands of the current opcode, 1-based
	flags   [6]bool              // b7612..: equal, not equal, less, greater, less-equal, greater-equal
	calls   []uint16             // p760A: return addresses
	events  [5]uint16            // w4FC3..w4FCB: entry points the area's header names

	posX, posY, facing byte // b7229, b722A, b722B
	wallHere, cellHere byte // b722C, b722D
	moved, redraw      bool // b8B82, b8B81
	look1, look2       byte // b7602, b7603

	viewChanged   bool // b8B84: an area word the view depends on changed
	justJoined    bool // b8B87: read once by the active-character value
	nameCleared   bool // b8B6D
	activeCleared bool // b8B6C

	random1, random2, random3 uint16 // w7604, w7606, w7608

	command, lastCommand               byte       // b7619, b7618
	block                              byte       // b8B78: the ECL block of this area
	stop                               bool       // b47F8: the event is over
	restart                            bool       // b47F9: a new script was loaded; run its start events
	restoreSelected                    bool       // b47FA: give back the selection the event started with
	keepScript                         bool       // b47FB: the loaded game's script is kept
	wallsLoaded                        bool       // b47FC
	mapLoaded                          bool       // b47FD
	monsterCount                       byte       // b47FE
	savedSelected                      *Character // p47FF
	pictureShown                       bool       // b8B62
	pictureBig                         bool       // b8B63
	pictureOn                          bool       // b8B7C
	viewDrawn                          bool       // b8B7D
	clearPicture                       bool       // b8B7E
	spriteOn                           bool       // b8B7F
	firstLoad                          bool       // b8B88
	inSpecial                          bool       // b8B7A
	waitedKey                          bool       // b8B80
	monstersLoaded                     bool       // b8B83
	inEncounter                        bool       // b8B85
	sprite, closeup                    byte       // b761B, b761C
	iconSlot                           byte       // b761D
	lookSave                           byte       // b8B79
	faceShown                          byte       // b8B86: the portrait the encounter shows
	encounterA, encounterB, encounterC uint16     // w8B66, w8B68, w8B6A
}

func (s *script) codeByte(a uint16) byte { return s.code[uint16(a-scriptCodeFirst)%scriptCodeSize] }
func (s *script) setCodeByte(a uint16, v byte) {
	s.code[uint16(a-scriptCodeFirst)%scriptCodeSize] = v
}

// addrKindOf = O04C:0723.
func addrKindOf(a uint16) int {
	switch {
	case a >= scriptAreaFirst && a <= scriptAreaLast:
		return addrArea
	case a >= scriptArea2First && a <= scriptArea2Last:
		return addrArea2
	case a >= scriptAuxFirst && a <= scriptAuxLast:
		return addrAux
	case a >= scriptCodeFirst && a <= scriptCodeLast:
		return addrCode
	}
	return addrSpecial
}

// readArgs = O04C:0032 (n): decode the opcode's n operands following pc and
// step pc past them.
func (e *Engine) readArgs(n int) {
	s := &e.sc
	texts := 0
	for k := 1; k <= n; k++ {
		s.argType[k] = s.codeByte(s.pc + 1)
		s.argLo[k] = s.codeByte(s.pc + 2)
		s.pc += 2
		switch s.argType[k] {
		case argVarByte, argAddress, argVarWord:
			s.pc++
			s.argHi[k] = s.codeByte(s.pc)
		case argText:
			texts++
			if s.argLo[k] > 0 {
				s.text[texts] = e.unpackText(int(s.argLo[k]))
			} else {
				s.text[texts] = ""
			}
		case argTextVar:
			texts++
			s.pc++
			s.argHi[k] = s.codeByte(s.pc)
			s.text[texts] = e.readTextVar(varAddr(s.argLo[k], s.argHi[k]))
		}
	}
	s.pc++
	if scriptTrace {
		for k := 1; k <= n; k++ {
			fmt.Fprintf(os.Stderr, "    arg%d type %d %d/%d = %d\n", k, s.argType[k], s.argLo[k], s.argHi[k], e.arg(k))
		}
	}
}

// arg = O04C:0168 (k): the value of operand k.
func (e *Engine) arg(k int) uint16 {
	s := &e.sc
	lo, hi := s.argLo[k], s.argHi[k]
	switch s.argType[k] {
	case argImmediate:
		return uint16(lo)
	case argVarByte, argVarWord, argText:
		return e.getVar(uint16(hi)<<8 | uint16(lo))
	case argAddress, argTextVar:
		return varAddr(lo, hi)
	}
	return 0
}

// varAddr = O04C:06D7 (lo, hi).
func varAddr(lo, hi byte) uint16 { return uint16(hi)<<8 | uint16(lo) }

// letterOf = O04C:06FB: a 6-bit text code as a character (1..31 are letters).
func letterOf(c byte) byte {
	if c > 31 {
		return c
	}
	return c + 64
}

// unpackText = O04C:1279 (len): the text packed 6 bits per character in the
// len bytes after pc (pc ends on the last one).
func (e *Engine) unpackText(n int) string {
	s := &e.sc
	var out []byte
	var cur, prev, c byte
	phase := 1
	for read := 0; read < n; {
		if phase < 4 {
			s.pc++
			prev = cur
			cur = s.codeByte(s.pc)
			read++
		}
		switch phase {
		case 1:
			c = cur >> 2
		case 2:
			c = (prev<<6 + cur>>2) >> 2
		case 3:
			c = (prev<<4 + cur>>4) >> 2
		case 4:
			c = cur & 63
		}
		if phase < 4 {
			phase++
		} else {
			phase = 1
		}
		if c != 0 {
			out = append(out, letterOf(c))
		}
	}
	if phase == 4 {
		if c = cur & 63; c != 0 {
			out = append(out, letterOf(c))
		}
	}
	return string(out)
}

// readTextVar = O04C:1421 (addr): the text kept one character per word (one
// per byte in the code) from addr on, up to a 0; the character's name for
// the first area2 address.
func (e *Engine) readTextVar(a uint16) string {
	var out []byte
	switch addrKindOf(a) {
	case addrArea:
		for i := uint16(0); wordAt(e.area, 2*int(a-scriptAreaFirst+i)) != 0; i++ {
			out = append(out, e.area[2*int(a-scriptAreaFirst+i)])
		}
	case addrArea2:
		if a == scriptArea2First {
			return e.selected().Name()
		}
		for i := uint16(0); wordAt(e.game, 2*int(a-scriptArea2First+i)) != 0; i++ {
			out = append(out, e.game[2*int(a-scriptArea2First+i)])
		}
	case addrAux:
		for i := uint16(0); wordAt(e.aux, 2*int(a-scriptAuxFirst+i)) != 0; i++ {
			out = append(out, e.aux[2*int(a-scriptAuxFirst+i)])
		}
	case addrCode:
		for i := uint16(0); e.sc.codeByte(a+i) != 0; i++ {
			out = append(out, e.sc.codeByte(a+i))
		}
	}
	return string(out)
}

// setTextVar = O04C:105D (addr, text): store text one character per word
// (per byte in the code) with a 0 after it; the first area2 address is the
// selected character's name.
func (e *Engine) setTextVar(a uint16, text string) {
	put := func(rec []byte, base uint16) {
		for i := 0; i < len(text); i++ {
			putWord(rec, 2*(int(a-base)+i), uint16(text[i]))
		}
		putWord(rec, 2*(int(a-base)+len(text)), 0)
	}
	switch addrKindOf(a) {
	case addrArea:
		put(e.area, scriptAreaFirst)
	case addrArea2:
		if a == scriptArea2First {
			e.selected().SetName(text)
			return
		}
		put(e.game, scriptArea2First)
	case addrAux:
		put(e.aux, scriptAuxFirst)
	case addrCode:
		for i := 0; i < len(text); i++ {
			e.sc.setCodeByte(a+uint16(i), text[i])
		}
		e.sc.setCodeByte(a+uint16(len(text)), 0)
	}
}

// compareValues = O04C:1A11 / 193B: set the six comparison flags.
func (e *Engine) compareValues(a, b uint16) {
	e.sc.flags = [6]bool{a == b, a != b, a < b, a > b, a <= b, a >= b}
}

func (e *Engine) compareTexts(a, b string) {
	e.sc.flags = [6]bool{a == b, a != b, a < b, a > b, a <= b, a >= b}
}

// scriptCall = O04C:1A77 (k): push pc and jump to the address in operand k.
func (e *Engine) scriptCall(k int) {
	e.sc.calls = append(e.sc.calls, e.sc.pc)
	e.sc.pc = varAddr(e.sc.argLo[k], e.sc.argHi[k])
}

// stepForward = O04C:1B01: one square ahead on the 16x16 map, wrapping at the
// edges; then the wall and cell values there.
func (e *Engine) stepForward() {
	s := &e.sc
	switch s.facing {
	case dirNorth:
		s.posY = byte((int(s.posY) + 15) % 16)
	case dirEast:
		s.posX = byte((int(s.posX) + 1) % 16)
	case dirSouth:
		s.posY = byte((int(s.posY) + 1) % 16)
	case dirWest:
		s.posX = byte((int(s.posX) + 15) % 16)
	}
	s.cellHere = e.MapCell(int(s.posX), int(s.posY))
	s.wallHere = e.WallAt(int(s.posX), int(s.posY), int(s.facing))
	s.moved = true
}

// Facings (multiples of 2: the 8 compass points).
const (
	dirNorth = 0
	dirEast  = 2
	dirSouth = 4
	dirWest  = 6
)
