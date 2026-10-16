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

// FreeAnimPic = O178:05ED.
func (e *Engine) FreeAnimPic() {
	e.pic = animPic{block: -1}
}

// drawPicAt = O178:0000 (x, y, overlay, pic): an overlay picture goes
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

// DrawAnimFrame = O178:07E3 (pic, frame): frame 1..3 of a sprite at its
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

// DrawPortrait = O178:0770 (x, y, both): the head at (x,y) and, when both
// is set, the body five rows below.
func (e *Engine) DrawPortrait(x, y int, both bool) {
	e.drawPicAt(x, y, false, e.port.head)
	if both {
		e.drawPicAt(x, y+5, false, e.port.body)
	}
}
