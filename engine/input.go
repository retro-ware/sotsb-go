package engine

// keyWaiting = R811:05DB as Turbo Pascal's Crt has it: true while the
// scan code of an extended key is still to be read, else the BIOS check.
// keyWaiting = Turbo Pascal KeyPressed: a pending scan code counts.
func (e *Engine) keyWaiting() bool { return e.pendingScan != 0 || e.KeyPressed() }

// crtReadKey = R811:05ED, Turbo Pascal's ReadKey: an extended key (zero
// ASCII) reads as 0, then its scan code on the next call. The convenience
// keys (F1, gamepad, mouse, extras.go and nav.go) are translated by
// Engine.ReadKey and never reach the game as such.
func (e *Engine) crtReadKey() byte {
	if e.pendingScan != 0 {
		c := e.pendingScan
		e.pendingScan = 0
		return c
	}
	k := e.ReadKey()
	if k&0xFF == 0 {
		e.pendingScan = byte(k >> 8)
		return 0
	}
	return byte(k)
}

// ReadKeyByte = R670:0112: one key as the game reads it. Ctrl-S toggles the
// sound; with the STING parameter Ctrl-D toggles the debug display and
// Ctrl-C quits. After a non-zero key, keys typed ahead are drained and the
// last one wins.
func (e *Engine) ReadKeyByte() byte {
	k := e.crtReadKey()
	if k == 0x13 {
		e.soundToggle()
	}
	if e.Cfg.Param == "STING" {
		if k == 4 {
			e.v49F7 ^= 1
		}
		if k == 3 {
			e.Exit() // R670:0000
		}
	}
	if k != 0 {
		e.draining = true
		for e.keyWaiting() {
			k = e.crtReadKey()
		}
		e.draining = false
	}
	return k
}

// DisplaySpaces = R540:0312 (x, y, colour, count): count blank cells.
func (e *Engine) DisplaySpaces(x, y, colour, count int) {
	if e.Trace != nil {
		e.Trace("R540:0312", "DisplaySpaces", x, y, colour, count)
	}
	if x > 0x27 || y > 0x18 {
		return
	}
	e.displayCharN(x, y, colour, count, ' ', true)
}

// ClearRow24 = O16B:0D08 redraw_screen: blank the message/menu row.
func (e *Engine) ClearRow24() { e.DisplaySpaces(0, 24, 0, 40) }

// InputString = R540:0748 (prompt, colour, maxlen): line editor on row 24.
// Letters and signs ('!'..'z') are taken while len < maxlen, a space only
// after the first character (and past maxlen); each is echoed in white one
// cell right of the last; backspace erases; extended keys are ignored;
// Enter or Esc ends. The result loses its trailing spaces and is upcased.
func (e *Engine) InputString(prompt string, colour, maxlen int) string {
	if len(prompt) > 40 {
		prompt = prompt[:40]
	}
	e.ClearRow24()
	e.DisplayString(0, 24, colour, prompt)
	e.ext.typing = true // hint line and mouse (nav.go)
	defer func() { e.ext.typing = false }()
	x := len(prompt)
	var s []byte
	for {
		k := e.ReadKeyByte()
		switch {
		case k == ' ':
			if len(s) > 0 {
				s = append(s, k)
				x++
				e.DisplayString(x, 24, colourWhite, " ")
			}
		case k > ' ' && k <= 'z':
			if len(s) < maxlen {
				s = append(s, k)
				x++
				e.DisplayString(x, 24, colourWhite, string(k))
			}
		case k == keyBackspace:
			if len(s) > 0 {
				s = s[:len(s)-1]
				e.DisplaySpaces(x, 24, 0, 1)
				x--
			}
		case k == 0:
			k = e.ReadKeyByte() // the extended key's scan code
		}
		if k == keyEnter || k == keyEsc || e.demo {
			break
		}
	}
	e.ClearRow24()
	for len(s) > 0 && s[len(s)-1] == ' ' {
		s = s[:len(s)-1]
	}
	for i := range s {
		s[i] = upper(s[i])
	}
	return string(s)
}
