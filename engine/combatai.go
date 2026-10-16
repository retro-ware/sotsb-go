package engine

// The computer-controlled turn (unit O090): monsters, and party members
// under computer control or fleeing. Called from the combat loop.

// Computer turn limits and values.
const (
	delayWaiting        = 20 // the Delay of one who waits for the end of the round
	aiMaxTries          = 20 // attack tries before the computer gives up
	aiPasses            = 7  // spell choice passes, from the strongest wish down
	aiPicksPerPass      = 3  // memorised spells tried each pass
	movePatternCount    = 6
	stepTries           = 5 // directions a move pattern tries
	noStepYet           = 8 // lastStep before the first step
	cloudkillLevelProof = 7 // from this many hit dice up the computer walks into Cloudkill
	moraleLimit         = 102
	statusSurrendered   = statusUnconscious // what KillCharacter makes of one who surrenders
	keyMagicToggle      = '2'
	keyEveryoneWaits    = ' '
	keyGodIntervene     = '-'
)

// anyRange: pickAttackTarget and buildNear look at every distance.
const anyRange = 255

// computerTurn = O090:000F (char): one computer turn. It picks the move
// pattern, may flee, uses an item or a spell, then moves and attacks until
// the turn is used.
func (e *Engine) computerTurn(c *Character) {
	done := e.aiKeys(c)
	e.ClearRow24()
	e.clearCombatText()
	if c.Okay == 0 {
		done = e.endTurn(c)
	}
	ci := c.Combat
	pattern := ci.Pattern
	if pattern == 0 || pattern > 4 || e.Dice(1, 4) == 1 {
		pattern = byte(e.Dice(1, 8))
		if pattern == 8 {
			pattern = byte(e.Dice(1, 2) + 4)
		} else {
			pattern = byte(e.Dice(1, 4)) // the first roll is thrown away
		}
	}
	ci.Pattern = pattern
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
		e.endTurn(c)
		return
	}
	if ci.Spell > 0 {
		e.CastSpell(ci.Spell, 1, 1, &done)
		e.endTurn(c)
		return
	}
	if e.aiCastKnown(c) {
		return
	}
	e.aiPickWeapon(c)
	done = e.aiKeys(c)
	for !done {
		if e.pickAttackTarget(c, anyRange, 1, 0) && int8(ci.Delay) > 0 && c.Okay != 0 {
			done = e.aiAttackLoop(c)
		} else {
			done = e.aiGiveUp(c)
		}
	}
}

// Spell table fields read by the computer's spell choice.
const (
	spellSaveTypeField = 9  // the saving throw kind against it
	spellAIPowerField  = 13 // the highest pass (7 down to 1) at which the computer uses it
	spellAIRadiusField = 15 // area radius; 0: one target only
)

// spellSaveNone: a spell whose save does nothing harms its own side freely.
const spellSaveNone = 1

// Save bonus of those caught by the computer's area spell: the party's
// members save worse, monsters better.
const (
	aiAreaSaveParty    = -2
	aiAreaSaveMonsters = 8
)

// canCast = O090:02F0 (char, spell, pass): may the computer cast the spell
// at this pass: the pass fits, and an area spell must not catch one of
// its own side who fails the save.
func (e *Engine) canCast(c *Character, spell, pass int) bool {
	s := spellTable[spell]
	if int(s[spellAIPowerField]) < pass {
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
	// O090:01F0: does the area at x, y catch one of the caster's side who
	// fails the save.
	catchesOwn := func(x, y int) bool {
		caught := false
		bonus := aiAreaSaveMonsters
		if e.selected().Side == sideParty {
			bonus = aiAreaSaveParty
		}
		e.buildNear(x, y, int(s[spellAIRadiusField]), anyRange, 1)
		foes := opposingSide(e.selected())
		for _, near := range e.near.list {
			w := e.combatant(near.index)
			if w.Side != foes && s[spellSaveField] != spellSaveNone &&
				!e.SavingThrow(w, int(s[spellSaveTypeField]), bonus) {
				caught = true
			}
		}
		return caught
	}
	for i := 1; i <= n; i++ {
		p := e.cpos[e.targets[i]]
		if catchesOwn(int(p.x), int(p.y)) {
			return false
		}
	}
	return true
}

// Item spells past this number are numbered 23 higher than the spells.
const (
	itemSpellShiftFrom = 56
	itemSpellShift     = 23
)

// aiCastFromItem = O090:03E9 (char): use a readied item that casts a spell
// the computer may cast; true when one was used.
func (e *Engine) aiCastFromItem(c *Character) bool {
	if c.Combat.CanUse == 0 || e.rv.side[opposingSide(c)] <= 0 || wordAt(e.area, 2*areaWordNoMagic) != 0 {
		return false
	}
	var found *Item
	pass := aiPasses
	rounds := e.Dice(1, aiPasses)
	for round := 1; round <= rounds; round++ {
		for _, it := range c.Items {
			if found != nil {
				break
			}
			spell := it.Spells[1]
			if !e.isScroll(it) && it.Spells[2] < itemPowerFlag && it.Readied != 0 && spell > 0 {
				if spell > itemSpellShiftFrom {
					spell -= itemSpellShift
				}
				if e.canCast(c, int(spell), pass) {
					found = it
				}
			}
		}
		pass--
	}
	if found == nil {
		return false
	}
	e.useItem(found)
	return true
}

// aiCastKnown = O090:0544 (char): cast one of the memorised spells the
// computer may cast, trying weaker wishes each pass; true when the turn
// was used.
func (e *Engine) aiCastKnown(c *Character) bool {
	ci := c.Combat
	if ci.HasSpells == 0 || ci.CanCast == 0 || e.FindAffect(c, affectFeebleminded) != nil {
		return false
	}
	var known []byte
	for i := 1; i < len(c.Memorized); i++ {
		if c.Memorized[i] > 0 {
			known = append(known, c.Memorized[i])
		}
	}
	if len(known) == 0 {
		ci.HasSpells = 0
		return false
	}
	found := 0
	pass := aiPasses
	rounds := e.Dice(1, aiPasses)
	if (c.Control > controlHired || e.computerMagic) && e.rv.side[opposingSide(c)] > 0 {
		for round := 1; round <= rounds && found == 0; round++ {
			for pick := 1; pick <= aiPicksPerPass && found == 0; pick++ {
				spell := int(known[e.Dice(1, len(known))-1])
				if e.canCast(c, spell, pass) {
					found = spell
				}
			}
			pass--
		}
	}
	if found == 0 {
		return false
	}
	var used bool
	e.spellMenuTargets(found, 1, &used)
	return used
}

// aiStepTurns: for move pattern p (1..6) and try i (1..5), the turn from
// the wanted direction at aiStepTurns[p][i-1] (DS:032C).
var aiStepTurns = [movePatternCount + 1][stepTries]byte{
	{},
	{8, 7, 6, 1, 2},
	{8, 1, 2, 7, 6},
	{7, 1, 8, 6, 2},
	{1, 7, 8, 2, 6},
	{8, 7, 6, 5, 4},
	{8, 1, 2, 3, 4},
}

func aiStepDir(pattern byte, try, dir int) int {
	return (int(aiStepTurns[pattern][try-1]) + dir) % 8
}

// groundImpassable: the movement cost of ground nobody can enter.
const groundImpassable = 255

// aiCanStep = O090:06BB (char, try, dir, var blocked): can the combatant
// take the try-th step of its pattern towards dir; blocked is set when the
// square is off the map. A cloud it is not proof against costs all its
// moves, unless it holds its breath (a save) in a stinking cloud or has
// the hit dice for Cloudkill.
func (e *Engine) aiCanStep(c *Character, try, dir int) (ok, blocked bool) {
	ci := c.Combat
	d := aiStepDir(ci.Pattern, try, dir)
	who, ground, stinking, poison := e.groundAt(c, d)
	if ground == 0 {
		return false, true
	}
	if e.groundInfo(ground)[0] == groundImpassable {
		return false, false
	}
	cost := stepCost(e.groundInfo(ground)[0], d)
	if who != 0 || cost >= int(ci.Moves) {
		return false, false
	}
	has := func(a byte) bool { return e.FindAffect(c, a) != nil }
	if stinking && !has(affectCoughing) && !has(affectMinorGlobe) && !has(affectGlobe) &&
		!has(affectPoisonImmune) && ci.Turned == 0 && !e.SavingThrow(c, saveVsParalysis, 0) {
		cost = int(ci.Moves) + 1
	}
	if poison && int8(c.MaxLevel) < cloudkillLevelProof && !has(affectPoisonImmune) && ci.Turned == 0 {
		cost = int(ci.Moves) + 1
	}
	return int(ci.Moves) >= cost, false
}

// aiMoveStep = O090:08CE (char): show the movement left and take one step
// towards the target (or away from the party's side when fleeing).
func (e *Engine) aiMoveStep(c *Character) {
	ci := c.Combat
	e.DisplayString(0, statusRow, colourGreen, "Move/Attack, Move Left = "+itoa(int(ci.Moves/2))+" ")
	if e.aiKeys(c) {
		return
	}
	// A hired one moves only while its morale holds against the party's; a
	// magic-user who is not fleeing moves only with armour readied (or as
	// a dual-classed human using the old class).
	willing := c.Control <= controlHired || int(e.partyMorale) <= int(e.rv.morale)+e.Dice(1, 100) || c.Side == sideMonsters
	armoured := ci.Fleeing != 0 || c.Slots[slotArmour] != nil || c.Class != clsMage || oldClassUsable(c)
	if ci.Moves/2 == 0 || int8(ci.Delay) <= 0 || !willing || !armoured {
		e.aiGiveUp(c)
		return
	}
	var dir int
	if ci.Fleeing == 0 {
		dir = e.directionTo(c, ci.Target)
	} else {
		ci.Pattern = byte(e.Dice(1, 2))
		facing := e.partyFacing()
		dir = facing - (facing+2)%4/2
		if c.Side == sideParty {
			dir += 4
		}
		dir %= 8
	}
	done := false
	try := 1
	for try <= stepTries && !done {
		ok, blocked := e.aiCanStep(c, try, dir)
		if ok {
			break
		}
		if ci.Fleeing != 0 && blocked {
			done = e.fleeBattle(c)
		} else {
			try++
		}
	}
	if done {
		ci.Moves = 0
		ci.Fleeing = 0
		e.endTurn(c)
		return
	}
	step := 0 // the original reads the table one past the last try
	if try <= stepTries {
		step = aiStepDir(ci.Pattern, try, dir)
	}
	if try > stepTries || e.lastStep == (step+4)%8 {
		e.stuckCount++
		ci.Pattern = ci.Pattern%movePatternCount + 1
		if e.stuckCount > 1 {
			ci.Target = nil
			if e.stuckCount > 2 {
				ci.Moves = 0
				done = true
			} else if !e.pickAttackTarget(c, anyRange, 1, 0) {
				done = e.aiGiveUp(c)
			}
		}
	}
	if try <= stepTries {
		e.lastStep = step
	} else {
		done = true
	}
	if done {
		return
	}
	e.showMoves = e.aiEngaged || e.combatOnScreen(c, false) || c.Side == sideParty
	e.combatTurn(c, byte(step), 0, false)
	e.leaveAttacks(c, int(ci.Facing))
	if c.Okay == 0 {
		e.endTurn(c)
		return
	}
	if ci.Moves > 0 {
		e.stepSquare(c, int(ci.Facing))
	}
	if c.Okay == 0 || e.isHelpless(c) {
		e.endTurn(c)
	}
	e.inPoisonCloud(c, true)
}

// aiAttackLoop = O090:0CD6 (char): move, pick a target and attack until
// the turn is over; true when it is.
func (e *Engine) aiAttackLoop(c *Character) bool {
	ci := c.Combat
	e.lastStep, e.stuckCount, e.stuckSteps = noStepYet, 0, 0
	tries := 0
	going, stop := true, false
	e.affectGroup(c, grpAI)
	if c.Side == sideParty && e.bandage(true) {
		ci.Delay = 0 // a party member under the computer bandages the dying first
	}
	if ci.Delay == 0 {
		going = false
	}
	for !stop && going {
		for ci.Fleeing != 0 && ci.Moves > 0 && int8(ci.Delay) > 0 && int8(ci.Delay) < delayWaiting {
			e.aiMoveStep(c)
		}
		if ci.Delay == 0 || ci.Delay == delayWaiting {
			going = false
			continue
		}
		tries++
		if tries > aiMaxTries {
			stop = true
			going = !e.aiGiveUp(c)
		}
		e.aiEngaged = false
		reach := 1
		if w := c.Slots[slotWeapon]; w != nil {
			reach = int(e.baseItem(w.Type, itemRange)) - 1
		}
		if reach <= 0 {
			reach = 1
		}
		target := ci.Target
		if target != nil && (target.Okay == 0 || target.Side == sideParty) {
			target = nil
		}
		if target != nil && e.mayAct(c, target) {
			tx, ty := e.combatX(target), e.combatY(target)
			distance := reach
			if e.lineOfSight(&distance, &tx, &ty, e.combatX(c), e.combatY(c)) && distance/2 <= reach {
				e.aiEngaged = true
			}
		}
		if !e.aiEngaged {
			n := e.nearEnemies(c, reach)
			if n == 0 {
				if e.pickAttackTarget(c, anyRange, 0, 0) {
					e.aiMoveStep(c)
				} else {
					stop = e.aiGiveUp(c)
				}
			} else {
				target = e.combatant(e.targets[e.Dice(1, n)])
				if e.rangedWeapon(c) && !e.throwable(c) && e.nearEnemies(c, 1) > 0 {
					e.aiPickWeapon(c)
					stop = true
					continue
				}
				if e.rangeTo(c, target) == 1 || e.mayAct(c, target) {
					e.aiEngaged = true
				}
			}
		}
		if !e.aiEngaged {
			continue
		}
		e.combatCentre(e.combatX(c), e.combatY(c), 2, e.directionTo(c, target))
		if e.sweepAttack(c, target) {
			stop = e.endTurn(c)
			continue
		}
		e.countAttack(c, target)
		var ammo *Item
		if e.rangedWeapon(c) {
			e.aiEngaged = e.ammoFor(c, &ammo)
			if e.throwable(c) && e.rangeTo(c, target) == 1 {
				ammo = nil
			}
		}
		stop = e.fullAttack(c, target, 0, ammo)
		if stop {
			going = false
		} else if target.Okay == 0 {
			stop = true
		}
	}
	return !going
}

// aiGiveUp = O090:1125 (char): the turn ends; a combatant that cannot go
// on is done, otherwise it stands guard.
func (e *Engine) aiGiveUp(c *Character) bool {
	e.clearCombatText()
	if e.isHelpless(c) || e.rangedWeapon(c) || c.Combat.Delay == 0 {
		return e.endTurn(c)
	}
	return e.guard(c)
}

// aiKeys = O090:1197 (char): keys during the computer turn: '2' toggles
// the computer's magic, space ends the computer's control of the party
// (this one then waits), '-' calls on the gods.
func (e *Engine) aiKeys(c *Character) bool {
	waits := false
	if e.keyWaiting() {
		k := e.ReadKeyByte()
		if k == 0 {
			k = e.ReadKeyByte()
		}
		switch k {
		case keyMagicToggle:
			e.computerMagic = !e.computerMagic
			if e.computerMagic {
				e.printMessage("Magic On")
			} else {
				e.printMessage("Magic Off")
			}
		case keyEveryoneWaits:
			for _, p := range e.party {
				if p.Control <= controlHired && p.Status != statusAnimated {
					p.Auto = 0
				}
			}
			if c.Auto == 0 {
				c.Combat.Delay = delayWaiting
				waits = true
			}
		case keyGodIntervene:
			e.godIntervene()
		}
	}
	e.ClearKeyboard()
	return waits
}

// gameWordPartyMorale: the game word with the party's morale (100 - it is
// what the computer's side must beat to stay).
const gameWordPartyMorale = 710

// aiFleeCheck = O090:12AC (char): a forced flight, or a morale check that
// makes a hired one or a monster flee or surrender.
func (e *Engine) aiFleeCheck(c *Character) bool {
	ci := c.Combat
	ci.Fleeing = 0
	if ci.Turned != 0 {
		ci.Fleeing = 1
		e.combatName(c, "is forced to flee", 10, true)
		return false
	}
	if c.Control <= controlHired {
		return false
	}
	e.rv.morale = c.Control & controlHired * 2
	if e.rv.morale > moraleLimit {
		e.rv.morale = 0
	}
	e.affectGroup(c, grpMorale)
	if int(e.rv.morale) >= 100-int(c.HP)*100/int(c.MaxHP) && e.rv.morale != 0 {
		return false
	}
	e.rv.morale = e.partyMorale
	e.affectGroup(c, grpMorale)
	if int(e.rv.morale) < 100-int(wordAt(e.game, 2*gameWordPartyMorale)) || e.rv.morale == 0 || c.Side == sideParty {
		if e.maxPartyMove(c) <= e.turnMoveRate(c)/2 {
			ci.Fleeing = 1
			return false
		}
		if c.Stats[statInt].Value > 5 {
			e.KillCharacter(c, statusSurrendered, "Surrenders")
			return e.endTurn(c)
		}
	}
	return false
}

// Base item flags and values read when the computer chooses weapons.
const (
	itemFlagLauncher        = 8 // needs ammunition (itemUsesArrows or itemUsesBolts)
	itemFlagThrown          = 16
	itemFlagStrength        = 4
	itemFlagDexterity       = 2
	itemFlagsThrownStrength = itemFlagThrown | itemFlagStrength
	itemFlagsSling          = itemFlagLauncher | itemFlagDexterity
	itemSlotWeapon          = 0
	itemSlotShield          = 1
	slotShield              = 1
	alignmentPower          = itemPowerFlag + 4 // burns those not of the item's alignment
	alignmentMask           = 15
	weaponSpellNever        = 83 // a weapon casting this is never chosen
)

// weaponScore = O090:1439 (char, item): how good the item is as the
// character's weapon (0 = unusable).
func (e *Engine) weaponScore(c *Character, it *Item) int {
	field := func(k int) byte { return e.baseItem(it.Type, k) }
	score := field(itemDiceNormal) * field(itemSidesNormal)
	if int8(it.Plus) > 0 {
		score += it.Plus * 8
	}
	if bonus := field(itemBonusNormal); int8(bonus) > 0 {
		score += bonus * 2
	}
	if field(itemFlags)&itemFlagLauncher != 0 {
		score += (field(itemAttacks) - 1) * 2
	}
	if field(itemHands) <= 1 {
		score += 3
	}
	if int(field(itemHands))+int(c.Hands) >= 3 ||
		it.Spells[2] == alignmentPower && it.Spells[1]&alignmentMask != c.Alignment ||
		it.Spells[1] == weaponSpellNever || it.Cursed != 0 {
		score = 0
	}
	return int(score)
}

// aiPickWeapon = O090:1544 (char): ready the best melee or missile weapon
// and the best shield.
func (e *Engine) aiPickWeapon(c *Character) {
	field := func(it *Item, k int) byte { return e.baseItem(it.Type, k) }
	if w := c.Slots[slotWeapon]; w != nil {
		c.Hands -= field(w, itemHands)
	}
	var missile, melee, shield *Item
	missileScore, shieldScore := byte(1), byte(0)
	meleeScore := c.BaseSides[0] * c.BaseDice[0]
	if int8(c.BaseBonus[0]) > 0 {
		meleeScore += c.BaseBonus[0] * 2
	}
	for _, it := range c.Items {
		usable := field(it, itemClassFlags)&c.ClassMask != 0
		if field(it, itemSlot) == itemSlotWeapon && usable {
			score := byte(e.weaponScore(c, it))
			flags := field(it, itemFlags)
			if flags&(itemFlagLauncher|itemFlagThrown) != 0 && score > missileScore {
				missile, missileScore = it, score
			}
			if flags&itemFlagLauncher == 0 && score > meleeScore {
				melee, meleeScore = it, score
			}
		}
		if field(it, itemSlot) == itemSlotShield && usable {
			score := byte(0)
			if int8(it.Plus) >= 0 {
				score = it.Plus + 1
			}
			if score > shieldScore {
				shield, shieldScore = it, score
			}
		}
	}
	farReaching := missile != nil && field(missile, itemRange) > 1 &&
		field(missile, itemFlags)&itemFlagsThrownStrength == itemFlagsThrownStrength
	var flags byte
	var ammo *Item
	if missile != nil {
		flags = field(missile, itemFlags)
		if flags&itemFlagThrown != 0 {
			ammo = missile
		}
		if flags&itemFlagLauncher != 0 {
			if flags&itemUsesArrows != 0 {
				ammo = c.Slots[slotArrows]
			}
			if flags&itemUsesBolts != 0 {
				ammo = c.Slots[slotBolts]
			}
		}
	}
	armed := ammo != nil || flags == itemFlagsSling
	chosen := melee
	if missile != nil && missileScore > meleeScore/2 && armed && (farReaching || e.nearEnemies(c, 1) == 0) {
		chosen = missile
	}
	changed := false
	if w := c.Slots[slotWeapon]; w == nil || w != chosen && w.Cursed == 0 {
		if w != nil {
			e.readyItem(w)
		}
		e.ApplyItems(c)
		if s := c.Slots[slotShield]; s != nil && s.Cursed == 0 {
			c.Hands -= field(s, itemHands)
		}
		if chosen != nil {
			e.readyItem(chosen)
		}
		changed = true
	}
	e.ApplyItems(c)
	e.setSpeed(c)
	shieldFree := true
	if s := c.Slots[slotShield]; s != nil && (s == shield || s.Cursed != 0) {
		shieldFree = false
	}
	if c.Hands > 2 {
		if s := c.Slots[slotShield]; s == nil || s.Cursed != 0 {
			if chosen != nil { // the original passes it even when there is none
				e.readyItem(chosen)
			}
		} else {
			e.readyItem(s)
		}
		changed = true
	} else if c.Hands < 2 && shieldFree {
		if s := c.Slots[slotShield]; s != nil {
			e.readyItem(s)
		}
		e.ApplyItems(c)
		if shield != nil {
			e.readyItem(shield)
		}
		changed = true
	}
	e.ApplyItems(c)
	if changed {
		e.panelDirty = true
		e.combatPanel(c)
	}
}
