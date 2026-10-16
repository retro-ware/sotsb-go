package engine

import "sotsb/dax"

// Pictures shown in the view window, ported from ovr29: animated PIC /
// SPRIT / FINAL blocks and the HEAD+BODY portraits.

// animPic = the DS:6ABE record: up to 0x42 bytes holding a frame count and
// per frame a delay and a picture pointer.
type animPic struct {
	name   string // DS:6B0A cache key
	block  int    // DS:6B1C (-1 = nothing loaded)
	frames []dax.SpriteFrame
	cur    int   // DS:6ABF frame shown while waiting for input (1-based)
	last   int64 // O16B:03C9 [bp-87]: tick of the last frame change in the current DisplayInput
}

// portrait = DS:6B00..6B09: the cached HEAD and BODY pictures.
type portrait struct {
	headID, bodyID int // DS:6B00 / 6B05 (-1 = none)
	head, body     *dax.Image
}

// LoadAnimPic = O110:0103 (name, block, masked): load <name><gfx>.DAX
// block into the view-window picture, unless it is already there. PIC and
// FINAL frames after the first are stored XORed against the first; the
// first PIC block is cut to 4 frames. Sprites are masked on colour 0.
func (e *Engine) LoadAnimPic(name string, block int, masked bool) {
	a := &e.pic
	if a.name == name && a.block == block {
		return
	}
	e.FreeAnimPic()
	a.name, a.block = name, block
	b := e.daxBlock(name+e.symDAX[4:], block)
	if b == nil {
		return
	}
	xor := name == "PIC" || name == "FINAL"
	fr := dax.ParseSprite(b, masked, 0, xor)
	if name == "PIC" && block == 1 && len(fr) > 4 {
		fr = fr[:4]
	}
	if masked && (e.Cfg.Graphics == 'E' || e.Cfg.Graphics == 'T') { // DS:4A1E != 0
		for _, f := range fr { // R2BC:1418(DS:0C6C, DS:0C7C, 0, 0): colour 13 -> opaque black
			e.recolourFrame(maskFrom[:], maskTo[:], 0, false, f.Image)
		}
	}
	a.frames = fr
	a.cur = 1
	e.ClearKeyboard()
}

var maskFrom = [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15} // DS:0C6C
var maskTo = [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 0, 14, 15}    // DS:0C7C

// FreeAnimPic = O110:0556.
func (e *Engine) FreeAnimPic() {
	e.pic = animPic{block: -1}
}

// drawPicAt = O110:000A (x, y, overlay, pic): an overlay picture goes
// through R2BC:0BB4 at page cell (x-1,y-1), which is screen cell (x,y)
// (the page starts one cell in); otherwise R2BC:0EFE.
func (e *Engine) drawPicAt(x, y int, overlay bool, im *dax.Image) {
	if im == nil {
		return
	}
	if overlay {
		e.drawPage(x-1, y-1, im, 0, true, false)
		e.Flip() // 0034
	} else {
		e.DrawPicture(x, y, im, 0)
	}
}

// DrawAnimFrame = O110:074D (pic, frame): frame 1..3 of a sprite at its
// own cell offset inside the view window.
func (e *Engine) DrawAnimFrame(frame int) {
	if frame < 1 || frame > 3 {
		e.MessageRow24b(0xF, "Illegal range in Show3DSprite.")
		e.Exit()
		return
	}
	if frame > len(e.pic.frames) {
		return
	}
	im := e.pic.frames[frame-1].Image
	e.drawPage(im.X+2, im.Y+2, im, 0, true, false) // R2BC:0BB4 at (x+3-1, y+3-1)
	e.Flip()
}

// LoadPortrait = O110:05D8 (head, body): HEAD<gfx>/BODY<gfx> blocks, kept
// while the ids do not change; 0xFF leaves that half alone.
func (e *Engine) LoadPortrait(head, body int) {
	p := &e.port
	d := e.symDAX[4:]
	if head != 0xFF && (p.headID == 0xFF || p.headID != head) {
		p.head = e.LoadPic("HEAD"+d, head)
		p.headID = head
	}
	if body != 0xFF && (p.bodyID == 0xFF || p.bodyID != body) {
		p.body = e.LoadPic("BODY"+d, body)
		p.bodyID = body
	}
	e.ClearKeyboard()
}

// DrawPortrait = O110:06DC (x, y, both): the head at (x,y) and, when both
// is set, the body five rows below (nested drawers O110:008D / O110:00AD).
func (e *Engine) DrawPortrait(x, y int, both bool) {
	e.drawPicAt(x, y, false, e.port.head)
	if both {
		e.drawPicAt(x, y+5, false, e.port.body)
	}
}
