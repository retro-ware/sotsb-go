package engine

// Item Look (extra): the shop's Look shows what an item does under the
// game's rules, read from its base record (DS:5582 + 16*type) and the
// affect handlers that test for it.

// classBits are the base record's +0xD class bits in the DS:0D38 order
// (the DS:05D6 masks).
var classBits = [8]byte{0x02, 0x10, 0x08, 0x40, 0x80, 0x01, 0x04, 0x20}
