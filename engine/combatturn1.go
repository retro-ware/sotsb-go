package engine

// Combat turn, first half (O0C9:0000..1DFE): setting up a combatant's turn,
// movement, attacks of opportunity, fleeing, sweeps, turning undead and the
// attack itself. The second half (from 1DFE) is in combatturn2.go.

// turnSetup = O0C9:0000 (char): fill the combat block for a new round. No
// one can cast inside a stinking cloud.
func (e *Engine) turnSetup(c *Character) {
	cb := c.Combat
	cb.Spell, cb.CanCast, cb.CanUse, cb.Attacked, cb.AttackSlot = 0, 1, 1, 0, 2
	if _, _, stinking, _ := e.groundAt(c, dirHere); stinking {
		cb.CanCast = 0
	}
	e.setSpeed(c)
	e.rv.attacks = c.BaseAttack[1]
	e.v681D = 0
	e.affectGroup(c, grpSpeed)
	c.Attacks[1] = byte(e.halfUp(e.rv.attacks))
	cb.Sweeps = c.FightLevel
	if c.Okay != 0 {
		cb.Delay = byte(int(dexMissile(c)) + e.Dice(1, 6))
		if int8(cb.Delay) < 1 {
			cb.Delay = 1
		}
		if byte((int(int8(c.Side))+1)&int(wordAt(e.game, area2Surprise))) != 0 { // only the low byte is tested
			cb.Delay -= 6
		}
		if int8(cb.Delay) < 0 || int8(cb.Delay) > 0x14 {
			cb.Delay = 0
		}
	} else {
		cb.Delay = 0
	}
	cb.Moves = byte(e.moveRate(c))
}

// rollDamage = O0C9:014C (att, vic, slot): damage roll of the attack slot (1 or 2),
// tripled-or-so for a backstab, then the
// affects of groups 4 (attacker) and 5 (victim) may change DS:6818.
func (e *Engine) rollDamage(a, b *Character, slot int) {
	d := int(int8(a.Bonus[slot-1])) + e.RollDice(int(a.Dice[slot-1]), int(a.Sides[slot-1]))
	e.rv.damage = byte(d)
	// (the original compares the byte with 0: never clamps)
	if e.canBackstab(a, b) {
		usable := 0
		if oldClassUsable(a) {
			usable = 1
		}
		lvl := int(int8(a.Levels[clsThief])) + int(int8(a.OldLevels[clsThief]))*usable
		e.rv.damage = byte(int(e.rv.damage) * (int(uint16(int16(lvl-1)))>>2 + 2))
	}
	e.rv.dmgType = 0
	e.affectGroup(a, grpDamageBonus)
	e.affectGroup(b, grpWeaponDamage)
}

// Attack text kinds.
const (
	attackPlain      = 0
	attackFromBehind = 1
	attackBackstab   = 2
	attackSlay       = 3
)

// Side panel cells of the attack text.
const (
	attackTextColumn = 23
	attackTextRow    = 12
	attackTextRight  = 38
	attackTextLines  = 3
)

// attackText = O0C9:02D5 backstab (att, vic, kind, damage, shown, hit): the
// attack text. kind: attackPlain .. attackSlay; damage is applied, shown
// printed; hit 0 is a miss.
func (e *Engine) attackText(a, b *Character, kind, damage, shown, hit int) {
	var s string
	switch kind {
	case attackBackstab:
		s = "-Backstabs-"
	case attackSlay:
		s = "slays helpless"
	default:
		s = "Attacks"
	}
	e.combatName(a, s, combatMsgRow, false)
	row := attackTextRow
	e.displayName(b, attackTextColumn, row, false)
	row++
	s = ""
	if kind == attackFromBehind {
		s = "(from behind) "
	}
	if hit != 0 {
		if kind == attackSlay {
			s = "with one cruel blow"
		} else {
			s += "Hitting for " + itoa(shown&0xFF)
			if shown&0xFF == 1 {
				s += " point "
			} else {
				s += " points "
			}
			s += "of damage"
		}
		e.damagePlayer(b, byte(damage))
	} else {
		s += "and Misses"
	}
	if b.Status != statusDisintegrated {
		e.DisplayTextBlock(attackTextColumn, row, attackTextRight, row+attackTextLines, colourGreen, false, s)
	}
	row = int(e.msgRow) + 1
	if damage&0xFF > 0 {
		b.Combat.CanCast = 0
		if b.Combat.Spell > 0 {
			e.pauseForMessage()
			e.combatName(b, "lost a spell", combatMsgRow+2, true)
			forgetSpell(b, b.Combat.Spell) // the spell stays chosen
		} else {
			e.pauseForMessage()
		}
	} else {
		e.pauseForMessage()
	}
	if b.Okay == 0 {
		e.combatName(b, "goes down", row, false)
		row += 2
		if b.Status == statusDying {
			e.DisplayString(attackTextColumn, row, colourGreen, "and is Dying")
		}
		if st := b.Status; st >= statusDead && st <= statusDisintegrated {
			e.combatName(b, "is killed", row, false)
		}
		e.downAffects(b)
		if b.Okay == 0 {
			e.combatRemoveFromMap(b)
		} else {
			e.pauseForMessage()
		}
	}
	e.clearCombatText()
}

// guardsStrike = O0C9:060E (char): the enemies next to a mover that were waiting (+7) lose
// their reaction, turn to it and attack it.
func (e *Engine) guardsStrike(c *Character) {
	n := e.nearEnemies(c, 1)
	for i := 1; i <= n; i++ {
		if c.Okay == 0 {
			continue
		}
		t := e.combatant(e.targets[i])
		if t.Combat.Guarding == 0 || e.isHelpless(t) {
			continue
		}
		e.combatCentre(e.combatX(c), e.combatY(c), 2, 8)
		t.Combat.Guarding = 0
		e.countAttack(t, c)
		e.fullAttack(t, c, 0, nil)
	}
}

// stepSquare = O0C9:06F7 (char, dir): step one square, paying the ground's
// step cost (stepCost) out of the moves left.
func (e *Engine) stepSquare(c *Character, dir int) {
	cb := c.Combat
	idx := e.combatIndex(c)
	ox, oy := e.cpos[idx].x, e.cpos[idx].y
	nx := ox + byte(stepX[dir])
	ny := oy + byte(stepY[dir])
	cost := stepCost(e.groundInfo(e.cmap.ground[ny][nx])[0], dir)
	if cost > int(cb.Moves) {
		cb.Moves = 0
	} else {
		cb.Moves -= byte(cost)
	}
	margin := 1
	if c.Auto != 0 {
		margin = 3
		if !e.combatInView(int(int8(nx))-e.cmap.viewX, int(int8(ny))-e.cmap.viewY) && e.showMoves {
			e.combatCentre(int(ox), int(oy), 2, 8)
		}
	}
	if e.showMoves {
		e.combatClearIcon(0, 0, idx)
	}
	e.cpos[idx].x, e.cpos[idx].y = nx, ny
	e.combatRedrawMap()
	if e.showMoves {
		e.combatCentre(int(nx), int(ny), margin, 8)
	}
	cb.Attackers, cb.Flank = 0, 0
	e.PlaySound(sndStep)
	e.guardsStrike(c)
	if c.Okay == 0 || e.isHelpless(c) {
		cb.Moves = 0
	}
}

// leaveAttacks = O0C9:0902 (char, dir): the mover steps (virtually) one
// square in dir; the enemies next to it that are able to fight and would
// no longer be next to it get a free attack, if they face it or have not
// yet acted or been attacked this round.
func (e *Engine) leaveAttacks(c *Character, dir int) {
	idx := e.combatIndex(c)
	var watching []byte
	for i := 1; i <= e.nearEnemies(c, 1); i++ {
		t := e.combatant(e.targets[i])
		if !e.isHelpless(t) && t.Combat.Turned == 0 {
			watching = append(watching, e.targets[i])
		}
	}
	if len(watching) == 0 {
		return
	}
	dx, dy := byte(stepX[dir]), byte(stepY[dir])
	e.cpos[idx].x += dx
	e.cpos[idx].y += dy
	stillNear := e.nearEnemies(c, 1)
	e.cpos[idx].x -= dx
	e.cpos[idx].y -= dy
	for i := range watching {
		for j := 1; j <= stillNear; j++ {
			if e.targets[j] == watching[i] {
				watching[i] = 0
			}
		}
	}
	for _, w := range watching {
		if w == 0 || c.Okay == 0 {
			continue
		}
		e.panelDirty, e.showMoves = true, true
		t := e.combatant(w)
		if !e.mayAct(t, c) || e.rangedWeapon(t) && !e.throwable(t) {
			continue
		}
		tcb := t.Combat
		for k := int(tcb.Facing) + 6; k <= int(tcb.Facing)+10; k++ {
			if int8(tcb.Delay) <= 0 && tcb.Attackers != 0 &&
				!e.inArc(e.combatX(t), e.combatY(t), e.combatX(c), e.combatY(c), k%8) {
				continue
			}
			hand := 1
			if t.BaseAttack[0] == 0 {
				hand = 2
			}
			for h := 1; h <= 2; h++ {
				if t.Attacks[h-1] > 0 {
					hand = h
				}
			}
			if t.Attacks[hand-1] == 0 {
				t.Attacks[hand-1] = 1
			}
			tcb.AttackSlot = byte(hand)
			saved := tcb.Target
			e.fullAttack(t, c, 1, nil)
			tcb.Target = saved
			if c.Okay != 0 {
				e.panelDirty = true
				e.combatPanel(c)
			}
			break
		}
	}
}

// fleeBattle = O0C9:0D3B flee_battle (char) bool: try to run away; "Got Away" removes the character.
func (e *Engine) fleeBattle(c *Character) bool {
	away := false
	if e.nearEnemies(c, 0xFF) == 0 {
		away = true
	} else {
		a := byte(e.moveRate(c)) >> 1
		b := byte(e.maxPartyMove(c))
		if b < a {
			away = true
		} else if b == a && e.Dice(1, 2) == 1 {
			away = true
		}
	}
	if away {
		e.KillCharacter(c, 3, "Got Away")
	} else {
		e.printMessage("Escape is blocked")
	}
	return e.endTurn(c)
}

// setSpeed = O0C9:0DF9 (char): the character's speed in +0x113 from its movement base
// (+0xA1) or the readied weapon's speed (DS:5582 record +5, at least 2).
func (e *Engine) setSpeed(c *Character) {
	flag := false
	old := c.Attacks[0]
	c.Attacks[0] = c.BaseAttack[0]
	var it *Item
	if e.rangedWeapon(c) && e.ammoFor(c, &it) {
		flag = true
		v := byte(0) // (the original reads the item at +0xCC unchecked)
		if c.Slots[0] != nil {
			v = e.baseItem(c.Slots[0].Type, 5)
		}
		if v < 2 {
			v = 2
		}
		e.rv.attacks = v
	} else {
		e.rv.attacks = c.Attacks[0]
	}
	e.v681D = 0
	e.affectGroup(c, 0x12)
	r := byte(e.halfUp(e.rv.attacks))
	if flag && it != nil {
		w := byte(1)
		if it.Count > w {
			w = it.Count
		}
		if w < r && it.Count > 0 {
			r = w
		}
	}
	if c.Combat.Attacked == 0 || r < old {
		c.Attacks[0] = r
	} else {
		c.Attacks[0] = old
	}
}

// halfUp = O0C9:0F12 (v): half of v, rounded up when DS:6DE3 is odd.
func (e *Engine) halfUp(v byte) int {
	if e.combatRound&1 != 0 {
		v++
	}
	return int(v >> 1)
}

// sweepAttack = O0C9:0F46 (att, vic) bool: a fighter with more attacks than... sweeps: the
// attacker (fewer attacks than +5 allows, victim not a "big" target) hits
// several adjacent weak enemies, one attack each.
func (e *Engine) sweepAttack(a, b *Character) bool {
	res := false
	acb := a.Combat
	if a.Attacks[0] < acb.Sweeps && b.MaxLevel == 0 && e.rangeTo(a, b) == 1 {
		n := e.nearEnemies(a, 1)
		cnt := 0
		bi := 0
		for i := 1; i <= n; i++ {
			t := e.combatant(e.targets[i])
			if t == b {
				bi = i
			}
			if t.MaxLevel == 0 {
				cnt++
			}
		}
		if byte(cnt) > a.Attacks[0] {
			if byte(cnt) > acb.Sweeps {
				cnt = int(acb.Sweeps)
			}
			e.combatName(a, "sweeps", 0xA, true)
			if e.combatant(e.targets[1]) != b {
				e.targets[bi] = e.targets[1] // (bi 0 when b was not found: DS:6DE4)
				e.targets[1] = byte(e.combatIndex(b))
			}
			for i := 1; i <= n; i++ {
				if cnt > 0 {
					t := e.combatant(e.targets[i])
					if t.MaxLevel == 0 {
						e.countAttack(a, t)
						a.Attacks[0] = 1
						e.fullAttack(a, t, 0, nil)
						cnt--
					}
				}
			}
			res = true
		}
	}
	return res
}

// mayAct = O0C9:1144 (t, c) bool: may t act against c? Affect group 1 of c
// can veto (DS:681E).
func (e *Engine) mayAct(t, c *Character) bool {
	if c == nil {
		return false
	}
	if t == c {
		return true
	}
	e.rv.unseen = 0
	e.affectGroup(c, grpAttackerExtra)
	return e.rv.unseen == 0
}

// attackSlots = O0C9:11DD (att, vic, w, var done): the attacker's attack slots against the
// victim; done is 1 when the attacker has used all its attacks.
func (e *Engine) attackSlots(a, b *Character, w int, it *Item) (done bool) {
	acb := a.Combat
	done = false
	e.rv.hits[1], e.rv.hits[2] = 0, 0
	e.shots6E2C[1], e.shots6E2C[2] = 0, 0
	any, stop := false, false
	e.rv.damage = 0
	acb.Attacked = 1
	if e.isHelpless(b) {
		e.PlaySound(sndBump)
		for acb.AttackSlot > 0 && a.Attacks[acb.AttackSlot-1] == 0 {
			acb.AttackSlot--
		}
		e.shots6E2C[acb.AttackSlot]++
		e.attackText(a, b, attackSlay, (int(b.HP)+5)&0xFF, 1, 1)
		e.removeInvisibility(a)
		for s := 1; s <= 2; s++ {
			a.Attacks[s-1] = 0
		}
		any = true
		done = true
	} else {
		if a.Slots[0] != nil && (b.Size > 0x80 || b.Size&7 > 1) {
			typ := a.Slots[0].Type
			a.Dice[0] = e.baseItem(typ, 2)
			a.Sides[0] = e.baseItem(typ, 3)
			a.Bonus[0] = byte(int(int8(a.Bonus[0])) - int(int8(e.baseItem(typ, 0xB))) + int(int8(e.baseItem(typ, 4))))
		}
		e.ApplyItems(b)
		e.affectGroup(b, grpArmour)
		hand := 0
		var need byte
		if e.canBackstab(a, b) {
			need = b.AC[1] - 4
		} else {
			if b.Combat.Attackers > 1 && byte(e.directionTo(a, b)) == b.Combat.Facing && b.Combat.Flank > 4 {
				w = 1
			}
			if w != 0 {
				hand = 1
			}
			need = b.AC[hand]
		}
		bonus := int(need)
		e.rangeBand(a, b, &bonus)
		need = byte(bonus)
		mode := attackPlain
		if w != 0 {
			mode = attackFromBehind
		}
		if e.canBackstab(a, b) {
			mode = attackBackstab
		}
		for s := int(acb.AttackSlot); s >= 1; s-- {
			for a.Attacks[s-1] != 0 && !stop {
				a.Attacks[s-1]--
				acb.AttackSlot = byte(s)
				e.shots6E2C[s]++
				if e.AttackHits(a, b, int(need)) || e.isHelpless(b) {
					e.rv.hits[s]++
					e.PlaySound(sndBump)
					any = true
					e.rollDamage(a, b, s)
					if b.Okay != 0 {
						e.attackText(a, b, mode, int(e.rv.damage), int(e.rv.damage), 1)
					}
					if b.Okay != 0 && e.rv.damage > 0 {
						e.affectGroup(a, s+1)
					}
					if b.Okay == 0 {
						stop = true
						if a.Slots[0] != nil && a.Slots[0].Type == itemTypeOneKill {
							a.Attacks[s-1] = 0
						}
					}
					if a.Okay == 0 { // a fire shield or the like killed the attacker
						a.Attacks[s-1] = 0
					}
				}
			}
		}
		if !any {
			e.PlaySound(sndMiss)
			e.attackText(a, b, mode, 0, 0, 0)
		}
		done = true
		for s := 1; s <= 2; s++ {
			if a.Attacks[s-1] > 0 {
				done = false
			}
		}
		acb.Sweeps = 0
	}
	if a.Okay == 0 {
		done = true
	}
	if done {
		e.endTurn(a)
	}
	return
}

// Weapons that animate their attack.
const (
	itemTypeAnimatedA = 28
	itemTypeAnimatedB = 29
)

// itemTypeOneKill: a weapon of this type stops attacking once its victim
// goes down.
const itemTypeOneKill = 66

// countAttack = O0C9:1638 (att, vic): the victim counts one more attack and turns towards
// the attacker (the turned amount accumulates in +0x12).
func (e *Engine) countAttack(att, vic *Character) {
	cb := vic.Combat
	cb.Attackers++
	d := (e.directionTo(vic, att) - int(cb.Facing) + 8) % 8
	if d > 4 {
		d = 8 - d
	}
	cb.Flank = byte((int(cb.Flank) + d) % 8)
}

// fullAttack = O0C9:16C7 (att, vic, flag, item, var done): a full attack of att on vic
// (turning both, then the slots of attackSlots). item is a thrown item that is
// spent; done is set when the attacker's turn ends.
func (e *Engine) fullAttack(a, b *Character, flag int, it *Item) (done bool) {
	e.showMoves = true
	e.panelDirty = true
	e.roundLimit = e.combatRound + combatRoundsWithoutAttack
	var dir byte
	if b.Combat.Attackers < 2 && flag == 0 {
		dir = byte(e.directionTo(b, a))
		b.Combat.Facing = dir
	} else if e.combatOnScreen(b, false) {
		dir = b.Combat.Facing
		if flag == 0 {
			b.Combat.Facing = (dir + 4) % 8
		}
	}
	if e.combatOnScreen(b, false) {
		e.combatTurn(b, dir, 0, true)
		e.combatRedrawMap()
	}
	dir = byte(e.directionTo(a, b))
	e.combatPanel(a)
	e.combatTurn(a, dir, 1, false)
	a.Combat.Target = b
	e.Delay(0x64)
	if it != nil {
		e.attackAnimation(a, b, it)
	}
	if w := a.Slots[0]; w != nil && (w.Type == itemTypeAnimatedA || w.Type == itemTypeAnimatedB) {
		e.attackAnimation(a, b, w)
	}
	done = true
	if a.Attacks[0] > 0 || a.Attacks[1] > 0 {
		saved := e.partySel
		for i, c := range e.party {
			if c == a {
				e.partySel = i
			}
		}
		done = e.attackSlots(a, b, flag, it)
		if it != nil {
			if it.Count > 0 {
				it.Count -= e.shots6E2C[1]
			}
			if it.Count == 0 {
				// a thrown weapon lands after the ones thrown before it
				if e.throwable(a) && it.Spells[1] != affectSpiritualHammer {
					cp := *it
					n := &cp
					at := len(e.treasureItems)
					for i, t := range e.treasureItems {
						if t == e.thrownItem {
							at = i + 1
							break
						}
					}
					e.treasureItems = append(e.treasureItems[:at], append([]*Item{n}, e.treasureItems[at:]...)...)
					n.Readied = 0
					e.loseItem(a, it)
					e.thrownItem = n
				} else {
					e.loseItem(a, it)
				}
			}
		}
		e.ApplyItems(a)
		e.partySel = saved
	}
	if done {
		done = e.endTurn(a)
	}
	if e.combatOnScreen(a, false) {
		e.combatTurn(a, a.Combat.Facing, 1, true)
		e.combatTurn(a, a.Combat.Facing, 0, false)
	}
	return
}

// rangeBand = O0C9:1A82 (att, vic, var need): the attacker's range band: every full
// weapon range band beyond the first adds 2, then 3, to the number to hit.
func (e *Engine) rangeBand(a, b *Character, need *int) {
	d := e.rangeTo(a, b)
	f := d
	if e.rangedWeapon(a) {
		f = (int(e.baseItem(a.Slots[0].Type, 0xC)) - 1) / 3
	}
	if d-f > 0 {
		d -= f
		*need += 2
	}
	if d-f > 0 {
		d -= f
		*need += 3
	}
}
