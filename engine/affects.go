package engine

// The affect handlers of O0A4, with the table O0A4:3969 fills. Each
// handler is (char, affect, flag); flag 1 means the affect is being
// removed. The rule globals DS:87D2.. are e.rv (see rules.go).

// AffectHandler is one entry of the DS:87EA table.
type AffectHandler func(e *Engine, c *Character, a *Affect, flag byte)

var affectHandlers [affectHandlerCount]AffectHandler

// itemHandler = O0A4:3908, the table's last entry: a readied (or
// unreadied) item that grants the affect its Spells[1] names.
func (e *Engine) itemHandler(c *Character, it *Item, flag byte) {
	e.itemCall = false
	typ := it.Spells[1]
	if flag != 0 {
		e.RemoveAffect(c, typ, nil)
		return
	}
	e.AddAffect(c, typ, 0, 255, 1)
}

// setupAffectHandlers = O0A4:3969, with the two breath weapons (83, 104)
// the magic overlay adds (O113:4650); the numbers it leaves out have no
// handler.
func setupAffectHandlers() {
	h := &affectHandlers
	h[affectBless] = (*Engine).blessHandler
	h[affectCurse] = (*Engine).curseHandler
	h[affectSticksToSnakes] = (*Engine).snakesHandler
	h[affectDispellingEvil] = (*Engine).dispellingEvilHandler
	h[affectDetectMagic] = (*Engine).emptyHandler
	h[affectElectricityImmune] = (*Engine).electricityImmuneHandler
	h[affectGnomeVsFoes] = (*Engine).gnomeVsFoesHandler
	h[affectProtFromEvil] = (*Engine).protFromEvilHandler
	h[affectProtFromGood] = (*Engine).protFromGoodHandler
	h[affectResistCold] = (*Engine).resistColdHandler
	h[affectCharmed] = (*Engine).charmHandler
	h[affectEnlarge] = (*Engine).emptyHandler
	h[affectBarkskin] = (*Engine).barkskinHandler
	h[affectFriendly] = (*Engine).emptyHandler
	h[affectPoisonDamage] = (*Engine).poisonDamageHandler
	h[affectReadMagic] = (*Engine).emptyHandler
	h[affectShield] = (*Engine).shieldHandler
	h[affectHalfElfResistance] = (*Engine).halfElfResistHandler
	h[affectFindTraps] = (*Engine).emptyHandler
	h[affectResistFire] = (*Engine).resistFireHandler
	h[affectSilenced] = (*Engine).silencedHandler
	h[affectSlowPoison] = (*Engine).slowPoisonHandler
	h[affectSpiritualHammer] = (*Engine).spiritualHammerHandler
	h[affectDetectInvisibility] = (*Engine).emptyHandler
	h[affectInvisible] = (*Engine).invisibleHandler
	h[affectDwarfVsFoes] = (*Engine).dwarfVsFoesHandler
	h[affectBefuddled] = (*Engine).loseTurnHandler
	h[affectMirrorImage] = (*Engine).mirrorImageHandler
	h[affectEnfeebled] = (*Engine).enfeebledHandler
	h[affectCoughing] = (*Engine).coughingHandler
	h[affectNausea] = (*Engine).loseTurnHandler
	h[affectDispelEvil] = (*Engine).dispelEvilHandler
	h[affectBlind] = (*Engine).blindHandler
	h[affectDiseased] = (*Engine).diseasedHandler
	h[affectConfused] = (*Engine).confusedHandler
	h[affectCursed] = (*Engine).cursedHandler
	h[affectBlinking] = (*Engine).blinkingHandler
	h[affectStrengthSpell] = (*Engine).emptyHandler
	h[affectHaste] = (*Engine).hasteHandler
	h[affectStinkingCloud] = (*Engine).stinkingCloudEndHandler
	h[affectProtNormalMissiles] = (*Engine).protNormalMissilesHandler
	h[affectSlow] = (*Engine).slowHandler
	h[affectWeakened] = (*Engine).weakenHandler
	h[affectWasting] = (*Engine).wastingHandler
	h[affectProtFromEvil10] = (*Engine).protFromEvilHandler
	h[affectProtFromGood10] = (*Engine).protFromGoodHandler
	h[affectDwarfDodges] = (*Engine).dwarfDodgesHandler
	h[affectGnomeDodges] = (*Engine).gnomeDodgesHandler
	h[affectPrayer] = (*Engine).prayerHandler
	h[affectFireShieldHot] = (*Engine).fireShieldHotHandler
	h[affectImmobile] = (*Engine).loseTurnHandler
	h[affectHeld] = (*Engine).loseTurnHandler
	h[affectAsleep] = (*Engine).loseTurnHandler
	h[affectFireShieldCold] = (*Engine).fireShieldColdHandler
	h[affectPoisoned] = (*Engine).emptyHandler
	h[affectInvisibleRing] = (*Engine).invisibleRingHandler
	h[affectGlobe] = (*Engine).globeHandler
	h[affectStoningGaze] = (*Engine).stoningGazeHandler
	h[affectGazeReflectable] = (*Engine).emptyHandler
	h[affectNeedsPlus3Weapon] = (*Engine).needsPlus3WeaponHandler
	h[affectFireResistRing] = (*Engine).fireResistRingHandler
	h[affectHealingOverTime] = (*Engine).healingOverTimeHandler
	h[affectMinorGlobe] = (*Engine).minorGlobeHandler
	h[affectPoisonSting] = (*Engine).poisonStingHandler
	h[affectStoningTouch] = (*Engine).stoningTouchHandler
	h[affectSwallowsWhole] = (*Engine).swallowsWholeHandler
	h[affectBreathFireClose] = (*Engine).breathFireCloseHandler
	h[affectFeebleminded] = (*Engine).feeblemindHandler
	h[affectInvisibleToAnimals] = (*Engine).invisibleToAnimalsHandler
	h[affectConfusingGaze] = (*Engine).confusingGazeHandler
	h[affectHighlighted] = (*Engine).highlightedHandler
	h[affectMirror] = (*Engine).emptyHandler
	h[affectDragonBreathImmune] = (*Engine).dragonBreathImmuneHandler
	h[affectMagicalBoots] = (*Engine).magicalBootsHandler
	h[affectBaneWeapon] = (*Engine).baneWeaponHandler
	h[affectDiseaseImmune] = (*Engine).emptyHandler
	h[affectBerserk] = (*Engine).berserkHandler
	h[affectMagicalStone] = (*Engine).magicalStoneHandler
	h[affectFireHeals] = (*Engine).fireHealsHandler
	h[affectBreathFire] = (*Engine).breathFireHandler
	h[affectSpitAcid] = (*Engine).spitAcidHandler
	h[affectFearAura] = (*Engine).fearAuraHandler
	h[84] = (*Engine).emptyHandler
	h[85] = (*Engine).emptyHandler
	h[affectPoisonStingMinus2] = (*Engine).poisonStingMinus2Handler
	h[87] = (*Engine).emptyHandler
	h[affectParalysingTouch] = (*Engine).paralysingTouchHandler
	h[affectMagicalCloak] = (*Engine).magicalCloakHandler
	h[affectBoltsHarmless] = (*Engine).boltsHarmlessHandler
	h[affectCloudkill] = (*Engine).cloudkillEndHandler
	h[affectFearless] = (*Engine).fearlessHandler
	h[affectFireImmune] = (*Engine).fireImmuneHandler
	h[affectHardyAll] = (*Engine).hardyAllHandler
	h[affectElfResistance] = (*Engine).elfResistHandler
	h[affectNoCharmSleep] = (*Engine).noCharmSleepHandler
	h[affectConSavingBonus] = (*Engine).conSaveHandler
	h[affectColdImmune] = (*Engine).coldImmuneHandler
	h[affectPoisonImmune] = (*Engine).poisonImmuneHandler
	h[affectDelayedBlast] = (*Engine).delayedBlastHandler
	h[101] = (*Engine).emptyHandler
	h[102] = (*Engine).emptyHandler
	h[affectPlainWeaponsImmune] = (*Engine).plainWeaponsImmuneHandler
	h[affectRangerVsGiants] = (*Engine).rangerVsGiantsHandler
	h[affectStunned] = (*Engine).loseTurnHandler
	h[affectBerserkFoe] = (*Engine).berserkFoeHandler
	h[affectAlwaysInvisible] = (*Engine).alwaysInvisibleHandler
	h[affectLaidOnHands] = (*Engine).emptyHandler
	h[affectPaladinCures] = (*Engine).paladinCuresHandler
	h[affectFrightened] = (*Engine).frightenedHandler
	h[affectZapsAttackers] = (*Engine).zapsHandler
	h[affectGiantStrength] = (*Engine).giantStrengthHandler
	h[affectBreathWeapon] = (*Engine).breathWeaponHandler
	h[affectBreathWeapon2] = (*Engine).breathWeaponHandler
}

func init() { setupAffectHandlers() }

// Spell table fields (spellTableData rows).
const (
	spellClassField = 0
	spellLevelField = 1
	spellSaveField  = 8  // what a saving throw does (0: nothing)
	spellImageField = 14 // non-zero: a mirror image can take the spell
)

// --- shared pieces --------------------------------------------------------

// blockAffect = O0A4:007F (type): the affect being added is refused when
// type is 0 or the one being added; the damage goes with it.
func (e *Engine) blockAffect(typ byte) {
	if typ == 0 || e.rv.affVal == typ {
		e.rv.damage = 0
		e.rv.affVal = 0
	}
}

// addAffectNow = O0A4:00A0 (char, type, data, time): adds the affect with
// its removal call; refused while an affect is being cured.
func (e *Engine) addAffectNow(c *Character, typ, data byte, time int) bool {
	if e.rv.cure != 0 {
		return false
	}
	e.AddAffect(c, typ, time, data, 1)
	return true
}

// endTurn = O151:2DEC (char): no initiative, spell, guard or moves left;
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

// readiedWeapon = O0A4:1381 (char): the ammunition the readied weapon
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

// avoidDistantAttack = O0A4:1308 (char, percent): an attack from afar by
// the selected character (who holds a weapon) may miss.
func (e *Engine) avoidDistantAttack(c *Character, pct int) {
	sel := e.selected()
	if sel.Slots[0] != nil && e.rangeTo(sel, c) > 1 && e.Dice(1, 100) <= pct {
		e.combatName(c, "Avoids it", combatMsgRow, true)
		e.rv.damage = 0
		e.rv.hitRoll = -1
		e.rv.hits[1]--
	}
}

// poisonAttack = O0A4:1A67 (char, bonus): the one the character fights is
// poisoned to death unless it saves. A victim with a poison sting of its
// own adds the sting's data to the saving throw, unless the bonus is
// negative, which then counts as none.
func (e *Engine) poisonAttack(c *Character, bonus int) {
	t := targetOf(c)
	e.rv.target = t
	if sting := e.FindAffect(t, affectPoisonSting); sting != nil {
		if bonus < 0 {
			bonus = 0
		} else {
			bonus += int(sting.Data)
		}
	}
	saved := e.SavingThrow(t, saveVsParalysis, bonus)
	e.isUnaffected(t, affectPoisoned, 0, 255, 0, 1, saved, "is "+affectNameOf(affectPoisoned))
	e.pauseForMessage()
	if e.FindAffect(t, affectPoisoned) != nil {
		e.killWithMessage(t, statusDead, "is killed")
	}
}

// paralyseTarget = O0A4:0014 (char, time, bonus): the one the character
// fights is paralysed unless it saves.
func (e *Engine) paralyseTarget(c *Character, time int, bonus int) {
	t := targetOf(c)
	e.rv.target = t
	saved := e.SavingThrow(t, saveVsParalysis, bonus)
	e.isUnaffected(t, affectHeld, time, 12, 0, 1, saved, "paralyzed by poison")
}

// selectedAligned: is the selected character's alignment one of these?
func (e *Engine) selectedAligned(a, b, c byte) bool {
	al := e.selected().Alignment
	return al == a || al == b || al == c
}

// --- the handlers ----------------------------------------------------------

func (e *Engine) emptyHandler(c *Character, a *Affect, flag byte) {}

// loseTurnHandler = O0A4:0274: the turn is over (befuddled, asleep, held).
func (e *Engine) loseTurnHandler(c *Character, a *Affect, flag byte) { e.endTurn(c) }

// saveAndHitUp = O0A4:0292.
func (e *Engine) saveAndHitUp(flag byte) {
	if flag == 0 {
		e.rv.save++
		e.rv.hitRoll++
	}
}

// blessHandler = O0A4:02A9.
func (e *Engine) blessHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 {
		e.rv.morale += 5
		e.rv.hitRoll++
	}
}

// curseHandler = O0A4:02C1.
func (e *Engine) curseHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	if e.rv.morale < 5 {
		e.rv.morale = 0
	} else {
		e.rv.morale -= 5
	}
	e.rv.hitRoll--
}

// snakesHandler = O0A4:02FF: the snakes eat the character's attacks.
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

// dispellingEvilHandler = O0A4:0394: enchanted attackers are hampered.
func (e *Engine) dispellingEvilHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 && e.selected().CreatureFlags&creatureEnchanted != 0 {
		e.rv.hitRoll -= 7
	}
}

// electricityImmuneHandler = O0A4:03B7.
func (e *Engine) electricityImmuneHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 && e.rv.dmgType&dmgElectricity != 0 {
		e.blockAffect(0)
	}
}

// gnomeVsFoesHandler = O0A4:03D6.
func (e *Engine) gnomeVsFoesHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	e.rv.target = targetOf(c)
	if e.rv.target.MonsterFlags&monsterGnomeFoe != 0 {
		e.rv.hitRoll++
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

// protFromEvilHandler = O0A4:0413: evil attackers.
func (e *Engine) protFromEvilHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 && e.selectedAligned(alignLawfulEvil, alignNeutralEvil, alignChaoticEvil) {
		e.rv.save += 2
		e.rv.hitRoll -= 2
	}
}

// protFromGoodHandler = O0A4:0450: good attackers.
func (e *Engine) protFromGoodHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 && e.selectedAligned(alignLawfulGood, alignNeutralGood, alignChaoticGood) {
		e.rv.save += 2
		e.rv.hitRoll -= 2
	}
}

// halveAndSave: damage of the type is halved and saves are better.
func (e *Engine) halveAndSave(flag, dmgType byte) {
	if flag == 0 && e.rv.dmgType&dmgType != 0 {
		e.rv.damage /= 2
		e.rv.save += 3
	}
}

// resistColdHandler = O0A4:048D.
func (e *Engine) resistColdHandler(c *Character, a *Affect, flag byte) {
	e.halveAndSave(flag, dmgCold)
}

// resistFireHandler = O0A4:06CA.
func (e *Engine) resistFireHandler(c *Character, a *Affect, flag byte) {
	e.halveAndSave(flag, dmgFire)
}

// Charm data bits: the charm taken hold, the side the charmed one came
// from, the side it now fights for.
const (
	charmTaken   = 32
	charmSideBit = 64
	charmNewSide = 128
	charmMorale  = 100
)

// charmHandler = O0A4:04B8: the character fights for the charmer; on
// removal back to its own side.
func (e *Engine) charmHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		c.Side = a.Data & charmSideBit / charmSideBit
		if c.Control == controlCharmed {
			c.Control = 0
			c.Auto = 0
		}
		return
	}
	if a.Data&charmTaken == 0 {
		a.Data = a.Data + charmTaken + c.Side*charmSideBit
		c.Side = a.Data / charmNewSide
		c.Auto = 1
		if c.Control <= controlHired {
			c.Control = controlCharmed
		}
		combatInfo(c).Target = nil
		e.countSides()
	}
	e.rv.morale = charmMorale
}

// barkskinHandler = O0A4:05AC.
func (e *Engine) barkskinHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	if e.rv.dmgType&dmgMagic == 0 {
		e.rv.save++
	}
	e.rv.hitRoll--
}

// Poison and wasting timers.
const (
	poisonDamageTime = 10
	weakenTime       = 60
	healingTime      = 60
)

// poisonDamageHandler = O0A4:05D5: a point of damage. What comes back
// later is affect 26 (the dwarf's bonus), not this one: the original's
// slip, kept.
func (e *Engine) poisonDamageHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 && e.addAffectNow(c, affectDwarfVsFoes, a.Data, poisonDamageTime) && c.HP > 1 {
		e.rv.dmgType = 0
		e.damagePerson(c, 1, saveNone, false)
		if e.gameState != stateCombat {
			e.playerSummary(e.selected())
		}
	}
}

// Armour values (60 - armour class).
const (
	acShield     = 57
	acCoughFloor = 50
)

// shieldHandler = O0A4:0665: armour class 3 at least, saves up; magic
// missiles do nothing.
func (e *Engine) shieldHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	if c.AC[0] < acShield {
		c.AC[0] = acShield
	}
	e.rv.save++
	if e.curSpell == spellMagicMissile {
		e.rv.damage = 0
	}
}

// resistCharmAndSleep: the chance in a hundred of shrugging off charm and
// sleep.
func (e *Engine) resistCharmAndSleep(flag byte, pct int) {
	if flag == 0 && e.Dice(1, 100) <= pct {
		e.blockAffect(affectCharmed)
		e.blockAffect(affectAsleep)
	}
}

// halfElfResistHandler = O0A4:069E: 30%.
func (e *Engine) halfElfResistHandler(c *Character, a *Affect, flag byte) {
	e.resistCharmAndSleep(flag, 30)
}

// elfResistHandler = O0A4:2F77: 90%.
func (e *Engine) elfResistHandler(c *Character, a *Affect, flag byte) {
	e.resistCharmAndSleep(flag, 90)
}

// noCharmSleepHandler = O0A4:2FA5.
func (e *Engine) noCharmSleepHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 {
		e.blockAffect(affectCharmed)
		e.blockAffect(affectAsleep)
	}
}

// silencedHandler = O0A4:06F5.
func (e *Engine) silencedHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	ci := combatInfo(c)
	if ci.CanUse != 0 {
		e.combatName(c, "is "+affectNameOf(affectSilenced), combatMsgRow, true)
	}
	ci.CanUse = 0
	ci.CanCast = 0
}

// slowPoisonHandler = O0A4:077A: when it ends, the poison kills.
func (e *Engine) slowPoisonHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	if e.FindAffect(c, affectPoisoned) != nil {
		e.killWithMessage(c, statusDead, "dies from poison")
		e.RemoveAffect(c, affectPoisoned, nil)
	}
	e.rv.cure = 1
	e.RemoveAffect(c, affectPoisonDamage, nil)
	e.rv.cure = 0
}

// The spiritual hammer item.
const (
	hammerType   = 6
	hammerWord1  = 6
	hammerWord2  = 121
	hammerPlus   = 1
	hammerFlags  = 128
	maxItemsHeld = 16
)

// spiritualHammerHandler = O0A4:080A: the hammer appears in the caster's
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
		hammer := &Item{itemRecord: itemRecord{Type: hammerType, Plus: hammerPlus}}
		hammer.Words[1] = hammerWord1
		hammer.Words[2] = hammerWord2
		hammer.Spells[1] = affectSpiritualHammer
		hammer.Spells[2] = hammerFlags
		added := e.AddItemCopy(c, hammer)
		if added != nil && c.Slots[0] == nil {
			saved := e.partySel
			e.partySel = e.partyIndex(c)
			e.readyItem(added)
			added.Readied = 1
			e.partySel = saved
		}
		e.combatName(c, "gains an item", combatMsgRow, true)
	}
	e.ApplyItems(c)
}

// unseenBy: the attacker cannot see the character, and hits it less.
func (e *Engine) unseenBy() {
	e.rv.unseen = 1
	e.rv.hitRoll -= 4
}

// invisibleHandler = O0A4:0A03: attackers who cannot see it are hampered.
func (e *Engine) invisibleHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 && e.FindAffect(e.selected(), affectDetectInvisibility) == nil {
		e.unseenBy()
	}
}

// dwarfVsFoesHandler = O0A4:0A38.
func (e *Engine) dwarfVsFoesHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	e.rv.target = targetOf(c)
	if e.rv.target.CreatureFlags&creatureDwarfFoe != 0 {
		e.rv.hitRoll++
	}
}

// Mirror image data: the images in the high nibble.
const (
	imageStep = 16
	imageLow  = 15
)

// mirrorImageHandler = O0A4:0A83: a blow or a single-target spell may
// strike an image instead.
func (e *Engine) mirrorImageHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 || e.Dice(1, int(a.Data/imageStep)+1) <= 1 {
		return
	}
	if e.curSpell > 0 && (e.rv.areaSpell != 0 || spellTableData[e.curSpell][spellImageField] == 0) {
		return
	}
	e.blockAffect(0)
	e.combatName(c, "lost an image", combatMsgRow, true)
	images := a.Data/imageStep - 1
	if images == 0 {
		e.RemoveAffect(c, affectMirrorImage, nil)
		return
	}
	a.Data = a.Data&imageLow | images*imageStep
}

// enfeebledHandler = O0A4:0B5D.
func (e *Engine) enfeebledHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 {
		e.rv.damage -= e.rv.damage / 4
	}
}

// coughingHandler = O0A4:0B8F: no spells or items, and worse armour.
func (e *Engine) coughingHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
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

// dispelEvilHandler = O0A4:0C8B: the enchanted foe is sent away unless it
// saves, and the spell is spent.
func (e *Engine) dispelEvilHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	t := targetOf(c)
	e.rv.target = t
	if t.CreatureFlags&creatureEnchanted != 0 && !e.SavingThrow(t, saveVsSpell, 0) {
		e.killWithMessage(t, statusDisintegrated, "is dispelled")
		e.RemoveAffect(c, affectDispellingEvil, nil)
		e.RemoveAffect(c, affectDispelEvil, nil)
		return
	}
	e.combatName(t, "resists dispel evil", combatMsgRow, true)
}

// blindHandler = O0A4:0D51.
func (e *Engine) blindHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	e.rv.hitRoll -= 4
	c.AC[0] -= 4
	c.AC[1] -= 4
	e.rv.save -= 4
}

// diseasedHandler = O0A4:0D82: weakness and wasting.
func (e *Engine) diseasedHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 {
		e.callAffect(affectWeakened, c, a, flag)
		e.callAffect(affectWasting, c, a, flag)
	}
}

// Times of the fits of confusion.
const (
	runAwayTime = 10
	berserkTime = 1
)

// confusedHandler = O0A4:0DED: each round the confused run, stand dazed,
// go berserk or rage; a saving throw may end it.
func (e *Engine) confusedHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	switch r := e.Dice(1, 100); {
	case r <= 10:
		e.RemoveAffect(c, affectConfused, nil)
		e.startFleeing(c)
		e.isUnaffected(c, affectFrightened, runAwayTime, 0, 1, 1, false, "runs away")
	case r <= 60:
		e.hitMessage(c, true, "is confused")
		e.clearCombatText()
		e.loseTurnHandler(c, a, 0)
	case r <= 80:
		e.isUnaffected(c, affectBerserk, berserkTime, c.Side, 1, 1, false, "goes berserk")
		e.callAffect(affectBerserk, c, nil, 0)
	default:
		e.hitMessage(c, true, "is enraged")
		e.clearCombatText()
	}
	if e.SavingThrow(c, saveVsSpell, -2) {
		e.RemoveAffect(c, affectConfused, nil)
	}
}

// startFleeing: the character turns to run, under the computer's control
// and with no target.
func (e *Engine) startFleeing(c *Character) {
	ci := combatInfo(c)
	ci.Turned = 1
	c.Auto = 1
	if c.Control <= controlHired {
		c.Control = controlCharmed
	}
	ci.Target = nil
}

// cursedHandler = O0A4:0FB9.
func (e *Engine) cursedHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 {
		e.rv.hitRoll -= 4
		e.rv.save -= 4
	}
}

// blinkingHandler = O0A4:0FD2: blinked away before its initiative came,
// it cannot be hit.
func (e *Engine) blinkingHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 && combatInfo(c).Delay == 0 {
		e.rv.unseen = 1
		e.rv.hitRoll = -1
	}
}

// Haste data: aged once.
const hasteAged = 16

// hasteHandler = O0A4:1002: a year older the first time, twice as fast.
func (e *Engine) hasteHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	if a.Data&hasteAged == 0 {
		a.Data += hasteAged
		e.combatName(c, "ages", combatMsgRow, true)
		c.Age++
		e.ageCheckDue = true
	}
	e.rv.moves *= 2
	e.rv.attacks *= 2
}

// protNormalMissilesHandler = O0A4:13E5: a missile without a plus misses.
func (e *Engine) protNormalMissilesHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	if w := e.readiedWeapon(e.selected()); w != nil && w.Plus == 0 {
		e.avoidDistantAttack(c, 100)
	}
}

// slowHandler = O0A4:1429.
func (e *Engine) slowHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 {
		e.rv.moves /= 2
		e.rv.attacks /= 2
	}
}

// Strength floor of weakness.
const weakestStrength = 3

// weakenHandler = O0A4:145F: strength drops while it lasts; at the
// bottom the victim retches.
func (e *Engine) weakenHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 || !e.addAffectNow(c, affectWeakened, a.Data, weakenTime) {
		return
	}
	if c.Stats[statStr].Value > weakestStrength {
		e.combatName(c, "is weakened", combatMsgRow, true)
		c.Stats[statStr].Value--
		return
	}
	e.AddAffect(c, affectNausea, 0, 255, 0)
}

// wastingHandler = O0A4:14F4: a point of damage while hit points last,
// then nausea.
func (e *Engine) wastingHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 || !e.addAffectNow(c, affectWasting, a.Data, poisonDamageTime) {
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
	e.AddAffect(c, affectNausea, 0, 255, 0)
}

// dwarfDodgesHandler = O0A4:159E.
func (e *Engine) dwarfDodgesHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 && e.selected().CreatureFlags&creatureGiantKin != 0 {
		e.rv.hitRoll -= 4
	}
}

// gnomeDodgesHandler = O0A4:15C3.
func (e *Engine) gnomeDodgesHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 && e.selected().CreatureFlags&creatureGnomeFoe != 0 {
		e.rv.hitRoll -= 4
	}
}

// Prayer data: the praying side.
const prayerSide = 16

// prayerHandler = O0A4:15E8: the praying side gains, the other loses.
func (e *Engine) prayerHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	if c.Side == a.Data&prayerSide/prayerSide {
		e.saveAndHitUp(flag)
		return
	}
	e.rv.hitRoll--
	e.rv.save--
}

// fireShieldHotHandler = O0A4:163D: cold is resisted, unsaved fire hurts
// double.
func (e *Engine) fireShieldHotHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	if e.rv.dmgType&dmgCold != 0 {
		e.rv.save += 2
		return
	}
	if e.rv.dmgType&dmgFire != 0 && !e.rv.saveMade {
		e.rv.damage *= 2
	}
}

// fireShieldColdHandler = O0A4:1678: fire is resisted, cold hurts double.
// The original tests the text-waiting flag where its twin tests the
// saving throw; kept.
func (e *Engine) fireShieldColdHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	if e.rv.dmgType&dmgFire != 0 {
		e.rv.save += 2
		return
	}
	if e.rv.dmgType&dmgCold != 0 && !e.saved {
		e.rv.damage *= 2
	}
}

// invisibleRingHandler = O0A4:16BC: invisible for the next round; in a
// fight taking the ring off ends it.
func (e *Engine) invisibleRingHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		if e.gameState == stateCombat {
			e.RemoveAffect(c, affectInvisible, nil)
		}
		return
	}
	e.AddAffect(c, affectInvisible, 1, 12, 0)
}

// Spell levels the globes stop.
const (
	globeMaxLevel      = 4
	minorGlobeMaxLevel = 3
)

// stopSpellsUpTo: spells of that level and below do not get through.
func (e *Engine) stopSpellsUpTo(flag byte, level byte) {
	if flag == 0 && e.curSpell > 0 && spellTableData[e.curSpell][spellLevelField] <= level {
		e.blockAffect(0)
	}
}

// globeHandler = O0A4:1708.
func (e *Engine) globeHandler(c *Character, a *Affect, flag byte) {
	e.stopSpellsUpTo(flag, globeMaxLevel)
}

// minorGlobeHandler = O0A4:1A2C.
func (e *Engine) minorGlobeHandler(c *Character, a *Affect, flag byte) {
	e.stopSpellsUpTo(flag, minorGlobeMaxLevel)
}

// Spells whose targeting the gaze and breath attacks borrow.
const (
	gazeAimSpell   = 61
	breathAimSpell = 68
)

// Missile looks.
const (
	gazeColours   = 18
	gazeFrames    = 4
	gazeDelay     = 45
	breathColours = 23
	breathFrames  = 1
	breathDelay   = 30
)

// aimWithSpell = the DS:8A8E target hook with a spell's targeting: whom the
// attack goes for, nil when there is no one.
func (e *Engine) aimWithSpell(spell int) *Character {
	if !e.spellTargetHook(spell, 1) {
		return nil
	}
	return e.spellTarget
}

// shootAt draws a missile of the colours from one combatant to another.
func (e *Engine) shootAt(from, to *Character, colours, frames, delay int) {
	e.hitColours(colours)
	e.flyMissile(e.combatX(from), e.combatY(from), e.combatX(to), e.combatY(to), frames, delay)
}

// opposingSide = O151:2AF8 (char): the side it fights against.
func opposingSide(c *Character) byte {
	if c.Side == 0 {
		return 1
	}
	return 0
}

// stoningGazeHandler = O0A4:1762: the one it looks at is turned to stone
// unless it saves; a gaze that can be reflected comes back from a
// mirror. The gaze goes off only at its own side's members: the
// original's test, kept.
func (e *Engine) stoningGazeHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	t := e.aimWithSpell(gazeAimSpell)
	if t == nil || t.Side == opposingSide(c) {
		return
	}
	e.rv.target = t
	e.combatName(c, "gazes...", combatMsgRow, false)
	e.shootAt(c, t, gazeColours, gazeFrames, gazeDelay)
	if e.FindAffect(c, affectGazeReflectable) != nil && e.FindAffect(t, affectMirror) != nil {
		e.combatName(t, "reflects it!", combatMsgRow+2, false)
		e.shootAt(t, c, gazeColours, gazeFrames, gazeDelay)
		e.rv.target = c
	}
	if !e.SavingThrow(e.rv.target, saveVsPetrify, 0) {
		e.killWithMessage(e.rv.target, statusStoned, "is turned to stone")
	}
}

// Weapon pluses that matter.
const plusNeeded = 3

// needsPlus3WeaponHandler = O0A4:1930.
func (e *Engine) needsPlus3WeaponHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	if w := e.readiedWeapon(e.selected()); w != nil && int8(w.Plus) >= plusNeeded {
		return
	}
	e.rv.damage = 0
}

// fireResistRingHandler = O0A4:1974: two less per die (at least one),
// saves up; fire that is not magic is stopped.
func (e *Engine) fireResistRingHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 || e.rv.dmgType&dmgFire == 0 {
		return
	}
	for die := byte(1); die <= e.rv.diceN; die++ {
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

// healingOverTimeHandler = O0A4:19D7: a hit point now and then.
func (e *Engine) healingOverTimeHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 && e.addAffectNow(c, affectHealingOverTime, a.Data, healingTime) && e.healPlayer(c, 1, true) {
		e.healedMessage(c)
	}
}

// poisonStingHandler = O0A4:1B49.
func (e *Engine) poisonStingHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 {
		e.poisonAttack(c, 0)
	}
}

// poisonStingMinus2Handler = O0A4:2B56.
func (e *Engine) poisonStingMinus2Handler(c *Character, a *Affect, flag byte) {
	if flag == 0 {
		e.poisonAttack(c, -2)
	}
}

// paralysingTouchHandler = O0A4:2B73.
func (e *Engine) paralysingTouchHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 {
		e.paralyseTarget(c, 100, 0)
	}
}

// stoningTouchHandler = O0A4:1B79.
func (e *Engine) stoningTouchHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	t := targetOf(c)
	e.rv.target = t
	if !e.SavingThrow(t, saveVsPetrify, 0) {
		e.killWithMessage(t, statusStoned, "is turned to stone")
	}
}

// Swallowing.
const (
	swallowRoll   = 75
	swallowFrames = 9
	swallowDelay  = 10
	belchChance   = 50
)

// swallowsWholeHandler = O0A4:1C26: a very good roll swallows the target;
// it flickers out and its body lies where the swallower stands.
func (e *Engine) swallowsWholeHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 || e.rv.hitRoll < swallowRoll {
		return
	}
	t := targetOf(c)
	e.rv.target = t
	e.combatName(t, "is swallowed whole", combatMsgRow, true)
	e.pauseForMessage()
	t.Status = statusDead
	t.Okay = 0
	t.HP = 0
	e.downAffects(t)
	idx := e.combatIndex(t)
	if !e.combatOnScreen(t, true) {
		e.combatCentre(e.combatX(t), e.combatY(t), 3, 8)
	}
	e.combatClearIcon(0, 0, idx)
	e.PlaySound(sndFall)
	for frame := 0; frame < swallowFrames; frame++ {
		for part := 0; part <= 3; part++ {
			dx, dy, ok := e.footprint(int(e.cpos[idx].size), part)
			sx, sy := int(e.combatScreenX[idx])+dx, int(e.combatScreenY[idx])+dy
			if ok && e.combatInView(sx, sy) {
				e.drawPage(sx*3, sy*3, e.icons[iconDeath][frame%2], 0, true, true)
			}
		}
		e.Flip()
		e.Delay(swallowDelay)
	}
	e.combatName(t, "is killed", combatMsgRow, false)
	e.combatClearIcon(0, 0, idx)
	if combatInfo(t).Extra == 0 && e.nbodies+1 < len(e.bodies) {
		x, y := e.combatX(c), e.combatY(c)
		e.nbodies++
		b := &e.bodies[e.nbodies]
		b.ground = e.cmap.ground[y][x]
		if b.ground != groundStinkingCloud {
			if c.Status == statusStoned {
				e.cmap.ground[y][x] = groundStoneBody
			} else {
				e.cmap.ground[y][x] = groundBody
			}
		}
		b.c, b.x, b.y = t, byte(x), byte(y)
	}
	if e.Dice(1, 100) <= belchChance {
		e.combatName(c, "belches in satisfaction", combatMsgRow, true)
	} else {
		e.combatName(t, "is digested instantly", combatMsgRow, true)
	}
	e.combatClearIcon(0, 0, idx)
	e.cpos[idx].size = 0
	e.combatRedrawMap()
	e.combatCentre(e.cmap.viewX+3, e.cmap.viewY+3, 3, 8)
	ci := combatInfo(t)
	ci.Delay, ci.Moves, ci.Spell, ci.Guarding = 0, 0, 0, 0
	e.pauseForMessage()
	e.clearCombatText()
}

// breathFireCloseHandler = O0A4:2094: once the fight is old enough, fire
// at a foe close by: 8 points, half on a save.
func (e *Engine) breathFireCloseHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 || e.combatRound == 0 || c.BaseAttack[0] < e.combatRound {
		return
	}
	e.rv.dmgType = dmgFire | dmgBreath
	t := targetOf(c)
	e.rv.target = t
	if e.rangeTo(c, t) >= 2 {
		return
	}
	e.combatName(c, "Breathes Fire", combatMsgRow, true)
	e.shootAt(c, t, breathColours, breathFrames, breathDelay)
	e.damagePerson(t, 8, saveHalf, e.SavingThrow(t, saveVsBreath, 0))
}

// Feeblemind's intelligence and wisdom.
const feebleScore = 3

// feeblemindHandler = O0A4:21BC.
func (e *Engine) feeblemindHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	c.Stats[statInt].Value = feebleScore
	c.Stats[statWis].Value = feebleScore
	e.combatName(c, "is made stupid", combatMsgRow, true)
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

// invisibleToAnimalsHandler = O0A4:2284. The original looks at the
// invisible one's own kind, not the attacker's; kept.
func (e *Engine) invisibleToAnimalsHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 && c.CreatureFlags&creatureAnimalsSeen != 0 && e.FindAffect(c, affectDetectInvisibility) == nil {
		e.unseenBy()
	}
}

// confusingGazeHandler = O0A4:22E4: a foe close by is confused for a few
// rounds unless it saves.
func (e *Engine) confusingGazeHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	t := e.aimWithSpell(breathAimSpell)
	if t == nil || e.rangeTo(c, t) >= 2 {
		return
	}
	e.rv.target = t
	e.combatName(c, "gazes...", combatMsgRow, false)
	e.shootAt(c, t, gazeColours, gazeFrames, gazeDelay)
	if !e.SavingThrow(t, saveVsSpell, 0) {
		e.isUnaffected(t, affectConfused, e.Dice(1, 10)+2, c.Side, 1, 1, false, "stares blankly")
	}
}

// highlightedHandler = O0A4:240F: easier to hit.
func (e *Engine) highlightedHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 {
		c.AC[0] -= 2
		c.AC[1] -= 2
	}
}

// dragonBreathImmuneHandler = O0A4:2444.
func (e *Engine) dragonBreathImmuneHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 && e.rv.dmgType&dmgBreath != 0 {
		e.blockAffect(0)
	}
}

// magicalBootsHandler = O0A4:2465: harder to hit.
func (e *Engine) magicalBootsHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 {
		c.AC[0] += 2
		c.AC[1] += 2
	}
}

// baneWeaponHandler = O0A4:2497: 6d6 and the strength bonus against its
// foes.
func (e *Engine) baneWeaponHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	t := targetOf(c)
	e.rv.target = t
	if t.MonsterFlags&monsterBane != 0 {
		e.rv.damage = byte(e.RollDice(6, 6) + int(strDamage(c)))
		e.rv.hitRoll++
	}
}

// berserkHandler = O0A4:250D: attacks the nearest one, of either side.
func (e *Engine) berserkHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		if c.Control == controlCharmed {
			c.Control = 0
			c.Auto = 0
		}
		c.Side = 0
		return
	}
	e.goBerserk(c)
	if e.gameState == stateCombat {
		combatInfo(c).CanCast = 0
		c.Side = targetOf(c).Side ^ 1
		e.combatName(c, "goes berserk", combatMsgRow, true)
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

// magicalStoneHandler = O0A4:2674.
func (e *Engine) magicalStoneHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 {
		e.rv.save++
	}
}

// fireHealsHandler = O0A4:2689: magical fire heals, electricity slows,
// nothing else gets through.
func (e *Engine) fireHealsHandler(c *Character, a *Affect, flag byte) {
	switch {
	case e.rv.dmgType&dmgFire != 0 && e.rv.dmgType&dmgMagic != 0:
		e.combatName(c, "is Healed", combatMsgRow, true)
		hp := int(c.HP) + int(e.rv.damage)
		if hp > int(c.MaxHP) {
			hp = int(c.MaxHP)
		}
		c.HP = byte(hp)
	case e.rv.dmgType&dmgElectricity != 0:
		e.AddAffect(c, affectSlow, 3, 255, 0)
		e.combatName(c, "is "+affectNameOf(affectSlow), combatMsgRow, true)
	}
	e.blockAffect(0)
}

// breathFireHandler = O0A4:278C: fire at a foe in reach, 7 points, half
// on a save.
func (e *Engine) breathFireHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	e.rv.dmgType = dmgFire | dmgBreath
	t := e.aimWithSpell(breathAimSpell)
	if t == nil {
		return
	}
	e.rv.target = t
	e.combatTurn(c, byte(e.directionTo(c, t)), 1, false)
	e.combatName(c, "Breathes Fire", combatMsgRow, true)
	e.shootAt(c, t, breathColours, breathFrames, breathDelay)
	e.damagePerson(t, 7, saveHalf, e.SavingThrow(t, saveVsBreath, 0))
}

// Acid spit: the nearer, the likelier.
const spitReach = 11

// spitAcidHandler = O0A4:28BC: 4d8 acid, half on a save; in the first
// round only one time in ten.
func (e *Engine) spitAcidHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	e.rv.dmgType = dmgAcid
	t := e.aimWithSpell(gazeAimSpell)
	if t == nil {
		return
	}
	e.rv.target = t
	chance := 0
	if r := e.rangeTo(c, t); r <= spitReach {
		if e.combatRound == 0 {
			chance = 10
		} else {
			chance = (spitReach - r) * 10
		}
	}
	if chance <= 0 {
		return
	}
	if e.Dice(1, 100) <= chance {
		e.combatName(c, "Spits...", combatMsgRow, true)
		e.shootAt(c, t, breathColours, breathFrames, breathDelay)
		e.damagePerson(t, e.RollDice(4, 8), saveHalf, e.SavingThrow(t, saveVsBreath, 0))
		return
	}
	e.combatName(c, "Spits and misses", combatMsgRow, true)
}

// The fear aura spares those of this level and above.
const fearLevel = 5

// fearAuraHandler = O0A4:2A36: foes below the level who are not fearless
// run in terror.
func (e *Engine) fearAuraHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	for _, foe := range e.party {
		if foe.Side == c.Side || e.FindAffect(foe, affectFearless) != nil ||
			e.FindAffect(foe, affectFrightened) != nil || int8(foe.MaxLevel) >= fearLevel {
			continue
		}
		e.isUnaffected(foe, affectFrightened, 0, 255, 1, 1, false, "is terrified")
		if e.FindAffect(foe, affectFrightened) != nil {
			e.startFleeing(foe)
		}
	}
}

// Magical cloak data.
const (
	cloakUsed = 16
	cloakKeep = 15
)

// magicalCloakHandler = O0A4:2B94: the first blow each round misses.
func (e *Engine) magicalCloakHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	if e.combatRound == 0 && e.rv.hitRoll == 0 {
		a.Data &= cloakKeep
		return
	}
	if a.Data&cloakUsed == 0 {
		e.rv.hitRoll = -1
		a.Data |= cloakUsed
	}
}

// Base item flags (itemFlags).
const (
	itemUsesArrows = 1
	itemUsesBolts  = 128
)

// boltsHarmlessHandler = O0A4:2BE7.
func (e *Engine) boltsHarmlessHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	if w := e.readiedWeapon(e.selected()); w != nil && e.baseItem(w.Type, itemFlags)&itemUsesBolts != 0 {
		e.rv.damage = 0
	}
}

// fearlessHandler = O0A4:2EB6.
func (e *Engine) fearlessHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 {
		e.blockAffect(affectFrightened)
	}
}

// fireImmuneHandler = O0A4:2ECE.
func (e *Engine) fireImmuneHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 && e.rv.dmgType&dmgFire != 0 {
		e.blockAffect(0)
	}
}

// coldImmuneHandler = O0A4:3045.
func (e *Engine) coldImmuneHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 && e.rv.dmgType&dmgCold != 0 {
		e.blockAffect(0)
	}
}

// constitutionSaveBonus: the hardy races' bonus, a point for each 3.5
// points of constitution. Below 4 the original adds an unset local; the
// port adds nothing.
func constitutionSaveBonus(c *Character) byte {
	switch con := c.Stats[statCon].Value; {
	case con >= 18:
		return 5
	case con >= 14:
		return 4
	case con >= 11:
		return 3
	case con >= 7:
		return 2
	case con >= 4:
		return 1
	}
	return 0
}

// hardyAllHandler = O0A4:2EEF: against paralysis and poison, rods and
// spells.
func (e *Engine) hardyAllHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	if t := e.rv.saveType; t == saveVsSpell || t == saveVsRod || t == saveVsParalysis {
		e.rv.save += constitutionSaveBonus(c)
	}
}

// conSaveHandler = O0A4:2FC4: against rods and spells.
func (e *Engine) conSaveHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	if t := e.rv.saveType; t == saveVsSpell || t == saveVsRod {
		e.rv.save += constitutionSaveBonus(c)
	}
}

// A saving throw nothing fails.
const certainSave = 100

// poisonImmuneHandler = O0A4:3066.
func (e *Engine) poisonImmuneHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	e.blockAffect(affectPoisoned)
	e.blockAffect(affectHeld)
	if e.rv.saveType == saveVsParalysis {
		e.rv.save = certainSave
	}
}

// The level below which a monster's plain weapon cannot hurt.
const plainWeaponLevel = 4

// plainWeaponsImmuneHandler = O0A4:3553: a weapon without a plus does
// nothing in the hands of a player race or a low-level monster.
func (e *Engine) plainWeaponsImmuneHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	sel := e.selected()
	if w := e.readiedWeapon(sel); w != nil && w.Plus != 0 {
		return
	}
	race := int8(sel.Race)
	if race > raceTribble && race < raceMonster || int8(sel.MaxLevel) < plainWeaponLevel {
		e.rv.damage = 0
	}
}

// rangerVsGiantsHandler = O0A4:35B2: a point per ranger level against
// giant-class foes close by.
func (e *Engine) rangerVsGiantsHandler(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	t := targetOf(c)
	e.rv.target = t
	if e.rangeTo(c, t) > 1 {
		return
	}
	if t.CreatureFlags&creatureRangerFoe != 0 {
		e.rv.damage += c.Levels[clsRanger]
	}
}

// berserkFoeHandler = O0A4:3637: attacks the nearest; afterwards back to
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
	c.Side = targetOf(c).Side ^ 1
}

// alwaysInvisibleHandler = O0A4:3757.
func (e *Engine) alwaysInvisibleHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 {
		e.AddAffect(c, affectInvisible, 255, 255, 0)
	}
}

// Paladin cures: one a week per five levels.
const paladinCureLevels = 5

// paladinCuresHandler = O0A4:3784: when the week ends, the cures are back.
func (e *Engine) paladinCuresHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 {
		return
	}
	usable := 0
	if oldClassUsable(c) {
		usable = 1
	}
	level := int(int8(c.Levels[clsPaladin])) + int(int8(c.OldLevels[clsPaladin]))*usable
	c.PaladinCures = byte((level-1)/paladinCureLevels + 1)
}

// frightenedHandler = O0A4:37D5: when the fear goes, so does the flight.
func (e *Engine) frightenedHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 {
		return
	}
	if c.Control == controlCharmed {
		c.Control = 0
		c.Auto = 0
	}
	combatInfo(c).Turned = 0
}

// zapsHandler = O0A4:3826: a close attacker takes double the damage it
// dealt, as magic.
func (e *Engine) zapsHandler(c *Character, a *Affect, flag byte) {
	sel := e.selected()
	if flag != 0 || e.rangeTo(sel, c) >= 2 {
		return
	}
	dealt := e.rv.damage
	e.rv.damage *= 2
	if e.rv.damage == 0 {
		return
	}
	dmgType := e.rv.dmgType
	e.rv.dmgType = dmgMagic
	e.combatName(sel, "gets zapped", combatMsgRow, true)
	e.damagePerson(sel, int(e.rv.damage), saveNone, false)
	e.rv.damage, e.rv.dmgType = dealt, dmgType
}

// giantStrengthHandler = O0A4:38BD.
func (e *Engine) giantStrengthHandler(c *Character, a *Affect, flag byte) {
	if flag == 0 {
		e.combatName(c, "gains "+affectNameOf(affectGiantStrength), combatMsgRow, true)
	}
}

// forgetSpellOf = O151:149F (char, spell): its first memorised copy goes.
func forgetSpellOf(c *Character, sp byte) {
	for i := range c.Memorized {
		if c.Memorized[i] == sp {
			c.Memorized[i] = 0
			return
		}
	}
}

// Breaths a fight allows (the affect's data counts them down).
const breathsPerCombat = 3

// The cone a breath fills.
const (
	breathConeWidth  = 3
	coldConeLength   = 7
	fireConeLength   = 9
	breathConeColour = 18
)

// breathWeaponHandler = O113:61AE (affects 83 and 104): a cone of cold or
// fire at the chosen square, as long as it catches only foes; each takes
// the breather's full hit points, half on a save. Three breaths a fight.
func (e *Engine) breathWeaponHandler(c *Character, a *Affect, flag byte) {
	if e.combatRound == 0 || a.Data > breathsPerCombat {
		a.Data = breathsPerCombat
	}
	if a.Data == 0 {
		return
	}
	length, what := coldConeLength, "cold"
	e.rv.dmgType = dmgCold | dmgBreath
	if a.Type == affectBreathWeapon2 {
		length, what = fireConeLength, "Fire"
		e.rv.dmgType = dmgFire | dmgBreath
	}
	x, y := e.combatX(c), e.combatY(c)
	if e.aimWithSpell(gazeAimSpell) == nil {
		return
	}
	e.coneTargets(x, y, e.tgtX, e.tgtY, breathConeWidth, length)
	for i := 1; i <= e.nSpellTargets; i++ {
		if t := e.tgt(i); t != nil && t.Side != opposingSide(c) {
			return
		}
	}
	if e.nSpellTargets == 0 {
		return
	}
	e.combatName(c, "breathes "+what, combatMsgRow, true)
	dir := e.directionTo(c, e.spellTarget)
	e.combatPanel(c)
	e.combatTurn(c, byte(dir), 1, false)
	e.shootAt(c, e.spellTarget, breathConeColour, breathFrames, breathDelay)
	for i := 1; i <= e.nSpellTargets; i++ {
		if t := e.tgt(i); t != nil {
			e.damagePerson(t, int(c.MaxHP), saveHalf, e.SavingThrow(t, saveVsBreath, 0))
		}
	}
	a.Data--
	e.endTurn(c)
}
