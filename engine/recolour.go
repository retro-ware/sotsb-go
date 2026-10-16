package engine

import (
	"sotsb/dax"
)

// Picture recolouring.

// recolourFrame = R2BC:1418 (from, to, frame, random, image), the EGA
// path (DS:4A1E = 1) on chunky pixels: for each of the 16 colours whose
// from and to differ, pixels that had that colour in the picture as passed
// in become `to` in a copy, which replaces the picture at the end (so one
// call never chains A->B->C). With random set, each 8-pixel cell of a row
// changes only the pixels selected by Random(256) & Random(256), leftmost
// pixel in bit 7. A negative frame means every frame.
func (e *Engine) recolourFrame(from, to []byte, frame int, random bool, im *dax.Image) {
	if im == nil {
		return
	}
	count := 1
	if frame < 0 {
		frame, count = 0, im.Count
	}
	if frame+count > im.Count {
		count = im.Count - frame
	}
	if count <= 0 {
		return
	}
	lo, hi := frame*im.BPP, (frame+count)*im.BPP
	src := append([]byte(nil), im.Data[lo:hi]...)
	dst := im.Data[lo:hi]
	for i := 0; i < 16; i++ {
		if from[i] == to[i] {
			continue
		}
		for cell := 0; cell < count*im.Height*im.Width; cell++ {
			mask := byte(0xFF)
			if random {
				a := byte(e.Random(0x100))
				mask = a & byte(e.Random(0x100))
			}
			for k := 0; k < 8; k++ {
				if mask&(0x80>>k) != 0 && src[cell*8+k] == from[i] {
					dst[cell*8+k] = to[i]
				}
			}
		}
	}
}
