package engine

import "encoding/binary"

// Words in the global records (area, area2, aux) and the party list.

func wordAt(b []byte, o int) uint16     { return binary.LittleEndian.Uint16(b[o:]) }
func putWord(b []byte, o int, v uint16) { binary.LittleEndian.PutUint16(b[o:], v) }

// selected is the selected member (player_ptr_6520).
func (e *Engine) selected() *Character {
	if len(e.party) == 0 {
		return &Character{}
	}
	if e.partySel >= len(e.party) {
		e.partySel = len(e.party) - 1
	}
	return e.party[e.partySel]
}

// partyIndex: c's place in the party list (O04C:077E), or the count when
// it is not in it.
func (e *Engine) partyIndex(c *Character) int {
	for i, p := range e.party {
		if p == c {
			return i
		}
	}
	return len(e.party)
}

// The party's square and facing on the area map (b7229, b722A, b722B).
func (e *Engine) partyX() int      { return int(int8(e.sc.posX)) }
func (e *Engine) partyY() int      { return int(int8(e.sc.posY)) }
func (e *Engine) partyFacing() int { return int(e.sc.facing) }

// stepBytes: a step table as the bytes the combat code indexes.
func stepBytes(t [9]int8) []byte {
	b := make([]byte, len(t))
	for i, v := range t {
		b[i] = byte(v)
	}
	return b
}

// stepTable: the column steps then the row steps (18 bytes, as DS:2696).
func stepTable() []byte { return append(stepBytes(stepX), stepBytes(stepY)...) }
