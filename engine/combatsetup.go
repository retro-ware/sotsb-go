package engine

// Combat setup (unit O097): the battle map, the combat blocks and where
// every combatant starts.

// combatSetup = O097:1564 battle_begins.
func (e *Engine) combatSetup() {
	e.FreeAnimPic()
	e.port = portrait{headID: noPicture, bodyID: noPicture}
	e.ClearRow24()
	e.DisplayString(0, statusRow, colourGreen, "A battle begins...")
	e.computerMagic, e.combatRound, e.rv.hitRoll = false, 0, 0
	e.roundLimit = combatRoundsWithoutAttack
	e.nbodies = 0
	e.stinkingClouds, e.poisonClouds, e.delayedBlasts = nil, nil, nil
	e.thrownItem = nil
	e.bodies = [10]body{}
	putWord(e.game, 2*gameWordAttackedAlly, 0)
	e.combatMapSetup()
	e.flushKey()
	e.combatBlocks()
	e.flushKey()
	e.placeCombatants()
	e.flushKey()
	e.spr = nil // init_dax_block (DS:75FA, 1, 4, 3, 0x18): iconToSprite sizes it
	if len(e.party) > 0 {
		e.cmap.viewX = e.combatX(e.party[0]) - 3
		e.cmap.viewY = e.combatY(e.party[0]) - 3
	}
	e.combatUseHook()
	for _, c := range e.party {
		e.affectGroup(c, grpRound)
	}
	e.monsterHPPercent()
	e.gameState = stateCombat
}

// flushKey = Pool 0899:0122: a waiting key is read and dropped.
func (e *Engine) flushKey() {
	if e.keyWaiting() {
		e.ReadKeyByte()
	}
}

// combatMapSetup = O097:094B: the tile set and the map. Silver Blades
// only fights in dungeons; outdoors (O097:0939, empty) the map stays bare.
func (e *Engine) combatMapSetup() {
	inside := wordAt(e.area, 2*areaWordFreeSteps) != 0
	if inside {
		e.LoadBackpacIcons("DungCom", 1, 0, combatIconCount)
	}
	e.cmap = &combatMap{cursorSize: 1}
	if inside {
		e.dungeonMap()
	}
}

// combatIconCount: the DungCom tiles.
const combatIconCount = 32

// combatBlocks = O097:09BF: a fresh combat block for every combatant;
// those past the party count are not party members, and such a one on the
// party's side without a sane control byte is run by the party's choice
// (game word 1420).
func (e *Engine) combatBlocks() {
	for i, c := range e.party {
		e.ApplyItems(c) // O0ED:0C20
		c.Combat = &CombatInfo{}
		if i+1 > int(wordAt(e.game, gsPartyCount)) {
			c.Combat.Extra = 1
		}
		c.Combat.Facing = combatFacings[e.partyFacing()/2]
		if c.Side == 1 {
			c.Combat.Facing = (c.Combat.Facing + 4) % 8
		}
		m := c.Control & 0x7F
		if c.Side == sideParty && c.Combat.Extra == 1 && (m == 0 || m > 0x66) {
			c.Control = byte(wordAt(e.game, 0x58C) + 0x80)
		}
		c.Combat.HasSpells = 1
	}
}

// --- the dungeon map (O097:0875) -------------------------------------

// cellSide = O097:0330 (x, y, dir): 1 wall, 0 open, 3 door. Off the
// 16x16 map only the party's row is open east and west.
func (e *Engine) cellSide(x, y, dir int) byte {
	if x >= 0 && x <= 0xF && y >= 0 && y <= 0xF {
		if e.DoorAt(x, y, dir) == 0 {
			return 1
		}
		if e.WallAt(x, y, dir) == 0 {
			return 0
		}
		return 3
	}
	if y == e.partyY() && (dir == 2 || dir == 6) {
		return 0
	}
	return 1
}

// cellEdge = O097:0330 (x, y, dir): the side seen from both cells.
func (e *Engine) cellEdge(x, y, dir int) byte {
	dx, dy := int(stepX[dir]), int(stepY[dir])
	return e.cellSide(x, y, dir) | e.cellSide(x+dx, y+dy, (dir+4)%8)
}

// dungeonMap = O097:0875: every cell within 6 east-west and 2 north-south
// of the party becomes a 6x5 block of tiles, rows skewed 5 squares.
func (e *Engine) dungeonMap() {
	for r := -2; r <= 2; r++ {
		for c := -6; c <= 6; c++ {
			x, y := e.partyX()+c, e.partyY()+r
			w, n, ea := e.cellEdge(x, y, 6), e.cellEdge(x, y, 0), e.cellEdge(x, y, 2)
			put := func(col, row int, t byte) { // Pool ovr11:0221
				px, py := col+c*6+r*5+0x15, row+r*5+0xA
				if px >= 0 && px <= 0x31 && py >= 0 && py <= 0x18 {
					e.cmap.ground[py][px] = t + 1
				}
			}
			// nested O05D:03FC: floor, west wall or door
			for row := 2; row <= 4; row++ {
				for col := 0; col <= 5; col++ {
					put(col, row, 0x16)
				}
			}
			if w == 1 {
				for i := 2; i <= 4; i++ {
					put(i-1, i, 4)
					put(i, i, 3)
					put(i+1, i, 0xD)
				}
			} else if w == 3 {
				put(1, 2, 8)
				put(5, 4, 0)
			}
			// nested Pool ovr11:0377: north wall
			if n == 1 {
				put(3, 0, 5)
				put(4, 0, 5)
				put(3, 1, 0xA)
				put(4, 1, 0xA)
			} else {
				put(3, 0, 0x16)
				put(4, 0, 0x16)
				put(3, 1, 0x16)
				put(4, 1, 0x16)
			}
			// nested O05D:051E: the north-west corner
			nw := e.cellEdge(x, y-1, 6) == 0 && e.cellEdge(x-1, y, 0) == 0
			pick := func(ok bool, a, b byte) byte {
				if ok {
					return a
				}
				return b
			}
			var a, b, cc, d byte
			if n == 0 {
				switch w {
				case 0:
					a = 0x16
				case 3:
					a = 0xD
				case 1:
					a = pick(nw, 0, 0xD)
				}
			} else if w == 0 {
				a = pick(nw, 0xF, 5)
			} else {
				a = pick(nw, 0x12, 2)
			}
			b = map[byte]byte{0: 0x16, 3: 0x11, 1: 5}[n]
			switch w {
			case 0:
				cc = pick(n == 0, 0x16, pick(nw, 0x10, 0xA))
			case 3:
				cc = pick(nw, 0x14, 7)
			case 1:
				cc = pick(nw, 1, 3)
			}
			if w == 0 || w == 3 {
				d = map[byte]byte{0: 0x16, 3: 0x17, 1: 0xA}[n]
			} else {
				d = map[byte]byte{0: 0xD, 3: 0x15, 1: 6}[n]
			}
			put(1, 0, a)
			put(2, 0, b)
			put(1, 1, cc)
			put(2, 1, d)
			// nested O05D:06F6: the north-east corner
			ne7, ne8 := e.cellEdge(x, y-1, 2), e.cellEdge(x+1, y, 0)
			ne := ne7 == 0 && ne8 == 0
			a = map[byte]byte{0: pick(ne7 == 1, 4, 0x16), 3: 0xF, 1: 5}[n]
			if n == 0 {
				switch ne7 {
				case 0:
					b = 0x16
				case 3:
					b = pick(ea == 0 && ne8 != 0, 0x18, 1)
				case 1:
					switch {
					case ea != 0:
						b = 3
					case ne8 != 0:
						b = 0xB
					default:
						b = 7
					}
				}
			} else {
				switch {
				case ea != 0:
					b = 9
				case ne8 != 0:
					b = 5
				default:
					b = pick(ne, 0x11, 0x13)
				}
			}
			cc = map[byte]byte{0: 0x16, 3: 0x10, 1: 0xA}[n]
			if n == 0 {
				switch {
				case ne7 == 0:
					d = 0x16
				case ea != 0:
					d = 4
				default:
					d = pick(ne8 == 0, 8, 0xC)
				}
			} else {
				switch {
				case ea != 0:
					d = 0xE
				default:
					d = pick(ne8 == 0, 0x17, 0xA)
				}
			}
			put(5, 0, a)
			put(6, 0, b)
			put(5, 1, cc)
			put(6, 1, d)
			rubble := e.MapCell(x, y)&cellRubble != 0
			e.dungeonRubble(c, r, n, ea, e.cellEdge(x, y, 4), w, rubble)
		}
	}
}

// Rubble on the dungeon floor.
const (
	cellRubble       = 0x40 // the map cell's flag for rubble in the room
	groundRubble     = 26
	groundRubbleEdge = 27
)

// rubbleSpread = DS:034C / DS:0350: the four squares round a rubble heap.
var rubbleSpread = [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}

// dungeonRubble = O097:0084 (for the cell at column c, row r with the
// edges n, e, s, w): in a cell flagged for it that is neither a corridor
// nor has a door, heaps of rubble on the floor.
func (e *Engine) dungeonRubble(c, r int, n, ea, s, w byte, flagged bool) {
	room := (n == 1 || ea == 1 || s == 1 || w == 1) &&
		(n != 1 || s != 1 || ea == 1 && w == 1) &&
		(ea != 1 || w != 1 || n == 1 && s == 1) &&
		n != 3 && ea != 3 && s != 3 && w != 3
	inMap := func(x, y int) bool { return x >= 0 && x < combatMapW && y >= 0 && y < combatMapH }
	for i := 2; i <= 3; i++ {
		for j := 2; j <= 4; j++ {
			x := c*6 + r*5 + 21 + i + j // sic: the row is added to the column too
			y := r*5 + 10 + j
			if !inMap(x, y) || !e.openGround(x, y) || !flagged || !room || e.Dice(1, 10) > 5 {
				continue
			}
			e.cmap.ground[y][x] = groundRubble
			for _, d := range rubbleSpread {
				nx, ny := x+d[0], y+d[1]
				if inMap(nx, ny) && e.openGround(nx, ny) && e.Dice(1, 10) <= 9 {
					e.cmap.ground[ny][nx] = groundRubbleEdge
				}
			}
		}
	}
}

// tileOpenGround: the groundInfo tile of open ground (things may be put there).
const tileOpenGround = 22

// combatMapW, combatMapH: the battle map's size in squares.
const combatMapW, combatMapH = 50, 25

func (e *Engine) openGround(x, y int) bool {
	return e.groundInfo(e.cmap.ground[y][x])[3] == tileOpenGround
}

// --- placement (O097:10DC) -------------------------------------------

// placement is the DS:4440..4658 state of O097:10DC: per side the
// formation templates (4 teams of 6 rows by 11), origin, size and facing.
type placement struct {
	slots  [2][4][6][11]byte // DS:4440
	ox, oy [2]int8           // DS:4650/4652
	rows   [2]byte           // DS:4654
	dir    [2]byte           // DS:4656
	side   byte              // DS:4658
}

// placeCombatants = O097:10DC.
func (e *Engine) placeCombatants() {
	e.countSides()
	e.cpos = [0x49]combatPos{}
	e.combatants = make([]*Character, 0x48)
	e.ncomb = 0
	e.combatRedrawMap()
	p := &e.place
	p.ox[0], p.oy[0] = 0, 0
	p.dir[0] = byte(e.partyFacing() / 2)
	dist := int(wordAt(e.game, 0x582))
	p.ox[1] = int8(dist*int(stepX[e.partyFacing()]) + int(p.ox[0]))
	p.oy[1] = int8(dist*int(stepY[e.partyFacing()]) + int(p.oy[0]))
	p.dir[1] = byte((e.partyFacing() + 4) % 8 / 2)
	for s := 0; s <= 1; s++ {
		p.rows[s] = byte((int(e.rv.side[s]) + 1) >> 1)
	}
	for s := 0; s <= 1; s++ {
		for team := 0; team <= 3; team++ {
			d := int(p.dir[s])
			if team == 1 {
				d = 4
			}
			for y := 0; y <= 5; y++ {
				for x := 0; x <= 0xA; x++ {
					v := byte(0)
					if int(formationRows[d][y*2]) <= x && int(formationRows[d][y*2+1]) >= x {
						v = 1
					}
					p.slots[s][team][y][x] = v
				}
			}
		}
	}
	idx := 1
	e.ncomb = 1
	e.cpos[0].size = 1 // original quirk: the absolute DS:5F2A is record 0's size, counted up per combatant
	for i := 0; i < len(e.party); i++ {
		c := e.party[i]
		e.flushKey()
		e.combatants[idx-1] = c
		p.side = c.Side
		e.cpos[idx].id = byte(idx)
		e.cpos[idx].size = c.Size & 7
		if e.placeOne(idx) {
			if c.Okay == 0 {
				e.cpos[idx].size = 0
				if c.Combat.Extra == 0 {
					x, y := e.cpos[idx].x, e.cpos[idx].y
					e.nbodies++
					e.bodies[e.nbodies] = body{c: c, x: x, y: y, ground: e.cmap.ground[y][x]}
					if c.Status == statusStoned {
						e.cmap.ground[y][x] = groundStoneBody
					} else {
						e.cmap.ground[y][x] = groundBody
					}
				}
			}
			e.combatRedrawMap()
			idx++
			e.cpos[0].size++
			e.ncomb++
			continue
		}
		e.cpos[idx].size = 0
		if c.Combat.Extra == 1 { // no room: the extra leaves
			e.combatants[idx-1] = nil
			e.partySel = i
			e.freePlayers(true, false)
			i--
			continue
		}
		idx++
		e.cpos[0].size++
		e.ncomb++
	}
}

// formationEdge = O097:0AEB (x, y): outside the 11x6 formation both ways.
func formationEdge(x, y int) bool {
	return (x < 0 || x >= 0xB) && (y < 0 || y >= 6)
}

// placeAt = O097:0B1C (idx, x, y, ox, oy, team): put combatant idx on
// formation slot (x,y) of the team block at (ox,oy) if the ground lets it.
func (e *Engine) placeAt(idx, x, y, ox, oy, team int) bool {
	p := &e.place
	if x < 0 || y < 0 || x > 0xA || y > 5 || p.slots[p.side][team][y][x] == 0 {
		return false
	}
	e.cpos[idx].x = byte(x + ox*6 + oy*5 + 0x16)
	e.cpos[idx].y = byte(oy*5 + y + 0xA)
	who, g, _, _ := e.groundAt(e.combatants[idx-1], dirHere)
	if who == 0 && g > 0 && e.groundInfo(g)[0] < 0xFF {
		p.slots[p.side][team][y][x] = 0
		return true
	}
	return false
}

// placeOne = O097:0C60 (idx): walk the formation of the combatant's side,
// row by row outwards from the middle, falling back on the other teams
// (blocks beside the first) until a free slot takes it.
func (e *Engine) placeOne(idx int) bool {
	p := &e.place
	s := p.side
	first, fail, state := true, false, 1
	var row, team, k, n int
	var bx, by, x, y int
	var dx, dy int
	ox, oy := int(p.ox[s]), int(p.oy[s])
	for {
		dir := int(teamDirs[p.dir[s]][team] >> 1)
		turn := func(q int) {
			d := int(combatFacings[(dir+q)%4])
			dx, dy = int(stepX[d]), int(stepY[d])
		}
		switch state {
		case 1:
			turn(2)
			t := 0
			if team > 0 {
				t = 1
			}
			bx = int(formationStartX[t][dir]) + dx*row
			by = int(formationStartY[t][dir]) + dy*row
			x, y = bx, by
			k, state, n = 1, 2, 1
		case 2:
			turn(1)
			x, y = bx+dx*k, by+dy*k
			state = 3
			n++
		case 3:
			turn(3)
			x, y = bx+dx*k, by+dy*k
			state = 2
			k++
			n++
		}
		out := x < 0 || y < 0 || x > 0xA || y > 5
		if state > 1 {
			full := out && !formationEdge(x, y)
			if !full {
				if first {
					full = n >= int(p.rows[s])
				} else {
					full = n > 0xB
				}
			}
			if full {
				row++
				if s == 0 && p.dir[0]&1 != 0 && team == 0 && row == 1 {
					cx, cy := e.partyX()+int(p.ox[0]), e.partyY()+int(p.oy[0])
					open := false
					for a := 1; a <= 3; a++ {
						d := int(teamSides[p.dir[s]][a])
						if e.gameState != stateWilderness && e.cellEdge(cx, cy, d) != 1 {
							open = true
						}
					}
					if open {
						row++
					}
				}
				state, first = 1, false
			}
		}
		ok := false
		if out && formationEdge(x, y) {
			state = 0
			for team < 3 && state != 1 {
				team++
				cx, cy := e.partyX()+int(p.ox[s]), e.partyY()+int(p.oy[s])
				d := int(teamSides[p.dir[s]][team])
				if e.gameState != stateWilderness && e.cellEdge(cx, cy, d) == 1 {
					continue
				}
				ox, oy = int(stepX[d])+int(p.ox[s]), int(stepY[d])+int(p.oy[s])
				row, state = 0, 1
			}
			if state != 1 {
				fail = true
			}
		}
		if !out {
			ok = e.placeAt(idx, x, y, ox, oy, team)
		}
		if ok || fail {
			return !fail
		}
	}
}
