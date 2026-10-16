package engine

// GAME.OVR overlay 14, second half (0x1BF8..0x3C72): choosing targets for
// spells and missiles, the attack animation and a few combat helpers.

// The combatant panel on the right of the combat screen.
const (
	panelLeft, panelTop     = 23, 1
	panelRight, panelBottom = 38, 21
)

// aimTarget is the 7-byte record the targeting routines fill in: who was
// aimed at (nil for an empty square) and the square.
type aimTarget struct {
	c    *Character
	x, y byte
}

// Spell table fields used in combat.
const (
	spellTargetsField  = 6  // low nibble: how targets are chosen; see spellTargets
	spellAffectField   = 10 // the affect the spell lays
	spellCombatField   = 11 // 0: camp only
	spellCastTimeField = 12 // segments; a third of it is rounds of delay
	spellAimField      = 14 // 0: the caster himself
)

// spellData returns field k of the spell's record.
func (e *Engine) spellData(spell, k int) byte { return spellTable[spell&0xFF][k] }

// healTarget = O0C9:1B5D (c, var who): the friend to heal around c: the most
// hurt of the same side on the nine squares round c (c itself only when
// under half its hit points), or the first body lying there when no one
// with fewer than 8 hit points is.
func (e *Engine) healTarget(c *Character, who **Character) bool {
	dxs, dys := stepBytes(stepX), stepBytes(stepY)
	var best *Character
	bestHP, body := byte(0xFF), 0
	for dir := 0; dir <= 8; dir++ {
		x := byte(e.combatX(c)) + dxs[dir]
		y := byte(e.combatY(c)) + dys[dir]
		idx, ground := e.mapAt(int(x), int(y))
		if idx > 0 {
			*who = e.combatant(idx)
			t := *who
			if t != nil && t.Side == c.Side && t.HP < t.MaxHP {
				if t.HP < bestHP || (t == c && t.HP < t.MaxHP>>1) {
					best, bestHP = t, t.HP
				}
			}
		} else if ground == groundBody {
			for b := 1; b <= e.nbodies; b++ {
				bd := e.bodies[b]
				// ponytail: the set [6,7,8] is the dead, stoned and gone
				if bd.c != nil && (bd.c.Status < 6 || bd.c.Status > 8) && bd.x == x && bd.y == y {
					body = b
				}
			}
		}
	}
	if bestHP >= 8 && body != 0 {
		*who = e.bodies[body].c
	} else {
		*who = best
	}
	return *who != nil
}

// Spell target kinds (the low nibble of spellTargetsField).
const (
	spellTargetsCaster    = 0
	spellTargetsHitDice   = 5  // creatures up to a rolled number of hit dice
	spellTargetsAreaFirst = 8  // 8..14: an area round a square, of radius kind&7
	spellTargetsAreaLast  = 14 // ...
	spellTargetsOneOrArea = 15 // one creature, or the area round an empty square
	spellTargetsCountMask = 3  // otherwise: (kind & 3) + 1 creatures
)

// Hit dice budget of the hit dice spells: 2d4, or Faerie Fire's level.
const (
	hitDiceBudgetDice  = 2
	hitDiceBudgetSides = 4
)

// chooseSpellTarget = O0C9:1D7C (spell, silent, allowGround, var t): one
// target for the selected character's spell. With silent 0 the player
// aims; otherwise one is picked for him. The square goes to DS:7573/7574.
func (e *Engine) chooseSpellTarget(spell, silent, allowGround int, t *aimTarget) bool {
	sel := e.selected()
	found := false
	if silent == 0 {
		allowCreature := b2i(spell != spellDimensionDoor)
		e.aimAt(sel, e.spellRange(spell), 0, allowGround, allowCreature, &found, t)
		sel.Combat.Target = t.c
	} else if e.spellData(spell, spellAimField) == 0 {
		t.c = sel
		if spell != spellCureLightWounds || e.healTarget(sel, &t.c) {
			t.x, t.y = byte(e.combatX(t.c)), byte(e.combatY(t.c))
			found = true
		}
	} else if e.pickAttackTarget(sel, e.spellRange(spell), 0, 1) {
		// one try (the loop counter starts at 1)
		ok := true
		tp := sel.Combat.Target
		if e.isHelpless(tp) {
			for _, a := range helplessAffectData[1:] {
				if e.spellData(spell, spellAffectField) == a {
					ok = false
				}
			}
		}
		if ok {
			t.c = tp
			t.x, t.y = byte(e.combatX(t.c)), byte(e.combatY(t.c))
			found = true
		}
	}
	if found {
		e.tgtX, e.tgtY = int(t.x), int(t.y)
	} else {
		*t = aimTarget{}
	}
	return found
}

// hitDiceWeight: what a creature counts against a hit dice spell's budget,
// by its hit dice (or, for Faerie Fire, its size).
func hitDiceWeight(spell int, c *Character) int {
	if spell == spellFaerieFire {
		switch c.Size {
		case 1:
			return 1
		case 2, 3:
			return 2
		case 4:
			return 4
		}
		return 0
	}
	switch c.MaxLevel {
	case 0, 1:
		return 1
	case 2:
		return 2
	case 3:
		return 4
	}
	return 8
}

// chooseSpellTargets = O0C9:1FA1 (spell, silent): the targets of the spell into
// DS:744B+4i (count DS:744E). False when none was chosen.
func (e *Engine) chooseSpellTargets(spell, silent int) (ok bool) {
	var t aimTarget
	ok = true
	e.nSpellTargets = 0
	e.rv.areaSpell = 0
	sel := e.selected()
	e.tgtX, e.tgtY = e.combatX(sel), e.combatY(sel)
	fromNear := func(rng int) {
		e.buildNear(e.tgtX, e.tgtY, rng, 0xFF, 1)
		for i, n := range e.near.list {
			e.setTgt(i+1, e.combatant(n.index))
		}
		e.nSpellTargets = len(e.near.list)
		e.rv.areaSpell = 1
	}
	already := func(n int) bool {
		for k := 1; k <= n; k++ {
			if e.tgt(k) == t.c {
				return true
			}
		}
		return false
	}
	kind := e.spellData(spell, spellTargetsField) & 0xF
	switch {
	case kind == spellTargetsCaster:
		e.setTgt(1, sel)
		e.nSpellTargets = 1
	case kind == spellTargetsHitDice:
		sum, n, done := 0, 0, false
		var budget int
		if spell == spellFaerieFire {
			budget = e.casterLevel(spell)
		} else {
			budget = e.Dice(hitDiceBudgetDice, hitDiceBudgetSides)
		}
		for !done && budget != 0 {
			if !e.chooseSpellTarget(spell, silent, 0, &t) {
				done = true
				continue
			}
			if !already(n) {
				n++
				e.setTgt(n, t.c)
				e.tgtX, e.tgtY = e.combatX(t.c), e.combatY(t.c)
				e.nSpellTargets++
				sum += hitDiceWeight(spell, t.c)
				if n > 1 && byte(sum) > byte(budget) {
					done = true
				}
			} else if silent != 0 {
				budget = int(byte(budget - 1))
			} else {
				e.printMessage("Already been targeted")
			}
			e.eraseCursor(int(t.x), int(t.y))
		}
	case kind == spellTargetsOneOrArea:
		if !e.chooseSpellTarget(spell, silent, 0, &t) {
			ok = false
		} else if sel.Combat.Target != nil {
			e.setTgt(1, sel.Combat.Target)
			e.nSpellTargets = 1
		} else {
			fromNear(int(kind) >> 4) // always 0, as in the original
		}
	case kind >= spellTargetsAreaFirst && kind <= spellTargetsAreaLast:
		if !e.chooseSpellTarget(spell, silent, 1, &t) {
			ok = false
		} else {
			fromNear(int(kind & 7))
		}
	default:
		left := int(kind&spellTargetsCountMask) + 1
		n := 0
		for left > 0 {
			if !e.chooseSpellTarget(spell, silent, 0, &t) {
				left = 0
				continue
			}
			if !already(n) {
				n++
				e.setTgt(n, t.c)
				left--
				e.tgtX, e.tgtY = e.combatX(t.c), e.combatY(t.c)
			} else if silent == 0 {
				e.printMessage("Already been targeted")
			} else {
				left--
			}
			e.eraseCursor(int(t.x), int(t.y))
		}
		e.nSpellTargets = n
		if n == 0 {
			ok = false
		} else {
			e.tgtX, e.tgtY = e.combatX(e.tgt(n)), e.combatY(e.tgt(n))
		}
	}
	return
}

// spellMenuTargets = O0C9:2558 spell_menu3 (spell, a, var quit): the selected character picks
// (spell 0: from his memorised spells) and casts a spell. Spells with a
// casting time of 3 or more only begin and are cast in a later round.
func (e *Engine) spellMenuTargets(spell, a int, quit *bool) {
	sel := e.selected()
	have, idx := true, -1 // bp-6 = 1, bp-5 = 0xFFFF
	*quit = false
	if spell == 0 {
		spell = int(e.SpellList(0, 1, &idx, &have))
	}
	if spell > 0 && e.spellData(spell, spellCombatField) == 0 {
		e.printMessage("Camp Only Spell")
		spell = 0
	}
	if a == 0 {
		e.combatUseHook()
		e.showMoves = true
		e.panelDirty = true
		e.combatRedrawChar(sel, 3, 1)
		e.panelDirty = true
		e.combatPanel(sel)
	}
	if spell > 0 {
		delay := int(e.spellData(spell, spellCastTimeField)) / 3
		if delay == 0 {
			e.CastSpell(byte(spell), byte(a), 1, quit)
			*quit = e.endTurn(sel)
		} else {
			*quit = true
			e.combatName(sel, "Begins Casting", 0xA, true)
			sel.Combat.Spell = byte(spell)
			if int(int8(sel.Combat.Delay)) > delay {
				sel.Combat.Delay -= byte(delay)
			} else {
				sel.Combat.Delay = 1
			}
		}
	}
}

// Weapons a thief backstabs with (item types, which are also their name
// words): club, dagger, broad, long and short sword.
const (
	itemTypeClub       = 3
	itemTypeDagger     = 4
	itemTypeBroadSword = 17
	itemTypeShortSword = 19
)

// sizeLarge: the Size bit of a combatant counted as large.
const sizeLarge = 128

// canBackstab = O0C9:2783 (c, target): a thief with no weapon or a short
// one, behind a man-sized target next to it that others are already
// fighting.
func (e *Engine) canBackstab(c, target *Character) bool {
	w := c.Slots[slotWeapon]
	thief := int8(c.Levels[clsThief]) > 0 || int8(c.OldLevels[clsThief]) > 0 && oldClassUsable(c)
	weapon := w == nil || w.Type == itemTypeClub || w.Type == itemTypeDagger ||
		w.Type >= itemTypeBroadSword && w.Type <= itemTypeShortSword
	return thief && weapon && target.Combat.Attackers > 1 && e.rangeTo(c, target) < 2 &&
		target.Size&^sizeLarge <= 1 && byte(e.directionTo(c, target)) == target.Combat.Facing
}

// Slopes (times 256) of the borders between the eight directions:
// tan 67.5 and tan 22.5 degrees.
const (
	steepSlope   = 618
	shallowSlope = 106
	slopeScale   = 256
)

// directionTo = O151:38C8 (from, to): the direction 0..7 from one combatant
// to the other (north first, clockwise); a diagonal needs both
// coordinates to differ.
func (e *Engine) directionTo(from, to *Character) int {
	x1, y1 := e.combatX(from), e.combatY(from)
	x2, y2 := e.combatX(to), e.combatY(to)
	dx, dy := x2-x1, y2-y1
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	steep := steepSlope * dx / slopeScale
	shallow := shallowSlope * dx / slopeScale
	diagonal := steep >= dy && shallow <= dy
	for dir := 0; dir < 8; dir++ {
		var fits bool
		switch dir {
		case 0:
			fits = y2 <= y1 && steep <= dy
		case 1:
			fits = y2 < y1 && x2 > x1 && diagonal
		case 2:
			fits = x2 >= x1 && shallow >= dy
		case 3:
			fits = y2 > y1 && x2 > x1 && diagonal
		case 4:
			fits = y2 >= y1 && steep <= dy
		case 5:
			fits = y2 > y1 && x2 < x1 && diagonal
		case 6:
			fits = x2 <= x1 && shallow >= dy
		case 7:
			fits = y2 < y1 && x2 < x1 && diagonal
		}
		if fits {
			return dir
		}
	}
	return 8 // never: the eight sectors cover the plane
}

// Sounds of the attack animation (DS:25BA, DS:25C0, DS:25C6).
const (
	sndAnimSwish = 6
	sndAnimBlast = 9
	sndAnimThrow = 12
)

// Item types (which are also their name words) with their own missile
// pictures.
const (
	itemTypeHandAxe     = 2
	itemTypeDart        = 5
	itemTypeHammer      = 6
	itemTypeJavelin     = 7
	itemTypeQuarrel     = 12
	itemTypeSling       = 28
	itemTypeSlingStaff  = 29
	itemTypeArrow       = 30
	itemTypeHornetsNest = 66
)

// Missile pictures: icon slots from missileIconBase and how they fly.
const (
	missileIconBase    = 13
	missileSpinOffset  = 3 // a spinning weapon (hit colours)
	missileStoneOffset = 8 // a sling stone
	missileOtherOffset = 7 // anything else
	missileDelay       = 10
	spinFrames         = 4
	spinDelay          = 50
	otherDelay         = 20
)

// attackAnimation = O0C9:2872 (from, to, weapon): the weapon's picture
// flies from one combatant to the other: arrows, quarrels, darts and
// javelins point their way, thrown axes, clubs and hammers spin, sling
// stones and everything else tumble.
func (e *Engine) attackAnimation(from, to *Character, weapon *Item) {
	e.PlaySound(sndAnimThrow)
	dir := e.directionTo(from, to)
	frames, delay := 1, missileDelay
	switch weapon.Type {
	case itemTypeHornetsNest, itemTypeDart, itemTypeJavelin, itemTypeQuarrel, itemTypeArrow:
		switch {
		case dir%2 == 1 && (dir == 3 || dir == 5):
			e.iconToSprite(missileIconBase+1, 1, 0, dir == 5)
		case dir%2 == 1:
			e.iconToSprite(missileIconBase+1, 0, 0, dir == 7)
		default:
			e.iconToSprite(missileIconBase+dir%4, dir/4, 0, false)
		}
		e.PlaySound(sndAnimThrow)
	case itemTypeHandAxe, itemTypeClub, itemTypeHammer:
		e.hitColours(missileIconBase + missileSpinOffset)
		frames, delay = spinFrames, spinDelay
		e.PlaySound(sndAnimBlast)
	case itemTypeSling, itemTypeSlingStaff:
		e.iconToSprite(missileIconBase+missileStoneOffset, 0, 0, false)
		e.iconToSprite(missileIconBase+missileStoneOffset, 1, 1, false)
		frames = 2
		e.PlaySound(sndAnimSwish)
	default:
		e.iconToSprite(missileIconBase+missileOtherOffset, 0, 0, false)
		e.iconToSprite(missileIconBase+missileOtherOffset, 1, 1, false)
		frames, delay = 2, otherDelay
		e.PlaySound(sndAnimBlast)
	}
	e.flyMissile(e.combatX(from), e.combatY(from), e.combatX(to), e.combatY(to), frames, delay)
}

// monsterHPPercent = O0C9:2A58: the party's morale is the monsters'
// remaining hit points in five percent steps (20 * hp / maximum, times 5;
// dead ones count in the maximum).
func (e *Engine) monsterHPPercent() {
	var hp, max uint16
	for _, c := range e.party {
		if c.Side == 1 {
			if c.Okay != 0 {
				hp += uint16(c.HP)
			}
			max += uint16(c.MaxHP)
		}
	}
	if max != 0 {
		e.partyMorale = byte(20 * hp / max * 5) // the 16-bit product, dx cleared before the divide
	}
}

// maxPartyMove = O0C9:2AE2 (c): the highest moveRate / 2 over the living
// combatants on the other side.
func (e *Engine) maxPartyMove(c *Character) int {
	best := byte(0)
	for _, t := range e.party {
		if b2i(isParty(c)) == int(t.Side) && t.Okay != 0 {
			if v := byte(e.moveRate(t)) >> 1; v > best {
				best = v
			}
		}
	}
	return int(best)
}

// mayAttack = O0C9:2B73 (c, target): may c attack the target? Attacking
// a friend asks first; yes turns every sane character of the party hostile.
func (e *Engine) mayAttack(c, target *Character) bool {
	if b2i(isParty(target)) == int(c.Side) || c.Auto != 0 {
		return true
	}
	if e.YesNo(0xD, 0xA, 0xF, "Attack Ally: ") != 'Y' {
		return false
	}
	e.v6DE2 = 1
	putWord(e.game, 0x666, 1)
	for _, m := range e.party {
		if m.Status == 0 && m.Control > 0x7F {
			m.Side = 1
			if m.Combat != nil {
				m.Combat.Target = nil
			}
		}
	}
	e.countSides()
	return true
}

// aimPrompt = O0C9:2C85 Aim_menu (c, cand, rng, showRange, valid, x, y):
// shows how far the aimed square (x,y) is and asks Next, Prev, Manual,
// Target, Center or Exit. The arrow keys come back as extended keys.
func (e *Engine) aimPrompt(c, cand *Character, rng, showRange int, valid bool, x, y int) (byte, bool) {
	dist, tx, ty := 0xFF, x, y
	saved := e.cmap.seeAll
	e.cmap.seeAll = 1
	e.lineOfSight(&dist, &tx, &ty, e.combatX(c), e.combatY(c))
	e.cmap.seeAll = saved
	dist /= 2
	menu := ""
	if showRange != 0 {
		e.DisplayString(0, 0x17, 0xA, "Range = "+itoa(dist)+"  ")
	}
	if rng >= dist && valid {
		var tmp *Item
		switch {
		case showRange == 0:
			menu = "Target "
		case cand == c:
		case !e.rangedWeapon(c):
			menu = "Target "
		case e.ammoFor(c, &tmp):
			if e.nearEnemies(c, 1) == 0 || e.throwable(c) {
				menu = "Target "
			}
		}
	}
	menu = cut("Next Prev Manual "+menu+"Center Exit", 0x28)
	if cand != nil {
		e.combatRedrawChar(cand, 3, 1)
		e.panelDirty = true
		e.combatPanel(cand)
	}
	return e.combatInput("Aim:", menu, true)
}

// combatInput = O104:03D8 with the picture not animated (prompt, menu,
// allowExtra): the key, and the DS:6E31 flag is dropped.
func (e *Engine) combatInput(prompt, menu string, allowExtra bool) (byte, bool) {
	return e.DisplayInput(prompt, menu, 0xD, 0xA, 0xF, allowExtra, false)
}

// aimConfirm = O0C9:2E7C (c, cand, mode, var found, var t): take cand as
// the target. With mode 1 (a missile attack) the attack is made at once.
func (e *Engine) aimConfirm(c, cand *Character, mode int, found *bool, t *aimTarget) {
	var res *Item
	*found = true
	if mode == 1 && !e.mayAttack(c, cand) {
		*found = false
	}
	if !*found {
		*t = aimTarget{}
		return
	}
	t.c = cand
	t.x, t.y = byte(e.combatX(cand)), byte(e.combatY(cand))
	e.cmap.cursor = 0
	e.combatCentre(e.cmap.viewX+viewW/2, e.cmap.viewY+3, 3, 8)
	if mode == 1 {
		if e.sweepAttack(c, cand) {
			*found = e.endTurn(c)
		} else {
			e.countAttack(c, cand)
			if e.rangedWeapon(c) && e.ammoFor(c, &res) && e.throwable(c) && e.rangeTo(c, cand) == 1 {
				res = nil
			}
			*found = e.fullAttack(c, cand, 0, res)
		}
	}
}

// aimManual = O0C9:302D (c, cand, rng, showRange, allowGround,
// allowCreature, var valid, var confirmed, var t, x, y): move a cursor
// over the map with the arrow keys from (x,y); Enter or T takes the square
// (and who stands on it), Esc or E gives up and puts the cursor back.
// valid tells whether the square taken may be targeted.
func (e *Engine) aimManual(c *Character, rng, showRange, allowGround, allowCreature int, startX, startY byte) (valid, confirmed bool, t aimTarget) {
	dxs, dys := stepBytes(stepX), stepBytes(stepY)
	x, y := int8(startX), int8(startY)
	key, dir := byte(0x20), 8
	e.cmap.cursor, e.cmap.cursorSize = 1, 1
	defer e.rawArrows("arrows, click=move the cursor|Enter, click it again=target|C=center|Esc=back")()
	for !(key == 0 || key == 13 || key == 'E' || key == 'T') {
		e.combatCentre(int(x), int(y), 2, dir)
		x += int8(dxs[dir])
		y += int8(dys[dir])
		x, y = min(max(x, 0), 0x31), min(max(y, 0), 0x18)
		who, ground := e.mapAt(int(x), int(y))
		e.ClearKeyboard()
		valid = false
		dist, tx, ty := 0xFF, int(x), int(y)
		if e.lineOfSight(&dist, &tx, &ty, e.combatX(c), e.combatY(c)) {
			valid = true
			if showRange != 0 {
				e.DisplayString(0, 0x17, 0xA, "Range = "+itoa(dist/2)+"  ")
			}
		} else if showRange != 0 {
			e.ClearCells(0, 0x17, 0x27, 0x17)
		}
		dist /= 2
		var cand *Character
		if valid {
			if who > 0 {
				cand = e.combatant(who)
			} else if ground == groundBody || ground == groundStoneBody {
				for b := 1; b <= e.nbodies; b++ {
					if int8(e.bodies[b].x) == x && int8(e.bodies[b].y) == y {
						cand = e.bodies[b].c
					}
				}
			}
		}
		if cand != nil {
			e.panelDirty = true
			e.combatPanel(cand)
		} else {
			e.ClearCells(panelLeft, panelTop, panelRight, panelBottom)
		}
		if rng < dist || e.groundInfo(ground)[0] == groundImpassable {
			valid = false
		}
		if cand != nil {
			if !e.mayAct(c, cand) || allowCreature == 0 {
				valid = false
			}
			if showRange == 1 {
				var ammo *Item
				if cand == c || (who == 0 && (ground == groundBody || ground == groundStoneBody)) {
					valid = false
				} else if e.rangedWeapon(c) && (!e.ammoFor(c, &ammo) || e.nearEnemies(c, 1) > 0 && !e.throwable(c)) {
					valid = false
				}
			}
		} else if allowGround == 0 {
			valid = false
		}
		menu := "Exit"
		if valid {
			menu = "Target"
		}
		mark := e.regionMark()
		{ // extra (nav.go): click a square to put the cursor there, again to target
			e.addRegion(region{x0: pageCol, y0: 1, x1: pageCol + viewW*3 - 1, y1: 21, click: func(cx, cy int) uint16 {
				sx, sy, in := e.combatSquare(cx, cy)
				switch {
				case !in:
					return 0
				case sx == int(x) && sy == int(y):
					if valid {
						return 'T'
					}
					return 0
				}
				e.ext.jump = &[2]int8{int8(sx), int8(sy)}
				return keyNone
			}})
		}
		key, _ = e.combatInput("(Use Cursor keys) ", menu, true)
		e.regionCut(mark)
		if j := e.ext.jump; j != nil { // the clicked square; the step below is dir 8 (none)
			e.ext.jump = nil
			e.eraseCursor(int(x), int(y))
			x, y = j[0], j[1]
		}
		switch key {
		case 13, 'T':
			confirmed = true
			t = aimTarget{c: cand, x: byte(x), y: byte(y)}
			e.cmap.cursor = 0
			e.eraseCursor(int(x), int(y))
		case scanUp:
			dir = 0
		case scanPageUp:
			dir = 1
		case scanRight:
			dir = 2
		case scanPageDown:
			dir = 3
		case scanDown:
			dir = 4
		case scanEnd:
			dir = 5
		case scanLeft:
			dir = 6
		case scanHome:
			dir = 7
		case ' ':
			e.combatCentre(int(x), int(y), 0, 8)
			dir = 8
		case 0, 'E':
			confirmed = false
			e.eraseCursor(int(x), int(y))
			t = aimTarget{x: startX, y: startY}
			e.combatCentre(int(startX), int(startY), 0, 8)
			e.cmap.cursor = 0
			e.eraseCursor(int(startX), int(startY))
		default:
			dir = 8
		}
	}
	return
}

// aimAt = O0C9:37DB (c, rng, showRange, allowGround, allowCreature, var
// found, var t): the player aims for c. Candidates are walked through
// nearest first (N next, P previous, M: manual cursor, T take, C centre,
// Esc/E give up). rng 0xFF takes the range of the readied weapon.
func (e *Engine) aimAt(c *Character, rangeArg, showRange, allowGround, allowCreature int, found *bool, t *aimTarget) {
	e.iconToSprite(0x19, 0, 0, false)
	*t = aimTarget{}
	*found = false
	rng := byte(rangeArg)
	if rng == 0xFF {
		if w := c.Slots[0]; w != nil {
			rng = e.baseItem(w.Type, 12) - 1
		} else {
			rng = 1
		}
	}
	if rng == 0 || rng == 0xFF {
		rng = 1
	}
	// the nested O0C9:353B and O0C9:3610 work on the caller's locals
	var list []nearEntry
	var cur byte
	var prevX, prevY byte
	ent := func(i byte) nearEntry {
		if int(i) < 1 || int(i) > len(list) {
			return nearEntry{}
		}
		return list[i-1]
	}
	// O0C9:353B: the combatants in sight that c may act against
	e.buildNear(e.combatX(c), e.combatY(c), 0x7F, 0xFF, e.combatSize(c))
	for _, n := range e.near.list {
		if e.mayAct(c, e.combatant(n.index)) {
			list = append(list, n)
		}
	}
	step := func(by byte, show bool) *Character { // O0C9:3610
		if show {
			prevX, prevY = e.cpos[ent(cur).index].x, e.cpos[ent(cur).index].y
		} else {
			e.eraseCursor(int(prevX), int(prevY))
		}
		cur += by
		if cur == 0 {
			cur = byte(len(list))
		}
		if int(cur) > len(list) {
			cur = 1
		}
		idx := ent(cur).index
		nx, ny := e.cpos[idx].x, e.cpos[idx].y
		if show {
			e.flyMissile(int(prevX), int(prevY), int(nx), int(ny), 1, 0)
			prevX, prevY = nx, ny
		}
		return e.combatant(idx)
	}
	cur = 1
	cand := step(0, true)
	x, y := byte(e.combatX(cand)), byte(e.combatY(cand))
	startX, startY := x, y
	valid := true
	key := byte(' ')
	for !(key == 0 || key == 13 || key == 'E' || key == 'T') {
		var ext bool
		key, ext = e.aimPrompt(c, cand, int(rng), showRange, valid, int(x), int(y))
		if ext {
			key = aimExtendedKey(key)
		}
		take := func() {
			*found = true
			if showRange == 1 {
				e.aimConfirm(c, t.c, showRange, found, t)
			}
			if !*found {
				e.cmap.cursor = 0
				e.combatCentre(int(x), int(y), 0, 8)
			}
		}
		switch key {
		case 'N':
			valid = true
			cand = step(1, true)
		case 'P':
			valid = true
			cand = step(0xFF, true)
		case 'M':
			var confirmed bool
			var mt aimTarget
			valid, confirmed, mt = e.aimManual(c, int(rng), showRange, allowGround, allowCreature, x, y)
			e.iconToSprite(0x19, 0, 0, false)
			cand, x, y = mt.c, mt.x, mt.y
			if valid && confirmed {
				key = 'T'
				*t = mt
				take()
			}
		case 'T':
			*t = aimTarget{c: cand, x: x, y: y}
			take()
		case 'C':
			e.cmap.cursor, e.cmap.cursorSize = 1, 1
			e.combatCentre(int(x), int(y), 0, 8)
			e.cmap.cursor = 0
		case 'E':
			e.cmap.cursor = 0
			e.combatCentre(int(x), int(y), 0, 8)
		case 0:
			e.cmap.cursor = 0
			e.eraseCursor(int(x), int(y))
			e.combatCentre(int(startX), int(startY), 0, 8)
		}
		if key == 'N' || key == 'P' || key == 'T' {
			if cand != nil {
				x, y = byte(e.combatX(cand)), byte(e.combatY(cand))
			}
		}
	}
	if showRange != 0 {
		e.ClearCells(0, 0x17, 0x27, 0x17)
	}
}

// aimExtendedKey: of the extended keys only the arrows count (as Manual).
func aimExtendedKey(k byte) byte {
	switch k {
	case scanUp, scanPageUp, scanRight, scanPageDown, scanDown, scanEnd, scanLeft, scanHome:
		return 'M'
	}
	return 0xFF
}

// pickAttackTarget = O0C9:3BF1 (c, rng, a, b): a target for c's attack from the
// enemies within rng, kept in c's combat block. Up to two passes of 20
// random picks; the second sees through walls (not with b set).
func (e *Engine) pickAttackTarget(c *Character, rng, a, b int) bool {
	first, found, second := false, false, false
	tgt := c.Combat.Target
	if b != 0 || tgt == nil || tgt.Side == c.Side || tgt.Okay == 0 || !e.mayAct(c, tgt) {
		c.Combat.Target = nil
	}
	if c.Combat.Target != nil {
		found = true
	}
	for !found && !second {
		second = first
		if first && b == 0 {
			e.cmap.seeAll = 1
		}
		tries := 0x14
		n := e.nearEnemies(c, rng)
		for tries > 0 && !found && n > 0 {
			tries--
			var r int
			if tries == 0x13 && e.FindAffect(c, affectCloudkill) != nil && e.combatant(e.targets[1]) == nil {
				r = 1 // the cloudkill swap of nearEnemies: the first pick is fixed
			} else {
				r = e.Dice(1, n)
			}
			if e.targets[r] > 0 {
				cand := e.combatant(e.targets[r])
				if (a != 0 && e.cmap.seeAll != 0) || e.mayAct(c, cand) {
					found = true
					c.Combat.Target = cand
				} else {
					e.targets[r] = 0
					for i := 1; i <= n; i++ {
						if e.targets[i] > 0 {
							found = true // sic, the original sets the result flag
						}
					}
					if found {
						found = false
					} else {
						n = 0
					}
				}
			}
		}
		if !first {
			first = true
		}
	}
	e.cmap.seeAll = 0
	return found
}

// godIntervene = O0C9:3E2D god_intervene: with the debug word (DS:8C94,
// "Wooden") as the second command line parameter, '-' kills every monster
// ("The Gods intervene!"). True when it was given.
func (e *Engine) godIntervene() bool {
	if e.Cfg.Param2 != debugWord {
		return false
	}
	e.printMessage("The Gods intervene!")
	for _, c := range e.party {
		if c.Side == sideMonsters {
			c.Okay = 0
			c.Status = statusDead
			if i := e.combatIndex(c); i == 0 {
				e.ncomb = 0 // DS:66C0 slot 0's size byte
			} else {
				e.cpos[i].size = 0
			}
		}
		e.endTurn(c)
	}
	e.combatCentre(e.cmap.viewX+viewW/2, e.cmap.viewY+viewH/2, 0xFF, 8)
	return true
}

// debugWord = DS:8C94 (copied from "Wooden" at DS:24EE): as the second
// command line parameter it skips the copy protection and lets the Gods
// intervene in combat.
const debugWord = "Wooden"

// Combat sides.
const (
	sideParty    = 0
	sideMonsters = 1
)
