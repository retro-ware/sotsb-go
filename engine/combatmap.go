package engine

// Combat map services (O11C/O121 and the START sprite blitter) that the
// rules call. They are placeholders until the combat overlays are ported;
// outside combat none of them is reached.

// nearEntry is one 3-byte record of the DS:6718 list built by O11C:08D8.
type nearEntry struct{ index, dist, dir byte }

type nearState struct{ list []nearEntry } // DS:6716..671A

// Flip = R226:12E9: show the drawing page. The port draws straight to the
// shown screen (Hold/Release frame the 3D view), so there is nothing to do.
func (e *Engine) Flip() { e.traceFlip() }

// combatant = DS:65B9[i]: the combatant behind a near list index.
func (e *Engine) combatant(i byte) *Character {
	if int(i) < 1 || int(i) > len(e.combatants) {
		return nil
	}
	return e.combatants[i-1]
}

// sign = O11C:0005.
func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	}
	return 0
}

// line is the O11C line walker record (0x18 bytes): O11C:01A5 starts it,
// O11C:024C takes one step. err (+8) counts in half units of the major
// axis; steps (+22) counts half squares, three for a diagonal.
type line struct {
	x1, y1, err, dx, dy, x, y, sx, sy, steps int
	dir                                      byte // +23
}

// newLine = O11C:01A5.
func newLine(x0, y0, x1, y1 int) *line {
	return &line{x1: x1, y1: y1, dx: abs(x1 - x0), dy: abs(y1 - y0), sx: sign(x1 - x0), sy: sign(y1 - y0), x: x0, y: y0}
}

// step = O11C:024C: one square along the major axis, and along the minor
// one when the error reaches the major distance; dir is the direction
// taken (8 = none).
func (e *Engine) step(l *line) bool {
	mx, my, moved := 0, 0, false
	if l.dx >= l.dy {
		if l.x != l.x1 {
			l.x += l.sx
			mx = l.sx
			l.err += 2 * l.dy
			l.steps += 2
			if l.err >= l.dx {
				l.err -= 2 * l.dx
				l.y += l.sy
				my = l.sy
				l.steps++
			}
			moved = true
		}
	} else if l.y != l.y1 {
		l.y += l.sy
		my = l.sy
		l.err += 2 * l.dx
		l.steps += 2
		if l.err >= l.dy {
			l.err -= 2 * l.dy
			l.x += l.sx
			mx = l.sx
			l.steps++
		}
		moved = true
	}
	l.dir = lineStepDirs[my+1][mx+1]
	return moved
}

// flyMissile = O0ED:1AA4 (x1, y1, x2, y2, frames, delay): the missile
// sprite (DS:6E36, loaded by hitColours) flies from square to square in
// thirds of a square, cycling its frames and scrolling the map along.
func (e *Engine) flyMissile(x1, y1, x2, y2, frames, delay int) {
	var path []byte
	l := newLine(x1*3, y1*3, x2*3, y2*3)
	for {
		moved := e.step(l)
		path = append(path, l.dir)
		if !moved {
			break
		}
	}
	n := len(path)
	if n < 2 {
		return
	}
	path = append(path, 8) // the rest of the 0x94 byte buffer
	ddx, ddy := stepBytes(stepX), stepBytes(stepY)
	dX, dY := x2-x1, y2-y1
	var done bool
	var cx, cy int
	if e.combatInView(x1-e.cmap.viewX, y1-e.cmap.viewY) && e.combatInView(x2-e.cmap.viewX, y2-e.cmap.viewY) {
		done, cx, cy = true, e.cmap.viewX+viewW/2, e.cmap.viewY+3
	} else if abs(dX) <= viewW-1 && abs(dY) <= 6 {
		done, cx, cy = true, x1+dX/2, y1+dY/2
	} else {
		cx, cy = e.cmap.viewX+viewW/2, e.cmap.viewY+3
	}
	e.combatCentre(cx, cy, 0xFF, 8)
	i, f, ox, oy := 0, 0, 0, 0
	for {
		px := (x1-e.cmap.viewX)*3 + ox
		py := (y1-e.cmap.viewY)*3 + oy
		sx, sy := x1, y1
		out := false
		for {
			if delay > 0 || px%3 == 0 || py%3 == 0 {
				e.drawHitFrame(f, px, py)
				e.Flip()
				e.Delay(delay)
				e.restoreHitFrame(px, py)
				if f++; f >= frames {
					f = 0
				}
			}
			i++
			d := path[i]
			mx, my := int(int8(ddx[d])), int(int8(ddy[d]))
			px += mx
			py += my
			out = px < 0 || px > viewW*3-3 || py < 0 || py > 0x12
			if !out && i < n {
				ox += mx
				oy += my
				if abs(ox) == 3 {
					x1 += sign(ox)
					cx += sign(ox)
					ox = 0
				}
				if abs(oy) == 3 {
					y1 += sign(oy)
					cy += sign(oy)
					oy = 0
				}
			}
			if i >= n || out {
				break
			}
		}
		if i < n {
			if sign(x1-sx) != 0 {
				if x1 >= e.cmap.viewX+viewW-1 {
					cx = x1 + 3
				} else if x1 <= e.cmap.viewX {
					cx = x1 - 3
				}
			}
			if sign(y1-sy) != 0 {
				if y1 >= e.cmap.viewY+6 {
					cy = y1 + 3
				} else if y1 <= e.cmap.viewY {
					cy = y1 - 3
				}
			}
			e.combatCentre(cx, cy, 0xFF, 8)
		} else {
			done = true
			if !e.combatInView(x2-e.cmap.viewX, y2-e.cmap.viewY) {
				e.combatCentre(x2, y2, 3, 8)
			}
			px, py := (x2-e.cmap.viewX)*3, (y2-e.cmap.viewY)*3
			e.drawHitFrame(f, px, py)
			if delay > 0 {
				e.Flip()
				e.Delay(delay)
				e.restoreHitFrame(px, py)
			}
		}
		if done {
			break
		}
	}
	e.Flip()
}

// isParty = O0ED:2564 (char): true for the party side.
func isParty(c *Character) bool { return c.Side == 0 }

// nearEnemies = O0ED:25E0 (char, range): combatants of the other side
// within range; the near list keeps only them, copied to DS:6DE4.
func (e *Engine) nearEnemies(c *Character, r int) int {
	e.nearCombatants(c, e.combatX(c), e.combatY(c), r)
	kept := e.near.list[:0]
	var want byte // the original keeps the targets whose side byte equals isParty(c)
	if isParty(c) {
		want = 1
	}
	for _, n := range e.near.list {
		if t := e.combatant(n.index); t != nil && t.Side == want {
			kept = append(kept, n)
		}
	}
	e.near.list = kept
	for i, n := range kept {
		e.targets[i+1] = n.index
	}
	return len(kept)
}

// rangeTo = O0ED:2708 (char, target): distance in squares (list distance
// halved) from char to target, the near list left as it was.
func (e *Engine) rangeTo(c, t *Character) int {
	saved := e.near
	e.near.list = append([]nearEntry(nil), saved.list...)
	e.cmap.seeAll = 1 // [DS:6716]+6: line of sight ignores walls
	e.nearCombatants(c, e.combatX(c), e.combatY(c), 0xFF)
	e.cmap.seeAll = 0
	var d byte
	for i, n := range e.near.list {
		if e.combatant(n.index) == t || i == len(e.near.list)-1 {
			d = n.dist
			break
		}
	}
	// only the 0xD8 entry bytes come back, the count (DS:671A) stays
	list := make([]nearEntry, len(e.near.list))
	copy(list, saved.list)
	e.near.list = list
	return int(d >> 1)
}

// Spell classes beyond spellids.go's: a monster's innate power.
const spellClassInnate = 3

// Levels at which paladins and rangers cast: their level less these.
const (
	paladinClericOffset = 8
	rangerDruidOffset   = 7
	rangerMageOffset    = 8
	innateCasterLevel   = 12
	itemCasterLevel     = 6
	paladinFirstCasting = 9
	rangerFirstCasting  = 8
)

// casterLevel = O0ED:286F (spell): the level the selected character casts
// the spell at. Paladins cast cleric spells, rangers druid and mage ones,
// from a few levels on; who casts none (and items, DS:757D) cast at 6.
func (e *Engine) casterLevel(spell int) int {
	c := e.selected()
	level := func(k int) int8 {
		l := int8(c.Levels[k])
		if oldClassUsable(c) {
			l += int8(c.OldLevels[k])
		}
		return l
	}
	if c.Levels[clsCleric] == 0 && c.Levels[clsMage] == 0 &&
		int8(c.Levels[clsPaladin]) < paladinFirstCasting && int8(c.Levels[clsRanger]) < rangerFirstCasting {
		return itemCasterLevel
	}
	class := spellTable[spell&127].class()
	var lvl int8
	switch class {
	case spellClassCleric:
		lvl = max(level(clsCleric), level(clsPaladin)-paladinClericOffset)
	case spellClassDruid:
		lvl = max(level(clsRanger)-rangerDruidOffset, 0)
	case spellClassMage:
		lvl = max(level(clsMage), level(clsRanger)-rangerMageOffset)
	case spellClassInnate:
		lvl = innateCasterLevel
	}
	if e.usingItem && class != spellClassInnate {
		return itemCasterLevel
	}
	return int(byte(lvl))
}
