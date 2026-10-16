package engine

// Pool of Radiance's interpreter globals still read by the por-derived
// combat, camp and treasure code (to go as those units are ported).

// eclRun holds the ovr04 globals that are not part of the ovr08 interpreter.
type eclRun struct {
	restoreSel  bool       // DS:4430: put DS:4436 back as the selected character
	savedSel    *Character // DS:4436 (a pointer: survives the list changing)
	scriptLoad  byte       // DS:4431
	needFrame   byte       // DS:4432: main screen not drawn yet
	walldefSet  byte       // DS:4433
	geoSet      byte       // DS:4434
	monCount    byte       // DS:4435: monsters appended to the party list
	areaChanged bool       // DS:442F
	quit        bool       // DS:49FF
	searchSave  byte       // DS:84DD
	f84DE       byte
	f84E4       byte
	f84E7       byte       // monsters were loaded
	sprite      int        // DS:6F81
	closeup     int        // DS:6F82
	doorTry     [3]byte    // DS:6DDE..6DE0: bash, pick, knock still allowed
	effTick     [0x49]byte // DS:471D: member i has timed effects to run down
	itemType    byte       // TREASURE: last random item type
	f84DF       byte
}

// cut: s at most n characters.
func cut(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

type eclState struct {
	argType                                                       [64]byte   // DS:7005
	argLo                                                         [64]byte   // DS:7085
	argHi                                                         [64]byte   // DS:7045
	str                                                           [64]string // DS:6FC8 + 0x100*k (k = 1..)
	retStack                                                      []int      // DS:6F70 chain
	flags                                                         [6]bool    // DS:6F78..6F7D: eq ne lt gt le ge
	op                                                            byte       // DS:6F7F
	prevOp                                                        byte       // DS:6F7E
	v6F68                                                         byte       // DS:6F68 (init 0x41)
	v6F69                                                         byte       // DS:6F69 (init 9)
	v6F6A                                                         uint16     // read-only specials 0xFB/0xFC/0xB1
	v6F6C                                                         uint16
	v6F6E                                                         uint16
	stop                                                          bool   // DS:442E
	events                                                        [5]int // DS:49E3,49E5,49E7,49E9,49EB
	wallAhead                                                     byte   // DS:6AB0
	cell                                                          byte   // DS:6AB1
	f84C8                                                         [2]byte
	f84CC, f84CE, f84D0                                           uint16
	f84D2, f84D3                                                  byte
	f84E0, f84E1, f84E2, f84E3, f84E5, f84E6, f84E8, f84E9, f84EA byte
	f70C6                                                         byte
	f49FD, f49FE                                                  byte
}
