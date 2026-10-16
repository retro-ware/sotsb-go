package engine

// The lingering clouds: Stinking Cloud (DS:7575, four squares) and
// Cloudkill (DS:7579, nine). Each is a list of 30-byte records.

// cloudRecord is one cloud: the caster, the ground each square had, which
// squares it covers, the centre and the caster's number for it.
type cloudRecord struct {
	caster  *Character // +0
	ground  [9]byte    // +8
	covered [9]byte    // +17
	x, y    byte       // +26
	number  byte       // +28 the caster's cloud number (affect data >> 4)
	stays   bool       // +29 the cloud withstood a Dispel Magic
}

// cloudKind ties a list to its squares, its ground and its affect.
type cloudKind struct {
	list    *[]*cloudRecord
	squares []byte
	ground  byte
	affect  byte
	message string
}

func (e *Engine) stinkingKind() cloudKind {
	return cloudKind{&e.stinkingClouds, cloudSquares4[:], groundStinkingCloud, affectStinkingCloud, "Creates a noxious cloud"}
}

func (e *Engine) poisonKind() cloudKind {
	return cloudKind{&e.poisonClouds, cloudSquares9[:], groundPoisonCloud, affectCloudkill, "Creates a poisonous cloud"}
}

// square is the map square of a cloud's k-th square (0-based).
func (r *cloudRecord) square(squares []byte, k int) (int, int) {
	d := squares[k]
	return int(int8(r.x)) + int(stepX[d]), int(int8(r.y)) + int(stepY[d])
}

// cloudOwner = O0E3:0D03 (x, y, list, squares): the caster of the first
// cloud covering the square. Default: DS:6524, the acting combatant, which
// in combat is the selected one.
func (e *Engine) cloudOwner(x, y int, list []*cloudRecord, squares []byte) *Character {
	for _, r := range list {
		for k := range squares {
			if r.covered[k] == 0 {
				continue
			}
			if sx, sy := r.square(squares, k); sx == x && sy == y {
				return r.caster
			}
		}
	}
	return e.selected()
}

// withSelected runs f with c as the selected character.
func (e *Engine) withSelected(c *Character, f func()) {
	keep := e.partySel
	for i, p := range e.party {
		if p == c {
			e.partySel = i
		}
	}
	f()
	e.partySel = keep
}

// groundUnderBody: the ground a body at x, y lies on, else g.
func (e *Engine) groundUnderBody(x, y int, g byte) byte {
	for b := 1; b <= e.nbodies; b++ {
		if int(int8(e.bodies[b].x)) == x && int(int8(e.bodies[b].y)) == y {
			g = e.bodies[b].ground
		}
	}
	return g
}

// olderGround looks through a list for another cloud covering x, y and
// gives the ground it saved there, skipping saved cloud grounds. stopFirst
// is Cloudkill's search, which takes the first found; Stinking Cloud's
// takes the last.
func olderGround(list []*cloudRecord, squares []byte, self *cloudRecord, x, y int, g byte, stopFirst bool) byte {
	for _, o := range list {
		if o == self {
			continue
		}
		for j := range squares {
			if o.covered[j] == 0 {
				continue
			}
			ox, oy := o.square(squares, j)
			if ox != x || oy != y {
				continue
			}
			og := o.ground[j]
			if og == groundStinkingCloud || stopFirst && og == groundPoisonCloud {
				continue
			}
			g = og
			if stopFirst {
				return g
			}
		}
	}
	return g
}

// createCloud = O0C2 create_noxious_cloud / spell_poisonous_cloud: the
// cloud is laid over the target square; who stands under it breathes it.
func (e *Engine) createCloud(k cloudKind) {
	e.rv.areaSpell = 1
	sel := e.selected()
	lvl := e.casterLevel(int(e.curSpell))
	n := 0
	list := *k.list
	// the last cloud of the list is not counted
	for i := 0; i < len(list)-1; i++ {
		if list[i].caster == sel {
			n++
		}
	}
	r := &cloudRecord{}
	*k.list = append(list, r)
	e.AddAffect(sel, k.affect, lvl, byte(n<<4+lvl), 1)
	r.caster = sel
	r.number = byte(n)
	r.x = byte(e.tgtX)
	r.y = byte(e.tgtY)
	who := make([]byte, len(k.squares))
	for i := range k.squares {
		x, y := r.square(k.squares, i)
		w, g := e.mapAt(x, y)
		who[i] = w
		if g > 0 && e.groundInfo(g)[0] < 255 {
			r.covered[i] = 1
		}
		switch {
		case g == groundStinkingCloud:
			g = olderGround(e.stinkingClouds, cloudSquares4[:], r, x, y, g, k.ground == groundPoisonCloud)
		case g == groundPoisonCloud && k.ground == groundPoisonCloud:
			g = olderGround(e.poisonClouds, cloudSquares9[:], r, x, y, g, true)
		case g == groundBody:
			g = e.groundUnderBody(x, y, g)
		}
		r.ground[i] = g
		if r.covered[i] != 0 {
			e.cmap.ground[y][x] = k.ground
		}
	}
	e.combatName(sel, k.message, combatMsgRow, false)
	e.combatCentre(e.tgtX, e.tgtY, 255, 8)
	e.gameDelay()
	e.clearCombatText()
	if k.ground == groundStinkingCloud {
		// one breath for each one under it
		for i := 1; i < len(who); i++ {
			for j := 0; j < len(who); j++ {
				if who[j] == who[i] && i != j {
					who[i] = 0
				}
			}
		}
	}
	for _, w := range who {
		if w > 0 {
			e.inPoisonCloud(e.combatant(w), true)
		}
	}
}

// clearCloud = O064:0C1D / O064:1AB9 (char, affect): the caster's cloud of
// that number lifts; the squares get back their ground (or a body's), and
// the remaining clouds are laid again.
func (e *Engine) clearCloud(k cloudKind, c *Character, a *Affect) {
	number := a.Data >> 4
	var r *cloudRecord
	at := -1
	for i, o := range *k.list {
		if o.caster == c && o.number == number {
			r, at = o, i
			break
		}
	}
	if r == nil {
		return
	}
	e.printMessage("The air clears a little...")
	for i := range k.squares {
		if r.covered[i] == 0 {
			continue
		}
		x, y := r.square(k.squares, i)
		body := false
		for b := 1; b <= e.nbodies; b++ {
			if e.bodies[b].c != nil && int(int8(e.bodies[b].x)) == x && int(int8(e.bodies[b].y)) == y {
				body = true
			}
		}
		if body {
			e.cmap.ground[y][x] = groundBody
		} else {
			e.cmap.ground[y][x] = r.ground[i]
		}
	}
	*k.list = append((*k.list)[:at], (*k.list)[at+1:]...)
	for _, o := range *k.list {
		for i := range k.squares {
			if o.covered[i] != 0 {
				x, y := o.square(k.squares, i)
				e.cmap.ground[y][x] = k.ground
			}
		}
	}
}

// stinkingCloudEndHandler = O064:0C1D.
func (e *Engine) stinkingCloudEndHandler(c *Character, a *Affect, flag byte) {
	e.clearCloud(e.stinkingKind(), c, a)
}

// cloudkillEndHandler = O064:1AB9.
func (e *Engine) cloudkillEndHandler(c *Character, a *Affect, flag byte) {
	e.clearCloud(e.poisonKind(), c, a)
}
