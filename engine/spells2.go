package engine

// Spell effect routines of O0C2 (1D60..35DE). CastSpell (O0C2:12E1) calls
// them through the table DS:72BA filled by O0C2:6592 setup_spells, indexed
// by the spell number (DS:6FB1); each is called without arguments and works
// on the spell globals: the target count DS:744E and list DS:744B (entry 1
// at DS:744F is the first target).

// fxBless = O0C2:1D60 is_Blessed: Bless, "is Blessed" on the selected character's side.
func (e *Engine) fxBless() {
	e.applySpellToSide(int(e.selected().Side), "is Blessed")
}

// fxBane = O0C2:1D91 is_Cursed: Bane, "is Cursed".
func (e *Engine) fxBane() {
	side := 0
	if isParty(e.selected()) {
		side = 1
	}
	e.applySpellToSide(side, "is Cursed")
}

// fxCureLight = O0C2:1DBC: Cure Light Wounds on the first target, 1d8.
func (e *Engine) fxCureLight() {
	if e.nSpellTargets != 0 {
		if e.Heal(e.tgt(1), e.Dice(1, 8), false) {
			e.healedMessage(e.tgt(1))
		}
	}
}

// fxCauseLight = O0C2:1DF8: 1d8 magic damage.
func (e *Engine) fxCauseLight() {
	e.applySpell(int(e.curSpell), 0, 0, e.RollDice(1, 8), 8, "")
}

// fxAffected = O0C2:1E3A is_affected: "is affected" (shared by several spells).
func (e *Engine) fxAffected() { e.applySpell(int(e.curSpell), 0, 0, 0, 0, "is affected") }

// fxProtected = O0C2:1E74 is_protected: "is protected".
func (e *Engine) fxProtected() { e.applySpell(int(e.curSpell), 0, 0, 0, 0, "is protected") }

// fxColdResistant = O0C2:1EB3 is_cold_resistant: "is cold-resistant".
func (e *Engine) fxColdResistant() { e.applySpell(int(e.curSpell), 0, 0, 0, 0, "is cold-resistant") }

// fxBurningHands = O0C2:1EE1: damage equal to the caster level, type 9.
func (e *Engine) fxBurningHands() {
	e.applySpell(int(e.curSpell), 0, 0, e.casterLevel(int(e.curSpell)), 9, "")
}

// fxCharmPerson = O0C2:1F2E is_charmed: Charm Person; large or strong targets are unaffected.
func (e *Engine) fxCharmPerson() {
	t := e.tgt(1)
	if t.Kind > 1 || t.Size > 1 {
		e.combatName(t, "is unaffected", 0xA, true)
		return
	}
	lvl := e.casterLevel(int(e.curSpell))
	e.applySpell(int(e.curSpell), (int(e.selected().Side)<<7+lvl)&0xFF, 1, 0, 0, "is charmed")
	if a := e.FindAffect(t, 0xB); a != nil {
		e.callAffect(0xB, e.tgt(1), a, 0)
	}
}

// spellStronger = O0C2:2005 is_stronger: Strength. The strength comes from
// the caster level: 18 with 18/00..18/100 for levels 1..6, then 19..22.
func (e *Engine) spellStronger() {
	t := e.spellTarget
	str, str00 := byte(strengthMaximum18), byte(0)
	switch e.casterLevel(int(e.curSpell)) {
	case 2:
		str00 = 1
	case 3:
		str00 = 51
	case 4:
		str00 = 76
	case 5:
		str00 = 91
	case 6:
		str00 = 100
	case 7:
		str = 19
	case 8:
		str = 20
	case 9:
		str = 21
	case 10, 11:
		str = 22
	}
	if code, ok := strongerThan(t, str, str00); ok {
		e.combatName(t, "is stronger", combatMsgRow, true)
		e.AddAffect(t, affectEnlarge, e.spellAmount(int(e.curSpell)), code, 1)
		e.statAffects(t, statStr)
		return
	}
	e.combatName(t, "is unaffected", combatMsgRow, true)
}

// spellReduce = O0C2:213A has_been_reduced: the magical strength (affect
// 12) goes unless the target saves.
func (e *Engine) spellReduce() {
	t := e.spellTarget
	if t != nil && e.nSpellTargets > 0 && !e.SavingThrow(t, saveVsSpell, 0) && e.FindAffect(t, affectEnlarge) != nil {
		e.RemoveAffect(t, affectEnlarge, nil)
		e.statAffects(t, statStr)
		e.combatName(t, "has been reduced", combatMsgRow, true)
	}
}

// spellFriends = O0C2:21DA is_friendly: the charisma affect at 2d4.
func (e *Engine) spellFriends() {
	e.applySpell(int(e.curSpell), e.Dice(2, 4), 1, 0, 0, "is friendly")
	e.statAffects(e.selected(), statCha)
}

// fxMagicMissile = O0C2:2221: magic damage of ((level+1)/2)d4 + (level+1)/2.
func (e *Engine) fxMagicMissile() {
	n := (e.casterLevel(int(e.curSpell)) + 1) & 0xFF >> 1
	e.applySpell(int(e.curSpell), 0, 0, (n+e.RollDice(n, 4))&0xFF, 8, "")
}

// fxShielded = O0C2:2284 is_shielded: "is shielded".
func (e *Engine) fxShielded() { e.applySpell(int(e.curSpell), 0, 0, 0, 0, "is shielded") }

// fxShockingGrasp = O0C2:22B2: caster level + 1d8 damage, type 0xC.
func (e *Engine) fxShockingGrasp() {
	lvl := e.casterLevel(int(e.curSpell))
	e.applySpell(int(e.curSpell), 0, 0, (lvl+e.RollDice(1, 8))&0xFF, 0xC, "")
}

// fxSleep = O0C2:2306 falls_asleep: Sleep. 4d4 hit dice of targets fall asleep, weakest
// first in list order; the others (and the already asleep) leave the list.
func (e *Engine) fxSleep() {
	e.rv.areaSpell = 1
	e.v4844 = byte(e.Dice(4, 4))
	for i := 1; i <= e.nSpellTargets; i++ {
		t := e.tgt(i)
		switch lvl := int8(t.MaxLevel); {
		case lvl >= 0 && lvl <= 1:
			e.v4845 = 1
		case lvl == 2:
			e.v4845 = 2
		case lvl == 3:
			e.v4845 = 4
		case lvl == 4:
			e.v4845 = 6
		case lvl == 5:
			if t.Race == 0 {
				e.v4845 = 0xA
			} else {
				e.v4845 = 0x14
			}
		default:
			e.v4845 = 0x14
		}
		if t.Status != 1 && e.FindAffect(t, 0x35) == nil && e.v4844 >= e.v4845 {
			e.v4844 -= e.v4845
		} else {
			e.setTgt(i, nil)
		}
	}
	e.applySpell(int(e.curSpell), 0, 0, 0, 0, "falls asleep")
}

// Spells with their own rules in holdTargets.
const (
	spellHoldPerson  = 23
	spellHoldAutoA   = 79 // the last target never saves
	spellHoldAutoB   = 81
	spellHoldAnySize = 94 // large and strong targets get no free save
)

// fxHold = O0C2:2444 is_held: Hold Person/Monster. The save bonus
// depends on the number of targets.
func (e *Engine) fxHold() {
	var bonus int8 // other counts leave the local unset (0)
	switch e.nSpellTargets {
	case 1:
		if e.curSpell == spellHoldPerson {
			bonus = -2
		} else {
			bonus = -3
		}
	case 2:
		bonus = -1
	}
	e.holdTargets(bonus, "is held")
}

// holdTargets = O0C2:1B24 (bonus, msg): each target, last first, gets a
// missile (but the first one shot) and the spell record's affect unless it
// saves.
func (e *Engine) holdTargets(bonus int8, msg string) {
	spell := int(e.curSpell)
	for i := e.nSpellTargets; i >= 1; i-- {
		t := e.tgt(i)
		if t == nil {
			continue
		}
		if i < e.nSpellTargets {
			e.PlaySound(sndSpell)
			e.hitColours(0x12)
			sel := e.selected()
			e.flyMissile(e.combatX(sel), e.combatY(sel), e.combatX(t), e.combatY(t), 4, 0x1E)
		}
		var saved bool
		var saveMode byte
		if (spell == spellHoldAutoA || spell == spellHoldAutoB) && i == e.nSpellTargets {
			saved, saveMode = true, 1
		} else {
			saved = e.SavingThrow(t, int(e.spellRec(byte(spell), spellFieldSaveType)), int(bonus))
			saveMode = e.spellRec(byte(spell), spellFieldSaveMode)
		}
		if (t.Kind > 1 || t.Size > 1) && spell != spellHoldAnySize {
			saved = true
		}
		e.isUnaffected(t, e.spellRec(byte(spell), spellFieldAffect), e.spellAmount(spell), byte(e.casterLevel(spell)), 0,
			saveMode, saved, msg)
	}
}

// fxFireResistant = O0C2:24A7 is_fire_resistant: "is fire resistant".
func (e *Engine) fxFireResistant() { e.applySpell(int(e.curSpell), 0, 0, 0, 0, "is fire resistant") }

// fxSilenced = O0C2:24E0 is_silenced: "is silenced".
func (e *Engine) fxSilenced() { e.applySpell(int(e.curSpell), 0, 0, 0, 0, "is silenced") }

// fxSlowPoison = O0C2:2519 is_affected2: Raise Dead style: a dead target is dropped from
// the list; one with affect 0x37 is brought to 1 hit point and stunned (0x0F).
func (e *Engine) fxSlowPoison() {
	t := e.tgt(1)
	if t.Status == 1 {
		e.setTgt(1, nil)
		return
	}
	if e.FindAffect(t, 0x37) == nil {
		return
	}
	if t.HP == 0 {
		t.HP = 1
	}
	e.applySpell(int(e.curSpell), 0xFF, 1, 0, 0, "is affected")
	e.callAffect(0x4E, t, nil, 1)
	e.AddAffect(t, 0xF, 10, 0xFF, 1)
}

// fxCharmSnakes = O0C2:25CD is_charmed2: the party members of kind 0xE whose hit points
// fit in the caster's hit points become the targets.
func (e *Engine) fxCharmSnakes() {
	e.v4844 = e.selected().HP
	e.nSpellTargets = 0
	for _, p := range e.party {
		if p.Kind == 0xE && e.v4844 >= p.HP {
			e.v4844 -= p.HP
			e.nSpellTargets++
			e.setTgt(e.nSpellTargets, p)
		}
	}
	e.applySpell(int(e.curSpell), 0, 0, 0, 0, "is charmed")
}

// fxSpiritualHammer = O0C2:2681: applies the spell, then runs the affect 0x17 handler on the first target.
func (e *Engine) fxSpiritualHammer() {
	e.applySpell(int(e.curSpell), 0, 1, 0, 0, "")
	e.callAffect(0x17, e.tgt(1), nil, 0)
}

// fxInvisibility = O0C2:26D4 is_invisible: "is invisible".
func (e *Engine) fxInvisibility() { e.applySpell(int(e.curSpell), 0, 0, 0, 0, "is invisible") }

// fxKnock = O0C2:270D knock_Knock: Knock.
func (e *Engine) fxKnock() { e.applySpell(int(e.curSpell), 0, 0, 0, 0, "Knock-Knock") }

// fxMirrorImage = O0C2:2748 is_duplicated: Mirror Image, 1d4 images in the high
// nibble of the level.
func (e *Engine) fxMirrorImage() {
	lvl := byte(e.Dice(1, 4))<<4 | byte(e.casterLevel(int(e.curSpell)))
	e.applySpell(int(e.curSpell), int(lvl), 0, 0, 0, "is duplicated")
}

// fxRayOfEnfeeblement = O0C2:27A6 is_weakened: "is weakened".
func (e *Engine) fxRayOfEnfeeblement() { e.applySpell(int(e.curSpell), 0, 0, 0, 0, "is weakened") }

// fxStinkingCloud = Stinking Cloud (O0C2 create_noxious_cloud).
func (e *Engine) fxStinkingCloud() { e.createCloud(e.stinkingKind()) }

// hasClass: a level in the class now, or an old one above the level of
// the class taken up after (a dual-classed character's first class).
func hasClass(c *Character, k int) bool {
	return int8(c.Levels[k]) > 0 || int8(c.OldLevels[k]) > int8(c.MulticlassLevel)
}

// spellStrength = O0C2:2C5B: Strength. 1d4 for a mage, 1d6 for a cleric or
// thief, 1d8 for a fighter (the last that applies) on the strength in use;
// above 18 the fighters, paladins and rangers get exceptional strength.
func (e *Engine) spellStrength() {
	var str00, add byte // add is an unset local when no class qualifies
	t := e.tgt(1)
	if hasClass(t, clsMage) {
		add = byte(e.Dice(1, 4))
	}
	if hasClass(t, clsCleric) || hasClass(t, clsThief) {
		add = byte(e.Dice(1, 6))
	}
	if hasClass(t, clsFighter) {
		add = byte(e.Dice(1, 8))
	}
	str := t.Stats[statStr].Value + add
	if str > strengthMaximum18 {
		if hasClass(t, clsFighter) || hasClass(t, clsPaladin) || hasClass(t, clsRanger) {
			str00 = t.Str00.Value + (str-strengthMaximum18)*10
			if str00 > 100 {
				str00 = 100
			}
		}
		str = strengthMaximum18
	}
	if _, ok := strongerThan(t, str, str00); ok {
		e.AddAffect(t, affectStrengthSpell, e.spellAmount(int(e.curSpell)), add+strengthEncodedBase, 1)
		e.statAffects(t, statStr)
	}
}

// fxAnimateDead = O0C2:2E19 is_animated: Animate Dead. Corpses of the party list (status 6,
// kind 0) are put back on their feet on the caster's side, one per caster level.
func (e *Engine) fxAnimateDead() {
	e.rv.areaSpell = 1
	lvl := e.casterLevel(int(e.curSpell))
	e.nSpellTargets = 0
	sel := e.selected()
	for _, p := range e.party {
		if lvl <= 0 {
			break
		}
		if p.Status != 6 || p.Kind != 0 || !e.combatSquareFree(p, e.combatX(p), e.combatY(p), true) {
			continue
		}
		data := byte(int(p.Side)<<4 + e.casterLevel(int(e.curSpell)))
		p.Side = sel.Side
		p.Auto = 1
		p.Undead = 2
		p.FightLevel = 0
		p.BaseMove = 6
		p.Memorized = [len(p.Memorized)]byte{}
		if p.Control > 0x7F {
			p.Control = 0xB2
		} else {
			p.Control = 0xB3
		}
		p.Kind = 4
		if e.gameState == 5 {
			p.Combat.Target = nil
		}
		lvl--
		if e.Revive(p, p.MaxHP) {
			e.isUnaffected(p, 0x20, 0, data, 1, 0, false, "is animated")
			p.Status = 1
		}
	}
}

// fxCureBlindness = O0C2:2FC4 can_see: cures blindness (affect 0x21): "can see".
func (e *Engine) fxCureBlindness() {
	if e.CureAffect(e.tgt(1), 0x21) {
		e.hitMessage(e.tgt(1), true, "can see")
	}
}

// fxBlindness = O0C2:300A is_blind: "is blind".
func (e *Engine) fxBlindness() { e.applySpell(int(e.curSpell), 0, 0, 0, 0, "is blind") }

// cureDiseaseAffects = O0C2:3037: with the cure flag (DS:681F) set, cures
// affects 0x22, 0x2B (and 0x2C, 0x1F) and 0x32 (and 0x39) of the first target.
func (e *Engine) cureDiseaseAffects() bool {
	cured := false
	e.rv.cure = 1
	t := e.tgt(1)
	if e.CureAffect(t, 0x22) {
		cured = true
	}
	if e.CureAffect(e.tgt(1), 0x2B) {
		cured = true
		e.RemoveAffect(e.tgt(1), 0x2C, nil)
		e.RemoveAffect(e.tgt(1), 0x1F, nil)
	}
	if e.CureAffect(e.tgt(1), 0x32) {
		cured = true
		e.RemoveAffect(e.tgt(1), 0x39, nil)
	}
	e.rv.cure = 0
	return cured
}

// fxCureDisease = O0C2:30DC: Cure Blindness spell body (the result is unused).
func (e *Engine) fxCureDisease() { e.cureDiseaseAffects() }

// fxDisease = O0C2:30F9 is_diseased: "is diseased".
func (e *Engine) fxDisease() { e.applySpell(int(e.curSpell), 0, 1, 0, 0, "is diseased") }

// Dispel Magic's odds: 50%, +5% for each caster level above the affect's,
// -2% for each below.
const (
	dispelOdds      = 50
	dispelPerAbove  = 5
	dispelPerBelow  = 2
	affectLevelMask = 0x0F // an affect's data: the caster level it was cast at
)

// dispelChance: the odds of a caster level against an affect's level.
func dispelChance(lvl, against byte) byte {
	switch {
	case lvl > against:
		return (lvl-against)*dispelPerAbove + dispelOdds
	case lvl < against:
		return dispelOdds - (against-lvl)*dispelPerBelow
	}
	return dispelOdds
}

// fxDispelMagic = O0C2:31E4 is_affected3: Dispel Magic. Each pass walks the
// affects of the first target (the loop variable only counts passes);
// each temporary affect may go. Then the clouds on the nine squares round
// the target square: a cloud whose caster's mage level does not stop it
// lifts, else it stays for good.
func (e *Engine) fxDispelMagic() {
	e.rv.areaSpell = 1
	lvl := byte(e.casterLevel(int(e.curSpell)))
	for i := 1; i <= e.nSpellTargets; i++ {
		hit := false
		t := e.tgt(1)
		for _, a := range append([]*Affect(nil), t.Affects...) {
			if a.Data == 0xFF {
				continue
			}
			e.v4845 = a.Data & affectLevelMask
			e.v4844 = dispelChance(lvl, e.v4845)
			if byte(e.Dice(1, 100)) <= e.v4844 {
				e.RemoveAffect(t, a.Type, a)
				hit = true
			}
		}
		if hit {
			e.hitMessage(t, true, "is affected")
		}
	}
	var x, y int8
	for k := 0; k <= 8; k++ {
		tx, ty := int8(e.tgtX), int8(e.tgtY)
		switch k {
		case 0:
			x, y = tx, ty
		case 1, 8:
			y = ty - 1
		case 2:
			x = tx + 1
		case 3, 7:
			y = ty
		case 4:
			y = ty + 1
		case 5:
			x = tx
		case 6:
			x = tx - 1
		}
		_, g := e.mapAt(int(x), int(y))
		var list []*cloudRecord
		var n int
		var affect byte
		switch g {
		case groundPoisonCloud:
			list, n, affect = e.poisonClouds, 9, affectCloudkill
		case groundStinkingCloud:
			list, n, affect = e.stinkingClouds, 4, affectStinkingCloud
		default:
			continue
		}
		for _, r := range append([]*cloudRecord(nil), list...) {
			for j := 0; j < n; j++ {
				if sx, sy := r.square(dispelCloudSquares[:], j); sx != int(x) || sy != int(y) || r.stays {
					continue
				}
				if !e.dispelsCloud(r.caster, lvl) {
					r.stays = true
					continue
				}
				for _, a := range r.caster.Affects {
					if a.Type == affect && a.Data>>4 == r.number {
						e.RemoveAffect(r.caster, affect, a)
						break
					}
				}
			}
		}
	}
}

// dispelsCloud = O0C2:3126 (caster): the dispel roll against the mage
// level of a cloud's caster (old levels counted by the selected
// character's dual class).
func (e *Engine) dispelsCloud(caster *Character, lvl byte) bool {
	old := byte(0)
	if oldClassUsable(e.selected()) {
		old = 1
	}
	e.v4845 = caster.OldLevels[clsMage]*old + caster.Levels[clsMage]
	e.v4844 = dispelChance(lvl, e.v4845)
	return byte(e.Dice(1, 100)) <= e.v4844
}

// fxPrayer = O0C2:359A is_praying: "is praying", cast at caster level with the side in the high nibble.
func (e *Engine) fxPrayer() {
	e.applySpell(int(e.curSpell), int(e.selected().Side)<<4+e.casterLevel(int(e.curSpell)), 0, 0, 0, "is praying")
}

// fxNeutralizePoison = O0C2:41DA cure_poison: an animated target drops
// out; a poisoned one (affect 55) is back on its feet with at least a hit
// point, without its poison affects.
func (e *Engine) fxNeutralizePoison() {
	t := e.tgt(1)
	if t.Status == statusAnimated {
		e.setTgt(1, nil)
		return
	}
	if e.FindAffect(t, affectPoisoned) != nil {
		if t.HP == 0 {
			t.HP = 1
		}
		e.rv.cure = 1
		e.RemoveAffect(t, affectPoisoned, nil)
		e.RemoveAffect(t, affectSlowPoison, nil)
		e.RemoveAffect(t, affectPoisonDamage, nil)
		e.rv.cure = 0
		e.combatName(t, "is unpoisoned", combatMsgRow, true)
		t.Okay = 1
		t.Status = statusOkay
		return
	}
	e.combatName(t, "is unaffected", combatMsgRow, true)
}

// fxPoison = O0C2:42D0: the spell's affect on the targets; then the
// caster's own target is tried against the poison sting (affect 64): when
// nothing there blocks it, the sting handler runs for the caster.
func (e *Engine) fxPoison() {
	e.applySpell(int(e.curSpell), 0, 0, 0, dmgMagic, "")
	sel := e.selected()
	e.rv.affVal = affectPoisonSting
	e.affectGroup(sel.Combat.Target, 9)
	if e.rv.affVal == affectPoisonSting {
		e.callAffect(affectPoisonSting, sel, nil, 0) // the affect argument is an unset local
	}
}

// sticksToSnakesTooBig: the hit dice from which a target is too big for the snakes.
const sticksToSnakesTooBig = 6

// fxSticksToSnakes = O0C2:435A cast_flattern: a small target gets the
// snakes (and their handler runs at once); a big one smashes them flat.
func (e *Engine) fxSticksToSnakes() {
	t := e.tgt(1)
	if int8(t.MaxLevel) < sticksToSnakesTooBig {
		e.applySpell(int(e.curSpell), e.casterLevel(int(e.curSpell)), 0, 0, dmgMagic, "")
		if a := e.FindAffect(t, affectSticksToSnakes); a != nil {
			e.callAffect(affectSticksToSnakes, t, a, 0)
		}
		return
	}
	e.combatName(t, "smashes them flat", combatMsgRow, true)
}

// fxCureCritical = O0C2:43F0: heals 3d8+3.
func (e *Engine) fxCureCritical() {
	if e.nSpellTargets == 0 {
		return
	}
	t := e.tgt(1)
	if e.Heal(t, e.Dice(3, 8)+3, false) {
		e.healedMessage(t)
	}
}

// fxCauseCritical = O0C2:4431: 3d8+3 of damage type 8.
func (e *Engine) fxCauseCritical() {
	e.applySpell(int(e.curSpell), 0, 0, e.RollDice(3, 8)+3, dmgMagic, "")
}

// fxDispelEvil = O0C2:4479 is_affected4: the caster is warded against
// enchanted creatures (affect 4, Dispel Evil's own amount), then the spell's
// affect on the targets.
func (e *Engine) fxDispelEvil() {
	e.isUnaffected(e.selected(), affectWardEnchanted, e.spellAmount(spellDispelEvil), 0, 0, 0, false, "")
	e.applySpell(int(e.curSpell), 0, 0, 0, 0, "is affected")
}

// fxFlameStrike = O0C2:44DA: 6d8 of damage type 9.
func (e *Engine) fxFlameStrike() {
	e.applySpell(int(e.curSpell), 0, 0, e.RollDice(6, 8), dmgMagic|dmgFire, "")
}

// fxRaiseDead = O0C2:451A cast_raise: a dead (or animated) target with
// constitution left and not an elf comes back with a hit point, a point of
// constitution less.
func (e *Engine) fxRaiseDead() {
	t := e.tgt(1)
	if (t.Status == statusDead || t.Status == statusAnimated) && t.Stats[statCon].Base > 0 && t.Race != raceElf {
		e.rv.cure = 1
		e.RemoveAffect(t, affectCollapse, nil)
		e.RemoveAffect(t, affectPoisoned, nil)
		e.rv.cure = 0
		t.Status = statusOkay
		t.Okay = 1
		t.Stats[statCon].Base--
		e.statAffects(t, statCon)
		t.HP = 1
		e.combatName(t, "is raised", combatMsgRow, true)
	}
}

// The squares beside a cone's far end, by the direction of its last step
// (DS:0A12, DS:0A1A and DS:0A22, DS:0A2A).
var (
	coneSideAX = [8]int8{-1, 0, 0, 1, 1, 0, 0, -1}
	coneSideAY = [8]int8{0, -1, -1, 0, 0, 1, 1, 0}
	coneSideBX = [8]int8{1, 1, 0, 0, -1, -1, 0, 0}
	coneSideBY = [8]int8{0, 0, 1, 1, 0, 0, -1, -1}
)

// The cone's open range: O0C2:17CF hands lineOfSight an unset local as the
// range; the port leaves the cone unlimited by range (walls still stop it).
const coneRange = 0x7FFF

// clampToMap = O0C2:16B7 (var x, var y): onto the combat map.
func clampToMap(x, y *int) {
	*x = min(max(*x, 0), cmapWidth-1)
	*y = min(max(*y, 0), cmapHeight-1)
}

// coneTargets = O0C2:17CF (x0, y0, x1, y1, width, length): the combatants
// in a cone from (x0,y0) towards (x1,y1): the line is drawn on, repeating
// its steps, to length squares (a diagonal step counts 1.5), stopped by
// walls, and walked; widths 2 and 3 add the lines to the squares beside
// its end. Everyone found but the caster becomes a target.
func (e *Engine) coneTargets(x0, y0, x1, y1, width, length int) {
	l := newLine(x0, y0, x1, y1)
	var dirs []byte
	for e.step(l) {
		dirs = append(dirs, l.dir)
	}
	last := len(dirs) - 1 // a byte in the asm: 255 with no step
	i := 0
	length *= 2
	walked := byte(l.steps)
	for walked < byte(length) {
		if x1 >= cmapWidth-1 || x1 <= 0 || y1 >= cmapHeight-1 || y1 <= 0 {
			break
		}
		d := byte(0)
		if i < len(dirs) {
			d = dirs[i]
		}
		if d%2 == 0 {
			walked += 2
		} else {
			walked += 3
		}
		x1 += int(stepX[d])
		y1 += int(stepY[d])
		if i == last {
			i = 0
		} else {
			i++
		}
	}
	clampToMap(&x1, &y1)
	rng := coneRange
	e.lineOfSight(&rng, &x1, &y1, x0, y0)
	var found [31]byte // [1..]: the combatants met, 0 ends the list
	n := 1
	dup := false
	var lastDir byte
	l = newLine(x0, y0, x1, y1)
	for e.step(l) {
		who, _ := e.mapAt(l.x, l.y)
		if who > 0 {
			if n > 1 {
				for k := 1; k <= n-1; k++ {
					if found[k] == who {
						dup = true // never cleared again: later combatants are left out
					}
				}
			} else {
				dup = false
			}
			if !dup && n < len(found) {
				found[n] = who
				n++
			}
		}
		lastDir = l.dir
	}
	side := func(sx, sy int) {
		s := newLine(x0, y0, sx, sy)
		for e.step(s) {
			who, _ := e.mapAt(s.x, s.y)
			if who == 0 {
				continue
			}
			for k := 1; k < len(found); k++ {
				if found[k] == who {
					break
				}
				if found[k] == 0 {
					found[k] = who
					break
				}
			}
		}
	}
	if width > 1 {
		ax, ay := x1+int(coneSideAX[lastDir%8]), y1+int(coneSideAY[lastDir%8])
		clampToMap(&ax, &ay)
		bx, by := x1+int(coneSideBX[lastDir%8]), y1+int(coneSideBY[lastDir%8])
		clampToMap(&bx, &by)
		side(bx, by)
		if width > 2 {
			side(ax, ay)
		}
	}
	e.nSpellTargets = 0
	sel := e.selected()
	for k := 1; k < len(found) && found[k] != 0; k++ {
		if c := e.combatant(found[k]); c != sel {
			e.nSpellTargets++
			e.setTgt(e.nSpellTargets, c)
		}
	}
}

// fxSlayLiving = O0C2:45FD cast_slay: the target's affects may stop the
// spell (they clear the damage, DS:6FAE, set to 67); else it dies unless it
// saves, then taking 2d8+1. The damage type stays 8 after.
func (e *Engine) fxSlayLiving() {
	t := e.tgt(1)
	e.rv.dmgType = dmgType64
	e.rv.damage = slayMark
	e.affectGroup(t, 9)
	if e.rv.damage != 0 {
		if !e.SavingThrow(t, saveVsSpell, 0) {
			e.killWithMessage(t, statusDead, "is slain")
		} else {
			e.rv.dmgType = dmgMagic
			e.damagePerson(t, e.RollDice(2, 8)+1, 0, false)
		}
		return
	}
	e.combatName(t, "is unaffected", combatMsgRow, true)
}

// slayMark: Slay Living's damage value before the target's affects see it.
const slayMark = 67

// Entangle's duration: O0C2:46BB asks for the amount of "spell 136", past
// the spell table; the bytes there (DS:4060) make 6 + 3 per caster level.
const (
	entangleBase     = 6
	entanglePerLevel = 3
	entangleAsSpell  = 136
)

// fxEntangle = O0C2:46BB: outdoors only, each target is rooted unless it
// saves.
func (e *Engine) fxEntangle() {
	if wordAt(e.area, 2*areaWordInside) != 0 {
		return
	}
	rounds := entangleBase + entanglePerLevel*e.casterLevel(entangleAsSpell)
	for i := 1; i <= e.nSpellTargets; i++ {
		t := e.tgt(i)
		if t == nil {
			continue
		}
		saved := e.SavingThrow(t, saveVsSpell, 0)
		e.isUnaffected(t, affectRooted, rounds, 0, 0, 1, saved, "is entangled")
	}
}

// fxFaerieFire = O0C2:4777 cast_highlisht.
func (e *Engine) fxFaerieFire() { e.holdTargets(0, "is highlighted") }

// fxInvisibleToAnimals = O0C2:47A4 cast_invisible2.
func (e *Engine) fxInvisibleToAnimals() { e.applySpell(int(e.curSpell), 0, 0, 0, 0, "is invisible") }

// fxCharmMonsters = O0C2:47DC cast_charmed: held like Hold Monsters; the
// charm handler runs at once for each one charmed.
func (e *Engine) fxCharmMonsters() {
	e.holdTargets(0, "is charmed")
	for i := 1; i <= e.nSpellTargets; i++ {
		if t := e.tgt(i); t != nil {
			if a := e.FindAffect(t, affectCharmed); a != nil {
				e.callAffect(affectCharmed, t, a, 0)
			}
		}
	}
}

// fxConfusion = O0C2:486D cast_confuse: up to 2d8 targets are confused
// unless they save.
func (e *Engine) fxConfusion() {
	if n := e.Dice(2, 8); e.nSpellTargets > n {
		e.nSpellTargets = n
	}
	for i := 1; i <= e.nSpellTargets; i++ {
		t := e.tgt(i)
		if t == nil {
			continue
		}
		saved := e.SavingThrow(t, saveVsSpell, 0)
		e.isUnaffected(t, affectConfused, e.spellAmount(spellConfusion), 0, 0, 1, saved, "is confused")
	}
}

// fxDimensionDoor = O0C2:4931 cast_teleport: an engulfed caster is let go
// by whoever holds it; it steps off the map and onto the target square.
func (e *Engine) fxDimensionDoor() {
	sel := e.selected()
	if e.FindAffect(sel, affectEngulfed) != nil {
		e.buildNear(e.combatX(sel), e.combatY(sel), 1, 0xFF, 1)
		for _, n := range e.near.list {
			c := e.combatant(n.index)
			a := e.FindAffect(c, affectHugging)
			if a == nil {
				a = e.FindAffect(c, affectEngulfing)
			}
			if a != nil && e.combatant(a.Data) == sel {
				e.RemoveAffect(c, affectHugging, nil)
				e.RemoveAffect(c, affectEngulfing, nil)
			}
		}
	}
	e.combatClearIcon(0, 0, e.combatIndex(sel))
	e.combatSquareFree(sel, e.tgtX, e.tgtY, false)
	e.combatCentre(e.combatX(sel), e.combatY(sel), 0, 8)
	e.combatName(sel, "teleports", combatMsgRow, true)
}

// The cone of Fear: 3 wide, 6 long.
const (
	fearConeWidth  = 3
	fearConeLength = 6
)

// fxFear = O0C2:4AE2 cast_terror: everyone in the cone that does not save
// runs, out of the player's hands.
func (e *Engine) fxFear() {
	sel := e.selected()
	e.coneTargets(e.combatX(sel), e.combatY(sel), e.tgtX, e.tgtY, fearConeWidth, fearConeLength)
	for i := 1; i <= e.nSpellTargets; i++ {
		t := e.tgt(i)
		if e.SavingThrow(t, saveVsSpell, 0) {
			e.combatName(t, "is unaffected", combatMsgRow, true)
			continue
		}
		e.isUnaffected(t, affectRunningAway, e.spellAmount(spellFear), 0, 1, 1, false, "runs in terror")
		t.Combat.Turned = 1
		t.Auto = 1
		if t.Control <= controlHired {
			t.Control = controlCharmed
		}
		t.Combat.Target = nil
	}
}

// fxFireShield = O0C2:4C42 cast_protection: the caster chooses hot (fire
// body) or cold (ice body) flames, with the shield that burns attackers;
// a computer caster picks at random.
func (e *Engine) fxFireShield() {
	sel := e.selected()
	rounds := func() int { return e.spellAmount(spellFireShield) }
	for done := false; !done; {
		var key byte
		if sel.Auto != 0 {
			if e.Dice(1, 10) > 5 {
				key = 'H'
			} else {
				key = 'C'
			}
		} else {
			key, _ = e.DisplayInput("flame type: ", "Hot Cold", colourLightMagenta, colourGreen, colourWhite, false, false)
		}
		switch key {
		case 'H':
			e.isUnaffected(sel, affectFireBody, rounds(), 0, 0, 0, false, "is protected")
			e.isUnaffected(sel, affectZapsAttackers, rounds(), 0, 0, 0, false, "")
			done = true
		case 'C':
			e.isUnaffected(sel, affectIceBody, rounds(), 0, 0, 0, false, "")
			e.isUnaffected(sel, affectZapsAttackers, rounds(), 0, 0, 0, false, "")
			done = true
		default:
			key, _ = e.DisplayInput("Abort spell? ", "Yes No", colourLightMagenta, colourGreen, colourWhite, false, false)
			done = key == 'Y'
		}
	}
}

// fxFumble = O0C2:4DF6 spell_slow: an unsaved target is stunned (clumsy),
// a saved one slowed; the handler runs at once. Then the spell's own affect.
func (e *Engine) fxFumble() {
	t := e.tgt(1)
	e.rv.dmgType = dmgType64
	if !e.SavingThrow(t, saveVsSpell, 0) {
		e.isUnaffected(t, affectStunned, e.spellAmount(spellFumble), 0, 0, 0, false, "is clumsy")
		if e.FindAffect(t, affectStunned) != nil {
			e.callAffect(affectStunned, t, nil, 0)
		}
	} else {
		e.isUnaffected(t, affectSlow, e.spellAmount(spellFumble), 0, 0, 0, false, "is slowed")
		if e.FindAffect(t, affectSlow) != nil {
			e.callAffect(affectSlow, t, nil, 0)
		}
	}
	e.applySpell(int(e.curSpell), 0, 1, 0, 0, "is clumsy")
}

// fxIceStorm = O0C2:4F0B: 3d10 of cold and magic.
func (e *Engine) fxIceStorm() {
	e.applySpell(int(e.curSpell), 0, 0, e.RollDice(3, 10), dmgCold|dmgMagic, "")
}

// fxMinorGlobe = O0C2:4F4E.
func (e *Engine) fxMinorGlobe() { e.applySpell(int(e.curSpell), 0, 0, 0, 0, "is protected") }

// fxCloudkill = O0C2 spell_poisonous_cloud.
func (e *Engine) fxCloudkill() { e.createCloud(e.poisonKind()) }

// The cone of Cone of Cold: 2 wide, half the caster level long.
const coneOfColdWidth = 2

// fxConeOfCold = O0C2:5550: level + level d4 of cold and magic.
func (e *Engine) fxConeOfCold() {
	sel := e.selected()
	lvl := byte(e.casterLevel(int(e.curSpell)))
	length := (int(byte(e.casterLevel(int(e.curSpell)))) + 1) >> 1
	if length < 1 {
		length = 1
	}
	e.coneTargets(e.combatX(sel), e.combatY(sel), e.tgtX, e.tgtY, coneOfColdWidth, length)
	e.applySpell(int(e.curSpell), 0, 0, int(lvl)+e.RollDice(int(lvl), 4), dmgCold|dmgMagic, "")
}

// fxFeeblemind = O0C2:55F2: the target's spell save is made harder for the
// spell (a cleric's -1, a mage's +4, others +2) and put back after; the
// feeblemind handler runs at once.
func (e *Engine) fxFeeblemind() {
	t := e.tgt(1)
	keep := t.Saves[saveVsSpell]
	switch t.Class {
	case clsCleric:
		t.Saves[saveVsSpell]--
	case clsMage:
		t.Saves[saveVsSpell] += 4
	default:
		t.Saves[saveVsSpell] += 2
	}
	e.applySpell(int(e.curSpell), 0, 0, 0, 0, "")
	if e.FindAffect(t, affectFeebleminded) != nil {
		e.callAffect(affectFeebleminded, t, nil, 0)
	}
	t.Saves[saveVsSpell] = keep
}

// fxPlain = O0C2:569E, 56CC, 56FA: the spell record alone.
func (e *Engine) fxPlain() { e.applySpell(int(e.curSpell), 0, 0, 0, 0, "") }

// The cone of spell 98: 1 wide, 3 long, 6d6.
const (
	spell98ConeWidth  = 1
	spell98ConeLength = 3
	kindSpell98Saves  = 18 // the only kind that rolls a save against it
)

// fxSpell98 = O0C2:5727: 6d6 in a narrow cone, taken as saved (halved)
// except by creatures of kind 18.
func (e *Engine) fxSpell98() {
	sel := e.selected()
	e.coneTargets(e.combatX(sel), e.combatY(sel), e.tgtX, e.tgtY, spell98ConeWidth, spell98ConeLength)
	for i := 1; i <= e.nSpellTargets; i++ {
		t := e.tgt(i)
		if t == nil {
			continue
		}
		e.damagePerson(t, e.RollDice(6, 6), e.spellRec(98, spellFieldSaveMode), t.Kind != kindSpell98Saves)
	}
}

// The breath and gaze handlers of O0C2 (setup_spells puts them in the
// affect table). Each is (char, affect, flag).

// Breaths a combat allows (an affect's data counts them down).
const breathsPerCombat = 3

// mirrorWord: the item name word of a mirror.
const mirrorWord = 118

// stoningGazeHandler = O0C2:585E spell_stone: the gaze goes at the chosen
// target; a gazer open to reflection is turned back by a readied mirror.
// Unless the one gazed at saves (petrify), it is stoned.
func (e *Engine) stoningGazeHandler(c *Character, a *Affect, flag byte) {
	c.Combat.Target = nil
	e.spellTargetHook(65, 1)
	t := c.Combat.Target
	if t == nil {
		return
	}
	e.combatName(c, "gazes...", combatMsgRow, false)
	e.hitColours(18)
	e.flyMissile(e.combatX(c), e.combatY(c), e.combatX(t), e.combatY(t), 4, 45)
	if e.FindAffect(c, affectGazeReflectable) != nil {
		for _, it := range t.Items {
			if it.Readied != 0 && (it.Words[0] == mirrorWord || it.Words[1] == mirrorWord || it.Words[2] == mirrorWord) {
				e.combatName(t, "reflects it!", 12, false)
				e.flyMissile(e.combatX(t), e.combatY(t), e.combatX(c), e.combatY(c), 4, 45)
				t = c
				break
			}
		}
	}
	if !e.SavingThrow(t, saveVsPetrify, 0) {
		e.killWithMessage(t, statusStoned, "is Stoned")
	}
}

// Spitting acid: within 6 squares, 30% to hit.
const (
	spitAcidRange  = 7
	spitAcidChance = 30
)

// spitAcidHandler = O0C2:5BEE spell_spit_acid: at the chosen target in
// range, its hit points in acid (half on a breath save), or a miss.
func (e *Engine) spitAcidHandler(c *Character, a *Affect, flag byte) {
	e.spellTargetHook(65, 1)
	t := c.Combat.Target
	roll := e.Dice(1, 100)
	if t == nil || e.rangeTo(c, t) >= spitAcidRange {
		return
	}
	if roll <= spitAcidChance {
		e.combatName(c, "Spits Acid", combatMsgRow, true)
		e.hitColours(23)
		e.flyMissile(e.combatX(c), e.combatY(c), e.combatX(t), e.combatY(t), 1, 30)
		e.damagePerson(t, int(c.MaxHP), 2, e.SavingThrow(t, saveVsBreath, 0))
		return
	}
	e.combatName(c, "Spits Acid and Misses", combatMsgRow, true)
}

// Cones of the breath weapons.
const (
	acidConeWidth  = 1
	acidConeLength = 6
	fireConeWidth  = 3
	fireConeLength = 9
)

// breathAcidHandler = O0C2:5D24 spell_breathes_acid: three breaths a
// combat in a cone, only when every one in it is a foe; its hit points
// each, halved on a breath save.
func (e *Engine) breathAcidHandler(c *Character, a *Affect, flag byte) {
	ok := false
	if e.combatRound == 0 {
		a.Data = breathsPerCombat
	}
	if a.Data == 0 {
		return
	}
	e.rv.dmgType = dmgAcid | dmgType32
	cx, cy := e.combatX(c), e.combatY(c)
	ok = e.spellTargetHook(0x3D, 1)
	if ok {
		e.coneTargets(cx, cy, e.tgtX, e.tgtY, acidConeWidth, acidConeLength)
	}
	for i := 1; i <= e.nSpellTargets; i++ {
		if e.tgt(i).Side != otherSide(c) {
			ok = false
		}
	}
	if !ok || e.nSpellTargets == 0 {
		return
	}
	e.combatName(c, "breathes acid", combatMsgRow, true)
	e.hitColours(18)
	e.flyMissile(cx, cy, e.combatX(e.tgt(1)), e.combatY(e.tgt(1)), 1, 30)
	for i := 1; i <= e.nSpellTargets; i++ {
		if t := e.tgt(i); t != nil {
			e.damagePerson(t, int(c.MaxHP), 2, e.SavingThrow(t, saveVsBreath, 0))
		}
	}
	a.Data--
	e.endTurn(c)
}

// otherSide = O0ED:2564 (c): the side c fights.
func otherSide(c *Character) byte {
	if c.Side == 0 {
		return 1
	}
	return 0
}

// breathFireHandler = O0C2:5F04 spell_breathes_fire: three breaths a
// combat in a wide cone; its hit points each, halved on a breath save.
func (e *Engine) breathFireHandler(c *Character, a *Affect, flag byte) {
	if e.combatRound == 0 {
		a.Data = breathsPerCombat
	}
	if a.Data == 0 {
		return
	}
	e.rv.dmgType = dmgFire | dmgType32
	cx, cy := e.combatX(c), e.combatY(c)
	if !e.spellTargetHook(61, 1) {
		return
	}
	e.coneTargets(cx, cy, e.tgtX, e.tgtY, fireConeWidth, fireConeLength)
	if e.nSpellTargets <= 0 {
		return
	}
	e.combatName(c, "breathes fire", combatMsgRow, true)
	e.hitColours(18)
	e.flyMissile(cx, cy, e.combatX(e.tgt(1)), e.combatY(e.tgt(1)), 1, 30)
	for i := 1; i <= e.nSpellTargets; i++ {
		if t := e.tgt(i); t != nil {
			e.damagePerson(t, int(c.MaxHP), 2, e.SavingThrow(t, saveVsBreath, 0))
		}
	}
	a.Data--
	e.endTurn(c)
}

// Breathing fire up close: half the time, at a target next to it, 7 points.
const (
	closeFireChance = 50
	closeFireRange  = 2
	closeFireDamage = 7
)

// breathFireCloseHandler = O0C2:6091 cast_breath_fire.
func (e *Engine) breathFireCloseHandler(c *Character, a *Affect, flag byte) {
	e.spellTargetHook(65, 1)
	t := c.Combat.Target
	if e.Dice(1, 100) <= closeFireChance && e.rangeTo(c, t) < closeFireRange && t != nil {
		e.rv.dmgType = dmgFire
		e.endTurn(c)
		e.combatName(c, "Breathes Fire", combatMsgRow, true)
		e.hitColours(23)
		e.flyMissile(e.combatX(c), e.combatY(c), e.combatX(t), e.combatY(t), 1, 30)
		e.damagePerson(t, closeFireDamage, 2, e.SavingThrow(t, saveVsBreath, 0))
	}
}

// Throwing lightning: only in the first rounds, 16d6 a strike.
const (
	throwLightningRounds = 4
	throwLightningDice   = 16
	throwLightningReach  = 10
)

// throwLightningHandler = O0C2:61B1 cast_throw_lightning: a bolt at the
// target square and on down the line (each strike rolls its own 16d6).
func (e *Engine) throwLightningHandler(c *Character, a *Affect, flag byte) {
	if e.combatRound >= throwLightningRounds {
		return
	}
	cx, cy := e.combatX(c), e.combatY(c)
	e.combatName(c, "throws lightning", combatMsgRow, true)
	e.spellTargetHook(51, 1)
	e.removeInvisibility(c)
	e.hitColours(19)
	e.flyMissile(cx, cy, e.tgtX, e.tgtY, 4, 50)
	var bounced bool // an unset local in 61B1
	e.strikeSquare(e.tgtX, e.tgtY, e.RollDice(throwLightningDice, 6), saveVsSpell, 0, &bounced)
	e.boltPath(throwLightningReach, e.RollDice(throwLightningDice, 6), saveVsSpell, 0)
	e.endTurn(c)
}

// Paralysing gaze: 60 rounds.
const paralysingGazeRounds = 60

// gazeParalyseHandler = O0C2:62A4 cast_gaze_paralyze: the chosen target
// is paralysed unless it saves (petrify).
func (e *Engine) gazeParalyseHandler(c *Character, a *Affect, flag byte) {
	c.Combat.Target = nil
	e.spellTargetHook(36, 1)
	t := c.Combat.Target
	if t == nil {
		return
	}
	e.combatName(c, "gazes...", combatMsgRow, false)
	e.hitColours(18)
	e.flyMissile(e.combatX(c), e.combatY(c), e.combatX(t), e.combatY(t), 4, 45)
	if !e.SavingThrow(t, saveVsPetrify, 0) {
		e.AddAffect(t, affectParalyzed, paralysingGazeRounds, 0xFF, 0)
		e.combatName(t, "is paralyzed", combatMsgRow, false)
	}
}
