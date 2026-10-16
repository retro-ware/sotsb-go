package engine

// The DAMAGE command (O03C:2958) and its helpers: the trap's hit roll
// (O0E3:11DD), damage with its message (O04C:2200) and damage_player
// (O0ED:23D5).

// DAMAGE's first operand: the count of random targets, or flags.
const (
	damageFlagged   = 128 // the flags below are used
	damageAll       = 64  // everyone (else: one member)
	damageAnyway    = 32  // no saving throw
	damageSaveHurts = 16  // a made save still hurts
	damageBonusMask = 31  // the save bonus
	saveSelected    = 128 // in the save operand: the selected member saves
	saveTypeMask    = 7
	deathMargin     = 10
)

// trapHits = O0E3:11DD (target, bonus): a d20 (1 misses, 20 counts as 100)
// with the defender's affects, plus the bonus, against the AC.
func (e *Engine) trapHits(c *Character, bonus byte) bool {
	e.rv.hitRoll = int8(e.Dice(1, 20))
	if e.rv.hitRoll <= 1 {
		return false
	}
	if e.rv.hitRoll == 20 {
		e.rv.hitRoll = 100
	}
	e.affectGroup(c, grpDefender)
	return e.rv.hitRoll >= 0 && int(e.rv.hitRoll)+int(bonus) > int(c.AC[0])
}

// damageWithMessage = O04C:2200 (damage, target): "is hit FOR n points of
// Damage." (or "dies." when it kills) in the text window, then the damage.
func (e *Engine) damageWithMessage(dmg byte, c *Character) {
	if c.Status == statusDead {
		return
	}
	var msg string
	if int(c.HP)+deathMargin < int(dmg) {
		msg = "  " + c.Name() + " dies. "
	} else {
		msg = "  " + c.Name() + " is hit FOR " + itoa(int(dmg)) + " points of Damage."
	}
	clear := false
	if e.msgRow > textBottom {
		e.msgRow = textTop
		clear = true
		e.displayAndWait(colourWhite, colourBlack, "press <enter>/<return> to continue")
	}
	e.msgCol = textRight
	e.DisplayTextBlock(textLeft, textTop, textRight, textBottom, colourWhite, clear, msg)
	e.damagePlayer(c, dmg)
	e.ClearCells(summaryLeftCol, 1, summaryRight, statusLineRow)
	e.playerSummary(e.selected())
}

// Status values damagePlayer sets.
const (
	statusOkay        = 0
	statusAnimated    = 1
	statusRunning     = 3
	statusUnconscious = 4
	statusDying       = 5
	statusDead        = 6
)

// damagePlayer = O0ED:23D5 damage_player (target, damage): hit points down
// to 0, then unconscious, dying (bleeding rounds) or dead at 10 below.
func (e *Engine) damagePlayer(c *Character, dmg byte) {
	var over, left byte
	if c.HP >= dmg {
		left = c.HP - dmg
	} else {
		over = dmg - c.HP
	}
	switch {
	case over > 9 || left == 0 && c.Status == statusAnimated:
		c.Status = statusDead
	case over > 0:
		c.Status = statusDying
		if e.gameState == stateCombat && c.Combat != nil {
			c.Combat.Dying = over
		}
	case left == 0:
		c.Status = statusUnconscious
	}
	if c.Status == statusOkay || c.Status == statusAnimated {
		c.HP = left
		return
	}
	c.Okay = 0
	c.HP = 0
	if e.gameState == stateCombat {
		e.rv.side[int8(c.Side)&1]--
		if c.Combat != nil {
			c.Combat.Delay = 0
		}
	}
}

// cmdDamage = O03C:2958 (targets, dice, sides, bonus, save): a trap or
// spell hurts party members.
func (e *Engine) cmdDamage() {
	saved := e.partySel
	e.readArgs(5)
	targets := byte(e.arg(1))
	dice := byte(e.arg(2))
	sides := byte(e.arg(3))
	bonus := byte(e.arg(4))
	save := byte(e.arg(5))
	roll := func() byte { return byte(int(bonus) + e.Dice(int(dice), int(sides))) }
	dmg := roll()
	saveHurts := targets&damageSaveHurts != 0
	all := targets&damageAll != 0
	var who byte
	if !all {
		who = byte(e.Dice(1, int(wordAt(e.game, 2*cvPartyCount))))
	}
	member := func(n byte) *Character { // the n-th (1-based) of the list
		if int(n) >= 1 && int(n) <= len(e.party) {
			return e.party[n-1]
		}
		return e.party[len(e.party)-1]
	}
	if targets&damageFlagged == 0 {
		for k := byte(1); k <= targets; k++ {
			c := member(byte(e.Dice(1, int(wordAt(e.game, 2*cvPartyCount)))))
			if e.trapHits(c, save) {
				e.damageWithMessage(dmg, c)
			}
			dmg = roll()
		}
	} else {
		saveBonus := targets & damageBonusMask
		saveType := save & saveTypeMask
		switch {
		case !all && save&saveSelected == 0:
			c := member(who)
			if !e.SavingThrow(c, int(saveType), int(saveBonus)) || saveHurts {
				e.damageWithMessage(dmg, c)
			}
		case !all:
			c := e.selected()
			if saveType != 0 && e.SavingThrow(c, int(saveType)-1, int(saveBonus)) {
				if saveHurts {
					e.damageWithMessage(dmg, c)
				}
			} else {
				e.damageWithMessage(dmg, c)
			}
		default:
			for _, c := range append([]*Character(nil), e.party...) {
				if targets&damageAnyway == 0 {
					if !e.SavingThrow(c, int(saveType), int(saveBonus)) || saveHurts {
						e.damageWithMessage(dmg, c)
					}
				} else {
					e.damageWithMessage(dmg, c)
				}
			}
		}
	}
	e.partyDead = true
	for _, c := range e.party {
		if c.Okay != 0 {
			e.partyDead = false
		}
	}
	if e.partyDead {
		e.drawScreenFrame()
		e.msgCol, e.msgRow = 2, 2
		e.DisplayTextBlock(1, 1, textRight, textBottom, colourGreen, true, "The entire party is killed!")
		e.Delay(partyKilledDelay)
	}
	e.partySel = saved
	e.displayAndWait(colourWhite, colourBlack, "press <enter>/<return> to continue")
}

const partyKilledDelay = 3000
