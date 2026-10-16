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
	HasSpells  byte       // 22 has memorised spells the computer may cast
}

// Rule globals (DS:87D2..87E8): what the affect handlers work on.
type ruleVars struct {
	save      byte       // b87D2 saving throw total
	affVal    byte       // b87D3 affect being added (0 = blocked)
	damage    byte       // b87D4
	dmgType   byte       // b87D5 1 fire 2 cold 4 electricity 8 magic 16 acid
	attacks   byte       // b87D6 attacks per round being worked out
	moves     byte       // b87D7 moves being worked out
	diceN     byte       // b87D9 dice of the damage roll
	unseen    byte       // b87DB the attacker cannot see its target
	cure      byte       // b87DC set while affects are being cured
	areaSpell byte       // b87DD an area spell: no image is lost
	hitRoll   int8       // b87DF
	hits      [3]byte    // b87E0+i, i = 1..2: hits of each attack slot
	morale    byte       // b87E2 flee/surrender roll
	target    *Character // p87E3 the one the handler looks at
	saveType  byte       // b87E7
	saveMade  bool       // b87E8 the last saving throw succeeded
	side      [2]int8    // standing members per side
}

// callAffect = O145:0423 (type, char, affect, flag): while an item is
// being used its handler runs instead.
func (e *Engine) callAffect(typ byte, c *Character, a *Affect, flag byte) {
	if e.itemCall {
		e.itemHandler(c, e.callItem, flag)
		return
	}
	if int(typ) < len(affectHandlers) && affectHandlers[typ] != nil {
		affectHandlers[typ](e, c, a, flag)
	}
}

// FindAffect = O151:28C8 (out, type, char): first affect of the type.
func (e *Engine) FindAffect(c *Character, typ byte) *Affect {
	for _, a := range c.Affects {
		if a.Type == typ {
			return a
		}
	}
	return nil
}

// RemoveAffect = O145:00A6 (char, type, affect): affect nil means the first
// one of the type. Its handler runs first when the node asks for it; losing
// friendliness or a strength affect recomputes that stat.
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
	case affectFriendly:
		e.statAffects(c, statCha)
	case affectEnlarge, affectStrengthSpell, affectGiantStrength:
		e.statAffects(c, statStr)
	}
}

// RemoveEffect is the old name of RemoveAffect used by the clock code.
func (e *Engine) RemoveEffect(c *Character, a *Affect) { e.RemoveAffect(c, a.Type, a) }

// AddAffect = O145:1235 (char, type, time, data, call): append a node.
func (e *Engine) AddAffect(c *Character, typ byte, time int, data, call byte) {
	a := &Affect{affectRecord: affectRecord{Type: typ, Time: uint16(time), Data: data, Call: call}}
	c.Affects = append(c.Affects, a)
}

// auraAffect = O145:0287's set: affects that also work on everyone near
// the one who has them.
func auraAffect(t byte) bool {
	return t == affectSilenced || t == affectProtFromEvil10 || t == affectProtFromGood10 || t == affectPrayer
}

// Squares an aura reaches: a prayer works further than the others.
const (
	auraRange   = 1
	prayerRange = 6
)

// checkAffect = O145:0287 (char, type): run the handler (flag 0) when the
// character has the affect, or for an aura affect when anyone in range
// (anyone at all outside combat) has it.
func (e *Engine) checkAffect(c *Character, typ byte) {
	a := e.FindAffect(c, typ)
	found := a != nil
	if !found && auraAffect(typ) {
		saved := e.nearList
		for _, p := range e.party {
			if found {
				break
			}
			pa := e.FindAffect(p, typ)
			if pa == nil {
				continue
			}
			a = pa
			if e.gameState != stateCombat {
				found = true
				continue
			}
			r := auraRange
			if typ == affectPrayer {
				r = prayerRange
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

// affectGroups = the lists of O145:0466: which affects take part in each
// kind of check (the group number is the caller's). The damage and the
// add-affect checks try magic resistance first.
var affectGroups = [][]byte{
	grpAttackerExtra: {affectInvisible, affectBlinking, affectInvisibleToAnimals},
	grpOnHit:         {affectStoningTouch, affectPoisonStingMinus2, affectParalysingTouch},
	grpOnDamage:      {affectPoisonSting, affectStoningTouch, affectPoisonStingMinus2, affectParalysingTouch},
	grpDamageBonus:   {affectEnfeebled, affectSwallowsWhole, affectBaneWeapon, affectRangerVsGiants, affectDispelEvil},
	grpWeaponDamage: {affectMirrorImage, affectProtNormalMissiles, affectNeedsPlus3Weapon, affectBoltsHarmless,
		affectPlainWeaponsImmune, affectZapsAttackers},
	grpDamage: {affectElectricityImmune, affectResistCold, affectShield, affectResistFire, affectMirrorImage,
		affectFireResistRing, affectMinorGlobe, affectGlobe, affectDragonBreathImmune, affectFireHeals,
		affectFireImmune, affectColdImmune},
	grpTurn:  {affectSticksToSnakes, affectBefuddled, affectNausea, affectImmobile, affectHeld, affectAsleep},
	grpRound: {affectInvisibleRing, affectFearAura, affectMagicalCloak, affectAlwaysInvisible},
	grpAddAffect: {affectHalfElfResistance, affectMirrorImage, affectMinorGlobe, affectDiseaseImmune,
		affectFireHeals, affectFearless, affectElfResistance, affectNoCharmSleep, affectPoisonImmune},
	grpAttacker: {affectBless, affectCurse, affectGnomeVsFoes, affectDwarfVsFoes, affectBlind, affectCursed,
		affectPrayer, affectBaneWeapon},
	grpArmour: {affectDispellingEvil, affectProtFromEvil, affectProtFromGood, affectBarkskin, affectShield,
		affectCoughing, affectBlind, affectProtFromEvil10, affectProtFromGood10, affectDwarfDodges,
		affectGnomeDodges, affectHighlighted, affectMagicalBoots},
	grpSave: {affectProtFromEvil, affectProtFromGood, affectResistCold, affectBarkskin, affectShield,
		affectResistFire, affectBlind, affectCursed, affectProtFromEvil10, affectProtFromGood10, affectPrayer,
		affectFireShieldHot, affectFireShieldCold, affectFireResistRing, affectMagicalStone, affectHardyAll},
	grpDown: {},
	grpAI: {affectStoningGaze, affectBreathFireClose, affectConfusingGaze, affectBreathFire, affectSpitAcid,
		affectBreathWeapon, affectBreathWeapon2, affectBerserkFoe},
	grpUse: {affectSticksToSnakes, affectDetectMagic, affectCharmed, affectSilenced, affectBefuddled,
		affectCoughing, affectBerserk, affectStunned, affectInvisible, affectBlinking, affectDwarfDodges,
		affectGnomeDodges, affectInvisibleToAnimals, affectMagicalCloak},
	grpDefender:      {},
	grpMorale:        {affectBless, affectCurse, affectCharmed},
	grpSpeed:         {affectHaste, affectSlow},
	grpRest:          {affectCharmed, affectSpiritualHammer, affectInvisibleRing, affectDelayedBlast},
	grpUnsavedDamage: {affectFireShieldHot, affectFireShieldCold},
	grpConfusion:     {affectConfused},
	grpMovement:      {affectHaste, affectSlow, affectMagicalBoots},
}

// Affect check groups by use (the O145:0466 selector).
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
	grpMovement
)

// affectGroup = O145:0466 (group, char).
func (e *Engine) affectGroup(c *Character, group int) {
	if group == grpDamage || group == grpAddAffect {
		e.magicResistance(c)
	}
	if group < len(affectGroups) {
		for _, t := range affectGroups[group] {
			e.checkAffect(c, t)
		}
	}
}

// Affects magic resistance never stops outright.
const (
	resistKeepsCloudkill = affectCloudkill
	resistKeepsFear      = affectFearAura
)

// magicResistance = O145:01E1 (char): an affect (with no damage, or with
// magical damage) is stopped by the character's magic resistance, which
// is 5% better for each caster level below 11.
func (e *Engine) magicResistance(c *Character) {
	if c.MagicResistance == 0 || e.rv.affVal == 0 {
		return
	}
	if e.rv.damage != 0 && e.rv.dmgType&dmgMagic == 0 {
		return
	}
	chance := int(c.MagicResistance) + (11-e.casterLevel(int(e.curSpell)))*5
	if e.Dice(1, 100) <= chance {
		e.rv.damage = 0
		if e.rv.affVal != resistKeepsCloudkill && e.rv.affVal != resistKeepsFear {
			e.rv.affVal = 0
		}
	}
}

// Cloud grounds.
const (
	groundDelayedBlast  = 26 // a delayed blast fireball waiting
	groundPoisonCloud   = 27
	groundStoneBody     = 28 // a body turned to stone
	groundStinkingCloud = 29
	groundBody          = 30
	groundPlain         = 23 // what groundAt starts from
)

// cloudsUnder: is any square the combatant covers a stinking or a poison
// cloud? (the two flags of O18B:0C80 with direction 8).
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

// Monsters the fumes of a stinking cloud do not touch.
const monsterNoBreath = 8

// inPoisonCloud = O145:0C52 (char, stinking): a character in a stinking
// cloud coughs (save) or is overcome by nausea, the selected character for
// the time being the cloud's caster; in a poison cloud the weak die.
func (e *Engine) inPoisonCloud(c *Character, checkStinking bool) {
	if c.Okay == 0 {
		return
	}
	stinking, poison := e.cloudsUnder(c)
	if stinking && checkStinking && c.Undead == 0 && e.FindAffect(c, affectNausea) == nil &&
		e.FindAffect(c, affectPoisonImmune) == nil && c.MonsterFlags&monsterNoBreath == 0 {
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
	e.pauseForMessage()
	e.AddAffect(c, affectPoisoned, 0, 255, 0)
	e.killWithMessage(c, statusDead, "is killed")
}

// AttackHits = O145:103A (attacker, defender, need): attacking ends
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

// SavingThrow = O145:1137 (char, type, bonus): 1 always fails, 20 always
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

// RollDice = O145:1211 (n, sides): Dice that also remembers n.
func (e *Engine) RollDice(n, sides int) int {
	e.rv.diceN = byte(n)
	return e.Dice(n, sides)
}

// killWithMessage = O145:0000 (char, status, message): the dead, stoned
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
	if c.Okay == 0 {
		e.combatRemoveFromMap(c)
	}
	e.pauseForMessage()
	e.clearCombatText()
	if e.gameState != stateCombat {
		e.summaryRow(c, c == e.selected())
	}
}

// Longest message a character line shows.
const maxMessageLen = 40

// Status of one who ran away: it keeps its hit points.
const statusFled = 3

// KillCharacter = O145:12E5 (char, status, message): leaves the fight
// (running away, surrendering) without a body.
func (e *Engine) KillCharacter(c *Character, status byte, msg string) {
	msg = cut(msg, maxMessageLen)
	if c.Okay == 0 {
		return
	}
	idx := e.combatIndex(c)
	e.combatRedrawChar(c, 3, 0)
	e.combatName(c, msg, combatMsgRow, true)
	c.Okay = 0
	c.Status = status
	if c.Status != statusFled {
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

// removeInvisibility = O145:13B1.
func (e *Engine) removeInvisibility(c *Character) {
	for a := e.FindAffect(c, affectInvisible); a != nil; a = e.FindAffect(c, affectInvisible) {
		e.RemoveAffect(c, affectInvisible, a)
	}
}

// downAffects = O145:13E9: the affects a fall ends; a berserk charmed one
// returns to its side.
func (e *Engine) downAffects(c *Character) {
	for _, t := range downAffectData[1:] {
		e.RemoveAffect(c, t, nil)
	}
	if e.FindAffect(c, affectBerserk) != nil && c.Control == controlCharmed {
		c.Side = 0
	}
}

// CureAffect = O145:1457 is_cured (char, type): the affect goes, silently
// (Curse said "is Cured").
func (e *Engine) CureAffect(c *Character, typ byte) bool {
	a := e.FindAffect(c, typ)
	if a == nil {
		return false
	}
	e.RemoveAffect(c, typ, a)
	return true
}

// What a successful save does to damage.
const (
	saveNone   = 0
	saveNegate = 1
	saveHalf   = 2
)

// damagePerson = O145:1D84 (char, damage, mode, saved): reports and
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
			e.pauseForMessage()
		} else {
			e.downAffects(c)
			e.affectGroup(c, grpDown)
			if c.Okay == 0 {
				e.combatRemoveFromMap(c)
			} else {
				e.pauseForMessage()
			}
		}
	}
	e.clearCombatText()
}

// isUnaffected = O145:20C8 (char, type, time, data, call, saveMode, saved,
// message): adds an affect unless something blocks it (or a save negates
// it); one already there lasts the longer of the two times.
func (e *Engine) isUnaffected(c *Character, typ byte, time int, data, call, saveMode byte, saved bool, msg string) {
	msg = cut(msg, maxMessageLen)
	e.rv.affVal = typ
	e.affectGroup(c, grpAddAffect)
	if e.rv.affVal == 0 || saved && saveMode == saveNegate {
		e.combatName(c, "is Unaffected", combatMsgRow, true)
		return
	}
	if a := e.FindAffect(c, e.rv.affVal); a != nil {
		if a.Time != 0 && uint16(time) > a.Time {
			a.Time = uint16(time)
		}
	} else {
		e.AddAffect(c, typ, time, data, call)
	}
	if msg != "" {
		e.hitMessage(c, true, msg)
		e.clearCombatText()
	}
}

// Heal = O145:21BF (char, amount, magical): only the living (okay,
// animated, unconscious, dying) heal; magic only the hurt. The dying stop
// bleeding, and outside combat the unconscious wake up.
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
			c.Status = statusOkay
			c.Okay = 1
		}
	}
	return true
}

// hitMessage = O151:270F (char, magical, message): in combat the target
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
		e.pauseForMessage()
	}
}

// forgetSpell = O151:149F (char, spell): the character's first memorised
// copy of the spell is gone.
func forgetSpell(c *Character, sp byte) {
	for i := range c.Memorized {
		if c.Memorized[i] == sp {
			c.Memorized[i] = 0
			break
		}
	}
}

// healedMessage = O151:2A75 (char): fully or partially healed; outside
// combat the character's summary row is drawn again.
func (e *Engine) healedMessage(c *Character) {
	msg := "is partially healed"
	if c.HP == c.MaxHP {
		msg = "is fully healed"
	}
	e.combatName(c, msg, combatMsgRow, true)
	if e.gameState != stateCombat {
		e.summaryRow(c, c == e.selected())
	}
}

// countSides = O151:2B1C: standing members of each side (DS:6814/6815).
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
