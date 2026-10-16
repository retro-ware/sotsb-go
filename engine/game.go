package engine

// The adventuring loop: Pool ovr04:31B4/32F8 (game loop and area change),
// ovr15 (command input, movement, doors) and ovr21 (clock).

// normalizeClock = O0B6:0317: carry each unit into the next; a year
// passing ages the party.
func (e *Engine) normalizeClock(t *[7]int) {
	for i := 0; i <= 6; i++ {
		if t[i] < int(clockLimits[i]) {
			continue
		}
		if i == 6 {
			for _, c := range e.party {
				c.Age = c.Age + 1
			}
			continue
		}
		t[i+1]++
		t[i] -= int(clockLimits[i])
	}
}

// AdvanceClock = O0B6:03FA (count, unit): add count units to the area
// clock (area+0x18C, 7 words), then run down timed effects.
func (e *Engine) AdvanceClock(count, unit int) {
	var t [7]int
	for i := range t {
		t[i] = int(wordAt(e.area, 0x18C+2*i))
	}
	for i := 1; i <= count; i++ {
		t[unit] = int(uint16(t[unit] + 1))
		e.normalizeClock(&t)
	}
	for i := range t {
		putWord(e.area, 0x18C+2*i, uint16(t[i]))
	}
	e.tickEffects(count, unit)
}

// tickEffects = O0B6:001E (count, unit): count down the timed entries in
// each member's effect list (+0x7F) in steps of up to 10 rounds; expired
// ones are removed. While camping only members flagged in DS:471D are
// looked at.
func (e *Engine) tickEffects(count, unit int) {
	r := &e.run
	if e.gameState != 2 {
		for i := 1; i < len(r.effTick); i++ { // FillChar(DS:471E, 0x48, 1)
			r.effTick[i] = 1
		}
	} else {
		any := false
		for i := 1; ; i++ { // repeat .. until: index 1 is always looked at
			if i < len(r.effTick) && r.effTick[i] != 0 {
				any = true
			}
			if any || i+1 > int(wordAt(e.game, 0x67C)) {
				break
			}
		}
		if !any {
			return
		}
	}
	w := count
	for u := unit; u > 1; u-- {
		w = int(uint16(w * int(clockLimits[u-1])))
	}
	next := func(c *Character, a *Affect) *Affect {
		for i, x := range c.Affects {
			if x == a && i+1 < len(c.Affects) {
				return c.Affects[i+1]
			}
		}
		return nil
	}
	head := func(c *Character) *Affect {
		if len(c.Affects) == 0 {
			return nil
		}
		return c.Affects[0]
	}
	for w > 0 {
		step := w
		if step > 10 {
			step = 10
		}
		for i, c := range e.party {
			if i+1 >= len(r.effTick) || r.effTick[i+1] == 0 {
				continue
			}
			r.effTick[i+1] = 0
			cur, pred := head(c), head(c) // bp-1B, bp-13
			var last *Affect              // bp-0F: the tail when the walk starts
			for a := cur; a != nil; a = next(c, a) {
				last = a
			}
			for done := false; cur != nil && !done; {
				done = cur == last
				d := int(cur.Time)
				switch {
				case d == 0:
					cur = next(c, cur)
				case step < d:
					cur.Time = uint16(d - step)
					r.effTick[i+1] = 1
					cur = next(c, cur)
				default:
					after := next(c, cur)
					e.RemoveEffect(c, cur) // O0E3:010A
					// Pool ovr21:0203 quirk: when the node before the removed one
					// is the head, the walk restarts at the head.
					if pred == head(c) {
						cur = head(c)
					} else {
						cur = after
					}
				}
				// O0B6:025E: pred = the node before cur (or cur itself).
				pred = head(c)
				for pred != nil && pred != cur && next(c, pred) != cur {
					pred = next(c, pred)
				}
			}
			for ; cur != nil; cur = next(c, cur) { // O0B6:02B3
				if cur.Time > 0 {
					r.effTick[i+1] = 1
				}
			}
		}
		if w > 10 {
			w -= 10
		} else {
			w = 0
		}
	}
}

// viewFacing: the facing the views show.
func (e *Engine) viewFacing() int { return int(e.sc.facing) }
