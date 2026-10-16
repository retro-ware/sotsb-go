package engine

// Combat setup (ovr11): the battle map, the combat blocks and where every
// combatant starts.

// combatSetup = O05D:battle_begins.
func (e *Engine) combatSetup() {
	e.FreeAnimPic()
	e.port = portrait{headID: 0xFF, bodyID: 0xFF}
	e.ClearRow24()
	e.gameDelay()
	e.DisplayString(0, 0x18, 0xA, "A battle begins...")
	e.v6E30, e.combatRound, e.rv.hitRoll = 0, 0, 0
	e.roundLimit = combatRoundsWithoutAttack
	e.nbodies = 0
	e.stinkingClouds, e.poisonClouds = nil, nil
	e.thrownItem = nil
	e.bodies = [10]body{}
	putWord(e.game, 0x666, 0)
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
		e.affectGroup(c, grpAlwaysInvisible)
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

// combatMapSetup = O05D:1030: the tile sets and the map.
func (e *Engine) combatMapSetup() {
	if wordAt(e.area, 2*areaWordFreeSteps) != 0 {
		e.LoadBackpacIcons("DungCom", 1, 0, 25)
	} else {
		e.LoadBackpacIcons("WildCom", 1, 0, 33)
	}
	e.LoadBackpacIcons("RandCom", 1, 34, 6)
	e.cmap = &combatMap{cursorSize: 1}
	if wordAt(e.area, 2*areaWordFreeSteps) != 0 {
		e.dungeonMap()
	} else {
		e.wildMap()
	}
}

// combatBlocks = O05D:10E0: a fresh combat block for every combatant;
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
	}
}

// --- the dungeon map (O05D:08CD) -------------------------------------

// cellSide = O05D:0306 (x, y, dir): 1 wall, 0 open, 3 door. Off the
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

// cellEdge = O05D:0388 (x, y, dir): the side seen from both cells.
func (e *Engine) cellEdge(x, y, dir int) byte {
	dx, dy := int(stepX[dir]), int(stepY[dir])
	return e.cellSide(x, y, dir) | e.cellSide(x+dx, y+dy, (dir+4)%8)
}

// dungeonMap = O05D:08CD: every cell within 6 east-west and 2 north-south
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

// dungeonRubble = O05D:00D3 (for the cell at column c, row r with the
// edges n, e, s, w): in a cell flagged for it that is neither a corridor
// nor has a door, heaps of rubble on the floor.
func (e *Engine) dungeonRubble(c, r int, n, ea, s, w byte, flagged bool) {
	room := (n == 1 || ea == 1 || s == 1 || w == 1) &&
		(n != 1 || s != 1 || ea == 1 && w == 1) &&
		(ea != 1 || w != 1 || n == 1 && s == 1) &&
		n != 3 && ea != 3 && s != 3 && w != 3
	inMap := func(x, y int) bool { return x >= 0 && x < wildMapW && y >= 0 && y < wildMapH }
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

// --- the wilderness map (O05D:0FC8) -----------------------------------

// Wilderness terrain flags (wildTerrain, by area byte 0x342).
const (
	terrainSparse   = 2   // fewer trees and rocks
	terrainThin     = 4   // a few fewer
	terrainForest   = 8   // more trees
	terrainHills    = 16  // a cliff now and then
	terrainMountain = 32  // a cliff more often
	terrainRough    = 64  // more of everything
	terrainBare     = 128 // no trees, much less of anything
)

// Combat ground values of the wilderness.
const (
	groundGrass        = 23 // the whole map to start with
	tileOpenGround     = 22 // groundInfo tile of open ground: things may be put there
	groundCliffLeft    = 59 // +1d2
	groundCliffRight   = 61 // +1d2
	groundCliffFoot    = 64
	groundCliffFoot2   = 65
	groundTreeTop      = 31 // +1d5, above groundTreeTrunk
	groundTreeTrunk    = 36 // +1d5
	groundShrub        = 41 // +1d2
	groundBoulder      = 57 // +1d2
	groundBush         = 47 // +1d2
	groundRocks        = 43 // +1d4
	groundStones       = 54 // +1d3
	groundFlowers      = 49 // +1d4
	areaByteTerrain    = 0x342
	wildMapW, wildMapH = 50, 25
)

// wildTerrain = O05D:0991: the terrain flags of the area (DS:0354 by DS:4A2C).
// Past the table the original reads the bytes after it; the port gives 0.
func (e *Engine) wildTerrain() byte {
	if int(e.terrain) >= len(wildTerrainFlags) {
		return 0
	}
	return wildTerrainFlags[e.terrain]
}

// wildMap = O05D:0FC8: grass everywhere, then perhaps a cliff, trees and
// the small things.
func (e *Engine) wildMap() {
	for y := range e.cmap.ground {
		for x := range e.cmap.ground[y] {
			e.cmap.ground[y][x] = groundGrass
		}
	}
	e.terrain = e.area[areaByteTerrain] // DS:4A2C
	e.wildCliff()
	e.wildTrees()
	e.wildScatter()
}

func (e *Engine) openGround(x, y int) bool {
	return e.groundInfo(e.cmap.ground[y][x])[3] == tileOpenGround
}

// wildCliff = O05D:0A00: in hills and mountains, now and then a cliff runs
// down the map, leaning right, with a fallen block at its foot here and
// there.
func (e *Engine) wildCliff() {
	chance := 0
	if e.wildTerrain()&terrainMountain != 0 {
		chance = 35
	}
	if e.wildTerrain()&terrainHills != 0 {
		chance = 75
	}
	if e.Dice(1, 100) > chance {
		return
	}
	x := 34 - e.Dice(5, 4)
	for (x+2)%7 > 0 {
		x--
	}
	for y := 0; y < wildMapH; y++ {
		if x > wildMapW-1 {
			continue
		}
		e.cmap.ground[y][x] = byte(e.Dice(1, 2) + groundCliffLeft)
		if x < wildMapW-1 {
			e.cmap.ground[y][x+1] = byte(e.Dice(1, 2) + groundCliffRight)
		}
		if e.Dice(1, 20) == 1 { // O05D:09AC
			if x < wildMapW-1 {
				e.cmap.ground[y][x+1] = groundCliffFoot
			}
			if y < wildMapH-1 && x < wildMapW-1 {
				e.cmap.ground[y+1][x+1] = groundCliffFoot2
			}
		}
		x++
	}
}

// wildTrees = O05D:0B0B: trees (two squares high) and shrubs on open
// ground, more in forests.
func (e *Engine) wildTrees() {
	t := e.wildTerrain()
	if t&terrainBare != 0 {
		return
	}
	chance := int8(10)
	if t&terrainSparse != 0 {
		chance -= 5
	}
	if t&terrainThin != 0 {
		chance -= 2
	}
	if t&terrainRough != 0 {
		chance += 5
	}
	if t&terrainForest != 0 {
		chance += 10
	}
	if chance < 0 {
		chance = 1
	}
	for x := 0; x < wildMapW; x++ {
		for y := 1; y < wildMapH; y++ {
			if !e.openGround(x, y) || !e.openGround(x, y-1) {
				continue
			}
			if int(chance) < e.Dice(1, 100) {
				continue
			}
			if int(chance) >= e.Dice(1, 100) {
				e.cmap.ground[y][x] = byte(e.Dice(1, 2) + groundShrub)
			} else {
				e.cmap.ground[y-1][x] = byte(e.Dice(1, 5) + groundTreeTop)
				e.cmap.ground[y][x] = byte(e.Dice(1, 5) + groundTreeTrunk)
			}
		}
	}
}

// scatterOdds are the percent chances of O05D:0CA2: a boulder, a bush,
// rocks, stones, flowers; the rest stays open.
type scatterOdds struct{ boulder, bush, rocks, stones, flowers int }

// wildScatter = O05D:0E4A: the small things on open ground, by how rough
// the terrain is.
func (e *Engine) wildScatter() {
	t := e.wildTerrain()
	rough := 50
	if t&terrainHills != 0 {
		rough += 10
	}
	if t&terrainMountain != 0 {
		rough += 30
	}
	if t&terrainRough != 0 {
		rough += 20
	}
	if t&terrainThin != 0 {
		rough -= 10
	}
	if t&terrainSparse != 0 {
		rough -= 20
	}
	if t&terrainBare != 0 {
		rough -= 50
	}
	var odds scatterOdds
	switch {
	case rough >= -30 && rough <= 9:
		odds = scatterOdds{0, 0, 0, 30, 15}
	case rough >= 10 && rough <= 29:
		odds = scatterOdds{0, 1, 5, 20, 10}
	case rough >= 30 && rough <= 69:
		odds = scatterOdds{0, 2, 5, 10, 5}
	case rough >= 70 && rough <= 89:
		odds = scatterOdds{10, 2, 10, 10, 1}
	case rough >= 90 && rough <= 110:
		odds = scatterOdds{15, 5, 15, 10, 1}
	default:
		return
	}
	for x := 0; x < wildMapW; x++ {
		for y := 0; y < wildMapH; y++ {
			if !e.openGround(x, y) {
				continue
			}
			r := e.Dice(1, 100)
			g := &e.cmap.ground[y][x]
			switch {
			case r <= odds.boulder:
				*g = byte(e.Dice(1, 2) + groundBoulder)
			case r <= odds.boulder+odds.bush:
				*g = byte(e.Dice(1, 2) + groundBush)
			case r <= odds.boulder+odds.bush+odds.rocks:
				*g = byte(e.Dice(1, 4) + groundRocks)
			case r <= odds.boulder+odds.bush+odds.rocks+odds.stones:
				*g = byte(e.Dice(1, 3) + groundStones)
			case r <= odds.boulder+odds.bush+odds.rocks+odds.stones+odds.flowers:
				*g = byte(e.Dice(1, 4) + groundFlowers)
			}
		}
	}
}

// --- placement (O05D:17FE) -------------------------------------------

// placement is the DS:4440..4658 state of O05D:17FE: per side the
// formation templates (4 teams of 6 rows by 11), origin, size and facing.
type placement struct {
	slots  [2][4][6][11]byte // DS:4440
	ox, oy [2]int8           // DS:4650/4652
	rows   [2]byte           // DS:4654
	dir    [2]byte           // DS:4656
	side   byte              // DS:4658
}

// placeCombatants = O05D:17FE.
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
				if !e.duelling && c.Combat.Extra == 0 {
					x, y := e.cpos[idx].x, e.cpos[idx].y
					e.nbodies++
					e.bodies[e.nbodies] = body{c: c, x: x, y: y, ground: e.cmap.ground[y][x]}
					e.cmap.ground[y][x] = 0x1F
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

// formationEdge = O05D:1202 (x, y): outside the 11x6 formation both ways.
func formationEdge(x, y int) bool {
	return (x < 0 || x >= 0xB) && (y < 0 || y >= 6)
}

// placeAt = O05D:1233 (idx, x, y, ox, oy, team): put combatant idx on
// formation slot (x,y) of the team block at (ox,oy) if the ground lets it.
func (e *Engine) placeAt(idx, x, y, ox, oy, team int) bool {
	p := &e.place
	if x < 0 || y < 0 || x > 0xA || y > 5 || p.slots[p.side][team][y][x] == 0 {
		return false
	}
	e.cpos[idx].x = byte(x + ox*6 + oy*5 + 0x16)
	e.cpos[idx].y = byte(oy*5 + y + 0xA)
	who, g, _, _ := e.groundAt(e.combatants[idx-1], 8)
	if who == 0 && g > 0 && e.groundInfo(g)[0] < 0xFF {
		p.slots[p.side][team][y][x] = 0
		return true
	}
	return false
}

// placeOne = O05D:1380 (idx): walk the formation of the combatant's side,
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
