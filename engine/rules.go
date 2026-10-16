package engine

// Rules: affects, to-hit rolls, saving throws, damage and healing, ported
// from ovr25 and the START helpers it leans on (O0ED:1788 ... 2637).

// CombatInfo is the 0x16-byte block at char+0x108 (allocated by ovr11).
type CombatInfo struct {
	Spell      byte       // +0x00 spell being cast (lost when hit)
	CanCast    byte       // +0x01
	CanUse     byte       // +0x02 may use an item
	Delay      byte       // +0x03 signed initiative; 0x14 = delayed
	AttackSlot byte       // +0x04 attack slot in use (1, 2)
	Sweeps     byte       // +0x05 sweeps left against weak foes
	Moves      byte       // +0x06 half squares left
	Guarding   byte       // +0x07
	Attacked   byte       // +0x08 has attacked this turn
	Facing     byte       // +0x09 0..7
	Target     *Character // +0x0A
	Dying      byte       // +0x0E rounds of bleeding left
	Attackers  byte       // +0x0F times attacked this round
	Turned     byte       // +0x10 turned by a cleric
	TurnUsed   byte       // +0x11 has tried to turn undead
	Flank      byte       // +0x12 accumulated turn from attackers
	Extra      byte       // +0x13 not one of the saved party
	Fleeing    byte       // +0x14
	Pattern    byte       // +0x15 AI move pattern, row of DS:02AE
}

// Rule globals (DS:6816..682A).
type ruleVars struct {
	save      byte    // DS:6816 saving throw total
	affVal    byte    // DS:6817 affect being added (0 = blocked)
	damage    byte    // DS:6818
	dmgType   byte    // DS:6819 1 fire 2 cold 4 electricity 8 magic 0x10 acid
	diceN     byte    // DS:681C
	hitRoll   int8    // DS:6822
	saveType  byte    // DS:682A
	cure      byte    // DS:6FB6 set while the temple removes affects
	saveMade  bool    // DS:6FC2 the last saving throw succeeded
	areaSpell byte    // DS:6FB7 an area spell: no image is lost
	side      [2]int8 // DS:6814 standing members per side
}

// callAffect = O0E3:00C7 (type, char, affect, flag): with DS:759C set
// the readied item's handler runs instead.
func (e *Engine) callAffect(typ byte, c *Character, a *Affect, flag byte) {
	if e.itemCall {
		e.itemHandler(c, e.callItem, flag)
		return
	}
	if int(typ) < len(affectHandlers) && affectHandlers[typ] != nil {
		affectHandlers[typ](e, c, a, flag)
	}
}

// FindAffect = O0ED:2345 (out, type, char): first affect of the type.
func (e *Engine) FindAffect(c *Character, typ byte) *Affect {
	for _, a := range c.Affects {
		if a.Type == typ {
			return a
		}
	}
	return nil
}

// RemoveAffect = O0E3:010A (char, type, affect): affect nil means the first
// one of the type. Its handler runs first when the node asks for it; losing
// a charisma or strength affect recomputes that stat.
func (e *Engine) RemoveAffect(c *Character, typ byte, a *Affect) {
	if a == nil {
		a = e.FindAffect(c, typ)
	}
	if a == nil {
		return
	}
	if a.Call != 0 {
		e.callAffect(typ, c, a, 1)
	}
	for i, x := range c.Affects {
		if x == a {
			c.Affects = append(c.Affects[:i], c.Affects[i+1:]...)
			break
		}
	}
	switch typ {
	case affectCharisma:
		e.statAffects(c, statCha)
	case affectEnlarge, affectStrengthSpell, affectGiantStrength:
		e.statAffects(c, statStr)
	}
}

// RemoveEffect is the old name of RemoveAffect used by the clock code.
func (e *Engine) RemoveEffect(c *Character, a *Affect) { e.RemoveAffect(c, a.Type, a) }

// AddAffect = O0E3:13F0 (char, type, time, data, call): append a node.
func (e *Engine) AddAffect(c *Character, typ byte, time int, data, call byte) {
	a := &Affect{affectRecord: affectRecord{Type: typ, Time: uint16(time), Data: data, Call: call}}
	c.Affects = append(c.Affects, a)
}

// auraAffect = O0E3:027A's set: affects that also work on everyone near the
// one who has them (prayer, chant ...).
func auraAffect(t byte) bool { return t == 0x15 || t == 0x2D || t == 0x2E || t == 0x31 }

// checkAffect = O0E3:027A (char, type): run the handler (flag 0) when the
// character has the affect, or for an aura affect when anyone in range
// (anyone at all outside combat) has it.
func (e *Engine) checkAffect(c *Character, typ byte) {
	a := e.FindAffect(c, typ)
	found := a != nil
	if !found && auraAffect(typ) {
		saved := e.nearList // DS:671B, 0xD8 bytes
		for _, p := range e.party {
			if found {
				break
			}
			pa := e.FindAffect(p, typ)
			if pa == nil {
				continue
			}
			a = pa
			if e.gameState != 5 {
				found = true
				continue
			}
			r := 1
			if typ == 0x31 {
				r = 6
			}
			e.nearCombatants(p, e.combatX(p), e.combatY(p), r)
			me := e.combatIndex(c)
			for _, n := range e.near.list {
				if int(n.index) == me {
					found = true
				}
			}
		}
		e.nearList = saved
	}
	if found {
		e.callAffect(typ, c, a, 0)
	}
}

// affectGroups = the lists of O0E3:0414: which affects take part in each
// kind of check (the group number is the caller's).
var affectGroups = [][]byte{
	grpAttackerExtra: {affectStumbling, affectInvisible, affectBlur, affectHiddenFromKind19},
	grpOnHit: {affectBurningTouch, affectAcidTouch, affectDispelEvil, affectEngulfs, affectHugs,
		affectParalysingGaze, affectFireTouch},
	grpOnDamage: {affectPoisonSting, affectPoisonSting4, affectPoisonSting2, affectParalysingTouch,
		affectPoisonStingMinus2, affectBurningTouch, affectRays},
	grpDamageBonus: {affectQuarterLessDamage, affectKindBane, affectBurns, affectVsKind3, affectVsKind8,
		affectRangerVsGiants},
	grpWeaponDamage: {affectMirrorImage, affectAvoidPlainWeapons, affectAvoidRanged60, affectAvoidsBows,
		affectRegenerates, affectEdgedHalved, affectMagicWeaponsHalved, affectNeedsMagicWeapon,
		affectRangedHalved, affectSlingStones, affectMagicWeaponsOnly, affectHalfDamage,
		affectSaveVsElements, affectArrowsScratch, affectSlainByArrow, affectZapsAttackers},
	grpDamage: {affectFireReduced, affectFireDamageReduced, affectResistCold, affectResistFire,
		affectMagicResist50, affectMagicResist15, affectFireImmune, affectElectricityHalved,
		affectColdHalved, affectShield, affectFireHalved, affectRegenerates, affectMirrorImage,
		affectColdImmune, affectUnaffectedByType32, affectSaveVsElements, affectElectricityHeals,
		affectMagicImmune, affectNoMindAffects, affectElectricityImmune, affectMinorGlobe},
	grpTurn: {affectSleep, affectParalyzed, affectHeld, affectNausea, affectSticksToSnakes,
		affectStunned, affectRooted},
	grpRound: {affectBacksUp, affectSaveVsElements, affectFirstStrike, affectFlicker95, affectFlicker},
	grpAddAffect: {affectMagicResist50, affectMagicResist15, affectElfResistSleep, affectNoCharmHold,
		affectNoParalysis, affectColdImmune, affectNoPoisonParalysis, affectFireImmune,
		affectHalfElfResistance, affectNoCharmHoldSleep, affectMinorGlobe, affectMagicImmune},
	grpAttacker: {affectBless, affectCurse, affectDazzled, affectHampered, affectPrayer, affectKindBane,
		affectGnomeVsManSized, affectDwarfVsOrc, affectVsKind3, affectVsKind8},
	grpArmour: {affectDazzled, affectShield, affectProtFromEvil, affectProtFromGood,
		affectProtFromEvil10, affectProtFromGood10, affectCoughing, affectArmourBonus},
	grpSave: {affectProtFromEvil, affectProtFromGood, affectResistCold, affectShield, affectResistFire,
		affectDazzled, affectHampered, affectProtFromEvil10, affectProtFromGood10, affectPrayer,
		affectFireDamageReduced, affectNoPoisonParalysis, affectNoCharmHoldSleep, affectConSavingBonus,
		affectFireBody, affectIceBody},
	grpDown: {affectBacksUp, affectRegrows, affectVsKind3},
	grpAI:   {83, 88, affectSpitsAcid, 86, affectRays, 90, 126, 128, 131, 132, affectEngulfing},
	grpUse:  {affectSilenced, affectCoughing, affectCharmed, affectSuffocating, affectBerserk},
	grpDefender: {affectInvisible, affectBlur, affectStumbling, affectVsGiants, affectGnomeExtra,
		affectFirstStrike, affectWardEnchanted},
	grpMorale:          {affectBless, affectCurse, affectCharmed},
	grpSpeed:           {affectHaste, affectSlow, affectEngulfed},
	grpRest:            {affectRegrowth, affectSpiritualHammer, affectFlicker95, affectFlicker, affectCharmed},
	grpUnsavedDamage:   {affectFireBody, affectIceBody},
	grpConfusion:       {affectConfused},
	grpAlwaysInvisible: {affectAlwaysInvisible},
	grpUnused:          {74},
}

// Affect check groups by use (the O0E3:0414 selector).
const (
	grpNone = iota
	grpAttackerExtra
	grpOnHit
	grpOnDamage
	grpDamageBonus
	grpWeaponDamage
	grpDamage
	grpTurn
	grpRound
	grpAddAffect
	grpAttacker
	grpArmour
	grpSave
	grpDown
	grpAI
	grpUse
	grpDefender
	grpMorale
	grpSpeed
	grpRest
	grpUnsavedDamage
	grpConfusion
	grpAlwaysInvisible
	grpUnused
)

// affectGroup = O0E3:0414 (group, char).
func (e *Engine) affectGroup(c *Character, group int) {
	if group < len(affectGroups) {
		for _, t := range affectGroups[group] {
			e.checkAffect(c, t)
		}
	}
}

// Cloud grounds.
const (
	groundStinkingCloud = 30
	groundPoisonCloud   = 28
	groundBody          = 31
	groundPlain         = 23 // what groundAt starts from
)

// cloudsUnder: is any square the combatant covers a stinking or a poison
// cloud? (the two flags of O121:0D04 with direction 8).
func (e *Engine) cloudsUnder(c *Character) (stinking, poison bool) {
	idx := e.combatIndex(c)
	for part := 0; part <= 3; part++ {
		dx, dy, ok := e.footprint(int(e.cpos[idx].size), part)
		if !ok {
			continue
		}
		_, g := e.mapAt(int(e.cpos[idx].x)+dx, int(e.cpos[idx].y)+dy)
		stinking = stinking || g == groundStinkingCloud
		poison = poison || g == groundPoisonCloud
	}
	return
}

// Hit dice a poison cloud kills outright, and the ones that save at -4.
const (
	cloudKillsDice  = 4
	cloudSavesMinus = 5
	cloudSavesPlain = 6
	cloudSaveMalus  = -4
)

// inPoisonCloud = O0E3:0E47 (char, stinking): a character in a stinking
// cloud coughs (save) or is overcome by nausea, the selected character for
// the time being the cloud's caster; in a poison cloud the weak die.
func (e *Engine) inPoisonCloud(c *Character, checkStinking bool) {
	if c.Okay == 0 {
		return
	}
	stinking, poison := e.cloudsUnder(c)
	if stinking && checkStinking && e.FindAffect(c, affectNausea) == nil &&
		e.FindAffect(c, affectCollapse) == nil && e.FindAffect(c, affectNoPoisonParalysis) == nil &&
		e.FindAffect(c, affectNoCharmHoldSleep) == nil {
		saved := e.SavingThrow(c, saveVsParalysis, 0)
		owner := e.cloudOwner(e.combatX(c), e.combatY(c), e.stinkingClouds, cloudSquares4[:])
		e.withSelected(owner, func() {
			if saved {
				e.isUnaffected(c, affectCoughing, 1, 255, 0, 0, saved, "starts to cough")
				if a := e.FindAffect(c, affectCoughing); a != nil {
					// the original passes a nil affect here
					e.callAffect(affectCoughing, c, nil, 0)
				}
			} else {
				e.isUnaffected(c, affectNausea, e.Dice(1, 4)+1, 255, 0, 0, saved, "chokes and gags from nausea")
				if e.FindAffect(c, affectNausea) != nil {
					e.callAffect(affectNausea, c, nil, 0)
				}
			}
		})
	}
	if !poison || c.Okay == 0 {
		return
	}
	switch hd := int8(c.MaxLevel); {
	case hd >= 0 && hd <= cloudKillsDice:
	case hd == cloudSavesMinus:
		if e.SavingThrow(c, saveVsParalysis, cloudSaveMalus) {
			return
		}
	case hd == cloudSavesPlain:
		if e.SavingThrow(c, saveVsParalysis, 0) {
			return
		}
	default:
		return
	}
	e.combatName(c, "is Poisoned", combatMsgRow, false)
	e.gameDelay()
	e.AddAffect(c, affectPoisoned, 0, 255, 0)
	e.killWithMessage(c, statusDead, "is killed")
}

// AttackHits = O0E3:1245 (attacker, defender, need): attacking ends
// invisibility; d20 + THAC0 + the side's modifier must reach need.
func (e *Engine) AttackHits(att, def *Character, need int) bool {
	e.removeInvisibility(att)
	e.rv.hitRoll = int8(e.Dice(1, 20))
	if e.rv.hitRoll <= 1 {
		return false
	}
	if e.rv.hitRoll == 20 {
		e.rv.hitRoll = 100
	}
	e.affectGroup(att, grpAttacker)
	e.affectGroup(def, grpDefender)
	adj := int(int8(e.game[area2PartyHit]))
	if att.Side != 0 {
		adj = int(int8(e.game[area2MonsterHit]))
	}
	return e.rv.hitRoll >= 0 && int(e.rv.hitRoll)+int(att.THAC0)+adj >= int(byte(need))
}

// SavingThrow = O0E3:12F1 do_saving_throw (char, type, bonus): 1 always fails, 20 always
// saves, otherwise d20 + bonus + the character's save bonus must reach the
// target for the type (0 paralysis ... 4 spells).
func (e *Engine) SavingThrow(c *Character, typ, bonus int) bool {
	e.rv.saveMade = true
	e.rv.save = byte(e.Dice(1, 20))
	switch e.rv.save {
	case 1:
		e.rv.saveMade = false
	case 20:
		e.rv.saveMade = true
	default:
		e.rv.save = byte(int(e.rv.save) + int(int8(c.SaveBonus)) + int(int8(bonus)))
		e.rv.saveType = byte(typ)
		e.affectGroup(c, grpSave)
		e.rv.saveMade = c.Saves[int(int8(typ))] <= e.rv.save
	}
	return e.rv.saveMade
}

// RollDice = O0E3:13CC (n, sides): Dice that also remembers n (DS:6FB2).
func (e *Engine) RollDice(n, sides int) int {
	e.rv.diceN = byte(n)
	return e.Dice(n, sides)
}

// killWithMessage = O0E3:0014 (char, status, message): the dead, stoned
// and gone stay as they are; others fall with the status.
func (e *Engine) killWithMessage(c *Character, status byte, msg string) {
	e.combatName(c, msg, combatMsgRow, false)
	switch c.Status {
	case statusDead, statusStoned, statusDisintegrated:
		return
	}
	c.Status = status
	c.Okay = 0
	c.HP = 0
	e.downAffects(c)
	e.affectGroup(c, grpDown)
	if c.Okay == 0 {
		e.combatRemoveFromMap(c)
	}
	e.gameDelay()
	e.clearCombatText()
	if e.gameState != stateCombat {
		e.playerSummary(e.selected())
	}
}

// KillCharacter = O0E3:14A7 (char, status, message): leaves the fight
// (running away, surrendering) without a body.
func (e *Engine) KillCharacter(c *Character, status byte, msg string) {
	msg = cut(msg, 0x28)
	if c.Okay == 0 {
		return
	}
	idx := e.combatIndex(c)
	e.combatRedrawChar(c, 3, 0)
	e.combatName(c, msg, combatMsgRow, true)
	c.Okay = 0
	c.Status = status
	if c.Status != 3 {
		c.HP = 0
	}
	e.combatClearIcon(0, 0, idx)
	e.Flip()
	e.cpos[idx].size = 0
	e.combatRedrawMap()
	if e.endTurn(c) {
		e.downAffects(c)
	}
}

// removeInvisibility = O0E3:1573.
func (e *Engine) removeInvisibility(c *Character) {
	for a := e.FindAffect(c, affectInvisible); a != nil; a = e.FindAffect(c, affectInvisible) {
		e.RemoveAffect(c, affectInvisible, a)
	}
}

// downAffects = O0E3:15AB: the affects a fall ends (DS:0A31); a berserk
// computer-charmed one returns to its side.
func (e *Engine) downAffects(c *Character) {
	for _, t := range downAffectData[1:] {
		e.RemoveAffect(c, t, nil)
	}
	if e.FindAffect(c, affectBerserk) != nil && c.Control == controlCharmed {
		c.Side = 0
	}
}

// heldAffects = DS:0A45: the affects of holding and being held.
var heldAffects = [4]byte{affectSuffocating, affectEngulfed, affectEngulfing, affectHugging}

// releaseHeld = O0E3:160D.
func (e *Engine) releaseHeld(c *Character) {
	for _, t := range heldAffects {
		e.RemoveAffect(c, t, nil)
	}
}

// CureAffect = O0E3:164D is_cured (char, type): "is Cured" and the affect goes.
func (e *Engine) CureAffect(c *Character, typ byte) bool {
	a := e.FindAffect(c, typ)
	if a == nil {
		return false
	}
	e.combatName(c, "is Cured", 0xA, true)
	e.RemoveAffect(c, typ, a)
	return true
}

// What a successful save does to damage.
const (
	saveNone   = 0
	saveNegate = 1
	saveHalf   = 2
)

// damagePerson = O0E3:1FBA (char, damage, mode, saved): reports and
// applies damage, the lost spell, and going down.
func (e *Engine) damagePerson(c *Character, dmg int, mode byte, saved bool) {
	e.rv.damage = byte(dmg)
	e.affectGroup(c, grpDamage)
	if saved {
		switch mode {
		case saveNegate:
			e.rv.damage = 0
		case saveHalf:
			e.rv.damage >>= 1
		}
	} else {
		e.affectGroup(c, grpUnsavedDamage)
	}
	if e.rv.damage == 0 {
		return
	}
	var msg string
	if e.rv.damage != 1 {
		msg = "takes " + itoa(int(e.rv.damage)) + " points of damage "
	} else {
		msg = "takes 1 point of damage "
	}
	switch e.rv.dmgType &^ dmgMagic {
	case dmgFire:
		msg += "from Fire"
	case dmgCold:
		msg += "from Cold"
	case dmgElectricity:
		msg += "from Electricity"
	case dmgAcid:
		msg += "from Acid"
	}
	if e.rv.dmgType&dmgMagic == e.rv.dmgType {
		msg += "from Magic"
	}
	e.hitMessage(c, false, msg)
	e.damagePlayer(c, e.rv.damage)
	if e.gameState == stateCombat {
		ci := combatInfo(c)
		ci.CanCast = 0
		if ci.Spell > 0 {
			e.combatName(c, "lost a spell", combatMsgRow+2, true)
			forgetSpellOf(c, ci.Spell)
			ci.Spell = 0
		}
	}
	if c.Okay == 0 {
		msg = "Goes Down"
		if c.Status == statusDying {
			msg += ", and is Dying"
		}
		if s := c.Status; s >= statusDead && s <= statusDisintegrated {
			msg = "is killed"
		}
		e.combatName(c, msg, int(e.msgRow)+1, false)
		if e.gameState != stateCombat {
			e.gameDelay()
		} else {
			e.downAffects(c)
			e.affectGroup(c, grpDown)
			if c.Okay == 0 {
				e.combatRemoveFromMap(c)
			} else {
				e.gameDelay()
			}
		}
	}
	e.clearCombatText()
}

// isUnaffected = O0E3:22E1 (char, type, time, data, call, saveMode, saved,
// message): adds an affect unless something blocks it (or a save negates
// it); a timed copy already there is replaced.
func (e *Engine) isUnaffected(c *Character, typ byte, time int, data, call, saveMode byte, saved bool, msg string) {
	msg = cut(msg, 0x28)
	e.rv.affVal = typ
	e.affectGroup(c, grpAddAffect)
	if e.rv.affVal == 0 || saved && saveMode == saveNegate {
		e.combatName(c, "is Unaffected", combatMsgRow, true)
		return
	}
	if a := e.FindAffect(c, typ); a != nil && a.Time > 0 {
		e.RemoveAffect(c, typ, a)
	}
	e.AddAffect(c, typ, time, data, call)
	if msg != "" {
		e.hitMessage(c, true, msg)
		e.clearCombatText()
	}
}

// Heal = O0E3:23D5 heal_player (char, amount, magical): only the living
// (okay, animated, unconscious, dying) heal; magic only the hurt.
func (e *Engine) Heal(c *Character, amount int, magical bool) bool {
	switch c.Status {
	case statusOkay, statusAnimated, statusUnconscious, statusDying:
	default:
		return false
	}
	if magical && c.HP >= c.MaxHP {
		return false
	}
	c.HP += byte(amount)
	if c.HP > c.MaxHP {
		c.HP = c.MaxHP
	}
	if c.Okay == 0 {
		if c.Status == statusDying {
			c.Status = statusUnconscious
		}
		if c.Status == statusUnconscious && e.gameState != stateCombat {
			e.callAffect(affectRecovering, c, nil, 0)
		}
	}
	return true
}

// Revive = O0E3:24C7 (char, hp): back on its feet if its square is free.
func (e *Engine) Revive(c *Character, hp byte) bool {
	if !e.combatSquareFree(c, e.combatX(c), e.combatY(c), true) {
		return false
	}
	c.Status = 0
	c.Okay = 1
	c.HP = hp
	if e.gameState == 5 {
		e.combatRedrawChar(c, 3, 0)
	}
	msg := "gets back up"
	if c.Side == 1 {
		msg = "stands up and grins"
	}
	e.combatName(c, msg, 0xA, true)
	e.countSides()
	return true
}

// hitMessage = O0ED:218A (char, magical, message): in combat the target
// flashes (once per message delay step for magic) with a sound.
func (e *Engine) hitMessage(c *Character, magical bool, msg string) {
	msg = cut(msg, 0x28)
	if e.gameState != 5 {
		e.combatName(c, msg, 0xA, true)
		return
	}
	sel := 0x17
	if magical {
		sel = 0x16
	}
	e.hitColours(sel)
	if !e.combatOnScreen(c, true) {
		e.combatCentre(e.combatX(c), e.combatY(c), 3, 8)
	}
	if magical {
		e.PlaySound(sndMagicHit)
	} else {
		e.PlaySound(sndHit)
	}
	e.combatName(c, msg, 0xA, false)
	n := 0
	if magical {
		n = int(e.gameSpeed)
	}
	idx := e.combatIndex(c)
	px, py := int(3*e.combatScreenX[idx]), int(3*e.combatScreenY[idx]) // byte products
	for i := 0; i <= n; i++ {
		for f := 0; f <= 3; f++ {
			e.drawHitFrame(f, px, py)
			e.Flip()
			e.Delay(0x46)
			e.restoreHitFrame(px, py)
		}
	}
	e.Flip()
	if n == 0 {
		e.gameDelay()
	}
}

// forgetSpell = O0ED:1520 (spell): the selected character's first
// memorised copy of the spell is gone.
func (e *Engine) forgetSpell(sp byte) {
	c := e.selected()
	if c == nil {
		return
	}
	for i := 0; i <= 0x14; i++ {
		if c.Memorized[i] == sp {
			c.Memorized[i] = 0
			break
		}
	}
}

// healedMessage = O0ED:24F7 (char): fully or partially healed; outside
// combat the selected character's summary is shown again.
func (e *Engine) healedMessage(c *Character) {
	msg := "is partially healed"
	if c.HP == c.MaxHP {
		msg = "is fully healed"
	}
	e.combatName(c, msg, combatMsgRow, true)
	if e.gameState != stateCombat {
		e.playerSummary(e.selected())
	}
}

// countSides = O0ED:2588: standing members of each side (DS:6814/6815).
func (e *Engine) countSides() {
	e.rv.side = [2]int8{}
	for _, c := range e.party {
		if c.Okay != 0 {
			e.rv.side[int8(c.Side)&1]++
		}
	}
}

// Words of the area2 record (DS:4FB6) the combat rules read.
const (
	area2Surprise       = 1430 // sides surprised: bit side+1
	area2MonsterHit     = 1760 // to-hit modifier of the monsters' side
	area2PartyHit       = 1762 // to-hit modifier of the party
	area2PartyMoveBonus = 1764 // added to the party's movement
)
