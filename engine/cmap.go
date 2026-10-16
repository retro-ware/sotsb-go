package engine

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"

	"sotsb/dax"
)

// The combat map (O11C, O121, O12A). Squares are 3x3 cells; the view
// shows 7x7 squares of the 50x25 map on the drawing page, which appears at
// screen cell (1,1) (R226:12E9 copies it 8 pixels down and right).

// combatMap is the 0x4E9-byte record at [DS:6716].
type combatMap struct {
	viewX, viewY int          // +2/+3 top left square of the view
	cursor       byte         // +4 draw the cursor icon
	cursorSize   byte         // +5 cursor footprint
	seeAll       byte         // +6 line of sight ignores walls
	ground       [25][50]byte // +7 ground type per square
}

// combatPos is the 4-byte DS:5F27 record of combatant i.
type combatPos struct{ x, y, id, size byte }

// body is a 7-byte DS:66D6 record: who fell where, and the ground it hid.
type body struct {
	c      *Character
	x, y   byte
	ground byte
}

const pageCells = 21 // the drawing page, 21x21 cells (outside combat)

// The combat view: viewW x viewH squares of 3x3 cells, the page at cell
// (pageCol, 1), the panel to its right.
const (
	viewW, viewH = 7, 7
	pageCol      = 1
	pageStride   = viewW * 24 // e.page row length in pixels
)

// pageGeom is where the drawing page sits on screen and how wide it is.
func (e *Engine) pageGeom() (x0, w int) {
	if e.gameState == 5 {
		return pageCol * 8, viewW * 24
	}
	return 8, pageCells * 8
}

// footprint = O18B:0000 (size, part, var dx, var dy): offset of one square
// of a combatant that covers several (DS:2844).
func (e *Engine) footprint(size, part int) (int, int, bool) {
	if size == 0 {
		return 0, 0, false
	}
	b := []byte{byte(footprints[size][part*2]), byte(footprints[size][part*2+1])}
	return int(int8(b[0])), int(int8(b[1])), int8(b[0]) >= 0
}

// groundInfo = DS:2744: 4 bytes per ground type: move cost (0xFF wall),
// view height, blocking height, tile.
func (e *Engine) groundInfo(g byte) []byte { return groundInfo[g%64][:] }

// tileOf: the original indexes the 50-wide grid flat, so a square just off a
// row reads its neighbour.
func (e *Engine) tileOf(x, y int) byte {
	i := y*50 + x
	if i < 0 || i >= 25*50 {
		return e.groundInfo(0)[3]
	}
	return e.groundInfo(e.cmap.ground[i/50][i%50])[3]
}

// drawPage draws a frame at page cell (x,y), clipped to the page. masked
// skips the image's mask colour; under (R226:0B94 mode 4) keeps what was
// there in DS:4A24.
func (e *Engine) drawPage(x, y int, im *dax.Image, frame int, masked, under bool) {
	if e.Trace != nil {
		mode := b2i(masked)
		if under {
			mode |= 4
		}
		e.Trace("R226:0B94", "DrawPage", x, y, frame, mode, picID(im, 0), "")
	}
	e.blitPage(x, y, im, frame, masked, under)
}

func (e *Engine) blitPage(x, y int, im *dax.Image, frame int, masked, under bool) {
	if im == nil || frame >= im.Count {
		return
	}
	d := im.Frame(frame)
	w := im.Width * 8
	x0, pw := e.pageGeom()
	e.S.Lock()
	defer e.S.Unlock()
	if under && len(e.sprUnder) != w*im.Height {
		e.sprUnder = make([]byte, w*im.Height)
	}
	for r := 0; r < im.Height; r++ {
		for c := 0; c < w; c++ {
			px, py := x*8+c, y*8+r
			if px < 0 || py < 0 || px >= pw || py >= pageCells*8 {
				continue
			}
			if under { // (off the page the buffer keeps what it had)
				e.sprUnder[r*w+c] = e.page[py*pageStride+px]
			}
			if p := d[r*w+c]; !masked || int(p) != im.Mask {
				e.S.Set(x0+px, 8+py, p)
				e.pageSet(px, py, p)
			}
		}
	}
}

// markSquare (extra) draws a dotted frame just inside screen square
// (sx, sy), into the page too so sprites restore it.
func (e *Engine) markSquare(sx, sy int, c uint8) {
	x0, _ := e.pageGeom()
	e.S.Lock()
	defer e.S.Unlock()
	for i := 2; i <= 21; i += 2 {
		for _, p := range [4][2]int{{i, 2}, {i, 21}, {2, i}, {21, i}} {
			px, py := sx*24+p[0], sy*24+p[1]
			e.S.Set(x0+px, 8+py, c)
			e.pageSet(px, py, c)
		}
	}
}

// restoreUnder = R226:0B94 (DS:4A24, mode 0): put back what the last
// sprite hid.
func (e *Engine) restoreUnder(x, y int, im *dax.Image) {
	if im == nil {
		return
	}
	w := im.Width * 8
	if len(e.sprUnder) != w*im.Height {
		e.sprUnder = make([]byte, w*im.Height)
	}
	if e.Trace != nil {
		s := md5.Sum(e.sprUnder)
		e.Trace("R226:0B94", "DrawPage", x, y, 0, 0, fmt.Sprintf("%dx%d:%s", im.Height, w, hex.EncodeToString(s[:])[:10]), "")
	}
	x0, pw := e.pageGeom()
	e.S.Lock()
	defer e.S.Unlock()
	for r := 0; r < im.Height; r++ {
		for c := 0; c < w; c++ {
			px, py := x*8+c, y*8+r
			if px < 0 || py < 0 || px >= pw || py >= pageCells*8 {
				continue
			}
			e.S.Set(x0+px, 8+py, e.sprUnder[r*w+c])
			e.pageSet(px, py, e.sprUnder[r*w+c])
		}
	}
}

// drawTile = O12A:00F7 (x, y, tile, page): a ground tile from the tile
// sets (DS:5E52, tiles from 0x80 in DS:5E56).
func (e *Engine) drawTile(x, y int, t byte) {
	im, f := e.sqrpaci[0], int(t)
	if t > 0x7F {
		im, f = e.sqrpaci[1], int(t&0x7F)
	}
	// R226:08D7 (page DS:4A20, tiles, mode 0, frame, x, y): the same blit
	// into the page as 0D71 unmasked.
	if e.Trace != nil {
		e.Trace("R226:08D7", "DrawPictureMasked", x, y, f, 0, picID(im, 0), "")
	}
	e.blitPage(x, y, im, f, false, false)
}

// drawIcon = O12A:0504 (x, y, dir, pose, slot): a combat icon on screen
// square (x,y), mirrored when facing directions 4..7.
func (e *Engine) drawIcon(x, y int, dir, pose, slot byte) {
	im := e.icons[slot][pose&1]
	if im == nil {
		return
	}
	if dir > 3 {
		im = im.Clone()
		im.FlipLeftToRight()
	}
	e.drawPage(x*3, y*3, im, 0, true, false)
}

// tileSetLoadLast: Load24x24Set's highest first tile.
const tileSetLoadLast = 48

// LoadBackpacIcons = O12A:002C Load24x24Set (name, block, first, n): a
// DAX block copied into the tile set from tile first on.
func (e *Engine) LoadBackpacIcons(name string, block, first, n int) {
	if first > tileSetLoadLast {
		return // the original stops: "Start range error in Load24x24Set"
	}
	im := e.LoadPic(name, block)
	t := e.sqrpaci[0]
	defer e.ClearKeyboard()
	if im == nil || t == nil {
		return
	}
	for f := 0; f < n && f < im.Count && first+f < t.Count && im.BPP == t.BPP; f++ {
		copy(t.Frame(first+f), im.Frame(f))
	}
}

// recolour = R226:15F3 with DS:0CD2 → to: map the colours of frame 0 (the
// original takes a frame number, always passed 0, so only that frame changes).
func (e *Engine) recolour(im *dax.Image, tt [16]byte) {
	if im == nil || im.Count == 0 {
		return
	}
	from := recolourFrom
	d := im.Frame(0)
	for i, p := range d {
		for k := 0; k < 16; k++ {
			if from[k] == p {
				d[i] = tt[k]
				break
			}
		}
	}
}

// LoadIconPic = O12A:01D0 (name, block, slot): the two poses of a combat
// icon, block and block+0x80, loaded masked on colour 0.
func (e *Engine) LoadIconPic(name string, block, slot int) {
	up := strings.ToUpper(name)
	load := func(n string, b int) *dax.Image { return e.LoadPicMasked(n, b, 0) }
	ic := &e.icons[slot]
	defer e.ClearKeyboard()
	switch {
	case strings.HasPrefix(up, "CHEAD") || strings.HasPrefix(up, "CBODY"):
		if up[len(up)-1] == 'T' {
			block += 0x40
		}
		n := name[:len(name)-1]
		ic[0], ic[1] = load(n, block), load(n, block+0x80)
	case up == "COMSPR" || up == "ICON":
		ic[0], ic[1] = load(name, block), load(name, block+0x80)
		if up == "ICON" {
			e.recolour(ic[0], recolourIcon)
			e.recolour(ic[1], recolourIcon)
		}
	default:
		n := name + e.symDAX[4:] // DS:5C08 game area digit
		ic[0], ic[1] = load(n, block), load(n, block+0x80)
		e.recolour(ic[0], recolourCombat)
		e.recolour(ic[1], recolourCombat)
	}
}

// freeIcon = O12A:0156 (slot).
func (e *Engine) freeIcon(n byte) { e.icons[n] = [2]*dax.Image{} }

// iconToSprite = Pool 00BA:1892 (slot, pose, frame, flip): copy an icon into a
// frame of the DS:6E36 sprite.
func (e *Engine) iconToSprite(slot, pose, frame int, flip bool) {
	im := e.icons[slot][pose]
	if im == nil {
		return
	}
	if e.spr == nil || e.spr.BPP != im.BPP {
		e.spr = dax.NewImage(4, im.Width, im.Height)
	}
	if flip {
		im = im.Clone()
		im.FlipLeftToRight()
	}
	e.spr.Mask = im.Mask
	copy(e.spr.Frame(frame), im.Frame(0))
}

// hitColours = O151:1A11 (slot): the four frames of the sprite from an icon.
func (e *Engine) hitColours(slot int) {
	e.iconToSprite(slot, 0, 0, false)
	e.iconToSprite(slot, 0, 1, true)
	e.iconToSprite(slot, 1, 2, true)
	e.iconToSprite(slot, 1, 3, false)
}

// drawHitFrame / restoreHitFrame = R226:0B94 with DS:6E36 (mode 5) and
// DS:4A24 (mode 0).
func (e *Engine) drawHitFrame(f, x, y int) { e.drawPage(x, y, e.spr, f, true, true) }
func (e *Engine) restoreHitFrame(x, y int) { e.restoreUnder(x, y, e.spr) }

// updateScreenPos = O18B:0068: screen squares of every combatant.
func (e *Engine) updateScreenPos() {
	for i := 1; i <= e.ncomb; i++ {
		e.combatScreenX[i] = int8(int(e.cpos[i].x) - e.cmap.viewX)
		e.combatScreenY[i] = int8(int(e.cpos[i].y) - e.cmap.viewY)
	}
}

// drawCursor = O18B:00F1 (x, y): the ground under the cursor, the cursor
// icon and whoever stands there.
func (e *Engine) drawCursor(x, y int) {
	sx, sy := x-e.cmap.viewX, y-e.cmap.viewY
	for part := 0; part <= 3; part++ {
		dx, dy, ok := e.footprint(int(e.cmap.cursorSize), part)
		if !ok || !combatInView(sx+dx, sy+dy) {
			continue
		}
		e.drawTile((sx+dx)*3, (sy+dy)*3, e.tileOf(x+dx, y+dy))
		if e.cmap.cursor != 0 {
			e.drawIcon(sx+dx, sy+dy, 0, 0, 0x19)
		}
	}
	e.drawOccupant(x, y)
}

func (e *Engine) drawOccupant(x, y int) {
	idx, _ := e.mapAt(x, y)
	if idx == 0 {
		return
	}
	c := e.combatant(idx)
	if c != nil && e.combatOnScreen(c, false) {
		e.drawIcon(int(e.combatScreenX[idx]), int(e.combatScreenY[idx]), c.Combat.Facing, 0, c.Order)
	}
}

// eraseCursor = O18B:029D (x, y).
func (e *Engine) eraseCursor(x, y int) {
	e.combatClearIcon(x-e.cmap.viewX, y-e.cmap.viewY, 0)
	e.drawOccupant(x, y)
}

// combatRedrawMap = O18B:0368: rebuild the occupancy grid (DS:60DB).
func (e *Engine) combatRedrawMap() {
	e.occ = [25][50]byte{}
	for i := 1; i <= e.ncomb; i++ {
		p := e.cpos[i]
		if p.size == 0 {
			continue
		}
		for part := 0; part <= 3; part++ {
			if dx, dy, ok := e.footprint(int(p.size), part); ok {
				x, y := int(p.x)+dx, int(p.y)+dy
				if x >= 0 && x < 50 && y >= 0 && y < 25 {
					e.occ[y][x] = byte(i)
				}
			}
		}
		e.combatScreenX[i] = int8(int(p.x) - e.cmap.viewX)
		e.combatScreenY[i] = int8(int(p.y) - e.cmap.viewY)
	}
}

// mapAt = O18B:0483 (x, y, var who, var ground).
func (e *Engine) mapAt(x, y int) (byte, byte) {
	if x < 0 || x > 0x31 || y < 0 || y > 0x18 {
		return 0, 0
	}
	return e.occ[y][x], e.cmap.ground[y][x]
}

// combatClearIcon = O18B:04F2 (x, y, idx): ground tiles over combatant
// idx (or over the occupant of screen square x,y when idx is 0).
func (e *Engine) combatClearIcon(sx, sy, idx int) {
	x, y := sx+e.cmap.viewX, sy+e.cmap.viewY
	if idx == 0 {
		i, _ := e.mapAt(x, y)
		idx = int(i)
	}
	if idx == 0 {
		if combatInView(sx, sy) {
			e.drawTile(sx*3, sy*3, e.tileOf(x, y))
		}
		return
	}
	sx, sy = int(e.combatScreenX[idx]), int(e.combatScreenY[idx])
	x, y = sx+e.cmap.viewX, sy+e.cmap.viewY
	for part := 0; part <= 3; part++ {
		dx, dy, ok := e.footprint(int(e.cpos[idx].size), part)
		if ok && combatInView(sx+dx, sy+dy) {
			e.drawTile((sx+dx)*3, (sy+dy)*3, e.tileOf(x+dx, y+dy))
		}
	}
}

// combatInView = O18B:06AC (x, y): a screen square inside the view.
func combatInView(x, y int) bool { return x >= 0 && x < viewW && y >= 0 && y < viewH }

func (e *Engine) combatInView(x, y int) bool { return combatInView(x, y) }

// combatOnScreen = O18B:06DD (char, all): any (or with all, every) square
// of the combatant in view.
func (e *Engine) combatOnScreen(c *Character, all bool) bool {
	idx := e.combatIndex(c)
	if e.cpos[idx].size == 0 {
		return false
	}
	r := true
	for part := 0; part <= 3; part++ {
		dx, dy, ok := e.footprint(int(e.cpos[idx].size), part)
		if !ok {
			continue
		}
		if !combatInView(int(e.combatScreenX[idx])+dx, int(e.combatScreenY[idx])+dy) {
			r = false
			if all {
				return r
			}
		} else {
			r = true
			if !all {
				return r
			}
		}
	}
	return r
}

// recentre = O18B:0797 (x, y, margin): scroll the view so (x,y) is at
// least margin squares inside it (0xFF: always centre); redraws the ground.
func (e *Engine) recentre(x, y, r int) bool {
	const half = viewW / 2
	cx, cy := e.cmap.viewX+half, e.cmap.viewY+3
	rr := r
	if r == 0xFF {
		rr = 0
	}
	if r != 0xFF && x >= cx-rr && x <= cx+rr && y >= cy-rr && y <= cy+rr {
		return false
	}
	if x < cx-rr {
		for x < cx && cx > half {
			cx--
		}
	} else if x > cx+rr {
		for x > cx && cx < 0x31-(viewW-1-half) {
			cx++
		}
	}
	if y < cy-rr {
		for y < cy && cy > 3 {
			cy--
		}
	} else if y > cy+rr {
		for y > cy && cy < 0x15 {
			cy++
		}
	}
	e.cmap.viewX, e.cmap.viewY = cx-half, cy-3
	for row := 0; row < viewH; row++ {
		for col := 0; col < viewW; col++ {
			e.drawTile(col*3, row*3, e.tileOf(e.cmap.viewX+col, e.cmap.viewY+row))
		}
	}
	e.updateScreenPos()
	return true
}

// combatCentre = O18B:0959 (x, y, margin, dir): bring the square next to
// (x,y) in direction dir (8 = itself) into view and draw the cursor there.
func (e *Engine) combatCentre(x, y, r, dir int) {
	dd := stepTable()
	nx, ny := x+int(int8(dd[dir])), y+int(int8(dd[9+dir]))
	if e.recentre(nx, ny, r) {
		for i := 1; i <= e.ncomb; i++ {
			c := e.combatant(byte(i))
			if c != nil && c.Okay != 0 && e.cpos[i].size > 0 && e.combatOnScreen(c, false) {
				e.drawIcon(int(e.combatScreenX[i]), int(e.combatScreenY[i]), c.Combat.Facing, 0, c.Order)
			}
		}
	}
	e.eraseCursor(x, y)
	if !combatInView(nx-e.cmap.viewX, ny-e.cmap.viewY) {
		nx = min(max(nx, 0), 0x31)
		ny = min(max(ny, 0), 0x18)
	}
	e.drawCursor(nx, ny)
	e.Flip()
}

// combatTurn = O18B:0AB9 (char, dir, pose, hide): face dir and draw the
// icon in the given pose (not with hide).
func (e *Engine) combatTurn(c *Character, dir, pose byte, hide bool) {
	idx := e.combatIndex(c)
	if !e.combatOnScreen(c, true) && e.showMoves {
		e.combatCentre(e.combatX(c), e.combatY(c), 3, 8)
	}
	if (dir>>2 != c.Combat.Facing>>2 || pose != 0 || hide) && e.showMoves {
		e.combatClearIcon(0, 0, idx)
	}
	c.Combat.Facing = dir
	if !hide && e.combatOnScreen(c, false) && e.showMoves {
		e.drawIcon(int(e.combatScreenX[idx]), int(e.combatScreenY[idx]), dir, pose, c.Order)
		e.Flip()
	}
}

func (e *Engine) combatX(c *Character) int    { return int(e.cpos[e.combatIndex(c)].x) }    // O18B:0BB0
func (e *Engine) combatY(c *Character) int    { return int(e.cpos[e.combatIndex(c)].y) }    // O18B:0BD8
func (e *Engine) combatSize(c *Character) int { return int(e.cpos[e.combatIndex(c)].size) } // O18B:0C00

// combatIndex = O18B:0C28 (char): its index in DS:65B9, 0 if none.
func (e *Engine) combatIndex(c *Character) int {
	for i := 1; i <= e.ncomb; i++ {
		if e.combatant(byte(i)) == c {
			return i
		}
	}
	return 0
}

// groundAt = O18B:0C80 (char, dir, var who, var ground, var stinking,
// var poison): who stands on and the dearest ground of the squares the
// combatant would cover one step in direction dir (8: where it stands);
// 0 = off the map. A cloud on any of them sets its flag instead.
func (e *Engine) groundAt(c *Character, dir int) (who, gr byte, stinking, poison bool) {
	gr = groundPlain
	best := byte(1)
	idx := e.combatIndex(c)
	for part := 0; part <= 3; part++ {
		dx, dy, ok := e.footprint(int(e.cpos[idx].size), part)
		if !ok {
			continue
		}
		o, g := e.mapAt(int(int8(e.cpos[idx].x))+dx+int(stepX[dir]), int(int8(e.cpos[idx].y))+dy+int(stepY[dir]))
		if int(o) == idx {
			o = 0
		}
		if o > 0 {
			who = o
		}
		switch {
		case g == 0 || gr == 0:
			gr = 0
		case g == groundStinkingCloud:
			stinking = true
		case g == groundPoisonCloud:
			poison = true
		case e.groundInfo(g)[0] >= best:
			best = e.groundInfo(g)[0]
			gr = g
		}
	}
	return
}

// combatRemoveFromMap = O18B:0DEB (char): a fallen combatant leaves the
// map: it flickers out, and unless summoned a body stays where it fell
// (a stone one for the petrified).
func (e *Engine) combatRemoveFromMap(c *Character) {
	if e.gameState != 5 {
		e.PlaySound(sndFall)
		e.gameDelay()
		return
	}
	if int(e.v70C7) < len(e.bodies) && e.bodies[e.v70C7].c == c {
		return // the original's search (DS:70C7, not its loop index)
	}
	idx := e.combatIndex(c)
	x, y := e.combatX(c), e.combatY(c)
	if !e.combatOnScreen(c, true) {
		e.combatCentre(x, y, 3, 8)
	}
	e.combatClearIcon(0, 0, idx)
	e.PlaySound(sndFall)
	for frame := 0; frame < deathFrames; frame++ {
		for part := 0; part <= 3; part++ {
			dx, dy, ok := e.footprint(int(e.cpos[idx].size), part)
			sx, sy := int(e.combatScreenX[idx])+dx, int(e.combatScreenY[idx])+dy
			if ok && e.combatInView(sx, sy) {
				e.drawPage(sx*3, sy*3, e.icons[iconDeath][frame%2], 0, true, true) // DS:66B0
			}
		}
		e.Flip()
		e.Delay(deathFrameDelay)
	}
	if c.Combat.Extra == 0 && e.nbodies+1 < len(e.bodies) {
		e.nbodies++
		b := &e.bodies[e.nbodies]
		b.ground = e.cmap.ground[y][x]
		if e.cmap.ground[y][x] != groundStinkingCloud {
			if c.Status == statusStoned {
				e.cmap.ground[y][x] = groundStoneBody
			} else {
				e.cmap.ground[y][x] = groundBody
			}
		}
		b.c, b.x, b.y = c, byte(x), byte(y)
	}
	e.gameDelay()
	e.combatClearIcon(0, 0, idx)
	e.cpos[e.combatIndex(c)].size = 0
	e.combatRedrawMap()
	e.combatCentre(e.cmap.viewX+3, e.cmap.viewY+3, 3, 8)
	ci := c.Combat
	ci.Delay, ci.Moves, ci.Spell, ci.Guarding = 0, 0, 0, 0
}

// The death flicker: icon slot 0x18's two poses, alternating.
const (
	iconDeath       = 0x18
	deathFrames     = 9
	deathFrameDelay = 10 // ms
)

// combatSquareFree = O18B:10FA (char, x, y, uncover): put the combatant on
// (x,y) if every square it needs is free ground; with uncover, its body is
// taken off the map.
func (e *Engine) combatSquareFree(c *Character, x, y int, uncover bool) bool {
	if e.gameState != 5 {
		return true
	}
	idx := e.combatIndex(c)
	p := &e.cpos[idx]
	p.size = c.Size & 0x7F
	p.x, p.y = byte(x), byte(y)
	who, g, _, _ := e.groundAt(c, dirHere)
	if who != 0 || g == 0 || e.groundInfo(g)[0] == 0xFF {
		p.size = 0
		return false
	}
	if uncover && c.Combat.Extra == 0 {
		for k := 1; k <= e.nbodies; k++ {
			if e.bodies[k].c == c {
				if gr := e.bodies[k].ground; gr != groundBody && gr != groundStoneBody {
					g = e.bodies[k].ground
				}
				e.bodies[k] = body{}
			}
		}
		covered := false
		for k := 1; k <= e.nbodies; k++ {
			if e.bodies[k].c != nil && int(e.bodies[k].x) == x && int(e.bodies[k].y) == y {
				covered = true
			}
		}
		if !covered {
			e.cmap.ground[y][x] = g
		}
	}
	e.combatRedrawMap()
	return true
}

// combatRedrawChar = O18B:1307 (char, margin, cursor): centre on the
// combatant, with the cursor icon over all its squares when asked.
func (e *Engine) combatRedrawChar(c *Character, r int, cursor byte) {
	e.cmap.cursor = cursor
	e.cmap.cursorSize = byte(e.combatSize(c))
	if e.showMoves { // O18B:1307 redraws only while moves are shown (b7600)
		e.combatCentre(e.combatX(c), e.combatY(c), r, 8)
	}
	e.cmap.cursor, e.cmap.cursorSize = 0, 1
}

// nearSort = O186:002E: order the near list by distance.
func (e *Engine) nearSort() {
	l := e.near.list
	for i := 0; i < len(l)-1; i++ {
		for j := i + 1; j < len(l); j++ {
			a, b := l[i], l[j]
			if b.dist < a.dist || (b.dist == a.dist && a.dir < b.dir && a.dir%2 <= b.dir%2) {
				l[i], l[j] = b, a
			}
		}
	}
}

// lineOfSight = O186:03E3 (map, var range, var tx, var ty, x, y): walk from
// (x,y) towards (tx,ty) until the target, a wall higher than the line or
// the range (in half squares) stops it.
func (e *Engine) lineOfSight(rng *int, tx, ty *int, x, y int) bool {
	limit := *rng*2 + 1
	a := newLine(x, y, *tx, *ty)
	h := int(e.groundInfo(e.cmap.ground[y][x])[1])
	b := newLine(0, h, max(a.dx, a.dy), h)
	for {
		if (e.cmap.seeAll == 0 && int(e.groundInfo(e.cmap.ground[a.y][a.x])[2]) > b.y) || a.steps&0xFF > limit {
			*tx, *ty = a.x, a.y
			*rng = a.steps & 0xFF
			return false
		}
		e.step(b)
		if !e.step(a) {
			*rng = a.steps & 0xFF
			return true
		}
	}
}

// inArc = O186:0543 (x1, y1, x2, y2, dir): (x2,y2) lies in the quarter
// facing dir from (x1,y1) (dir 8: anywhere).
func (e *Engine) inArc(x1, y1, x2, y2, dir int) bool {
	if x1 < 0 || x1 > 0x31 || y1 < 0 || y1 > 0x18 || x2 < 0 || x2 > 0x31 || y2 < 0 || y2 > 0x18 {
		return false
	}
	if dir == 0xFF {
		dir = 8
	}
	dd := stepTable()
	nx, ny := x1+int(int8(dd[dir])), y1+int(int8(dd[9+dir]))
	if (x1 == x2 && y1 == y2) || (nx == x2 && ny == y2) {
		return true
	}
	switch dir {
	case 0:
		return (x2 >= nx && y2 <= ny+nx-x2) || (x2 <= nx && y2 <= ny+x2-nx)
	case 1:
		return (x2 >= nx && y2 <= ny+nx-x2) || (x2 >= nx+ny-y2 && y2 <= ny)
	case 2:
		return (x2 >= nx+ny-y2 && y2 <= ny) || (x2 >= nx+y2-ny && y2 >= ny)
	case 3:
		return (x2 >= nx+y2-ny && y2 >= ny) || (x2 >= nx && y2 >= ny+x2-nx)
	case 4:
		return (x2 >= nx && y2 >= ny+x2-nx) || (x2 <= nx && y2 >= ny+nx-x2)
	case 5:
		return (x2 <= nx && y2 >= ny+nx-x2) || (x2 <= nx+ny-y2 && y2 >= ny)
	case 6:
		return (x2 <= nx+ny-y2 && y2 >= ny) || (x2 <= nx+y2-ny && y2 <= ny)
	case 7:
		return (x2 <= nx+y2-ny && y2 <= ny) || (x2 <= nx && y2 <= ny+x2-nx)
	}
	return true
}

// buildNear = O186:08D1 (x, y, range, dir, size, map): every combatant on
// the map seen from the squares of a combatant of that size at (x,y),
// within range and the quarter dir (0xFF: all round), nearest first.
func (e *Engine) buildNear(x, y, rng, dir, size int) {
	var sx, sy [4]int
	var sok [4]bool
	for k := 0; k <= 3; k++ {
		dx, dy, ok := e.footprint(size, k)
		sx[k], sy[k], sok[k] = x+dx, y+dy, ok
	}
	e.near.list = e.near.list[:0]
	for i := 1; i <= e.ncomb; i++ {
		p := e.cpos[i]
		if p.size == 0 {
			continue
		}
		var tx, ty [4]int
		var tok [4]bool
		for m := 0; m <= 3; m++ {
			dx, dy, ok := e.footprint(int(p.size), m)
			tx[m], ty[m], tok[m] = int(p.x)+dx, int(p.y)+dy, ok
		}
		found, best, bk, bm := false, 0xFF, 0, 0
		for m := 0; m <= 3; m++ {
			if !tok[m] {
				continue
			}
			for k := 0; k <= 3; k++ {
				if !sok[k] || !e.inArc(sx[k], sy[k], tx[m], ty[m], dir) {
					continue
				}
				ax, ay, r := tx[m], ty[m], rng
				if e.lineOfSight(&r, &ax, &ay, sx[k], sy[k]) {
					found = true
					if r < best {
						best, bk, bm = r, k, m
					}
				}
			}
		}
		if !found {
			continue
		}
		d := 0
		if dir < 8 {
			d = dir
		} else {
			for !e.inArc(sx[bk], sy[bk], tx[bm], ty[bm], d) {
				d++
			}
		}
		e.near.list = append(e.near.list, nearEntry{byte(i), byte(best), byte(d)})
	}
	e.nearSort()
}

// nearCombatants = O18B:0C00 + O186:08D1: everyone the combatant at (x,y)
// sees within range r, all round.
func (e *Engine) nearCombatants(c *Character, x, y, r int) {
	e.buildNear(x, y, r, 0xFF, e.combatSize(c))
}

// BuildCharIcon = O0A1:0A90 (recolour): the character's combat icon: the
// CBODY poses with the CHEAD poses laid over them (Pool ovr18:032F), in icon
// slot +0xBF; with recolour the six icon colour pairs (+0xC1..) replace
// the DS:3CF5 colours. The original works on the selected character.
func (e *Engine) BuildCharIcon(c *Character, recolour bool) {
	size := iconSizeLetters[c.IconSize]
	slot := int(c.Order)
	e.LoadIconPic("CHEAD"+size, int(c.IconHead), iconWorkHead)
	e.LoadIconPic("CBODY"+size, int(c.IconBody), slot)
	for p := 0; p < 2; p++ {
		head, body := e.icons[iconWorkHead][p], e.icons[slot][p]
		if head == nil || body == nil {
			continue
		}
		// data planes are ORed, mask planes ANDed: a transparent pixel is
		// all-zero data, opaque where either picture is
		for i, v := range head.Data {
			if i >= len(body.Data) || int(v) == head.Mask {
				continue
			}
			if int(body.Data[i]) == body.Mask {
				body.Data[i] = v
			} else {
				body.Data[i] |= v
			}
		}
	}
	if recolour {
		var to [16]byte
		for i := range to {
			to[i] = byte(i)
		}
		for i := 1; i <= len(c.IconColour); i++ {
			v := c.IconColour[i-1]
			pos := defaultIconColours[i] // the icon part's colour index
			to[pos] = v & 15
			to[int(pos)+8] = v >> 4
		}
		for p := 0; p < 2; p++ {
			recolourPicture(e.icons[slot][p], to)
		}
	}
	e.freeIcon(iconWorkHead)
	e.ClearKeyboard()
}

// Icon size letters (DS 1303): the file letter of small (S) and large (T) icons.
var iconSizeLetters = [3]string{"", "S", "T"}

// iconWorkHead: the icon slot the head picture is loaded into while building.
const iconWorkHead = 11

// The combat map's size in squares.
const (
	cmapWidth  = 50
	cmapHeight = 25
)
