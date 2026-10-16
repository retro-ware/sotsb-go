package engine

// The computer-controlled turn, ported from O05A: monster AI, and party
// members under AI control or fleeing. Called from the combat loop (ovr09).

// o10_000F = O05A:004B (char): one computer turn. Picks the move pattern,
// then spells, then moves and attacks until the turn is used.
func (e *Engine) o10_000F(c *Character) {
	done := e.aiKeys(c)
	e.ClearRow24()
	e.clearCombatText()
	if c.Okay == 0 {
		done = e.endTurn(c)
	}
	ci := c.Combat
	d := ci.Pattern
	if d == 0 || d > 4 || e.Dice(1, 4) == 1 {
		d = byte(e.Dice(1, 8))
		if d == 8 {
			d = byte(e.Dice(1, 2) + 4)
		} else {
			d = byte(e.Dice(1, 4)) // the first roll is thrown away
		}
	}
	ci.Pattern = d
	if !done {
		done = e.aiFleeCheck(c)
	}
	if ci.Fleeing != 0 && ci.Turned == 0 {
		e.combatName(c, "flees in panic", 10, true)
	}
	if done {
		return
	}
	if e.aiCastFromItem(c) {
		done = e.endTurn(c)
		return
	}
	if ci.Spell > 0 {
		e.CastSpell(ci.Spell, 1, 1, &done)
		done = e.endTurn(c)
		return
	}
	if e.aiTryCast(c) {
		done = e.endTurn(c)
		return
	}
	if e.aiCastKnown(c) {
		return
	}
	e.aiPickWeapon(c)
	done = e.aiKeys(c)
	for !done {
		if e.pickAttackTarget(c, 0xFF, 1, 0) && int8(ci.Delay) > 0 && c.Okay != 0 {
			done = e.aiAttackLoop(c)
		} else {
			done = e.aiGiveUp(c)
		}
	}
}

// aiTryCast = O05A:024B (char): a cleric (or one who was a cleric before
// changing class) with no turn used turns the undead in reach.
func (e *Engine) aiTryCast(c *Character) bool {
	if c.Combat.TurnUsed == 0 && (int8(c.Levels[clsCleric]) > 0 ||
		int8(c.OldLevels[clsCleric]) > int8(c.MulticlassLevel)) {
		var t *Character
		if e.weakestFoe(c, &t) {
			e.turnUndead(c)
			return true
		}
	}
	return false
}

// Spell table fields read by the computer's spell choice (DS:37E9..37EB;
// the table starts at DS:37DC). spellAimField (14) says whether it looks
// for targets.
const (
	spellSaveTypeField = 9  // the saving throw kind against it
	spellAIPowerField  = 13 // the highest pass (7 down to 1) at which the AI uses it
	spellAIRadiusField = 15 // area radius; 0: one target only
)

// spellSaveNone: a spell whose save does nothing harms its own side freely.
const spellSaveNone = 1

// Save bonus of those caught by the AI's area spell: the party's members
// save worse, monsters better.
const (
	aiAreaSaveParty    = -2
	aiAreaSaveMonsters = 8
)

// canCast = O05A:03B1 (char, spell, level): may the computer cast the
// spell at this pass: the pass fits and an area spell must not catch
// one of its own side who fails the save.
func (e *Engine) canCast(c *Character, spell, level int) bool {
	s := spellTable[spell]
	if int(s[spellAIPowerField]) < level {
		return false
	}
	if spell != spellCureLightWounds && s[spellAimField] == 0 {
		return true
	}
	if spell == spellCureLightWounds {
		var t *Character
		if e.healTarget(c, &t) {
			return true
		}
	}
	n := e.nearEnemies(c, e.spellRange(spell))
	if n == 0 {
		return false
	}
	if s[spellAIRadiusField] == 0 {
		return true
	}
	// O05A:02AF (nested): does the area at x, y catch one of the caster's
	// side who fails the save.
	catchesOwn := func(x, y int) bool {
		r := false
		bonus := aiAreaSaveMonsters
		if e.selected().Side == 0 {
			bonus = aiAreaSaveParty
		}
		e.buildNear(x, y, int(s[spellAIRadiusField]), 0xFF, 1)
		foes := otherSide(e.selected())
		for i := 1; i <= len(e.near.list); i++ {
			w := e.combatant(e.near.list[i-1].index)
			if w.Side == foes || s[spellSaveField] == spellSaveNone {
				continue
			}
			if !e.SavingThrow(w, int(s[spellSaveTypeField]), bonus) {
				r = true
			}
		}
		return r
	}
	for i := 1; i <= n; i++ {
		p := e.cpos[e.targets[i]]
		if catchesOwn(int(p.x), int(p.y)) {
			return false
		}
	}
	return true
}

// aiCastFromItem = O05A:04AA (char): try to use an item that casts a
// spell the computer can cast; true when one was used.
func (e *Engine) aiCastFromItem(c *Character) bool {
	r := false
	var found *Item
	lvl := 7
	rounds := e.Dice(1, 7)
	if c.Combat.CanUse == 0 || e.rv.side[otherSide(c)] == 0 || wordAt(e.area, 0x1CA) != 0 {
		return false
	}
	for k := 1; k <= rounds; k++ {
		for _, p := range c.Items {
			if found != nil {
				break
			}
			sp := p.Spells[1]
			if !e.isScroll(p) && p.Spells[2] < 0x80 && p.Readied != 0 && sp > 0 {
				if sp > 0x38 {
					sp -= 0x17
				}
				if e.canCast(c, int(sp), lvl) {
					found = p
				}
			}
		}
		lvl--
	}
	if found != nil {
		e.useItem(found)
		r = true
	}
	return r
}

// aiCastKnown = O05A:060B (char): pick a memorised spell the
// computer may cast, trying lower levels each round; true when the spell
// call (spellMenuTargets) returned a non-empty string.
func (e *Engine) aiCastKnown(c *Character) bool {
	var arr [len(charRecord{}.Memorized)]byte
	n := 0
	if c.Combat.CanCast != 0 {
		for i := 1; i < len(c.Memorized); i++ {
			if c.Memorized[i] > 0 {
				arr[n] = c.Memorized[i]
				n++
			}
		}
	}
	found := 0
	lvl := 7
	rounds := e.Dice(1, 7)
	if n > 0 && (c.Control > 0x7F || e.v6E30 != 0) && e.rv.side[otherSide(c)] > 0 {
		for it := 1; it <= rounds && found == 0; it++ {
			for t := 1; t < 4 && found == 0; t++ {
				sp := int(arr[e.Dice(1, n)-1])
				if e.canCast(c, sp, lvl) {
					found = sp
				}
			}
			lvl--
		}
	}
	if found > 0 {
		var ok bool
		e.spellMenuTargets(found, 1, &ok)
		return ok
	}
	return false
}

// aiStepTurns = DS:02B8: for move pattern p (1..6) and try i (1..5), the
// turn from the wanted direction at aiStepTurns[p*5+i]. The first bytes
// are other data; pattern 0 never occurs.
var aiStepTurns = [36]byte{137, 139, 142, 144, 31, 0,
	8, 7, 6, 1, 2,
	8, 1, 2, 7, 6,
	7, 1, 8, 6, 2,
	1, 7, 8, 2, 6,
	8, 7, 6, 5, 4,
	8, 1, 2, 3, 4}

// cloudkillLevelProof: from this many hit dice up the AI walks into Cloudkill.
const cloudkillLevelProof = 7

func aiStepDir(pattern byte, i, dir int) int {
	return (int(aiStepTurns[int(pattern)*5+i]) + dir) % 8
}

// aiCanStep = O05A:073B (char, i, dir, var blocked): can the combatant
// take the i-th step of its pattern towards dir; blocked is set when the
// square is off the map. A cloud it is not proof against costs all its
// moves unless it holds its breath (a save) or, for Cloudkill, is strong.
func (e *Engine) aiCanStep(c *Character, i, dir int) (ok, blocked bool) {
	ci := c.Combat
	d := aiStepDir(ci.Pattern, i, dir)
	who, ground, stinking, poison := e.groundAt(c, d)
	if ground == 0 {
		return false, true
	}
	if e.groundInfo(ground)[0] == 0xFF {
		return false, false
	}
	cost := stepCost(e.groundInfo(ground)[0], d)
	if who == 0 && cost < int(ci.Moves) {
		has := func(a byte) bool { return e.FindAffect(c, a) != nil }
		if stinking && !has(affectCollapse) && !has(affectCoughing) && !has(affectNoPoisonParalysis) &&
			!has(affectNoCharmHoldSleep) && !has(affectMagicImmune) && !has(affectMinorGlobe) &&
			ci.Turned == 0 && !e.SavingThrow(c, saveVsParalysis, 0) {
			cost = int(ci.Moves) + 1
		}
		if poison && int8(c.MaxLevel) < cloudkillLevelProof && !has(affectMagicImmune) &&
			!has(affectNoPoisonParalysis) && !has(affectNoMindAffects) && ci.Turned == 0 &&
			!has(affectNoCharmHoldSleep) {
			cost = int(ci.Moves) + 1
		}
		if int(ci.Moves) >= cost {
			return true, false
		}
	}
	return false, false
}

// aiMoveStep = O05A:09D1 (char): show the movement left and take one
// step towards the target (or a wandering direction when fleeing).
func (e *Engine) aiMoveStep(c *Character) {
	ci := c.Combat
	e.DisplayString(0, 0x18, 0xA, "Move/Attack, Move Left = "+itoa(int(ci.Moves>>1))+" ")
	if e.aiKeys(c) {
		return
	}
	// hired ones only move while the monsters are not too hurt; a mage
	// that is not fleeing moves only with armour readied
	willing := c.Control < 0x80 || int(e.v6E2F) <= int(e.v6825)+e.Dice(1, 100) || c.Side == 1
	if ci.Moves>>1 <= 0 || int8(ci.Delay) <= 0 || !willing ||
		(ci.Fleeing == 0 && c.Slots[slotArmour] == nil && c.Class == clsMage) {
		e.aiGiveUp(c)
		return
	}
	var dir int
	if ci.Fleeing == 0 {
		dir = e.directionTo(c, ci.Target)
	} else {
		ci.Pattern = byte(e.Dice(1, 2))
		f := int(e.partyFacing())
		dir = f - ((f+2)%4)>>1
		if c.Side == 0 {
			dir += 4
		}
		dir %= 8
		dir = int(byte(dir))
	}
	var ok, blocked bool
	done := false
	i := 1
	for i < 6 && !done {
		if ok, blocked = e.aiCanStep(c, i, dir); ok {
			break
		}
		if ci.Fleeing != 0 && blocked {
			done = e.fleeBattle(c)
		} else {
			i++
		}
	}
	var d int
	if done {
		ci.Moves = 0
		ci.Fleeing = 0
		done = e.endTurn(c)
	} else {
		d = aiStepDir(ci.Pattern, i, dir)
		if i == 6 || int(e.v443C) == (d+4)%8 {
			e.v443D++
			ci.Pattern = byte(int(ci.Pattern)%6 + 1)
			if e.v443D > 1 {
				ci.Target = nil
				if e.v443D > 2 {
					ci.Moves = 0
					done = true
				} else if !e.pickAttackTarget(c, 0xFF, 1, 0) {
					done = e.aiGiveUp(c)
				}
			}
		}
		if i < 6 {
			e.v443C = byte(d)
		} else {
			done = true
		}
	}
	if done {
		return
	}
	e.showMoves = e.aiEngaged || e.combatOnScreen(c, false) || c.Side == 0
	e.combatTurn(c, byte(d), 0, false)
	e.leaveAttacks(c, int(ci.Facing))
	if c.Okay == 0 {
		done = e.endTurn(c)
		return
	}
	if ci.Moves > 0 {
		e.stepSquare(c, int(ci.Facing))
	}
	if c.Okay == 0 {
		done = e.endTurn(c)
	}
	e.inPoisonCloud(c, true)
}

// aiAttackLoop = O05A:0DB1 (char): move, pick a target and attack until
// the turn is over; true when it is.
func (e *Engine) aiAttackLoop(c *Character) bool {
	ci := c.Combat
	e.v443C, e.v443D, e.v443E = 8, 0, 0
	iter := 0
	cont, stop := true, false
	e.affectGroup(c, 0xE)
	if c.Side == 0 && e.bandage(true) {
		ci.Delay = 0 // a party member under the computer bandages the dying first
	}
	if ci.Delay == 0 {
		cont = false
	}
	for !stop && cont {
		for ci.Fleeing != 0 && ci.Moves > 0 && int8(ci.Delay) > 0 && int8(ci.Delay) < 0x14 {
			e.aiMoveStep(c)
		}
		if ci.Delay == 0 || ci.Delay == 0x14 {
			cont = false
		} else {
			stop = false
		}
		if stop || !cont {
			continue
		}
		iter++
		if iter > 0x14 {
			stop = true
			cont = !e.aiGiveUp(c)
		}
		e.aiEngaged = false
		rng := 1
		if w := c.Slots[0]; w != nil {
			rng = int(e.baseItem(w.Type, 0xC) - 1)
		}
		if rng == 0 || rng == 0xFF {
			rng = 1
		}
		tgt := ci.Target
		if tgt != nil && (tgt.Okay == 0 || tgt.Side == 0) {
			tgt = nil
		}
		if tgt != nil && e.mayAct(c, tgt) {
			tx, ty := e.combatX(tgt), e.combatY(tgt)
			r := rng
			if e.lineOfSight(&r, &tx, &ty, e.combatX(c), e.combatY(c)) && r>>1 <= rng {
				e.aiEngaged = true
			}
		}
		if !e.aiEngaged {
			n := e.nearEnemies(c, rng)
			if n == 0 {
				if e.pickAttackTarget(c, 0xFF, 0, 0) {
					e.aiMoveStep(c)
				} else {
					stop = e.aiGiveUp(c)
				}
			} else {
				r := e.Dice(1, n)
				tgt = e.combatant(e.targets[r])
				if e.rangedWeapon(c) && !e.throwable(c) && e.nearEnemies(c, 1) > 0 {
					e.aiPickWeapon(c)
					stop = true
					continue
				}
				if e.rangeTo(c, tgt) == 1 || e.mayAct(c, tgt) {
					e.aiEngaged = true
				}
			}
		}
		if e.aiEngaged {
			e.combatCentre(e.combatX(c), e.combatY(c), 2, e.directionTo(c, tgt))
		}
		if !e.aiEngaged {
			continue
		}
		if e.sweepAttack(c, tgt) {
			stop = e.endTurn(c)
			continue
		}
		e.countAttack(c, tgt)
		var weapon *Item
		if e.rangedWeapon(c) {
			e.aiEngaged = e.ammoFor(c, &weapon)
			if e.throwable(c) && e.rangeTo(c, tgt) == 1 {
				weapon = nil
			}
		}
		stop = e.fullAttack(c, tgt, 0, weapon)
		if stop {
			cont = false
		} else if tgt.Okay == 0 {
			stop = true
		}
	}
	return !cont
}

// aiGiveUp = O05A:11F7 (char): the turn ends; a combatant that cannot go
// on is cleared, otherwise it stands guard (O0ED:2833).
func (e *Engine) aiGiveUp(c *Character) bool {
	e.clearCombatText()
	if e.isHelpless(c) || e.rangedWeapon(c) || c.Combat.Delay == 0 {
		return e.endTurn(c)
	}
	return e.guard(c)
}

// aiKeys = O05A:1269 (char): keys during the computer turn: '2' toggles
// magic, space ends the turn of everyone not yet moved, '-' (godIntervene).
func (e *Engine) aiKeys(c *Character) bool {
	r := false
	if e.keyWaiting() {
		k := e.ReadKeyByte()
		if k == 0 {
			k = e.ReadKeyByte()
		}
		switch k {
		case 0x32:
			e.v6E30 ^= 1
			if e.v6E30 != 0 {
				e.printMessage("Magic On")
			} else {
				e.printMessage("Magic Off")
			}
		case 0x20:
			for _, p := range e.party {
				if p.Control < 0x80 && p.Status != 1 {
					p.Auto = 0
				}
			}
			if c.Auto == 0 {
				c.Combat.Delay = 0x14
				r = true
			}
		case 0x2D:
			e.godIntervene()
		}
	}
	e.ClearKeyboard()
	return r
}

// aiFleeCheck = O05A:137F (char): a forced flee, or a morale check that
// makes the combatant flee or surrender.
func (e *Engine) aiFleeCheck(c *Character) bool {
	ci := c.Combat
	r := false
	ci.Fleeing = 0
	e.releaseHeld(c)
	if ci.Turned != 0 {
		ci.Fleeing = 1
		e.combatName(c, "is forced to flee", 10, true)
		return r
	}
	if c.Control <= 0x7F {
		return r
	}
	e.v6825 = byte(int(c.Control&0x7F) * 2)
	if e.v6825 > 0x66 {
		e.v6825 = 0
	}
	e.affectGroup(c, 0x11)
	t := 100 - int(c.HP)*100/int(c.MaxHP) // cdq clobbers dx: signed word divide
	if !(int(e.v6825) < t || e.v6825 == 0) {
		return r
	}
	e.v6825 = e.v6E2F
	e.affectGroup(c, 0x11)
	t2 := uint16(100 - wordAt(e.game, 0x58C))
	if !(uint16(e.v6825) < t2 || e.v6825 == 0 || c.Side == 0) {
		return r
	}
	a := e.maxPartyMove(c)
	if a <= e.turnMoveRate(c)>>1 {
		ci.Fleeing = 1
		e.RemoveAffect(c, affectMiscasting, nil)
		e.RemoveAffect(c, affectVsKind3, nil)
		return r
	}
	if c.Stats[statInt].Base > 5 {
		e.KillCharacter(c, 4, "Surrenders")
		r = e.endTurn(c)
	}
	return r
}

// weaponScore = O05A:1535 (char, item): how good the item is as the
// character's weapon (0 = unusable).
func (e *Engine) weaponScore(c *Character, it *Item) int {
	var t [16]byte
	for i := range t {
		t[i] = e.baseItem(it.Type, i)
	}
	s := byte(int(t[9]) * int(t[10]))
	if int8(it.Plus) > 0 {
		s += byte(int(int8(it.Plus)) << 3)
	}
	if int8(t[11]) > 0 {
		s += byte(int(int8(t[11])) << 1)
	}
	if it.Type == 0x55 && c.Combat.Target != nil && int8(c.Combat.Target.Undead) > 0 {
		s = 8
	}
	if t[14]&8 != 0 {
		s += byte((int(t[5]) - 1) << 1)
	}
	if t[1] <= 1 {
		s += 3
	}
	if int(c.Hands)+int(t[1]) > 3 {
		s = 0
	}
	if it.Spells[2] == 0x84 && it.Spells[1]&0xF != c.Alignment {
		s = 0
	}
	if it.Spells[1] == 0x53 {
		s = 0
	}
	if it.Cursed != 0 {
		s = 0
	}
	return int(s)
}

// aiPickWeapon = O05A:1673 (char): choose the best melee or missile
// weapon, and the best second item, and ready them.
func (e *Engine) aiPickWeapon(c *Character) {
	tb := func(it *Item, n int) byte { return e.baseItem(it.Type, n) }
	if w := c.Slots[0]; w != nil {
		c.Hands -= tb(w, 1)
	}
	if w := c.Slots[1]; w != nil {
		c.Hands -= tb(w, 1)
	}
	var best1, best2, best3 *Item
	bs1, bs3 := byte(1), byte(0)
	bs2 := byte(int(c.BaseDice[0]) * int(c.BaseSides[0]))
	if int8(c.BaseBonus[0]) > 0 {
		bs2 += byte(int(int8(c.BaseBonus[0])) << 1)
	}
	for _, p := range c.Items {
		if tb(p, 0) == 0 && tb(p, 0xD)&c.ClassMask != 0 {
			sc := byte(e.weaponScore(c, p))
			f := tb(p, 0xE)
			if (f&8 != 0 || f&0x10 != 0) && sc > bs1 {
				best1, bs1 = p, sc
			}
			if f&8 == 0 && sc > bs2 {
				best2, bs2 = p, sc
			}
		}
		if tb(p, 0) == 1 && tb(p, 0xD)&c.ClassMask > 0 {
			sc := byte(0)
			if int8(p.Plus) >= 0 {
				sc = byte(int(int8(p.Plus)) + 1)
			}
			if sc > bs3 {
				best3, bs3 = p, sc
			}
		}
	}
	ranged := best1 != nil && tb(best1, 0xC) > 1 && tb(best1, 0xE)&0x14 == 0x14
	var pend *Item
	var f byte
	if best1 != nil {
		f = tb(best1, 0xE)
		if f&0x10 != 0 {
			pend = best1
		}
		if f&8 != 0 {
			if f&1 != 0 {
				pend = c.Slots[11]
			}
			if f&0x80 != 0 {
				pend = c.Slots[12]
			}
		}
	}
	ok := pend != nil || f == 0x0A
	chosen := best2
	if best1 != nil && int(bs1) > int(bs2)>>1 && ok && (ranged || e.nearEnemies(c, 1) == 0) {
		chosen = best1
	}
	done := false
	ok = true
	if w := c.Slots[0]; w != nil && (w == chosen || w.Cursed != 0) {
		ok = false
	}
	if ok {
		if c.Slots[0] != nil {
			e.readyItem(c.Slots[0])
		}
		e.ApplyItems(c)
		if w := c.Slots[1]; w != nil && w.Cursed == 0 {
			c.Hands -= tb(w, 1)
		}
		if chosen != nil {
			e.readyItem(chosen)
		}
		done = true
	}
	e.ApplyItems(c)
	e.setSpeed(c)
	ok = true
	if w := c.Slots[1]; w != nil && (w == best3 || w.Cursed != 0) {
		ok = false
	}
	if c.Hands > 2 {
		// the original passes chosen even when nil
		if w := c.Slots[1]; w == nil || w.Cursed != 0 {
			if chosen != nil {
				e.readyItem(chosen)
			}
		} else {
			e.readyItem(w)
		}
		done = true
	} else if c.Hands < 2 && ok {
		if c.Slots[1] != nil {
			e.readyItem(c.Slots[1])
		}
		e.ApplyItems(c)
		if best3 != nil {
			e.readyItem(best3)
		}
		done = true
	}
	e.ApplyItems(c)
	if done {
		e.panelDirty = true
		e.combatPanel(c)
	}
}
