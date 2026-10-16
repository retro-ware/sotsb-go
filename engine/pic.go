package engine

import "sotsb/dax"

// symFix = DS:2682 symbol_set_fix: the first symbol number of each of the
// five 8x8 symbol sets; symLast (DS 9868) the last.
var (
	symFix  = [5]int{1, 46, 116, 186, 256}
	symLast = [5]int{45, 115, 185, 255, 295}
)

// LoadSymbolSet = Pool ovr35:003B — load 8X8D<n>.DAX block into set index,
// masked on colour 13.
func (e *Engine) LoadSymbolSet(set, block int) {
	if set < 0 || set > 4 {
		return
	}
	e.sym[set] = e.LoadPicMasked(e.symDAX, block, 0xD)
	if e.sym[set] == nil {
		e.MessageRow24b(0xF, "Unable to load 8x8D"+e.symDAX[4:]+"-"+itoa(block))
		e.Exit()
	}
}

// LoadPicMasked = R2BC:0118 with the mask flag set.
func (e *Engine) LoadPicMasked(name string, id int, mask byte) *dax.Image {
	if e.Trace != nil {
		e.Trace("R2BC:0118", "LoadPic", name, id, 1, int(mask), "")
	}
	b := e.daxBlock(name, id)
	if b == nil {
		return nil
	}
	return dax.ParseImage(b, true, mask)
}

// LoadPic = R2BC:0118 (no masking).
func (e *Engine) LoadPic(name string, id int) *dax.Image {
	if e.Trace != nil {
		e.Trace("R2BC:0118", "LoadPic", name, id, 0, 0, "")
	}
	b := e.daxBlock(name, id)
	if b == nil {
		return nil
	}
	return dax.ParseImage(b, false, 0)
}

// DrawPicture = R2BC:0EFE draw_picture(x, y, pic): blit frame at cell (x,y).
func (e *Engine) DrawPicture(x, y int, im *dax.Image, frame int) {
	if e.Trace != nil {
		e.Trace("R2BC:0EFE", "DrawPicture", x, y, picID(im, frame))
	}
	if im == nil || frame >= im.Count {
		return
	}
	d := im.Frame(frame)
	w := im.Width * 8
	e.S.Lock()
	defer e.S.Unlock()
	for r := 0; r < im.Height; r++ {
		for c := 0; c < w; c++ {
			e.S.Set(x*8+c, y*8+r, d[r*w+c])
		}
	}
}

// Put8x8Symbol = Pool ovr35:015D Put8x8Symbol(x, y, id) without overlay: copy
// the symbol's frame from its set and draw it at cell (x,y).
func (e *Engine) Put8x8Symbol(x, y, id int) { e.put8x8(x, y, id, false, 0) }

// Put8x8SymbolMasked = Pool ovr35:015D with the overlay flag (R2BC:08F7): the
// set's mask colour is left transparent.
func (e *Engine) Put8x8SymbolMasked(x, y, id int) { e.put8x8(x, y, id, true, 1) }

// Put8x8SymbolPage = Pool ovr35:015D with the page flag and mode 0: drawn
// opaque into the drawing page (the area map).
func (e *Engine) Put8x8SymbolPage(x, y, id int) { e.put8x8(x, y, id, true, 0) }

func (e *Engine) put8x8(x, y, id int, masked bool, mode int) {
	set := -1
	for i := range symFix { // O19C:0171
		if id >= symFix[i] && id <= symLast[i] {
			set = i
		}
	}
	if set < 0 || e.sym[set] == nil {
		return
	}
	if masked {
		e.DrawPictureMasked(x, y, e.sym[set], id-symFix[set], mode)
	} else {
		e.DrawPicture(x, y, e.sym[set], id-symFix[set])
	}
}

// DrawPictureMasked = R2BC:08F7 (x, y, frame, mode): like DrawPicture;
// with mode bit 0 and a masked picture the mask pixels are not drawn.
func (e *Engine) DrawPictureMasked(x, y int, im *dax.Image, frame, mode int) {
	if e.Trace != nil {
		px, py := x, y
		if e.pageDraw {
			px, py = x-1, y-1
		}
		e.Trace("R2BC:08F7", "DrawPictureMasked", px, py, frame, mode, picID(im, 0), "")
	}
	masked := mode&1 != 0
	if im == nil || frame >= im.Count {
		return
	}
	d := im.Frame(frame)
	w := im.Width * 8
	e.S.Lock()
	defer e.S.Unlock()
	for r := 0; r < im.Height; r++ {
		for c := 0; c < w; c++ {
			if p := d[r*w+c]; !masked || int(p) != im.Mask {
				e.S.Set(x*8+c, y*8+r, p)
				if e.pageDraw {
					e.pageSet(x*8+c-8, y*8+r-8, p)
				}
			}
		}
	}
}
