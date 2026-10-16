package engine

// The affect handlers of O064 (and the five O064's setup_spells2 takes from
// O08D), with the table they fill. Each handler is (char, affect, flag);
// flag 1 means the affect is being removed. The rule globals DS:6FAA.. are
// e.rv and the scratch fields of Engine (see rules.go).

// AffectHandler is one entry of the DS:6FC0 table.
type AffectHandler func(e *Engine, c *Character, a *Affect, flag byte)

var affectHandlers [affectHandlerCount]AffectHandler

// itemHandler = DS:720C, O064:26D3: a readied (or unreadied) item that
// grants the affect its Spells[1] names.
func (e *Engine) itemHandler(c *Character, it *Item, flag byte) {
	e.itemCall = false
	typ := it.Spells[1]
	if flag != 0 {
		e.RemoveAffect(c, typ, nil)
		return
	}
	e.AddAffect(c, typ, 0, 255, 1)
	if e.gameState != stateCombat {
		e.callAffect(typ, c, nil, 0)
	}
}

// setupAffectHandlers = O064:2AD8 setup_spells2.
func setupAffectHandlers() {
	h := &affectHandlers
	h[affectBless] = (*Engine).blessHandler
	h[affectCurse] = (*Engine).curseHandler
	h[affectSticksToSnakes] = (*Engine).snakesHandler
	h[affectWardEnchanted] = (*Engine).wardEnchantedHandler
	h[5] = (*Engine).emptyHandler
	h[affectKindBane] = (*Engine).kindBaneHandler
	h[affectArmourBonus] = (*Engine).armourBonusHandler
	h[affectProtFromEvil] = (*Engine).protFromEvilHandler
	h[affectProtFromGood] = (*Engine).protFromGoodHandler
	h[affectResistCold] = (*Engine).resistColdHandler
	h[affectCharmed] = (*Engine).charmHandler
	h[affectEnlarge] = (*Engine).emptyHandler
	h[affectSuffocating] = (*Engine).suffocateHandler
	h[affectCharisma] = (*Engine).emptyHandler
	h[affectPoisonDamage] = (*Engine).poisonDamageHandler
	h[16] = (*Engine).emptyHandler
	h[affectShield] = (*Engine).shieldHandler
	h[affectGnomeVsManSized] = (*Engine).vsGoblinoidHandler
	h[19] = (*Engine).emptyHandler
	h[affectResistFire] = (*Engine).resistFireHandler
	h[affectSilenced] = (*Engine).silencedHandler
	h[affectSlowPoison] = (*Engine).slowPoisonHandler
	h[affectSpiritualHammer] = (*Engine).spiritualHammerHandler
	h[affectDetectInvisibility] = (*Engine).emptyHandler
	h[affectInvisible] = (*Engine).invisibleHandler
	h[affectDwarfVsOrc] = (*Engine).vsOrcHandler
	h[affectStunned] = (*Engine).loseTurnHandler
	h[affectMirrorImage] = (*Engine).mirrorImageHandler
	h[affectQuarterLessDamage] = (*Engine).quarterLessHandler
	h[affectCoughing] = (*Engine).coughingHandler
	h[affectNausea] = (*Engine).loseTurnHandler
	h[affectCollapse] = (*Engine).collapseHandler
	h[affectDazzled] = (*Engine).dazzledHandler
	h[affectWeakAndWasting] = (*Engine).weakAndWastingHandler
	h[affectConfused] = (*Engine).confusedHandler
	h[affectHampered] = (*Engine).hamperedHandler
	h[affectStumbling] = (*Engine).stumblingHandler
	h[affectStrengthSpell] = (*Engine).emptyHandler
	h[affectHaste] = (*Engine).hasteHandler
	h[affectStinkingCloud] = (*Engine).stinkingCloudEndHandler
	h[affectAvoidPlainWeapons] = (*Engine).avoidPlainWeaponsHandler
	h[affectSlow] = (*Engine).slowHandler
	h[affectWeakened] = (*Engine).weakenHandler
	h[affectWasting] = (*Engine).wastingHandler
	h[affectProtFromEvil10] = (*Engine).protFromEvilHandler
	h[affectProtFromGood10] = (*Engine).protFromGoodHandler
	h[affectVsGiants] = (*Engine).vsGiantsHandler
	h[affectGnomeExtra] = (*Engine).gnomeExtraHandler
	h[affectPrayer] = (*Engine).prayerHandler
	h[affectFireBody] = (*Engine).fireBodyHandler
	h[affectSleep] = (*Engine).loseTurnHandler
	h[affectParalyzed] = (*Engine).loseTurnHandler
	h[affectHeld] = (*Engine).loseTurnHandler
	h[affectIceBody] = (*Engine).iceBodyHandler
	h[affectPoisoned] = (*Engine).emptyHandler
	h[affectFlicker] = (*Engine).flickerHandler
	h[affectEngulfs] = (*Engine).engulfsHandler
	h[affectEngulfed] = (*Engine).engulfedHandler
	h[affectRegenerating] = (*Engine).regeneratingHandler
	h[affectMagicWeaponsOnly] = (*Engine).magicWeaponsOnlyHandler
	h[affectFireDamageReduced] = (*Engine).fireDamageReducedHandler
	h[affectHealingOverTime] = (*Engine).healingOverTimeHandler
	h[affectMinorGlobe] = (*Engine).minorGlobeHandler
	h[affectPoisonSting] = (*Engine).poisonStingHandler
	h[affectPoisonSting4] = (*Engine).poisonSting4Handler
	h[affectPoisonSting2] = (*Engine).poisonSting2Handler
	h[affectParalysingTouch] = (*Engine).paralysingTouchHandler
	h[affectFeebleminded] = (*Engine).feeblemindHandler
	h[affectHiddenFromKind19] = (*Engine).hiddenFromKind19Handler
	h[affectPoisonStingMinus2] = (*Engine).poisonStingMinus2Handler
	h[affectBlur] = (*Engine).blurHandler
	h[affectFlicker95] = (*Engine).flicker95Handler
	h[affectUnaffectedByType32] = (*Engine).unaffectedByType32Handler
	h[74] = (*Engine).emptyHandler
	h[affectVsKind3] = (*Engine).vsKind3Handler
	h[affectVsKind8] = (*Engine).vsKind8Handler
	h[affectBerserk] = (*Engine).berserkHandler
	h[affectRecovering] = (*Engine).recoveringHandler
	h[affectBurningTouch] = (*Engine).burningTouchHandler
	h[affectAcidTouch] = (*Engine).acidTouchHandler
	h[affectHalfDamage] = (*Engine).halfDamageHandler
	h[affectSaveVsElements] = (*Engine).saveVsElementsHandler
	h[affectElectricityHeals] = (*Engine).electricityHealsHandler
	h[affectArrowsScratch] = (*Engine).arrowsScratchHandler
	h[affectRays] = (*Engine).raysHandler
	h[affectFirstStrike] = (*Engine).firstStrikeHandler
	h[affectCloudkill] = (*Engine).cloudkillEndHandler
	h[92] = (*Engine).emptyHandler
	h[affectFireHalved] = (*Engine).fireHalvedHandler
	h[affectRangedHalved] = (*Engine).rangedHalvedHandler
	h[affectFallsDead] = (*Engine).fallsDeadHandler
	h[affectHugs] = (*Engine).hugsHandler
	h[affectConSavingBonus] = (*Engine).conSaveHandler
	h[affectRegrowth] = (*Engine).regrowthHandler
	h[affectBacksUp] = (*Engine).backsUpHandler
	h[affectRegrows] = (*Engine).regrowsHandler
	h[affectRegenerates] = (*Engine).regeneratesHandler
	h[affectRegrowing] = (*Engine).regrowingHandler
	h[affectBurns] = (*Engine).burnsHandler
	h[affectAvoidRanged60] = (*Engine).avoidRanged60Handler
	h[affectMagicResist50] = (*Engine).magicResist50Handler
	h[affectMagicResist15] = (*Engine).magicResist15Handler
	h[affectElfResistSleep] = (*Engine).elfResistHandler
	h[affectNoCharmHold] = (*Engine).noCharmHoldHandler
	h[affectNoParalysis] = (*Engine).noParalysisHandler
	h[affectColdImmune] = (*Engine).coldImmuneHandler
	h[affectNoPoisonParalysis] = (*Engine).noPoisonParalysisHandler
	h[affectFireImmune] = (*Engine).fireImmuneHandler
	h[affectFireReduced] = (*Engine).fireReducedHandler
	h[affectElectricityHalved] = (*Engine).electricityHalvedHandler
	h[affectEdgedHalved] = (*Engine).edgedHalvedHandler
	h[affectMagicWeaponsHalved] = (*Engine).magicWeaponsHalvedHandler
	h[affectSlingStones] = (*Engine).slingStonesHandler
	h[affectColdHalved] = (*Engine).coldHalvedHandler
	h[affectNeedsMagicWeapon] = (*Engine).needsMagicWeaponHandler
	h[affectAvoidsBows] = (*Engine).avoidsBowsHandler
	h[affectSpitsAcid] = (*Engine).spitsAcidHandler
	h[affectParalysingGaze] = (*Engine).paralysingGazeHandler
	h[affectFireTouch] = (*Engine).fireTouchHandler
	h[affectHalfElfResistance] = (*Engine).halfElfResistHandler
	h[affectNoCharmHoldSleep] = (*Engine).noCharmHoldSleepHandler
	h[127] = (*Engine).emptyHandler
	h[affectMagicImmune] = (*Engine).magicImmuneHandler
	h[affectSlainByArrow] = (*Engine).slainByArrowHandler
	h[affectNoMindAffects] = (*Engine).noMindAffectsHandler
	h[affectRangerVsGiants] = (*Engine).rangerVsGiantsHandler
	h[affectElectricityImmune] = (*Engine).electricityImmuneHandler
	h[affectRooted] = (*Engine).rootedHandler
	h[affectBerserkFoe] = (*Engine).berserkFoeHandler
	h[affectAlwaysInvisible] = (*Engine).alwaysInvisibleHandler
	h[affectEngulfing] = (*Engine).engulfingHandler
	h[140] = (*Engine).emptyHandler
	h[affectPaladinCures] = (*Engine).paladinCuresHandler
	h[affectRunningAway] = (*Engine).runningAwayHandler
	h[affectZapsAttackers] = (*Engine).zapsHandler
	h[affectHugging] = (*Engine).huggingHandler
	h[affectDispelEvil] = (*Engine).dispelEvilHandler
	h[affectGiantStrength] = (*Engine).emptyHandler
}

func init() { setupAffectHandlers() }

// --- shared pieces --------------------------------------------------------

// blockAffect = O064:0019 (type): the affect being added is refused when
// type is 0 or the one being added; the damage goes with it.
func (e *Engine) blockAffect(typ byte) {
	if typ == 0 || e.rv.affVal == typ {
		e.rv.damage = 0
		e.rv.affVal = 0
	}
}

// addAffectNow = O064:0038 (char, type, data, time): adds the affect with
// its removal call; refused while an affect is being cured.
func (e *Engine) addAffectNow(c *Character, typ, data byte, time int) bool {
	if e.rv.cure != 0 {
		return false
	}
	e.AddAffect(c, typ, time, data, 1)
	return true
}

// endTurn = O0ED:27E4 (char): no initiative, spell, guard or moves left;
// always true (the caller's "turn over").
func (e *Engine) endTurn(c *Character) bool {
	ci := combatInfo(c)
	ci.Delay, ci.Spell, ci.Guarding, ci.Moves = 0, 0, 0, 0
	return true
}

// combatInfo is c.Combat; outside a fight the original reads whatever its
// pointer holds, we read a blank block.
func combatInfo(c *Character) *CombatInfo {
	if c.Combat == nil {
		return &CombatInfo{}
	}
	return c.Combat
}

// targetOf is the combatant c is fighting (its combat block's target).
func targetOf(c *Character) *Character {
	if t := combatInfo(c).Target; t != nil {
		return t
	}
	return &Character{}
}

// readiedWeapon = O064:0F77 (char): the ammunition the readied weapon
// shoots, else the weapon itself.
func (e *Engine) readiedWeapon(c *Character) *Item {
	var it *Item
	if c.Slots[0] != nil {
		if !e.ammoFor(c, &it) || it == nil {
			it = c.Slots[0]
		}
	}
	return it
}

// avoidDistantAttack = O064:0F06 (char, percent): an attack from afar by
// the selected character (who holds a weapon) may miss.
func (e *Engine) avoidDistantAttack(c *Character, pct int) {
	sel := e.selected()
	if sel.Slots[0] != nil && e.rangeTo(sel, c) > 1 && e.Dice(1, 100) <= pct {
		e.combatName(c, "Avoids it", combatMsgRow, true)
		e.rv.damage = 0
		e.rv.hitRoll = -1
		e.hits6822[1]--
	}
}

// poisonAttack = O064:1407 (char, bonus): the one the character fights is
// poisoned to death unless it saves.
func (e *Engine) poisonAttack(c *Character, bonus int) {
	t := targetOf(c)
	e.v6826 = t
	if !e.SavingThrow(t, saveVsParalysis, bonus) {
		e.combatName(t, "is Poisoned", combatMsgRow, false)
		e.gameDelay()
		e.AddAffect(t, affectPoisoned, 0, 255, 0)
		e.killWithMessage(t, statusDead, "is killed")
	}
}

// paralyseTarget = O064:14AE (char, time): the one the character fights is
// paralysed for the time unless it saves.
func (e *Engine) paralyseTarget(c *Character, time int) {
	t := targetOf(c)
	e.v6826 = t
	if !e.SavingThrow(t, saveVsParalysis, 0) {
		e.hitMessage(t, true, "is Paralyzed")
		e.AddAffect(t, affectParalyzed, time, 12, 0)
	}
}

// resistRoll = O064:20EE (base): magic resistance, base% plus 5% for each
// caster level under 11, against the affect or magic damage.
func (e *Engine) resistRoll(base int) {
	lvl := e.casterLevel(int(e.curSpell))
	chance := byte(base + (11-lvl)*5)
	if e.rv.affVal == 0 && e.rv.dmgType&dmgMagic == 0 {
		return
	}
	if byte(e.Dice(1, 100)) <= chance {
		e.blockAffect(0)
	}
}

// alignedIn: is the selected character's alignment one of these?
func (e *Engine) selectedAligned(a, b, c byte) bool {
	al := e.selected().Alignment
	return al == a || al == b || al == c
}

// --- the handlers ----------------------------------------------------------

func (e *Engine) emptyHandler(c *Character, a *Affect, flag byte) {}

// loseTurnHandler = O064:0071: the turn is over (stunned, sleeping, held).
func (e *Engine) loseTurnHandler(c *Character, a *Affect, flag byte) { e.endTurn(c) }

// saveAndHitUp = O064:0087.
func (e *Engine) saveAndHitUp() {
	e.rv.save++
	e.rv.hitRoll++
}

// blessHandler = O064:0096.
func (e *Engine) blessHandler(c *Character, a *Affect, flag byte) {
	e.v6825 += 5
	e.rv.hitRoll++
}

// curseHandler = O064:00A6.
func (e *Engine) curseHandler(c *Character, a *Affect, flag byte) {
	if e.v6825 < 5 {
		e.v6825 = 0
	} else {
		e.v6825 -= 5
	}
	e.rv.hitRoll--
}

// snakesHandler = O064:00DC: the snakes eat the character's attacks.
func (e *Engine) snakesHandler(c *Character, a *Affect, flag byte) {
	n := c.Attacks[0] + c.Attacks[1]
	if a.Data > n {
		a.Data -= n
	} else {
		e.RemoveAffect(c, affectSticksToSnakes, nil)
	}
	e.hitMessage(c, true, "is fighting with snakes")
	e.clearCombatText()
	e.endTurn(c)
}

// wardEnchantedHandler = O064:015F.
func (e *Engine) wardEnchantedHandler(c *Character, a *Affect, flag byte) {
	if e.selected().Pad331&creatureEnchanted != 0 {
		e.rv.hitRoll -= 7
	}
}

// Monster kinds the bane and bonus affects look at.
const (
	kindBane1  = 10
	kindBane2a = 9
	kindBane2b = 12
	kindBane3  = 4
	kindKind3  = 3
	kindKind8  = 8
	kindKind19 = 19
)

// kindBaneHandler = O064:017A: to hit and damage by the target's kind.
func (e *Engine) kindBaneHandler(c *Character, a *Affect, flag byte) {
	e.v6826 = targetOf(c)
	var k byte
	switch e.v6826.Kind {
	case kindBane1:
		k = 1
	case kindBane2a, kindBane2b:
		k = 2
	case kindBane3:
		k = 3
	}
	e.rv.hitRoll += int8(k)
	e.rv.damage += k
	e.rv.dmgType = dmgFire | dmgMagic
}

// Armour values (60 - armour class).
const (
	acCeiling     = 60
	acShield      = 57
	acCoughFloor  = 50
	acArmourBonus = 2
)

// armourBonusHandler = O064:01DF: two better, up to armour class 0.
func (e *Engine) armourBonusHandler(c *Character, a *Affect, flag byte) {
	for i := range c.AC {
		if c.AC[i] < acCeiling-acArmourBonus {
			c.AC[i] += acArmourBonus
		} else {
			c.AC[i] = acCeiling
		}
	}
}

// Alignments.
const (
	alignLawfulGood = iota
	alignLawfulNeutral
	alignLawfulEvil
	alignNeutralGood
	alignTrueNeutral
	alignNeutralEvil
	alignChaoticGood
	alignChaoticNeutral
	alignChaoticEvil
)

// protFromEvilHandler = O064:0224: evil attackers.
func (e *Engine) protFromEvilHandler(c *Character, a *Affect, flag byte) {
	if e.selectedAligned(alignLawfulEvil, alignNeutralEvil, alignChaoticEvil) {
		e.rv.save += 2
		e.rv.hitRoll -= 2
	}
}

// protFromGoodHandler = O064:0259: good attackers.
func (e *Engine) protFromGoodHandler(c *Character, a *Affect, flag byte) {
	if e.selectedAligned(alignLawfulGood, alignNeutralGood, alignChaoticGood) {
		e.rv.save += 2
		e.rv.hitRoll -= 2
	}
}

// resistColdHandler = O064:028E.
func (e *Engine) resistColdHandler(c *Character, a *Affect, flag byte) {
	if e.rv.dmgType&dmgCold != 0 {
		e.rv.damage >>= 1
		e.rv.save += 3
	}
}

// Charm data bits: the charmed one's own side, the charm taken hold, the
// side it now fights for.
const (
	charmOwnSide   = 16
	charmTaken     = 32
	charmSideShift = 6
	charmSideBit   = 64
	charmNewShift  = 7
	charmMorale    = 100
)

// charmHandler = O064:02AD: the character fights for the charmer; on
// removal back to its own side.
func (e *Engine) charmHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		c.Side = (a.Data & charmSideBit) >> charmSideShift
		if c.Control == controlCharmed {
			c.Control = 0
		}
		return
	}
	if a.Data&charmTaken == 0 {
		a.Data = a.Data + charmTaken + c.Side*charmSideBit
		c.Side = a.Data >> charmNewShift
		c.Auto = 1
		if c.Control <= controlHired {
			c.Control = controlCharmed
		}
		combatInfo(c).Target = nil
		e.countSides()
	}
	e.v6825 = charmMorale
}

// suffocateHandler = O064:0379.
func (e *Engine) suffocateHandler(c *Character, a *Affect, flag byte) {
	if a.Data == 0 {
		e.killWithMessage(c, statusDead, "Suffocates")
		return
	}
	a.Data--
}

// Poison and wasting timers.
const (
	poisonDamageTime = 10
	weakenTime       = 60
	healingTime      = 60
)

// poisonDamageHandler = O064:03BC: a point of damage, again later.
func (e *Engine) poisonDamageHandler(c *Character, a *Affect, flag byte) {
	if e.addAffectNow(c, affectPoisonDamage, a.Data, poisonDamageTime) && c.HP > 1 {
		e.rv.dmgType = 0
		e.damagePerson(c, 1, saveNone, false)
		if e.gameState != stateCombat {
			e.playerSummary(e.selected())
		}
	}
}

// shieldHandler = O064:041F: armour class 3 at least, saves up; magic
// missiles (spell 15) do nothing.
func (e *Engine) shieldHandler(c *Character, a *Affect, flag byte) {
	if c.AC[0] < acShield {
		c.AC[0] = acShield
	}
	e.rv.save++
	if e.curSpell == spellMagicMissile {
		e.rv.damage = 0
	}
}

// vsGoblinoidHandler = O064:044A.
func (e *Engine) vsGoblinoidHandler(c *Character, a *Affect, flag byte) {
	e.v6826 = targetOf(c)
	if e.v6826.Pad331&creatureGoblinoid != 0 {
		e.rv.hitRoll++
	}
}

// resistFireHandler = O064:0480.
func (e *Engine) resistFireHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 && e.rv.dmgType&dmgFire != 0 {
		e.rv.damage >>= 1
		e.rv.save += 3
	}
}

// silencedHandler = O064:04B1.
func (e *Engine) silencedHandler(c *Character, a *Affect, flag byte) {
	ci := combatInfo(c)
	if ci.CanUse != 0 {
		e.combatName(c, "is silenced", combatMsgRow, true)
	}
	ci.CanUse = 0
	ci.CanCast = 0
}

// slowPoisonHandler = O064:0517: when it ends, the poison kills.
func (e *Engine) slowPoisonHandler(c *Character, a *Affect, flag byte) {
	if e.FindAffect(c, affectPoisoned) != nil {
		e.killWithMessage(c, statusDead, "dies from poison")
	}
	e.rv.cure = 1
	e.RemoveAffect(c, affectPoisonDamage, nil)
	e.rv.cure = 0
}

// The spiritual hammer item.
const (
	hammerType   = 20
	hammerWord1  = 20
	hammerWord2  = 243
	hammerPlus   = 1
	hammerFlags  = 160
	maxItemsHeld = 16
	hammerAffect = affectSpiritualHammer
)

// spiritualHammerHandler = O064:0583: the hammer appears in the caster's
// hand and goes with the spell.
func (e *Engine) spiritualHammerHandler(c *Character, a *Affect, flag byte) {
	var found *Item
	for _, it := range c.Items {
		if it.Type == hammerType && it.Words[2] == hammerWord2 {
			found = it
			break
		}
	}
	if flag != 0 && found != nil {
		e.loseItem(c, found)
	}
	if flag == 0 && found == nil && c.ItemCount < maxItemsHeld {
		tmp := &Item{Type: hammerType, Plus: hammerPlus}
		tmp.Words[1] = hammerWord1
		tmp.Words[2] = hammerWord2
		tmp.Spells[1] = hammerAffect
		tmp.Spells[2] = hammerFlags
		e.AddItemCopy(c, tmp)
		e.readyItem(tmp) // the original readies its template, not the copy
		e.combatName(c, "Gains an item", combatMsgRow, true)
	}
	e.ApplyItems(c)
}

// invisibleHandler = O064:06C6: attackers who cannot see it are hampered.
func (e *Engine) invisibleHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 && e.FindAffect(e.selected(), affectDetectInvisibility) == nil {
		e.v681E = 1
		e.rv.hitRoll -= 4
	}
}

// vsOrcHandler = O064:06FB.
func (e *Engine) vsOrcHandler(c *Character, a *Affect, flag byte) {
	e.v6826 = targetOf(c)
	if e.v6826.Pad331&creatureOrc != 0 {
		e.rv.hitRoll++
	}
}

// Mirror image data: the images in the high nibble.
const imageShift = 4

// mirrorImageHandler = O064:073F: a single-target spell or blow may strike
// an image instead.
func (e *Engine) mirrorImageHandler(c *Character, a *Affect, flag byte) {
	if e.Dice(1, int(a.Data>>imageShift)+1) > 1 && e.curSpell > 0 && e.rv.areaSpell == 0 {
		e.blockAffect(0)
		e.combatName(c, "lost an image", combatMsgRow, true)
		a.Data--
		if a.Data == 0 {
			e.RemoveAffect(c, affectMirrorImage, nil)
		}
	}
}

// quarterLessHandler = O064:07C0.
func (e *Engine) quarterLessHandler(c *Character, a *Affect, flag byte) {
	e.rv.damage -= e.rv.damage >> 1 >> 1
}

// coughingHandler = O064:07E8: no spells or items, and worse armour.
func (e *Engine) coughingHandler(c *Character, a *Affect, flag byte) {
	ci := combatInfo(c)
	if ci.CanUse != 0 {
		e.combatName(c, "is coughing", combatMsgRow, true)
	}
	ci.CanUse = 0
	ci.CanCast = 0
	e.ApplyItems(c)
	if c.AC[1] > acCoughFloor+2 {
		c.AC[1] -= 2
	} else {
		c.AC[1] = acCoughFloor
	}
	c.AC[0] = c.AC[1]
	if c == e.selected() {
		e.combatPanel(c)
	}
}

// Animated creatures' data: the side in the high nibble.
const (
	collapseSideShift = 4
	animatedMove      = 12
)

// collapseHandler = O064:089E: the animated creature falls; brought back to
// a plain fighter of the side it was raised for.
func (e *Engine) collapseHandler(c *Character, a *Affect, flag byte) {
	a.Call = 0
	if e.rv.cure == 0 {
		e.killWithMessage(c, statusDead, "collapses")
	}
	c.Side = a.Data >> collapseSideShift
	c.Auto = 1
	c.Undead = 0
	usable := byte(0)
	if oldClassUsable(c) {
		usable = 1
	}
	c.FightLevel = c.Levels[clsFighter] + c.OldLevels[clsFighter]*usable
	c.BaseMove = animatedMove
	if c.Control == controlCharmed {
		c.Control = 0
	}
	c.Kind = 0
}

// dazzledHandler = O064:0951.
func (e *Engine) dazzledHandler(c *Character, a *Affect, flag byte) {
	e.rv.hitRoll -= 4
	c.AC[0] -= 4
	c.AC[1] -= 4
	e.rv.save -= 4
}

// weakAndWastingHandler = O064:0974.
func (e *Engine) weakAndWastingHandler(c *Character, a *Affect, flag byte) {
	e.callAffect(affectWeakened, c, a, flag)
	e.callAffect(affectWasting, c, a, flag)
}

// Run-away and berserk times.
const (
	runAwayTime = 10
	berserkTime = 1
)

// confusedHandler = O064:09D9: each round the confused run, stand dazed,
// go berserk or rage; a saving throw may end it.
func (e *Engine) confusedHandler(c *Character, a *Affect, flag byte) {
	switch r := e.Dice(1, 100); {
	case r <= 10:
		e.RemoveAffect(c, affectConfused, nil)
		ci := combatInfo(c)
		ci.Turned = 1
		c.Auto = 1
		if c.Control <= controlHired {
			c.Control = controlCharmed
		}
		ci.Target = nil
		e.isUnaffected(c, affectRunningAway, runAwayTime, 0, 1, 1, false, "runs away")
	case r <= 60:
		e.hitMessage(c, true, "is confused")
		e.clearCombatText()
		e.loseTurnHandler(c, a, 0)
	case r <= 80:
		e.isUnaffected(c, affectBerserkFoe, berserkTime, c.Side, 1, 1, false, "goes berserk")
		e.callAffect(affectBerserkFoe, c, nil, 0)
	default:
		e.hitMessage(c, true, "is enraged")
		e.clearCombatText()
	}
	if e.SavingThrow(c, saveVsSpell, -2) {
		e.RemoveAffect(c, affectConfused, nil)
	}
}

// hamperedHandler = O064:0B6F.
func (e *Engine) hamperedHandler(c *Character, a *Affect, flag byte) {
	e.rv.hitRoll -= 4
	e.rv.save -= 4
}

// stumblingHandler = O064:0B80 has_action_timedout: with no initiative
// left the attack misses.
func (e *Engine) stumblingHandler(c *Character, a *Affect, flag byte) {
	if combatInfo(c).Delay == 0 {
		e.v681E = 1
		e.rv.hitRoll = -1
	}
}

// Haste data: aged once.
const hasteAged = 16

// hasteHandler = O064:0BA5 spl_age: a year older the first time, twice as
// fast.
func (e *Engine) hasteHandler(c *Character, a *Affect, flag byte) {
	if a.Data&hasteAged == 0 {
		a.Data += hasteAged
		e.combatName(c, "ages", combatMsgRow, true)
		c.Age++
	}
	e.v681A *= 2
}

// avoidPlainWeaponsHandler = O064:0FE0: a non-magical weapon misses.
func (e *Engine) avoidPlainWeaponsHandler(c *Character, a *Affect, flag byte) {
	if w := e.readiedWeapon(e.selected()); w != nil && w.Plus == 0 {
		e.avoidDistantAttack(c, 100)
	}
}

// slowHandler = O064:101B.
func (e *Engine) slowHandler(c *Character, a *Affect, flag byte) { e.v681A >>= 1 }

// Strength floor of weakness.
const weakestStrength = 3

// weakenHandler = O064:1038 weaken: strength drops while it lasts; at the
// bottom the victim retches.
func (e *Engine) weakenHandler(c *Character, a *Affect, flag byte) {
	if !e.addAffectNow(c, affectWeakened, a.Data, weakenTime) {
		return
	}
	if c.Stats[statStr].Value > weakestStrength {
		e.combatName(c, "is weakened", combatMsgRow, true)
		c.Stats[statStr].Value--
		return
	}
	if e.FindAffect(c, affectNausea) == nil {
		e.AddAffect(c, affectNausea, 0, 255, 0)
	}
}

// wastingHandler = O064:10C2: a point of damage while hit points last,
// then nausea.
func (e *Engine) wastingHandler(c *Character, a *Affect, flag byte) {
	if !e.addAffectNow(c, affectWasting, a.Data, poisonDamageTime) {
		return
	}
	if c.HP > 1 {
		e.rv.dmgType = 0
		e.damagePerson(c, 1, saveNone, false)
		if e.gameState != stateCombat {
			e.playerSummary(e.selected())
		}
		return
	}
	if e.FindAffect(c, affectNausea) == nil {
		e.AddAffect(c, affectNausea, 0, 255, 0)
	}
}

// Creature sizes and kinds the racial affects look at.
const (
	sizeMask     = 127
	sizeLarge    = 2
	kindGiantA   = 2
	kindGiantB   = 10
	kindManSized = 1
)

// vsGiantsHandler = O064:1153: large giants hit a dwarf or gnome less.
func (e *Engine) vsGiantsHandler(c *Character, a *Affect, flag byte) {
	e.v6826 = targetOf(c)
	sel := e.selected()
	if (sel.Kind == kindGiantA || sel.Kind == kindGiantB) && sel.Size&sizeMask == sizeLarge {
		e.rv.hitRoll -= 4
	}
}

// gnomeExtraHandler = O064:11A2.
func (e *Engine) gnomeExtraHandler(c *Character, a *Affect, flag byte) {
	sel := e.selected()
	if sel.Kind == kindManSized && sel.Size&sizeMask == sizeLarge {
		e.rv.hitRoll -= 4
	}
}

// Prayer data: the praying side.
const prayerSideShift = 4

// prayerHandler = O064:11C9: the praying side gains, the other loses.
func (e *Engine) prayerHandler(c *Character, a *Affect, flag byte) {
	if c.Side == (a.Data&charmOwnSide)>>prayerSideShift {
		e.saveAndHitUp()
		return
	}
	e.rv.hitRoll--
	e.rv.save--
}

// fireBodyHandler = O064:1212.
func (e *Engine) fireBodyHandler(c *Character, a *Affect, flag byte) {
	if e.rv.dmgType&dmgCold != 0 {
		e.rv.save += 2
		return
	}
	if e.rv.dmgType&dmgFire != 0 && !e.rv.saveMade {
		e.rv.damage *= 2
	}
}

// iceBodyHandler = O064:1243. The original tests DS:5D0B (the game-saved
// flag) where its twin above tests the saving throw; the port tests the
// saving throw (deliberate deviation).
func (e *Engine) iceBodyHandler(c *Character, a *Affect, flag byte) {
	if e.rv.dmgType&dmgFire != 0 {
		e.rv.save += 2
		return
	}
	if e.rv.dmgType&dmgCold != 0 && !e.rv.saveMade {
		e.rv.damage *= 2
	}
}

// flickerHandler = O064:127B: invisible for the next round.
func (e *Engine) flickerHandler(c *Character, a *Affect, flag byte) {
	e.AddAffect(c, affectInvisible, 1, 12, 0)
}

// engulfedHandler = O064:129A: no moves.
func (e *Engine) engulfedHandler(c *Character, a *Affect, flag byte) {
	combatInfo(c).Moves = 0
	if e.v681D != 0 {
		e.v681A = 0
	}
}

// regeneratingHandler = O064:12BA.
func (e *Engine) regeneratingHandler(c *Character, a *Affect, flag byte) {
	e.AddAffect(c, affectRegrowth, 0, 255, 0)
}

// Weapon pluses that matter.
const magicWeaponFull = 3

// magicWeaponsOnlyHandler = O064:12D8: plain weapons do nothing, weak
// magic ones half.
func (e *Engine) magicWeaponsOnlyHandler(c *Character, a *Affect, flag byte) {
	w := e.readiedWeapon(e.selected())
	if w == nil || w.Plus == 0 {
		e.rv.damage = 0
		return
	}
	if int8(w.Plus) < magicWeaponFull {
		e.rv.damage >>= 1
	}
}

// fireDamageReducedHandler = O064:132B: two less per die (at least one),
// saves up; fire that is not magic is stopped.
func (e *Engine) fireDamageReducedHandler(c *Character, a *Affect, flag byte) {
	if e.rv.dmgType&dmgFire == 0 {
		return
	}
	for i := byte(1); i <= e.rv.diceN; i++ {
		e.rv.damage -= 2
		if e.rv.damage < e.rv.diceN {
			e.rv.damage = e.rv.diceN
		}
	}
	e.rv.save += 4
	if e.rv.dmgType&dmgMagic == 0 {
		e.blockAffect(0)
	}
}

// healingOverTimeHandler = O064:1386: a hit point now and then.
func (e *Engine) healingOverTimeHandler(c *Character, a *Affect, flag byte) {
	if e.addAffectNow(c, affectHealingOverTime, a.Data, healingTime) && e.Heal(c, 1, true) {
		e.healedMessage(c)
	}
}

// Spell levels the minor globe stops.
const globeMaxLevel = 4

// minorGlobeHandler = O064:13CA.
func (e *Engine) minorGlobeHandler(c *Character, a *Affect, flag byte) {
	if e.curSpell > 0 && spellTableData[e.curSpell][spellLevelField] < globeMaxLevel {
		e.blockAffect(0)
	}
}

func (e *Engine) poisonStingHandler(c *Character, a *Affect, flag byte)       { e.poisonAttack(c, 0) }  // O064:1520
func (e *Engine) poisonSting4Handler(c *Character, a *Affect, flag byte)      { e.poisonAttack(c, 4) }  // O064:1534
func (e *Engine) poisonSting2Handler(c *Character, a *Affect, flag byte)      { e.poisonAttack(c, 2) }  // O064:1548
func (e *Engine) poisonStingMinus2Handler(c *Character, a *Affect, flag byte) { e.poisonAttack(c, -2) } // O064:1671

// paralysingTouchHandler = O064:155C.
func (e *Engine) paralysingTouchHandler(c *Character, a *Affect, flag byte) {
	e.paralyseTarget(c, e.Dice(2, 8))
}

// Feeblemind's intelligence and wisdom.
const feebleScore = 7

// feeblemindHandler = O064:1595 spell_stupid.
func (e *Engine) feeblemindHandler(c *Character, a *Affect, flag byte) {
	c.Stats[statInt].Value = feebleScore
	c.Stats[statWis].Value = feebleScore
	e.combatName(c, "is stupid", combatMsgRow, true)
	if e.gameState != stateCombat {
		return
	}
	ci := combatInfo(c)
	ci.CanCast = 0
	if ci.Spell > 0 {
		e.combatName(c, "lost a spell", combatMsgRow+2, true)
		forgetSpellOf(c, ci.Spell)
		ci.Spell = 0
	}
}

// hiddenFromKind19Handler = O064:1636.
func (e *Engine) hiddenFromKind19Handler(c *Character, a *Affect, flag byte) {
	if e.selected().Kind == kindKind19 && e.FindAffect(e.selected(), affectDetectInvisibility) == nil {
		e.v681E = 1
		e.rv.hitRoll -= 4
	}
}

// blurHandler = O064:1685.
func (e *Engine) blurHandler(c *Character, a *Affect, flag byte) {
	e.v681E = 1
	e.rv.hitRoll -= 4
}

// flicker95Handler = O064:1696.
func (e *Engine) flicker95Handler(c *Character, a *Affect, flag byte) {
	if e.Dice(1, 100) <= 95 {
		e.AddAffect(c, affectInvisible, 1, 12, 0)
	}
}

// unaffectedByType32Handler = O064:16D2.
func (e *Engine) unaffectedByType32Handler(c *Character, a *Affect, flag byte) {
	if e.rv.dmgType&dmgType32 > 0 {
		e.blockAffect(0)
		e.combatName(c, "is unaffected", combatMsgRow, true)
	}
}

// vsKind3Handler = O064:171A: a heavy blow against kind 3.
func (e *Engine) vsKind3Handler(c *Character, a *Affect, flag byte) {
	e.v6826 = targetOf(c)
	if e.v6826.Kind == kindKind3 {
		e.rv.damage = byte(e.Dice(1, 12)*3 + 4 + int(strDamage(c)))
		e.rv.hitRoll += 2
	}
}

// vsKind8Handler = O064:1772.
func (e *Engine) vsKind8Handler(c *Character, a *Affect, flag byte) {
	e.v6826 = targetOf(c)
	if e.v6826.Kind == kindKind8 {
		e.rv.hitRoll += 3
		e.rv.damage += 3
	}
}

// berserkHandler = O064:17B3 spl_berzerk: attacks the nearest one, of
// either side.
func (e *Engine) berserkHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		if c.Control == controlCharmed {
			c.Control = 0
		}
		c.Side = 0
		return
	}
	e.goBerserk(c)
	if e.gameState == stateCombat {
		combatInfo(c).CanCast = 0
		c.Side = targetOf(c).Side ^ 1
		e.combatName(c, "goes berzerk", combatMsgRow, true)
	}
}

// goBerserk is the start the two berserk handlers share: computer
// controlled, against the nearest combatant.
func (e *Engine) goBerserk(c *Character) {
	c.Auto = 1
	if c.Control <= controlHired || c.Control == controlCharmed {
		c.Control = controlCharmed
	} else {
		c.Control = controlBerserk
	}
	if e.gameState != stateCombat {
		return
	}
	ci := combatInfo(c)
	ci.Target = nil
	e.buildNear(e.combatX(c), e.combatY(c), 255, 255, e.combatSize(c))
	if len(e.near.list) > 1 {
		ci.Target = e.combatant(e.near.list[1].index)
	}
}

// recoveringHandler = O064:18D9: back up when there is room, else later.
func (e *Engine) recoveringHandler(c *Character, a *Affect, flag byte) {
	if !e.Revive(c, c.HP) {
		e.addAffectNow(c, affectRecovering, a.Data, 1)
	}
}

// burningTouchHandler = O064:1919.
func (e *Engine) burningTouchHandler(c *Character, a *Affect, flag byte) {
	e.rv.dmgType = dmgFire | dmgMagic
	e.damagePerson(targetOf(c), e.RollDice(2, 10), saveNone, false)
}

// acidTouchHandler = O064:194C.
func (e *Engine) acidTouchHandler(c *Character, a *Affect, flag byte) {
	e.rv.dmgType = dmgAcid
	e.damagePerson(targetOf(c), e.RollDice(1, 4), saveNone, false)
}

// halfDamageHandler = O064:197F.
func (e *Engine) halfDamageHandler(c *Character, a *Affect, flag byte) { e.rv.damage >>= 1 }

// Spell table fields.
const (
	spellClassField = 0
	spellLevelField = 1
	spellSaveField  = 8 // what a saving throw does (0: nothing)
)

// saveVsElementsHandler = O064:1990: fire and cold are halved, or with a
// saving throw against a spell that allows one, stopped.
func (e *Engine) saveVsElementsHandler(c *Character, a *Affect, flag byte) {
	if e.rv.dmgType&(dmgFire|dmgCold) == 0 {
		return
	}
	if e.SavingThrow(c, saveVsSpell, 0) && spellTableData[e.curSpell][spellSaveField] != 0 {
		e.rv.damage = 0
		return
	}
	e.rv.damage >>= 1
}

// electricityHealsHandler = O064:19E1.
func (e *Engine) electricityHealsHandler(c *Character, a *Affect, flag byte) {
	if e.rv.dmgType&dmgElectricity != 0 {
		e.blockAffect(0)
		c.HP += byte(e.Dice(1, 8))
	}
}

// Base item flags (itemFlags).
const (
	itemUsesArrows = 1
	itemUsesBolts  = 128
)

// arrowsScratchHandler = O064:1A14.
func (e *Engine) arrowsScratchHandler(c *Character, a *Affect, flag byte) {
	if w := e.readiedWeapon(e.selected()); w != nil && e.baseItem(w.Type, itemFlags) == itemUsesArrows {
		e.rv.damage = 1
	}
}

// First strike data.
const (
	firstStrikeDone = 16
	firstStrikeKeep = 15
)

// firstStrikeHandler = O064:1A55: in the first round an attack that rolled
// nothing still... the original's note: misses once per round.
func (e *Engine) firstStrikeHandler(c *Character, a *Affect, flag byte) {
	if e.combatRound == 0 && e.rv.hitRoll == 0 {
		a.Data &= firstStrikeKeep
		return
	}
	if a.Data&firstStrikeDone == 0 {
		e.rv.hitRoll = -1
		a.Data |= firstStrikeDone
	}
}

// fireHalvedHandler = O064:1D98.
func (e *Engine) fireHalvedHandler(c *Character, a *Affect, flag byte) {
	if e.rv.dmgType&dmgFire != 0 {
		e.rv.damage >>= 1
	}
}

// rangedHalvedHandler = O064:1DB2.
func (e *Engine) rangedHalvedHandler(c *Character, a *Affect, flag byte) {
	if w := e.readiedWeapon(e.selected()); w != nil && e.baseItem(w.Type, itemFlags)&(itemUsesArrows|itemUsesBolts) != 0 {
		e.rv.damage >>= 1
	}
}

// fallsDeadHandler = O064:1E06.
func (e *Engine) fallsDeadHandler(c *Character, a *Affect, flag byte) {
	a.Call = 0
	if c.Okay != 0 {
		e.killWithMessage(c, statusDead, "Falls dead")
	}
}

// conSaveHandler = O064:1E42: dwarves' and halflings' constitution bonus
// against rods and spells. Below constitution 4 the original adds an unset
// local; the port adds nothing.
func (e *Engine) conSaveHandler(c *Character, a *Affect, flag byte) {
	if e.rv.saveType != saveVsSpell && e.rv.saveType != saveVsRod {
		return
	}
	var bonus byte
	switch con := c.Stats[statCon].Value; {
	case con >= 4 && con <= 6:
		bonus = 1
	case con >= 7 && con <= 10:
		bonus = 2
	case con >= 11 && con <= 13:
		bonus = 3
	case con >= 14 && con <= 17:
		bonus = 4
	case con >= 18 && con <= 20:
		bonus = 5
	}
	e.rv.save += bonus
}

// regrowthHandler = O064:1EB8.
func (e *Engine) regrowthHandler(c *Character, a *Affect, flag byte) {
	c.HP += 3
	if c.HP > c.MaxHP {
		c.HP = c.MaxHP
	}
}

// Hit points a fallen troll gets back up with.
const backUpHP = 6

// backsUpHandler = O064:1EE8: a dying or unconscious troll gets up, to
// fall dead a few rounds later.
func (e *Engine) backsUpHandler(c *Character, a *Affect, flag byte) {
	var hp byte
	if c.Status == statusDying && combatInfo(c).Dying < backUpHP {
		hp = backUpHP - combatInfo(c).Dying
	}
	if c.Status == statusUnconscious {
		hp = backUpHP
	}
	if hp > 0 && e.Revive(c, hp) {
		e.AddAffect(c, affectFallsDead, e.Dice(1, 4)+1, 255, 1)
		a.Call = 0
		e.RemoveAffect(c, affectBacksUp, a)
	}
}

// regrowsHandler = O064:1F91: wounds other than fire and acid grow back.
func (e *Engine) regrowsHandler(c *Character, a *Affect, flag byte) {
	if e.rv.dmgType&dmgFire == 0 && e.rv.dmgType&dmgAcid == 0 {
		e.AddAffect(c, affectRegrowing, e.Dice(3, 6), 255, 1)
	}
}

// regeneratesHandler = O064:1FCC sp_regenerate.
func (e *Engine) regeneratesHandler(c *Character, a *Affect, flag byte) {
	if e.FindAffect(c, affectRegrowth) == nil && e.FindAffect(c, affectRegenerating) == nil {
		e.AddAffect(c, affectRegenerating, 3, 255, 1)
	}
}

// regrowingHandler = O064:201E: back up whole when there is room.
func (e *Engine) regrowingHandler(c *Character, a *Affect, flag byte) {
	if !e.Revive(c, c.MaxHP) {
		e.addAffectNow(c, affectRegrowing, a.Data, 1)
	}
}

// burnsHandler = O064:205D: d6 more unless the target resists fire.
func (e *Engine) burnsHandler(c *Character, a *Affect, flag byte) {
	t := targetOf(c)
	e.v6826 = t
	if e.FindAffect(t, affectResistFire) == nil && e.FindAffect(t, affectIceBody) == nil &&
		e.FindAffect(t, affectFireDamageReduced) == nil {
		e.rv.damage += byte(e.Dice(1, 6))
	}
}

func (e *Engine) avoidRanged60Handler(c *Character, a *Affect, flag byte) {
	e.avoidDistantAttack(c, 60)
}                                                                         // O064:20DA
func (e *Engine) magicResist50Handler(c *Character, a *Affect, flag byte) { e.resistRoll(50) } // O064:214F
func (e *Engine) magicResist15Handler(c *Character, a *Affect, flag byte) { e.resistRoll(15) } // O064:215D

// elfResistHandler = O064:216B: 90% against sleep and charm.
func (e *Engine) elfResistHandler(c *Character, a *Affect, flag byte) {
	if e.Dice(1, 100) <= 90 {
		e.blockAffect(affectHeld)
		e.blockAffect(affectCharmed)
	}
}

// noCharmHoldHandler = O064:218F.
func (e *Engine) noCharmHoldHandler(c *Character, a *Affect, flag byte) {
	e.blockAffect(affectCharmed)
	e.blockAffect(affectHeld)
}

// noParalysisHandler = O064:21A4.
func (e *Engine) noParalysisHandler(c *Character, a *Affect, flag byte) {
	e.blockAffect(affectParalyzed)
}

// coldImmuneHandler = O064:21B2.
func (e *Engine) coldImmuneHandler(c *Character, a *Affect, flag byte) {
	if e.rv.dmgType&dmgCold != 0 {
		e.blockAffect(0)
	}
}

// A save that always succeeds.
const sureSave = 100

// noPoisonParalysisHandler = O064:21C9.
func (e *Engine) noPoisonParalysisHandler(c *Character, a *Affect, flag byte) {
	e.blockAffect(affectPoisoned)
	e.blockAffect(affectParalyzed)
	if e.rv.saveType == saveVsParalysis {
		e.rv.save = sureSave
	}
}

// fireImmuneHandler = O064:21EA.
func (e *Engine) fireImmuneHandler(c *Character, a *Affect, flag byte) {
	if e.rv.dmgType&dmgFire != 0 {
		e.blockAffect(0)
	}
}

// fireReducedHandler = O064:2201: a point less per die, at least one.
func (e *Engine) fireReducedHandler(c *Character, a *Affect, flag byte) {
	if e.rv.dmgType&dmgFire == 0 {
		return
	}
	for i := byte(1); i <= e.rv.diceN; i++ {
		e.rv.damage--
		if e.rv.damage < e.rv.diceN {
			e.rv.damage = e.rv.diceN
		}
	}
}

// electricityHalvedHandler = O064:2246.
func (e *Engine) electricityHalvedHandler(c *Character, a *Affect, flag byte) {
	if e.rv.dmgType&dmgElectricity != 0 {
		e.rv.damage >>= 1
	}
}

// edgedHalvedHandler = O064:2260: melee weapons and bows do half.
func (e *Engine) edgedHalvedHandler(c *Character, a *Affect, flag byte) {
	if w := e.readiedWeapon(e.selected()); w != nil {
		if f := e.baseItem(w.Type, itemFlags); f == 0 || f&itemUsesArrows != 0 {
			e.rv.damage >>= 1
		}
	}
}

// magicWeaponsHalvedHandler = O064:22BF.
func (e *Engine) magicWeaponsHalvedHandler(c *Character, a *Affect, flag byte) {
	if w := e.readiedWeapon(e.selected()); w != nil && int8(w.Plus) > 0 {
		e.rv.damage >>= 1
	}
}

// Item types the defences look at.
const (
	itemTypeSling    = 85
	itemTypeLongBow  = 87
	itemTypeShortBow = 88
	itemTypeArrow    = 28
	slayingArrowWord = 135
)

// slingStonesHandler = O064:22F9: a sling does d6+1.
func (e *Engine) slingStonesHandler(c *Character, a *Affect, flag byte) {
	if w := e.selected().Slots[0]; w != nil && w.Type == itemTypeSling {
		e.rv.damage = byte(e.RollDice(1, 6) + 1)
	}
}

// coldHalvedHandler = O064:233C.
func (e *Engine) coldHalvedHandler(c *Character, a *Affect, flag byte) {
	if e.rv.dmgType&dmgCold != 0 {
		e.rv.damage >>= 1
	}
}

// Monster attackers' hit dice that still hurt.
const strongMonsterDice = 4

// needsMagicWeaponHandler = O064:2356: only magic weapons, or monsters of
// four or more hit dice, hurt it.
func (e *Engine) needsMagicWeaponHandler(c *Character, a *Affect, flag byte) {
	w := e.readiedWeapon(e.selected())
	sel := e.selected()
	if (w == nil || w.Plus == 0) && (int8(sel.Race) > 0 || int8(sel.MaxLevel) < strongMonsterDice) {
		e.rv.damage = 0
	}
}

// avoidsBowsHandler = O064:23A2.
func (e *Engine) avoidsBowsHandler(c *Character, a *Affect, flag byte) {
	if w := e.selected().Slots[0]; w != nil && (w.Type == itemTypeLongBow || w.Type == itemTypeShortBow) {
		e.avoidDistantAttack(c, 50)
	}
}

// Acid spit.
const (
	spitChance  = 25
	spitRange   = 4
	spitColours = 23
	spitFrames  = 1
	spitDelay   = 30
)

// spitsAcidHandler = O064:23F6: a quarter of the time, once, at a near
// target: 8d4, half on a save.
func (e *Engine) spitsAcidHandler(c *Character, a *Affect, flag byte) {
	t := targetOf(c)
	e.v6826 = t
	if e.Dice(1, 100) > spitChance || e.rangeTo(c, t) >= spitRange {
		return
	}
	e.endTurn(c)
	e.combatName(c, "Spits Acid", combatMsgRow, true)
	e.hitColours(spitColours)
	e.flyMissile(e.combatX(c), e.combatY(c), e.combatX(t), e.combatY(t), spitFrames, spitDelay)
	dmg := e.RollDice(8, 4)
	saved := e.SavingThrow(t, saveVsBreath, 0)
	e.damagePerson(t, dmg, saveHalf, saved)
	e.RemoveAffect(c, affectSpitsAcid, a)
	e.RemoveAffect(c, affectAcidTouch, nil)
}

// paralysingGazeHandler = O064:2523 spl_paralyze.
func (e *Engine) paralysingGazeHandler(c *Character, a *Affect, flag byte) {
	t := targetOf(c)
	e.v6826 = t
	if !e.SavingThrow(t, saveVsParalysis, 0) {
		e.AddAffect(t, affectParalyzed, 0, 255, 0)
		e.combatName(t, "is paralyzed", combatMsgRow, true)
	}
}

// fireTouchHandler = O064:259D.
func (e *Engine) fireTouchHandler(c *Character, a *Affect, flag byte) {
	e.rv.dmgType = dmgCold | dmgMagic
	e.damagePerson(targetOf(c), e.RollDice(2, 8), saveNone, false)
}

// halfElfResistHandler = O064:25D0: 30% against sleep and charm.
func (e *Engine) halfElfResistHandler(c *Character, a *Affect, flag byte) {
	if e.Dice(1, 100) <= 30 {
		e.blockAffect(affectCharmed)
		e.blockAffect(affectHeld)
	}
}

// noCharmHoldSleepHandler = O064:25F4.
func (e *Engine) noCharmHoldSleepHandler(c *Character, a *Affect, flag byte) {
	e.blockAffect(affectCharmed)
	e.blockAffect(affectHeld)
	e.blockAffect(affectParalyzed)
	e.blockAffect(affectPoisoned)
	if e.rv.saveType == saveVsParalysis {
		e.rv.save = sureSave
	}
}

// magicImmuneHandler = O064:2623.
func (e *Engine) magicImmuneHandler(c *Character, a *Affect, flag byte) {
	if e.rv.affVal == 0 && e.rv.dmgType&dmgMagic == 0 {
		return
	}
	e.blockAffect(0)
}

// slainByArrowHandler = O064:2643: an arrow of slaying kills outright.
func (e *Engine) slainByArrowHandler(c *Character, a *Affect, flag byte) {
	var it *Item
	if e.ammoFor(e.selected(), &it) && it != nil && it.Type == itemTypeArrow && it.Words[2] == slayingArrowWord {
		c.Status = statusGone
		c.Okay = 0
		c.HP = 0
		e.downAffects(c)
		e.affectGroup(c, grpDown)
		if c.Okay == 0 {
			e.combatRemoveFromMap(c)
		}
	}
}

// noMindAffectsHandler = O064:2750.
func (e *Engine) noMindAffectsHandler(c *Character, a *Affect, flag byte) {
	e.blockAffect(affectRunningAway)
	e.blockAffect(affectQuarterLessDamage)
	e.blockAffect(affectFeebleminded)
	if e.rv.dmgType&dmgType64 != 0 {
		e.blockAffect(0)
	}
}

// rangerVsGiantsHandler = O064:277C: a point per ranger level against
// giants.
func (e *Engine) rangerVsGiantsHandler(c *Character, a *Affect, flag byte) {
	e.v6826 = targetOf(c)
	if e.v6826.Pad331&creatureGiant != 0 {
		e.rv.damage += c.Levels[clsRanger]
	}
}

// electricityImmuneHandler = O064:27B5.
func (e *Engine) electricityImmuneHandler(c *Character, a *Affect, flag byte) {
	if e.rv.dmgType&dmgElectricity != 0 {
		e.blockAffect(0)
	}
}

// rootedHandler = O064:27CC.
func (e *Engine) rootedHandler(c *Character, a *Affect, flag byte) { combatInfo(c).Moves = 0 }

// berserkFoeHandler = O064:27E0: attacks the nearest; afterwards back to
// the side its data keeps.
func (e *Engine) berserkFoeHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		if c.Control == controlCharmed {
			c.Control = 0
		}
		c.Side = a.Data
		return
	}
	e.goBerserk(c)
	if e.gameState == stateCombat {
		c.Side = targetOf(c).Side ^ 1
	}
}

// alwaysInvisibleHandler = O064:28D0 add_affect_19.
func (e *Engine) alwaysInvisibleHandler(c *Character, a *Affect, flag byte) {
	e.AddAffect(c, affectInvisible, 255, 255, 0)
}

// Paladin cures: one a week per five levels.
const paladinCureLevels = 5

// paladinCuresHandler = O064:28EF: when the week ends, the cures are back.
func (e *Engine) paladinCuresHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 {
		return
	}
	usable := 0
	if oldClassUsable(c) {
		usable = 1
	}
	lvl := int(int8(c.Levels[clsPaladin])) + int(int8(c.OldLevels[clsPaladin]))*usable
	c.PaladinCures = byte((lvl-1)/paladinCureLevels + 1)
}

// runningAwayHandler = O064:2932.
func (e *Engine) runningAwayHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 {
		return
	}
	if c.Control == controlCharmed {
		c.Control = 0
		c.Auto = 0
	}
	combatInfo(c).Turned = 0
}

// zapsHandler = O064:2975: a close attacker takes double the damage it
// dealt, as magic.
func (e *Engine) zapsHandler(c *Character, a *Affect, flag byte) {
	sel := e.selected()
	if e.rangeTo(sel, c) >= 2 {
		return
	}
	dmg, typ := e.rv.damage, e.rv.dmgType
	e.rv.damage *= 2
	e.rv.dmgType = dmgMagic
	e.combatName(sel, "gets zapped", combatMsgRow, true)
	e.damagePerson(sel, int(e.rv.damage), saveNone, false)
	e.rv.damage, e.rv.dmgType = dmg, typ
}

// dispelEvilHandler = O064:2A19 sp_dispel_evil.
func (e *Engine) dispelEvilHandler(c *Character, a *Affect, flag byte) {
	t := targetOf(c)
	e.v6826 = t
	if t.Pad331&creatureEnchanted != 0 && !e.SavingThrow(t, saveVsSpell, 0) {
		e.killWithMessage(t, statusGone, "is dispelled")
		e.RemoveAffect(e.selected(), affectWardEnchanted, nil)
		e.RemoveAffect(e.selected(), affectDispelEvil, nil)
		return
	}
	e.combatName(t, "resists dispel evil", combatMsgRow, true)
}

// --- O08D's handlers --------------------------------------------------------

// Engulf and hug.
const (
	engulfAttacks = 2
	hugAttacks    = 1
	grabDice      = 2
	grabSides     = 8
	hugMinRoll    = 18
)

// engulfsHandler = O08D:4032 engulfs: two hits engulf the target, which
// then suffocates.
func (e *Engine) engulfsHandler(c *Character, a *Affect, flag byte) {
	t := targetOf(c)
	if e.hits6822[1] != 2 || t.Okay == 0 || e.FindAffect(t, affectEngulfed) != nil ||
		e.FindAffect(t, affectSuffocating) != nil {
		return
	}
	e.combatName(c, "engulfs "+t.Name(), combatMsgRow+2, true)
	e.AddAffect(t, affectEngulfed, 0, byte(e.combatIndex(t)), 0)
	e.callAffect(affectEngulfed, t, nil, 0)
	e.AddAffect(t, affectSuffocating, 0, byte(e.Dice(2, 4)), 0)
	e.AddAffect(c, affectEngulfing, 0, byte(e.combatIndex(t)), 1)
}

// holdTarget is the shared body of the engulfing and hugging handlers
// (O08D:45C6 and 46FC): the held one is hit every round until either
// falls; flag set or a fall frees it.
func (e *Engine) holdTarget(c *Character, a *Affect, flag byte, self byte, extra []byte, attacks byte) {
	e.v6826 = e.combatant(a.Data)
	t := e.v6826
	if t == nil {
		t = &Character{}
	}
	free := func() {
		for _, x := range extra {
			e.RemoveAffect(t, x, nil)
		}
	}
	if flag != 0 || c.Okay == 0 || t.Okay == 0 {
		free()
		if flag == 0 {
			a.Call = 0
			e.RemoveAffect(c, self, a)
		}
		return
	}
	c.Attacks[0] = attacks
	c.Attacks[1] = 0
	c.Dice[0] = grabDice
	c.Sides[0] = grabSides
	e.attackOnce(c, t, 1, nil)
	e.endTurn(c)
	if t.Okay == 0 {
		e.RemoveAffect(c, self, nil)
		free()
	}
}

// engulfingHandler = O08D:45C6.
func (e *Engine) engulfingHandler(c *Character, a *Affect, flag byte) {
	e.holdTarget(c, a, flag, affectEngulfing, []byte{affectEngulfed, affectSuffocating}, engulfAttacks)
}

// huggingHandler = O08D:46FC.
func (e *Engine) huggingHandler(c *Character, a *Affect, flag byte) {
	e.holdTarget(c, a, flag, affectHugging, []byte{affectEngulfed}, hugAttacks)
}

// hugsHandler = O08D:480C hugs: a roll of 18 or more grabs the target.
func (e *Engine) hugsHandler(c *Character, a *Affect, flag byte) {
	if e.rv.hitRoll < hugMinRoll {
		return
	}
	t := targetOf(c)
	e.v6826 = t
	e.combatName(c, "hugs "+t.Name(), combatMsgRow+2, true)
	e.AddAffect(t, affectEngulfed, 0, byte(e.combatIndex(t)), 0)
	e.callAffect(affectEngulfed, t, nil, 0)
	e.AddAffect(c, affectHugging, 0, byte(e.combatIndex(t)), 1)
}

// Ray ranges and the spells the eyes cast after them.
const (
	rayTries          = 4
	disintegrateRange = 2
	stoneRange        = 3
	deathRayRange     = 4
	woundRange        = 5
	rayColour         = 13
	raySmallColours   = 5
	rayStoneColours   = 10
	rayFrames         = 1
	rayDelay          = 30
	eyeSpell1         = 84
	eyeSpell2         = 55
	eyeSpell3         = 21
)

// raysHandler = O08D:42F2 attack_or_kill: an eye tyrant's rays, each used
// once, at whoever is in reach, then its spells.
func (e *Engine) raysHandler(c *Character, a *Affect, flag byte) {
	ci := combatInfo(c)
	var used byte
	tries := rayTries
	ci.Target = nil
	retarget := func(first bool) {
		// O08D:41C1 also checks the line of fire, which only sets the
		// spell's quit flag that O0C2:12E1 overwrites.
		e.pickTarget(c, 255, 0, first)
	}
	shoot := func(colours byte, t *Character) {
		e.hitColours(int(colours) + rayColour)
		e.flyMissile(e.combatX(c), e.combatY(c), e.combatX(t), e.combatY(t), rayFrames, rayDelay)
	}
	retarget(true)
	for {
		t := ci.Target
		r := e.rangeTo(c, t)
		tries--
		if t != nil {
			var done bool
			switch {
			case r <= disintegrateRange && used&1 == 0:
				used |= 1
				e.combatName(c, "fires a disintegrate ray", combatMsgRow, true)
				shoot(raySmallColours, t)
				if !e.SavingThrow(t, saveVsBreath, 0) {
					e.killWithMessage(t, statusGone, "is disintergrated")
				}
				retarget(false)
			case r <= stoneRange && used&2 == 0:
				used |= 2
				e.combatName(c, "fires a stone to flesh ray", combatMsgRow, true)
				shoot(rayStoneColours, t)
				if !e.SavingThrow(t, saveVsPetrify, 0) {
					e.killWithMessage(t, statusStoned, "is Stoned")
				}
				retarget(false)
			case r <= deathRayRange && used&4 == 0:
				used |= 4
				e.combatName(c, "fires a death ray", combatMsgRow, true)
				shoot(raySmallColours, t)
				if !e.SavingThrow(t, saveVsParalysis, 0) {
					e.killWithMessage(t, statusDead, "is killed")
				}
				retarget(false)
			case r <= woundRange && used&8 == 0:
				used |= 8
				e.combatName(c, "wounds you", combatMsgRow, true)
				shoot(raySmallColours, t)
				e.damagePerson(t, e.RollDice(2, 8)+1, saveNone, false)
				retarget(false)
			case used&16 == 0:
				e.castInnate(eyeSpell1, 1, 1, &done)
				used |= 16
			case used&32 == 0:
				e.castInnate(eyeSpell2, 1, 1, &done)
				used |= 32
			case used&64 == 0:
				e.castInnate(eyeSpell3, 1, 1, &done)
				used |= 64
			}
		}
		if tries <= 0 || ci.Target == nil {
			break
		}
	}
}

// forgetSpellOf = O0ED:1520 (char, spell): its first memorised copy goes.
func forgetSpellOf(c *Character, sp byte) {
	for i := range c.Memorized {
		if c.Memorized[i] == sp {
			c.Memorized[i] = 0
			return
		}
	}
}

// Bridges to the combat overlays, until O08D and O0C2 are ported:
// attackOnce = O08D:19DB, pickTarget = O08D:3E44, castInnate = O0C2:12E1.

func (e *Engine) attackOnce(c, t *Character, flag int, it *Item) { e.fullAttack(c, t, flag, it) }

func (e *Engine) pickTarget(c *Character, rng, a int, first bool) bool {
	b := 0
	if first {
		b = 1
	}
	return e.pickAttackTarget(c, rng, a, b)
}

func (e *Engine) castInnate(spell, a, b int, quit *bool) { e.spellMenuTargets(spell, a, quit) }
