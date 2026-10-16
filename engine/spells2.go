package engine

// Spell effect routines of O113 (1F0D..60ED). CastSpell (O113:1403) calls
// them through the table DS:8A8E filled by O113:67A1, indexed by the spell
// number (DS:87D8); each is called without arguments and works on the
// spell globals: the target count DS:8C66 and list DS:8C63 (entry 1 at
// DS:8C67 is the first target).

// fxBless = O113:1F14: Bless, "is Blessed" on the selected character's side.
func (e *Engine) fxBless() {
	e.applySpellToSide(int(e.selected().Side), "is Blessed")
}

// fxBane = O113:1F47: Bane, "is Cursed".
func (e *Engine) fxBane() {
	side := 0
	if isParty(e.selected()) {
		side = 1
	}
	e.applySpellToSide(side, "is Cursed")
}

// fxCureLight = O113:1F7E and O113:5815: Cure Light Wounds on the first target, 1d8.
func (e *Engine) fxCureLight() {
	if e.nSpellTargets != 0 {
		if e.healPlayer(e.tgt(1), byte(e.Dice(1, 8)), false) {
			e.healedMessage(e.tgt(1))
		}
	}
}

// fxCauseLight = O113:1FBC: 1d8 magic damage.
func (e *Engine) fxCauseLight() {
	e.applySpell(int(e.curSpell), 0, 0, e.RollDice(1, 8), 8, "")
}

// fxAffected = O113:1FF2: "is Affected" (shared by several spells).
func (e *Engine) fxAffected() { e.applySpell(int(e.curSpell), 0, 0, 0, 0, "is Affected") }

// fxProtected = O113:202B: "is Protected".
func (e *Engine) fxProtected() { e.applySpell(int(e.curSpell), 0, 0, 0, 0, "is Protected") }

// fxColdResistant = O113:2064: "is Resistant to Cold".
func (e *Engine) fxColdResistant() {
	e.applySpell(int(e.curSpell), 0, 0, 0, 0, "is Resistant to Cold")
}

// fxBurningHands = O113:209E: damage equal to the caster level, type 9.
func (e *Engine) fxBurningHands() {
	e.applySpell(int(e.curSpell), 0, 0, e.casterLevel(int(e.curSpell)), 9, "")
}

// fxCharmPerson = O113:20D1: Charm Person works on the demihuman races
// and on person-like monsters.
func (e *Engine) fxCharmPerson() {
	t := e.tgt(1)
	if t == nil {
		return
	}
	if int8(t.Race) < raceHuman || t.MonsterFlags&monsterPersonLike != 0 {
		lvl := e.casterLevel(int(e.curSpell))
		e.applySpell(int(e.curSpell), (int(e.selected().Side)<<7+lvl)&0xFF, 1, 0, 0, "is Charmed")
		if a := e.FindAffect(t, affectCharmed); a != nil {
			e.callAffect(affectCharmed, t, a, 0)
		}
		return
	}
	e.combatName(t, "is Unaffected", combatMsgRow, true)
}

// spellStronger = O113:21AF: Enlarge. The strength comes from the caster
// level: 18 with 18/00..18/100 for levels 1..6, then 19..22, and 23 above.
func (e *Engine) spellStronger() {
	t := e.tgt(1)
	str, str00 := byte(strengthMaximum18), byte(0)
	switch e.casterLevel(int(e.curSpell)) {
	case 1:
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
	default:
		str = 23
	}
	if code, ok := strongerThan(t, str, str00); ok {
		e.isUnaffected(t, affectEnlarge, e.spellAmount(int(e.curSpell)), code, 0, 0, false, "is Enlarged")
		e.statAffects(t, statStr)
		return
	}
	e.combatName(t, "is Unaffected", combatMsgRow, true)
}

// spellReduce = O113:22F6 has_been_reduced: the magical strength (affect
// 12) goes unless the target saves.
func (e *Engine) spellReduce() {
	t := e.tgt(1)
	if t != nil && e.nSpellTargets > 0 && !e.SavingThrow(t, saveVsSpell, 0) && e.FindAffect(t, affectEnlarge) != nil {
		e.RemoveAffect(t, affectEnlarge, nil)
		e.statAffects(t, statStr)
		e.combatName(t, "has been reduced", combatMsgRow, true)
	}
}

// spellFriends = O113:2389: the charisma affect at 2d4.
func (e *Engine) spellFriends() {
	e.applySpell(int(e.curSpell), e.Dice(2, 4), 1, 0, 0, "is Friendly")
	e.statAffects(e.selected(), statCha)
}

// fxMagicMissile = O113:23DC: magic damage of ((level+1)/2)d4 + (level+1)/2.
func (e *Engine) fxMagicMissile() {
	n := (e.casterLevel(int(e.curSpell)) + 1) & 0xFF >> 1
	e.applySpell(int(e.curSpell), 0, 0, (n+e.RollDice(n, 4))&0xFF, 8, "")
}

// fxShielded = O113:243B: "is Shielded".
func (e *Engine) fxShielded() { e.applySpell(int(e.curSpell), 0, 0, 0, 0, "is Shielded") }

// fxShockingGrasp = O113:2475: caster level + 1d8 electrical damage.
func (e *Engine) fxShockingGrasp() {
	e.rv.dmgType = dmgElectricity | dmgMagic
	lvl := e.casterLevel(int(e.curSpell))
	e.applySpell(int(e.curSpell), 0, 0, (lvl+e.RollDice(1, 8))&0xFF, dmgElectricity|dmgMagic, "")
}

// fxSleep = O113:24C8: Sleep. 4d4 hit dice of targets fall asleep, weakest
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
		if t.Status != 1 && e.FindAffect(t, affectAsleep) == nil && e.v4844 >= e.v4845 {
			e.v4844 -= e.v4845
		} else {
			e.setTgt(i, nil)
		}
	}
	e.applySpell(int(e.curSpell), 0, 0, 0, 0, "falls Asleep")
}

// fxHold = O113:260A: Hold Person/Monster. The save bonus depends on the
// number of targets.
func (e *Engine) fxHold() {
	var bonus int8 // more targets leave the local unset (0)
	switch e.nSpellTargets {
	case 1:
		if e.curSpell == spellHoldPersonCleric {
			bonus = -2
		} else {
			bonus = -3
		}
	case 2:
		bonus = -1
	}
	e.holdTargets(bonus, "is Held")
}

// holdTargets = O113:1CC8 (bonus, msg): each target, last first, gets a
// missile (but the first one shot) and the spell record's affect unless it
// saves. Hold Person never holds a monster or a human.
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
		if (spell == spellFaerieFire || spell == spellCharmMonsters) && i == e.nSpellTargets {
			saved, saveMode = true, 1
		} else {
			saved = e.SavingThrow(t, int(e.spellRec(byte(spell), spellFieldSaveType)), int(bonus))
			saveMode = e.spellRec(byte(spell), spellFieldSaveMode)
		}
		if (t.Race == raceTribble || t.Race == raceMonster) && (spell == spellHoldPersonCleric || spell == spellHoldPersonMU) {
			saved = true
		}
		if !saved {
			var call byte
			if spell == spellCharmMonsters {
				call = 1
			}
			e.isUnaffected(t, e.spellRec(byte(spell), spellFieldAffect), e.spellAmount(spell), byte(e.casterLevel(spell)), call,
				saveMode, saved, msg)
		}
	}
}

// fxFireResistant = O113:2667.
func (e *Engine) fxFireResistant() {
	e.applySpell(int(e.curSpell), 0, 0, 0, 0, "is Fire Resistant (spell)")
}

// fxSilenced = O113:26A0: "is Silenced".
func (e *Engine) fxSilenced() { e.applySpell(int(e.curSpell), 0, 0, 0, 0, "is Silenced") }

// fxSlowPoison = O113:26D9: a poisoned target is brought back to at least
// 1 hit point and on its feet, taking a point of poison damage now and then.
func (e *Engine) fxSlowPoison() {
	t := e.tgt(1)
	if e.FindAffect(t, affectPoisoned) == nil {
		return
	}
	if t.HP == 0 {
		t.HP = 1
	}
	t.Status = statusOkay
	t.Okay = 1
	e.applySpell(int(e.curSpell), 0xFF, 1, 0, 0, "is Affected")
	e.AddAffect(t, affectPoisonDamage, 10, 0xFF, 1)
}

// fxCharmSnakes = O113:2773: the active snakes of the party list whose hit
// points fit in the caster's hit points become the targets.
func (e *Engine) fxCharmSnakes() {
	budget := e.selected().HP
	e.nSpellTargets = 0
	for _, p := range e.party {
		if p.CreatureFlags&creatureSnake != 0 && p.Okay != 0 && e.FindAffect(p, affectImmobile) == nil && budget >= p.HP {
			budget -= p.HP
			e.nSpellTargets++
			e.setTgt(e.nSpellTargets, p)
		}
	}
	e.applySpell(int(e.curSpell), 0, 0, 0, 0, "is Charmed")
}

// fxSpiritualHammer = O113:2857: applies the spell, then runs the
// Spiritual Hammer handler on the first target.
func (e *Engine) fxSpiritualHammer() {
	e.applySpell(int(e.curSpell), 0, 1, 0, 0, "")
	e.callAffect(affectSpiritualHammer, e.tgt(1), nil, 0)
}

// fxInvisibility = O113:28A4: "is invisible".
func (e *Engine) fxInvisibility() { e.applySpell(int(e.curSpell), 0, 0, 0, 0, "is invisible") }

// fxKnock = O113:28E9: Knock.
func (e *Engine) fxKnock() { e.applySpell(int(e.curSpell), 0, 0, 0, 0, "Knock-Knock") }

// fxMirrorImage = O113:2916: Mirror Image, 1d4 images in the high
// nibble of the level.
func (e *Engine) fxMirrorImage() {
	lvl := byte(e.Dice(1, 4))<<4 | byte(e.casterLevel(int(e.curSpell)))
	e.applySpell(int(e.curSpell), int(lvl), 0, 0, 0, "is Duplicated")
}

// fxRayOfEnfeeblement = O113:2974: "is Weakened".
func (e *Engine) fxRayOfEnfeeblement() { e.applySpell(int(e.curSpell), 0, 0, 0, 0, "is Weakened") }

// fxStinkingCloud = O113:29C5.
func (e *Engine) fxStinkingCloud() { e.createCloud(e.stinkingKind()) }

// hasClass: a level in the class now, or an old one above the level of
// the class taken up after (a dual-classed character's first class).
func hasClass(c *Character, k int) bool {
	return int8(c.Levels[k]) > 0 || int8(c.OldLevels[k]) > int8(c.MulticlassLevel)
}

// spellStrength = O113:2E2C: Strength. 1d4 for a mage, 1d6 for a cleric
// or thief, 1d8 for a fighter (the last that applies) on the strength in use;
// above 18 the fighters and rangers get exceptional strength.
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
		if hasClass(t, clsFighter) || hasClass(t, clsRanger) {
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

// fxHeal = O113:304E: Heal cures diseases, blindness and feeblemind, then
// brings the target to its maximum hit points less 1d4.
func (e *Engine) fxHeal() {
	t := e.tgt(1)
	if e.cureDiseaseAffects() {
		e.combatName(t, "is cured", combatMsgRow, false)
	}
	if e.CureAffect(t, affectBlind) {
		e.combatName(t, "is no longer blind", combatMsgRow, false)
	}
	if e.CureAffect(t, affectFeebleminded) {
		e.combatName(t, "regains intellect", combatMsgRow, false)
		e.statAffects(t, statInt)
		e.statAffects(t, statWis)
	}
	short := byte(e.Dice(1, 4))
	var amount byte
	if int(t.HP)+int(short) <= int(t.MaxHP) {
		amount = t.MaxHP - t.HP - short
	}
	if e.healPlayer(t, amount, false) {
		e.combatName(t, "is Healed", combatMsgRow, false)
	}
}

// fxCureBlindness = O113:31C5: "can see".
func (e *Engine) fxCureBlindness() {
	if e.CureAffect(e.tgt(1), affectBlind) {
		e.hitMessage(e.tgt(1), true, "can see")
	}
}

// fxBlindness = O113:3202: "is Blind".
func (e *Engine) fxBlindness() { e.applySpell(int(e.curSpell), 0, 0, 0, 0, "is Blind") }

// cureDiseaseAffects = O113:2FBD: with the cure flag set, cures the
// diseases of the first target and works out its scores again.
func (e *Engine) cureDiseaseAffects() bool {
	cured := false
	e.rv.cure = 1
	for _, a := range diseaseAffects[1:] {
		if e.CureAffect(e.tgt(1), a) {
			cured = true
		}
	}
	if cured {
		e.statAffects(e.tgt(1), statStr)
	}
	e.rv.cure = 0
	return cured
}

// fxCureDisease = O113:323B (the result is unused).
func (e *Engine) fxCureDisease() { e.cureDiseaseAffects() }

// fxDisease = O113:324C: "is Diseased".
func (e *Engine) fxDisease() { e.applySpell(int(e.curSpell), 0, 1, 0, 0, "is Diseased") }

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

// fxDispelMagic = O113:3337: Dispel Magic. Each pass walks the
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
			e.hitMessage(t, true, "is Affected")
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
		case groundDelayedBlast:
			list, n, affect = e.delayedBlasts, 1, affectDelayedBlast
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
		e.combatCentre(e.tgtX, e.tgtY, 3, 8)
	}
}

// dispelsCloud = O113:3285 (caster): the dispel roll against the mage
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

// fxPrayer = O113:36F6: "is Affected by Prayer", cast at caster level with the side in the high nibble.
func (e *Engine) fxPrayer() {
	e.applySpell(int(e.curSpell), int(e.selected().Side)<<4+e.casterLevel(int(e.curSpell)), 0, 0, 0, "is Affected by Prayer")
}

// fxNeutralizePoison = O113:42DF: an animated target drops
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
		e.RemoveAffect(t, affectPoisoned, nil)
		e.RemoveAffect(t, affectSlowPoison, nil)
		e.RemoveAffect(t, affectPoisonDamage, nil)
		e.combatName(t, "is unpoisoned", combatMsgRow, true)
		t.Okay = 1
		t.Status = statusOkay
		return
	}
	e.combatName(t, "is unaffected", combatMsgRow, true)
}

// fxPoison = O113:43E1: the spell's affect on the targets; then the
// caster's own target is tried against the poison sting (affect 64): when
// nothing there blocks it, the sting handler runs for the caster.
func (e *Engine) fxPoison() {
	e.applySpell(int(e.curSpell), 0, 0, 0, dmgMagic, "")
	sel := e.selected()
	e.rv.affVal = affectPoisonSting
	e.affectGroup(sel.Combat.Target, grpAddAffect)
	if e.rv.affVal == affectPoisonSting {
		e.callAffect(affectPoisonSting, sel, nil, 0) // the affect argument is an unset local
	}
}

// sticksToSnakesTooBig: the hit dice from which a target is too big for the snakes.
const sticksToSnakesTooBig = 6

// fxSticksToSnakes = O113:447E: a small target gets the snakes (and their
// handler runs at once); a big one smashes them flat.
func (e *Engine) fxSticksToSnakes() {
	t := e.tgt(1)
	if int8(t.MaxLevel) < sticksToSnakesTooBig {
		e.combatName(e.selected(), "turns sticks into snakes", combatMsgRow, true)
		e.applySpell(int(e.curSpell), e.casterLevel(int(e.curSpell)), 0, 0, dmgMagic, "")
		if a := e.FindAffect(t, affectSticksToSnakes); a != nil {
			e.callAffect(affectSticksToSnakes, t, a, 0)
		}
		return
	}
	e.combatName(t, "smashes them flat", combatMsgRow, true)
}

// fxCureCritical = O113:4536: heals 3d8+3.
func (e *Engine) fxCureCritical() {
	if e.nSpellTargets == 0 {
		return
	}
	t := e.tgt(1)
	if e.healPlayer(t, byte(e.Dice(3, 8)+3), false) {
		e.healedMessage(t)
	}
}

// fxCauseCritical = O113:4579: 3d8+3 of damage type 8.
func (e *Engine) fxCauseCritical() {
	e.applySpell(int(e.curSpell), 0, 0, e.RollDice(3, 8)+3, dmgMagic, "")
}

// fxDispelEvil = O113:45B5: the caster is warded against enchanted
// creatures (Dispel Evil's own amount), then the spell's affect on the
// targets.
func (e *Engine) fxDispelEvil() {
	e.rv.areaSpell = 0
	e.rv.dmgType = dmgType64
	e.isUnaffected(e.selected(), affectDispellingEvil, e.spellAmount(spellDispelEvil), 0, 0, 0, false, "")
	e.applySpell(int(e.curSpell), 0, 0, 0, 0, "is Affected")
}

// fxFlameStrike = O113:462C: 6d8 of damage type 9.
func (e *Engine) fxFlameStrike() {
	e.applySpell(int(e.curSpell), 0, 0, e.RollDice(6, 8), dmgMagic|dmgFire, "")
}

// fxRaiseDead = O113:4669: a dead (or animated) target with constitution
// left and not an elf comes back with a hit point, a point of constitution
// less.
func (e *Engine) fxRaiseDead() {
	t := e.tgt(1)
	if (t.Status == statusDead || t.Status == statusAnimated) && t.Stats[statCon].Base > 0 && t.Race != raceElf {
		e.RemoveAffect(t, affectPoisoned, nil)
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

// The cone's open range: O113:1910 hands lineOfSight an unset local as the
// range; the port leaves the cone unlimited by range (walls still stop it).
const coneRange = 0x7FFF

// clampToMap = O113:1910 (nested) (var x, var y): onto the combat map.
func clampToMap(x, y *int) {
	*x = min(max(*x, 0), cmapWidth-1)
	*y = min(max(*y, 0), cmapHeight-1)
}

// coneTargets = O113:1910 (x0, y0, x1, y1, width, length): the combatants
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

// fxSlayLiving = O113:4722: the target's affects may stop the spell (they
// clear the damage), as does a lifeless one; else it dies unless it saves,
// then taking 2d8+1.
func (e *Engine) fxSlayLiving() {
	t := e.tgt(1)
	e.rv.dmgType = dmgType64
	e.rv.damage = slayMark
	e.affectGroup(t, grpAddAffect)
	if t.MonsterFlags&monsterNoBreath != 0 {
		e.rv.damage = 0
	}
	if e.rv.damage != 0 {
		if !e.SavingThrow(t, saveVsSpell, 0) {
			e.killWithMessage(t, statusDead, "is slain")
		} else {
			e.rv.dmgType = dmgMagic
			e.damagePerson(t, e.RollDice(2, 8)+1, 0, false)
		}
		return
	}
	e.combatName(t, "is Unaffected", combatMsgRow, true)
}

// slayMark: Slay Living's damage value before the target's affects see it.
const slayMark = 255

// fxNothing = O113:47FB and O113:1F0D: spells without an effect of their own.
func (e *Engine) fxNothing() {}

// fxFaerieFire = O113:4805.
func (e *Engine) fxFaerieFire() { e.holdTargets(0, "is Magically Highlighted") }

// fxInvisibleToAnimals = O113:4831.
func (e *Engine) fxInvisibleToAnimals() {
	e.applySpell(int(e.curSpell), 0, 0, 0, 0, "is Invisible (spell)")
}

// fxCharmMonsters = O113:486A: held like Hold Monsters; the
// charm handler runs at once for each one charmed.
func (e *Engine) fxCharmMonsters() {
	e.holdTargets(0, "is Charmed")
	for i := 1; i <= e.nSpellTargets; i++ {
		if t := e.tgt(i); t != nil {
			if a := e.FindAffect(t, affectCharmed); a != nil {
				e.callAffect(affectCharmed, t, a, 0)
			}
		}
	}
}

// fxConfusion = O113:48F9: up to 2d8 targets are confused
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
		e.isUnaffected(t, affectConfused, e.spellAmount(spellConfusion), 0, 0, 1, saved, "is Confused")
	}
}

// fxDimensionDoor = O113:49CA: the caster steps off the map and onto the
// target square.
func (e *Engine) fxDimensionDoor() {
	sel := e.selected()
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

// fxFear = O113:4A63: everyone in the cone that does not save
// runs, out of the player's hands.
func (e *Engine) fxFear() {
	sel := e.selected()
	e.coneTargets(e.combatX(sel), e.combatY(sel), e.tgtX, e.tgtY, fearConeWidth, fearConeLength)
	for i := 1; i <= e.nSpellTargets; i++ {
		t := e.tgt(i)
		if e.SavingThrow(t, saveVsSpell, 0) {
			e.combatName(t, "is Unaffected", combatMsgRow, true)
			continue
		}
		e.isUnaffected(t, affectFrightened, e.spellAmount(spellFear), 0, 1, 1, false, "flees in terror")
		t.Combat.Turned = 1
		t.Auto = 1
		if t.Control <= controlHired {
			t.Control = controlCharmed
		}
		t.Combat.Target = nil
	}
}

// fxFireShield = O113:4BC9: the caster chooses hot or cold flames, with the shield that burns attackers;
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
			key, _ = e.menuChoice("flame type: ", "Hot Cold", colourLightMagenta, colourGreen, colourWhite, false, false)
		}
		switch key {
		case 'H':
			e.isUnaffected(sel, affectFireShieldHot, rounds(), 0, 0, 0, false, "is protected")
			e.isUnaffected(sel, affectZapsAttackers, rounds(), 0, 0, 0, false, "")
			done = true
		case 'C':
			e.isUnaffected(sel, affectFireShieldCold, rounds(), 0, 0, 0, false, "")
			e.isUnaffected(sel, affectZapsAttackers, rounds(), 0, 0, 0, false, "")
			done = true
		default:
			key, _ = e.menuChoice("Abort spell? ", "Yes No", colourLightMagenta, colourGreen, colourWhite, false, false)
			done = key == 'Y'
		}
	}
}

// fxFumble = O113:4D7C: an unsaved target is befuddled, a saved one slowed; the handler runs at once. Then the spell's own affect.
func (e *Engine) fxFumble() {
	t := e.tgt(1)
	e.rv.dmgType = dmgType64
	if !e.SavingThrow(t, saveVsSpell, 0) {
		e.isUnaffected(t, affectBefuddled, e.spellAmount(spellFumble), 0, 0, 0, false, "is Befuddled")
		if e.FindAffect(t, affectBefuddled) != nil {
			e.callAffect(affectBefuddled, t, nil, 0)
		}
	} else {
		e.isUnaffected(t, affectSlow, e.spellAmount(spellFumble), 0, 0, 0, false, "is Slowed")
		if e.FindAffect(t, affectSlow) != nil {
			e.callAffect(affectSlow, t, nil, 0)
		}
	}
	e.applySpell(int(e.curSpell), 0, 1, 0, 0, "is clumsy")
}

// fxIceStorm = O113:4EB0: 3d10 of cold and magic.
func (e *Engine) fxIceStorm() {
	e.applySpell(int(e.curSpell), 0, 0, e.RollDice(3, 10), dmgCold|dmgMagic, "")
}

// fxMinorGlobe = O113:4EE6.
func (e *Engine) fxMinorGlobe() { e.applySpell(int(e.curSpell), 0, 0, 0, 0, "is Protected") }

// fxCloudkill = O113:4F39.
func (e *Engine) fxCloudkill() { e.createCloud(e.poisonKind()) }

// The cone of Cone of Cold: 2 wide, half the caster level long.
const coneOfColdWidth = 2

// fxConeOfCold = O113:5499: level + level d4 of cold and magic.
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

// fxFeeblemind = O113:5549: the target's spell save is made harder by its
// class (clerics -1; rangers +1 at level 8, +2 from 9; paladins +4 from
// level 9; mages +4 for a human, else +2; the other spell casting
// combinations +2) and put back after; the feeblemind handler runs at
// once. A cleric's case leaves the "applies" flag unset in the asm; the
// port takes it as applying.
func (e *Engine) fxFeeblemind() {
	t := e.tgt(1)
	keep := t.Saves[saveVsSpell]
	applies := true
	switch t.Class {
	case clsCleric, classClericFighter, classClericThief:
		t.Saves[saveVsSpell]--
	case clsRanger:
		if t.MaxLevel == 8 {
			t.Saves[saveVsSpell]++
		}
		if int8(t.MaxLevel) >= 9 {
			t.Saves[saveVsSpell] += 2
		}
	case clsPaladin:
		if int8(t.MaxLevel) >= 9 {
			t.Saves[saveVsSpell] += 4
		}
	case clsMage, classFighterMage, classFighterMageThief, classMageThief:
		if t.Race == raceHuman {
			t.Saves[saveVsSpell] += 4
		} else {
			t.Saves[saveVsSpell] += 2
		}
	case classClericFighterMage, classClericRanger, classClericMage:
		t.Saves[saveVsSpell] += 2
	default:
		applies = false
	}
	if applies {
		e.applySpell(int(e.curSpell), 0, 0, 0, 0, "")
	}
	if e.FindAffect(t, affectFeebleminded) != nil {
		e.callAffect(affectFeebleminded, t, nil, 0)
	} else {
		e.combatName(t, "is unaffected", combatMsgRow, true)
	}
	t.Saves[saveVsSpell] = keep
}

// fxPlain = O113:57E8 and O113:58AD: the spell record alone.
func (e *Engine) fxPlain() { e.applySpell(int(e.curSpell), 0, 0, 0, 0, "") }

// fxCharmMammal = O113:56FF: Charm Person or Mammal works on the
// demihuman races and on charmable creatures.
func (e *Engine) fxCharmMammal() {
	t := e.tgt(1)
	if t == nil {
		return
	}
	if int8(t.Race) < raceHuman || t.CreatureFlags&creatureCharmable != 0 {
		lvl := e.casterLevel(int(e.curSpell))
		e.applySpell(int(e.curSpell), (int(e.selected().Side)<<7+lvl)&0xFF, 1, 0, 0, "is Charmed")
		if a := e.FindAffect(t, affectCharmed); a != nil {
			e.callAffect(affectCharmed, t, a, 0)
		}
		return
	}
	e.combatName(t, "is not Charmed", combatMsgRow, true)
}

// deathSpellCost = O113:58E5's price of a victim, out of the 4d20*2 budget,
// by its hit dice.
func deathSpellCost(c *Character) byte {
	switch lvl := int8(c.MaxLevel); {
	case lvl == 0:
		return 1
	case lvl == 1:
		return 2
	case lvl >= 2 && lvl <= 3:
		return 8
	case lvl >= 4 && lvl <= 5:
		return 16
	case lvl == 6:
		if c.MaxHP <= 51 {
			return 16
		}
		return 20
	case lvl >= 7 && lvl <= 8:
		return 20
	}
	return 162 // more than any budget
}

// fxDeathSpell = O113:58E5: victims drop dead in list order while the
// budget lasts; lifeless ones are passed over.
func (e *Engine) fxDeathSpell() {
	e.rv.areaSpell = 1
	e.rv.dmgType = dmgType64
	budget := byte(e.Dice(4, 20) * 2)
	for i := 1; i <= e.nSpellTargets; i++ {
		t := e.tgt(i)
		cost := deathSpellCost(t)
		if t.MonsterFlags&monsterNoBreath == 0 && budget >= cost {
			budget -= cost
			e.killWithMessage(t, statusDead, "drops dead")
		} else {
			e.setTgt(i, nil)
		}
	}
}

// killUnlessSaved = O113:5A11 and O113:5CD8: unless the target's affects
// stop it, a failed save leaves it in the given state.
func (e *Engine) killUnlessSaved(status byte, msg string) {
	e.rv.dmgType = dmgType64
	t := e.tgt(1)
	e.rv.damage = slayMark
	e.affectGroup(t, grpAddAffect)
	if e.rv.damage != 0 && !e.SavingThrow(t, saveVsSpell, 0) {
		e.killWithMessage(t, status, msg)
	}
}

// fxDisintegrate = O113:5A11.
func (e *Engine) fxDisintegrate() { e.killUnlessSaved(statusDead, "disintegrates") }

// fxFleshToStone = O113:5CD8.
func (e *Engine) fxFleshToStone() { e.killUnlessSaved(statusStoned, "is turned to stone") }

// fxStoneToFlesh = O113:5AD5: a statue comes back with a hit point; in
// combat its body leaves the map and it stands again on its square. Out
// of combat, casting it on someone not stoned may be confirmed anyway.
func (e *Engine) fxStoneToFlesh() {
	t := e.tgt(1)
	if t.Status == statusStoned {
		t.Status = statusOkay
		t.Okay = 1
		t.HP = 1
		e.combatName(t, "is no longer a statue", combatMsgRow, false)
		if e.gameState == stateCombat {
			k := 1
			for e.bodies[k].c != t {
				k++
			}
			b := e.bodies[k]
			e.cmap.ground[int(b.y)][int(b.x)] = b.ground
			copy(e.bodies[k:e.nbodies], e.bodies[k+1:e.nbodies+1])
			e.nbodies--
			if e.combatSquareFree(t, e.combatX(t), e.combatY(t), true) {
				e.combatCentre(e.tgtX, e.tgtY, 0, 8)
			}
		}
	} else if e.gameState == stateCamp {
		if e.castCureAnyway(t, "is not stoned.") == 'y' {
			e.combatName(t, "is Unaffected", combatMsgRow, false)
		}
	}
	e.applySpell(int(e.curSpell), 0, 0, 0, 0, "")
}

// The longest wait of a delayed blast fireball, in segments (ten to a round).
const (
	delayedBlastMaxWait   = 50
	segmentsPerRound      = 10
	delayedBlastFireLevel = dmgFire | dmgMagic
)

// fxDelayedBlast = O113:5D90: the caster sets the delay; none goes off at
// once like a fireball. Otherwise the fireball waits on the target square
// as the caster's next numbered one, with the delay as its affect time.
func (e *Engine) fxDelayedBlast() {
	e.rv.areaSpell = 1
	lvl := e.casterLevel(spellDelayedBlastFireball)
	delay := (int(e.askAmount(delayedBlastMaxWait, "Blast Delay in segments (1-50)? ", colourGreen)) + 5) / segmentsPerRound
	if delay == 0 {
		e.applySpell(spellDelayedBlastFireball, 0, 0, lvl+e.RollDice(lvl, 6), delayedBlastFireLevel, "")
		return
	}
	sel := e.selected()
	var number byte
	for _, r := range e.delayedBlasts {
		if r.caster == sel {
			number++
		}
	}
	if n := len(e.delayedBlasts); n > 0 && e.delayedBlasts[n-1].caster == sel {
		number-- // the asm counts all but the last record
	}
	r := &cloudRecord{caster: sel, number: number, x: byte(e.tgtX), y: byte(e.tgtY)}
	e.delayedBlasts = append(e.delayedBlasts, r)
	e.AddAffect(sel, affectDelayedBlast, delay, byte(lvl)+number<<4, 1)
	_, g := e.mapAt(e.tgtX, e.tgtY)
	if g > 0 && groundInfo[g%64][0] < groundStopper {
		r.covered[0] = 1
	}
	for _, o := range e.delayedBlasts {
		if o == r {
			continue
		}
		if o.covered[0] != 0 && int(o.x) == e.tgtX && int(o.y) == e.tgtY {
			if og := o.ground[0]; og != groundStinkingCloud && og != groundPoisonCloud && og != groundDelayedBlast {
				g = og
				break
			}
		}
	}
	r.ground[0] = g
	if r.covered[0] != 0 {
		e.cmap.ground[e.tgtY][e.tgtX] = groundDelayedBlast
	}
	e.combatName(sel, "Sets a delayed blast fireball.", combatMsgRow, false)
	e.combatCentre(e.tgtX, e.tgtY, 255, 8)
}

// fxMassInvisibility = O113:6038: the caster's side, but those with an
// enemy next to them in combat.
func (e *Engine) fxMassInvisibility() {
	e.rv.areaSpell = 1
	side := e.selected().Side
	for i := 1; i <= e.nSpellTargets; i++ {
		t := e.tgt(i)
		if t.Side != side || e.gameState == stateCombat && e.nearEnemies(t, 1) > 0 {
			e.setTgt(i, nil)
		}
	}
	e.applySpell(int(e.curSpell), 0, 0, 0, 0, "is Invisible (spell)")
}

// fxPowerWordStun = O113:60ED: the stun lasts by the target's hit points:
// up to 30 4d4 rounds, 31..60 2d4, 61..90 1d4, more none.
func (e *Engine) fxPowerWordStun() {
	t := e.tgt(1)
	var rounds int
	switch {
	case t.HP >= 1 && t.HP <= 30:
		rounds = e.Dice(4, 4)
	case t.HP >= 31 && t.HP <= 60:
		rounds = e.Dice(2, 4)
	case t.HP >= 61 && t.HP <= 90:
		rounds = e.Dice(1, 4)
	}
	spell := byte(spellPowerWordStun)
	e.isUnaffected(t, e.spellRec(spell, spellFieldAffect), rounds, byte(e.casterLevel(spellPowerWordStun)), 0,
		e.spellRec(spell, spellFieldSaveMode), false, "is STUNNED")
}
