package engine

import (
	"strconv"

	"sotsb/dax"
)

// The view window's pictures (unit O110): animated PIC / FINAL / SPRIT
// blocks, the HEAD and BODY portraits and the wilderness big pictures.

// Where the view window's pictures go.
const (
	viewPicX       = 3
	viewPicY       = 3
	portraitBodyDy = 5
	noPicture      = 255
	bigPicWild     = 121 // the wilderness map's picture
)

// areaDigit is the game-area digit in the .DAX names (ECL2, PIC2 ...).
func (e *Engine) areaDigit() string { return strconv.Itoa(int(e.gameArea)) }

// drawViewPicture = O110:000A (x, y, overlay, picture): an overlay (or any
// picture while the area word asks for recolouring) is drawn through the
// page one cell up and left; otherwise straight to the screen.
func (e *Engine) drawViewPicture(x, y int, overlay bool, f int) {
	if f < 0 || f >= len(e.pic.frames) {
		return
	}
	im := e.pic.frames[f].Image
	if wordAt(e.area, 2*areaWordRecolour) > 0 || overlay {
		if wordAt(e.area, 2*areaWordRecolour) > 0 {
			e.recolourFrame(recolourFrom[:], recolourCombat[:], 0, true, im)
		}
		e.drawPage(x-1, y-1, im, 0, true, false)
		e.Flip()
		return
	}
	e.DrawPicture(x, y, im, 0)
}

// areaWordRecolour: non-zero recolours the pictures (night).
const areaWordRecolour = 511

// loadViewPicture = O110:0103 load_pic_final (name, block, flags): the
// <name><area>.DAX block into the view picture unless it is already there.
// PIC and FINAL frames after the first are stored XORed against the first.
func (e *Engine) loadViewPicture(name string, block byte, flags byte) {
	a := &e.pic
	if a.name == name && byte(e.mapMode) == block || block == noPicture {
		return
	}
	if e.loadingMessage {
		e.ClearRow24()
		e.displayStringBg(0, statusRow, colourGreen, colourBlack, "Loading...Please Wait")
	}
	e.restoreBackground()
	a.name = name
	e.mapMode = block
	b := e.daxBlock(name+e.areaDigit(), int(block))
	if b == nil {
		e.displayAndWait(colourYellow, colourBlack, "PIC not found")
		return
	}
	xor := name == "PIC" || name == "FINAL"
	a.frames = dax.ParseSprite(b, flags&1 != 0 && e.videoMode != 0, 0, xor)
	a.cur = 1
	e.ClearKeyboard()
	if e.loadingMessage {
		e.ClearRow24()
	}
}

// restoreBackground = O110:0556 (picture): free the view picture.
func (e *Engine) restoreBackground() {
	e.pic = animPic{block: -1}
	e.spriteShown = false
	e.mapMode = noPicture
}

// loadPortrait = O110:05D8 (head, body): HEAD / BODY blocks, kept while
// the ids do not change; 255 leaves that half alone.
func (e *Engine) loadPortrait(head, body byte) {
	p := &e.port
	if head != noPicture && (p.headID == noPicture || p.headID != int(head)) {
		p.head = e.LoadPic("HEAD"+e.areaDigit(), int(head))
		if p.head == nil {
			e.displayAndWait(colourYellow, colourBlack, "head not found")
		}
		p.headID = int(head)
	}
	if body != noPicture && (p.bodyID == noPicture || p.bodyID != int(body)) {
		p.body = e.LoadPic("BODY"+e.areaDigit(), int(body))
		if p.body == nil {
			e.displayAndWait(colourYellow, colourBlack, "head not found")
		}
		p.bodyID = int(body)
	}
	e.ClearKeyboard()
}

// drawPortrait = O110:06DC (x, y, both): the head, and the body five rows
// below it.
func (e *Engine) drawPortrait(x, y int, both bool) {
	if both {
		e.DrawPortrait(x, y, true)
		return
	}
	e.DrawPortrait(x, y, false)
}

// showSprite = O110:074D Show3DSprite (frame): frame 1..3 of the loaded
// sprite at its own offset in the view.
func (e *Engine) showSprite(frame int) { e.DrawAnimFrame(frame) }

// bigPicture = O110:080C bigpic (block): load the wilderness picture.
func (e *Engine) bigPicture(block byte) {
	e.restoreBackground()
	if e.areaKind == block {
		return
	}
	e.bigPic = e.LoadPic("bigpic"+e.areaDigit(), int(block))
	e.areaKind = block
}

// drawWildernessFrame: the screen frame with the rule under the picture
// (R19D:0000 and 00C0; Silver Blades has no single routine for it).
func (e *Engine) drawWildernessFrame() {
	e.drawScreenFrame()
	e.drawRule(viewRuleRow)
}

// drawBigPic draws the big picture at (1,1) (O110:087A's draw_picture).
func (e *Engine) drawBigPic() {
	if e.bigPic != nil {
		e.DrawPicture(1, 1, e.bigPic, 0)
	}
}

// loadPicture = O151:3031 load_pic: put back what belongs in the view
// window for the game state. While the wilderness map shows the party the
// view is left alone.
func (e *Engine) loadPicture() {
	e.viewReset = true
	mapUp := wordAt(e.area, 2*areaWordPartyOnMap) != 0
	switch e.gameState {
	case stateStartMenu:
		e.drawScreenFrame()
	case stateShop, stateVault:
		if e.shopFrame {
			e.drawAdventureFrame()
		}
		e.drawViewPicture(viewPicX, viewPicY, true, 0)
		e.playerSummary(e.selected())
		e.statusLine()
	case stateCamp:
		e.drawAdventureFrame()
		e.loadViewPicture("PIC", campPicture, 0)
		e.playerSummary(e.selected())
		e.statusLine()
	case stateDungeon:
		e.drawAdventureFrame()
		if !mapUp {
			e.drawView()
		}
		e.playerSummary(e.selected())
		e.statusLine()
		e.sc.firstLoad = false
	case stateWilderness:
		if e.mapMode != mapPicture && !mapUp {
			e.drawView()
		}
	case stateAfterCombat:
		e.drawAdventureFrame()
		e.loadViewPicture("PIC", afterCombatPicture, 0)
		e.playerSummary(e.selected())
	}
}

// campPicture: the camp fire.
const (
	campPicture        = 59
	afterCombatPicture = 60
)
