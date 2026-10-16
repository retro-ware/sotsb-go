package engine

// The combat main loop and the player's command menus (O089).

// Combat = O089:011D: set up the battle, then rounds until one side is
// down (or the party stops fighting). Each round everyone acts in the
// order of their initiative.
func (e *Engine) Combat() {
	e.gameState = stateCombat
	e.ext.panel = nil
	e.combatTargets = true // DS:8A8E = O0C9:1FA1
	e.combatSetup()
	done := e.rv.side[0] == 0 || e.rv.side[1] == 0
	for !done {
		e.countSides()
		for _, c := range e.party {
			e.turnSetup(c)
		}
		putWord(e.game, area2Surprise, 0) // the surprise lasts one round
		e.buildTurnOrder()
		for e.turnIndex = 0; e.turnIndex < len(e.turnOrder); e.turnIndex++ {
			e.takeTurn(e.turnOrder[e.turnIndex])
		}
		done = e.endRound()
	}
	if e.ageCheckDue {
		e.oldAgeDeaths()
	}
	e.combatEnd()
}

// maxTurns is the length of the turn order (DS:8E2A, 72 entries).
const maxTurns = 72

// combatEnd = O089:0000: free the clouds and the map; a charmed party
// member stays with the enemy if any are left, and the affects a fall
// ends go.
func (e *Engine) combatEnd() {
	e.stinkingClouds, e.poisonClouds, e.delayedBlasts = nil, nil, nil
	for _, c := range e.party {
		if e.FindAffect(c, affectCharmed) != nil && c.Status == statusOkay {
			if e.rv.side[1] > 1 {
				c.Status = statusRunning
			} else {
				c.Status = statusOkay
			}
		}
		e.downAffects(c)
	}
	e.cmap = nil
	e.spr = nil
	e.combatTargets = false // DS:8A8E = O113:1253 cast_spell_on
}

// buildTurnOrder = O089:0741: everyone sorted by initiative (combat block
// +3, signed), ties settled by a coin toss as each one is put in.
func (e *Engine) buildTurnOrder() {
	e.turnOrder = e.turnOrder[:0]
	for _, c := range e.party {
		ini := int8(c.Combat.Delay)
		at := 0
		for at < len(e.turnOrder) {
			other := int8(e.turnOrder[at].Combat.Delay)
			if ini > other || ini == other && e.Dice(1, 2) == 1 {
				break
			}
			at++
		}
		if at >= maxTurns {
			continue
		}
		e.turnOrder = append(e.turnOrder[:at], append([]*Character{c}, e.turnOrder[at:]...)...)
		if len(e.turnOrder) > maxTurns {
			e.turnOrder = e.turnOrder[:maxTurns]
		}
	}
}

// delayTurn: c (at the current place in the turn order) moves to the end
// of the order and acts again there, last.
func (e *Engine) delayTurn(c *Character) {
	e.requeueTurn(c, func(*Character) bool { return true })
}

// requeueTurn takes c from the current place in the turn order and puts it
// back after the following turns for which passes holds, then steps back
// so the turn now at the current place is taken next.
func (e *Engine) requeueTurn(c *Character, passes func(next *Character) bool) {
	i := e.turnIndex
	for i+1 < len(e.turnOrder) && passes(e.turnOrder[i+1]) {
		e.turnOrder[i] = e.turnOrder[i+1]
		i++
	}
	e.turnOrder[i] = c
	e.turnIndex--
}

// oldAgeLimits are the ages from which a character of each race may die
// of old age after a battle (O089:0387); ten years before it they look
// very old.
var oldAgeLimits = [...]uint16{
	raceElf:      1600,
	raceHalfElf:  325,
	raceDwarf:    525,
	raceGnome:    750,
	raceHalfling: 199,
	raceHuman:    120,
}

// oldAgeWarning is how many years before the limit a character looks old.
const oldAgeWarning = 10

// oldAgeDeaths = O089:0387: after a battle in which someone aged, those
// past the age limit of their race die, and the others mourn them.
func (e *Engine) oldAgeDeaths() {
	const textTop, textLeft, textRight, textBottom = 14, 23, 38, 21
	wait := func() { e.displayAndWait(colourLightMagenta, colourBlack, "press <enter>/<return> to continue") }
	for _, c := range e.party {
		if c.Status >= statusDead || int(c.Race) >= len(oldAgeLimits) || oldAgeLimits[c.Race] == 0 {
			continue
		}
		limit := oldAgeLimits[c.Race]
		if c.Age <= limit {
			if c.Age > limit-oldAgeWarning {
				e.combatName(c, "is looking very old", combatMsgRow, true)
			}
			continue
		}
		say := func(colour int, text string) {
			e.ClearCells(textLeft, textTop, textRight, textBottom)
			e.DisplayTextBlock(textLeft, textTop, textRight, textBottom, colour, false, text)
		}
		say(colourWhite, "The strain of this last battle has worn heavily on "+c.Name()+".  ")
		wait()
		pronoun := "He"
		if c.Sex != 0 {
			pronoun = "She"
		}
		say(colourWhite, pronoun+" breathes deeply one last time and crumples to the ground.")
		wait()
		e.ClearCells(textLeft, textTop, textRight, textBottom)
		e.killWithMessage(c, statusDisintegrated, "dies from age")
		mourners := 0
		var mourner *Character
		for _, m := range e.party {
			if m.Okay == 1 && m.Status == statusOkay && m.Side != 1 {
				mourners++
				mourner = m
			}
		}
		var text string
		switch mourners {
		case 0:
			text = "Sadly, no party member witnesses the passing of the champion."
		case 1:
			text = mourner.Name() + " weeps at the loss of such a venerable companion."
		default:
			text = " The remaining party members are overcome with grief as they search " + c.Name() + "'s body for valuables."
		}
		say(colourYellow, text)
		wait()
	}
	e.ageCheckDue = false
}

// takeTurn = O089:089E (char).
func (e *Engine) takeTurn(c *Character) {
	ci := c.Combat
	ci.Attackers, ci.Flank, ci.Guarding = 0, 0, 0
	e.affectGroup(c, 7)
	if int8(ci.Delay) <= 0 {
		return // no cursor erase when the turn is skipped
	}
	if ci.Delay == 0x14 {
		ci.Delay = 0x13
	}
	e.selectChar(c)
	e.showMoves = c.Side == sideParty || e.combatOnScreen(c, false)
	e.combatRedrawChar(c, 2, 1)
	e.ApplyItems(c)
	e.panelDirty = true
	e.combatPanel(c)
	e.affectGroup(c, 0xF)
	if ci.Spell == 0 {
		e.affectGroup(c, 0x15)
	}
	if int8(ci.Delay) > 0 {
		if c.Auto != 0 {
			e.computerTurn(c)
		} else {
			e.playerTurn(c)
		}
	}
	e.eraseCursor(e.combatX(c), e.combatY(c))
}

// selectChar makes c the selected member (DS:5D92).
func (e *Engine) selectChar(c *Character) {
	for i, p := range e.party {
		if p == c {
			e.partySel = i
		}
	}
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Combat menu keys: the menu letters, and the extended keys (scan codes).
const (
	keyAltQ       = 16 // Alt+Q: the computer fights for the whole party
	keyAltA       = 30 // Alt+A: the same as Alt+Q
	keyAltX       = 45 // Alt+X: god_intervene
	keyAlt2       = 50 // toggles whether the computer casts for the party
	keyAltT       = 19
	quickTurnWait = 200
)

// playerTurn = O089:0A05 camp_menu (char): a spell being cast goes off,
// otherwise the combat menu until the turn is used.
func (e *Engine) playerTurn(c *Character) {
	if c.Okay == 0 {
		e.endTurn(c)
		return
	}
	ci := c.Combat
	if ci.Spell > 0 {
		s := ci.Spell
		ci.Spell = 0
		var quit bool
		e.CastSpell(s, 0, 1, &quit)
		e.endTurn(c)
		return
	}
	done := false
	for !done {
		key, ext := e.combatMenu(c)
		if !ext {
			switch key {
			case 'Q':
				e.quickOne(c)
				e.ClearRow24()
				e.ClearKeyboard()
				e.Delay(quickTurnWait)
				done = true
				e.computerTurn(c)
			case 'M':
				done = e.moveAttack(c, ' ')
			case 'V':
				done = e.viewPlayer() // ItemsMenu sets it
				e.setSpeed(c)
				if !done {
					e.combatUseHook()
				}
			case 'A':
				var aim aimTarget
				e.aimAt(c, 0xFF, 1, 0, 1, &done, &aim)
			case 'U':
				e.menuSel = 2
				done = e.ItemsMenu()
				e.setSpeed(c)
				if !done {
					e.combatUseHook()
				}
			case 'C':
				e.spellMenuTargets(0, 0, &done)
			case 'Y': // not on the menu: end the turn
				done = e.endTurn(c)
			case 'D':
				done = e.otherMenu(c)
			case ' ':
				for _, p := range e.party {
					if p.Control < controlHired+1 {
						p.Auto = 0
					}
				}
			}
		} else {
			switch key {
			case keyAlt2:
				e.computerMagic = !e.computerMagic
				if e.computerMagic {
					e.printMessage("Magic On")
				} else {
					e.printMessage("Magic Off")
				}
			case keyAltQ, keyAltA:
				e.turnIndex-- // this turn is taken again, by the computer
				ci.Delay = 20
				for _, p := range e.party {
					e.quickOne(p)
				}
				e.ClearRow24()
				e.Delay(quickTurnWait)
				done = true
			case keyAltX:
				if e.godIntervene() {
					e.combatRedrawChar(c, 3, 0)
					done = true
				} else {
					e.printMessage("That doesn't work")
				}
			}
		}
		if !done {
			e.combatRedrawChar(c, 2, 1)
			e.panelDirty = true
			e.combatPanel(c)
		}
	}
}

// combatMenu = O089:0D63 combat_menu (char, var key): the main combat menu.
func (e *Engine) combatMenu(c *Character) (byte, bool) {
	ci := c.Combat
	m := ""
	if ci.Moves > 0 {
		m += "Move "
	}
	m += "View Aim "
	if c.ItemCount > 0 {
		m += "Use "
	}
	spells := false
	for _, sp := range c.Memorized {
		if sp > 0 {
			spells = true
		}
	}
	if spells && ci.CanCast != 0 && wordAt(e.area, 2*areaWordNoMagic) == 0 {
		m += "Cast "
	}
	m += "Quick Done"
	if e.debug {
		m += "*"
	}
	for {
		key, ext := e.DisplayInput("", m, 0xD, 0xA, 0xF, true, false)
		if ext && !combatExtraKeys[key] {
			key = 0
		}
		if combatMenuKeys[key] {
			e.ClearRow24()
			return key, ext
		}
	}
}

// combatExtraKeys are the extended keys combat_menu keeps: Alt+Q, Alt+A,
// Alt+T, Alt+X and Alt+2.
var combatExtraKeys = keySet(keyAltQ, keyAltA, keyAltT, keyAltX, keyAlt2)

// combatMenuKeys are the keys that leave combat_menu: the extended ones,
// space and the menu letters.
var combatMenuKeys = keySet(keyAltQ, keyAltT, keyAltA, ' ', keyAltX, keyAlt2, 'A', 'C', 'D', 'M', 'Q', 'T', 'U', 'V', 'Y')

// keySet is a lookup table of key codes.
func keySet(keys ...byte) (set [256]bool) {
	for _, k := range keys {
		set[k] = true
	}
	return
}

// combatRoundsWithoutAttack: a fight ends after this many rounds without
// an attack.
const combatRoundsWithoutAttack = 15

// endRound = O089:0F61 battle01 (var done): the clock, the dying, the end
// check.
func (e *Engine) endRound() (done bool) {
	e.AdvanceClock(1, 1)
	e.combatRound++
	e.monsterHPPercent()
	for _, c := range e.party {
		e.affectGroup(c, grpRest)
		e.inPoisonCloud(c, false)
		if c.Status == 5 {
			c.Combat.Dying++
			if c.Combat.Dying > 9 {
				c.Status = 6
			}
		}
	}
	if e.bandage(false) {
		e.printMessage("Your Teammate is Dying")
	}
	e.countSides()
	e.combatCentre(e.cmap.viewX+viewW/2, e.cmap.viewY+3, 0xFF, 8)
	if e.rv.side[0] == 0 || e.rv.side[1] == 0 || e.combatRound >= e.roundLimit {
		done = true
	}
	if e.rv.side[0] > 0 && e.rv.side[1] == 0 && !e.demo {
		if e.YesNo(0xD, 0xA, 0xF, "Continue Battle:") == 'Y' {
			done = false
		}
	}
	return
}

// moveAttack = O089:1194 (char, key, var done): step around with the keypad
// (key: the first step, or space to ask) until Done or the turn is over;
// stepping into someone attacks, off the map flees. Escape undoes the
// moves of this menu.
func (e *Engine) moveAttack(c *Character, key byte) (done bool) {
	ci := c.Combat
	moves, face := ci.Moves, ci.Facing
	x, y := e.combatX(c), e.combatY(c)
	done = false
	defer e.rawArrows("keypad, arrows, click a square=step, attack, walk to a framed one|Enter, click self=done|Esc=undo the moves")()
	mark := e.regionMark()
	defer e.regionCut(mark)
	var reach map[[2]int][]int
	var path []int // extra: the steps to a clicked square, taken one by one
	{              // extra (nav.go): a click on a square next to the character steps there, on a farther reachable one walks there
		e.addRegion(region{x0: pageCol, y0: 1, x1: pageCol + viewW*3 - 1, y1: 21, click: func(cx, cy int) uint16 {
			sx, sy, in := e.combatSquare(cx, cy)
			dx, dy := sx-e.combatX(c), sy-e.combatY(c)
			if !in {
				return 0
			}
			if dx == 0 && dy == 0 {
				return 13
			}
			if dx < -1 || dx > 1 || dy < -1 || dy > 1 {
				if p, ok := reach[[2]int{sx, sy}]; ok {
					path = append([]int(nil), p...)
					return keyNone
				}
				return 0
			}
			dxs, dys := stepBytes(stepX), stepBytes(stepY)
			for d := 0; d < 8; d++ {
				if int(int8(dxs[d])) == dx && int(int8(dys[d])) == dy {
					return stepKeys[d]
				}
			}
			return 0
		}})
	}
	for key != 0 && key != 0xD && key != 'D' && !done {
		if len(path) > 0 { // walking to a clicked square: the next step, while it is still free
			d := path[0]
			path = path[1:]
			if who, g, _, _ := e.groundAt(c, d); who > 0 || g == 0 || stepCost(e.groundInfo(g)[0], d) > int(ci.Moves) {
				path = nil
				continue
			}
			key = stepLetters[d]
			e.Delay(0x32)
		} else if key == ' ' {
			reach = e.moveReach(c)
			e.markReach(reach, true)
			e.ClearCells(0x18, 0x18, 0x27, 0x18)
			key, _ = e.DisplayInput("Move/Attack, Move Left = "+itoa(int(ci.Moves>>1)), " Done", 0xD, 0xA, 0xF, true, false)
			e.markReach(reach, false)
		}
		d := 8
		switch key {
		case 0:
			ci.Moves = moves
			e.combatClearIcon(0, 0, e.combatIndex(c))
			done = !e.combatSquareFree(c, x, y, false)
			e.combatCentre(e.combatX(c), e.combatY(c), 0, 8)
			ci.Facing = face
		case scanUp:
			d = 0
		case scanPageUp:
			d = 1
		case scanRight:
			d = 2
		case scanPageDown:
			d = 3
		case scanDown:
			d = 4
		case scanEnd:
			d = 5
		case scanLeft:
			d = 6
		case scanHome:
			d = 7
		}
		if d >= 8 {
			continue
		}
		e.combatTurn(c, byte(d), 0, false)
		who, g, _, _ := e.groundAt(c, d)
		switch {
		case who > 0:
			done = e.attackTarget(c, e.combatant(who), done)
		case g == 0:
			switch e.YesNo(0xD, 0xA, 0xF, "Flee:") {
			case 'Y':
				done = e.fleeBattle(c)
			case 'N':
				done = false
			}
		case stepCost(e.groundInfo(g)[0], d) > int(ci.Moves):
			e.printMessage("can't go there")
		default:
			e.leaveAttacks(c, d)
			if c.Okay == 0 {
				done = e.endTurn(c)
				break
			}
			if ci.Moves > 0 {
				e.stepSquare(c, d)
			}
			if c.Okay == 0 {
				done = e.endTurn(c)
			}
			e.inPoisonCloud(c, true)
			if e.isHelpless(c) {
				done = e.endTurn(c)
			}
		}
		if key != 0 && key != 0xD && key != 'D' {
			key = ' '
		}
	}
	if ci.Moves < 2 {
		ci.Moves = 0
	}
	return
}

// stepCost = O089:1194's price of a step onto ground of the given cost:
// three times it diagonally, twice straight; 255 never passes.
func stepCost(ground byte, dir int) int {
	if ground == groundImpassable {
		return groundImpassable
	}
	if dir&1 != 0 {
		return int(byte(int(ground) * 3))
	}
	return int(byte(int(ground) * 2))
}

// stepLetters are the keypad letters moveAttack reads for the 8 directions.
var stepLetters = [8]byte{scanUp, scanPageUp, scanRight, scanPageDown, scanDown, scanEnd, scanLeft, scanHome}

// moveReach (extra) finds the squares c can walk to with the moves it has
// left, as moveAttack's steps allow them (2 a step, 3 diagonally, free
// ground no dearer than the moves left), each with the steps that keep
// the most moves.
func (e *Engine) moveReach(c *Character) map[[2]int][]int {
	idx := e.combatIndex(c)
	ox, oy := e.cpos[idx].x, e.cpos[idx].y
	defer func() { e.cpos[idx].x, e.cpos[idx].y = ox, oy }()
	type node struct {
		left int
		path []int
	}
	best := map[[2]int]node{{int(ox), int(oy)}: {left: int(c.Combat.Moves)}}
	queue := [][2]int{{int(ox), int(oy)}}
	dd := stepTable()
	for len(queue) > 0 {
		at := queue[0]
		queue = queue[1:]
		n := best[at]
		if n.left <= 1 {
			continue
		}
		e.cpos[idx].x, e.cpos[idx].y = byte(at[0]), byte(at[1])
		for d := 0; d < 8; d++ {
			cost := 2 + d&1
			who, g, _, _ := e.groundAt(c, d)
			if who > 0 || g == 0 || stepCost(e.groundInfo(g)[0], d) > n.left {
				continue
			}
			to := [2]int{at[0] + int(int8(dd[d])), at[1] + int(int8(dd[9+d]))}
			if b, ok := best[to]; ok && b.left >= n.left-cost {
				continue
			}
			best[to] = node{n.left - cost, append(append([]int(nil), n.path...), d)}
			queue = append(queue, to)
		}
	}
	out := map[[2]int][]int{}
	for k, n := range best {
		if len(n.path) > 0 {
			out[k] = n.path
		}
	}
	return out
}

// markReach (extra) shows the reachable squares in view with a dotted
// frame (on) or draws them plain again (off).
func (e *Engine) markReach(reach map[[2]int][]int, on bool) {
	for k := range reach {
		sx, sy := k[0]-e.cmap.viewX, k[1]-e.cmap.viewY
		if !combatInView(sx, sy) {
			continue
		}
		if on {
			e.markSquare(sx, sy, 0xB)
		} else {
			e.drawTile(sx*3, sy*3, e.tileOf(k[0], k[1]))
			e.drawOccupant(k[0], k[1])
		}
	}
	e.Flip()
}

// rangedWeapon = O151:3466 (char): the readied weapon shoots (base item
// +0xC above 1).
func (e *Engine) rangedWeapon(c *Character) bool {
	w := c.Slots[0]
	return w != nil && e.baseItem(w.Type, 0xC) > 1
}

// throwable = O151:34A9 (char): a ranged weapon that also works in melee
// (base item flags 0x14), or the one weapon type that is always thrown.
func (e *Engine) throwable(c *Character) bool {
	if !e.rangedWeapon(c) {
		return false
	}
	t := c.Slots[0].Type
	return e.baseItem(t, 0xE)&0x14 == 0x14 || t == itemTypeAlwaysThrown
}

// itemTypeAlwaysThrown is thrown whatever its base item flags say.
const itemTypeAlwaysThrown = 67

// ammoFor = O151:34FD (char, var item): the item a ranged attack uses up
// (the weapon itself, the arrows in slot 11 or the bolts in slot 12); true
// when there is one or the weapon needs none (flags 0x0A).
func (e *Engine) ammoFor(c *Character, it **Item) bool {
	w := c.Slots[0]
	*it = nil
	var f byte
	if w != nil {
		f = e.baseItem(w.Type, 0xE)
		if f&0x10 != 0 {
			*it = w
		}
		if f&8 != 0 {
			if f&1 != 0 {
				*it = c.Slots[11]
			}
			if f&0x80 != 0 {
				*it = c.Slots[12]
			}
		}
	}
	return *it != nil || f == 0xA
}

// attackTarget = O089:1515 (char, target, var done).
// done passes through when no attack is made.
func (e *Engine) attackTarget(c, t *Character, done bool) bool {
	if e.rangedWeapon(c) && !e.throwable(c) {
		e.printMessage("Not with that weapon")
		return done
	}
	if !e.mayAttack(c, t) {
		return done
	}
	c.Combat.Target = t
	if e.sweepAttack(c, t) {
		return true
	}
	e.countAttack(c, t)
	return e.fullAttack(c, t, 0, nil)
}

// otherMenu = O089:1614 delay_menu (char, var done): Done's submenu.
func (e *Engine) otherMenu(c *Character) (done bool) {
	done = false
	m := ""
	if !e.rangedWeapon(c) || e.throwable(c) {
		m += "Guard "
	}
	m += "Delay Quit "
	if e.bandage(false) {
		m += "Bandage "
	}
	m += "Speed Exit"
	for key := byte(0x20); key != 0 && key != 'E' && !done; {
		key, _ = e.DisplayInput("", m, 0xD, 0xA, 0xF, false, false)
		switch key {
		case 'G':
			done = e.guard(c)
		case 'D':
			c.Combat.Delay = 1
			e.delayTurn(c)
			done = true
		case 'Q':
			done = e.endTurn(c)
		case 'B':
			done = e.bandage(true)
			done = e.endTurn(c)
		case 'S':
			e.speedMenu()
		}
	}
	return
}

// guard = O151:2E3B (char): end the turn on guard (not when turned to flee).
func (e *Engine) guard(c *Character) bool {
	r := e.endTurn(c)
	if c.Combat.Turned == 0 {
		c.Combat.Guarding = 1
		e.printMessage("Guarding")
	}
	return r
}

// bandage = O151:35CB (doIt): is a party member dying? With doIt the
// first one is bandaged.
func (e *Engine) bandage(doIt bool) bool {
	found := false
	for _, c := range e.party {
		if c.Combat.Extra == 0 && c.Side == 0 && c.Status == 5 {
			found = true
			if doIt {
				c.Status = 4
				c.Combat.Dying = 0
				e.combatName(c, "is bandaged", 0xA, true)
				doIt = false
			}
		}
	}
	return found
}

// speedMenu = O089:188C set_gamespeed: the game speed (DS:4FC2).
func (e *Engine) speedMenu() {
	for key := byte(0x20); key != 0 && key != 'E'; {
		m := " "
		if e.gameSpeed < 9 {
			m += "Slower "
		}
		if e.gameSpeed > 0 {
			m += "Faster "
		}
		m += "Exit"
		key, _ = e.DisplayInput("GameSpeed ("+itoa(int(e.gameSpeed))+") :", m, 0xD, 0xA, 0xF, false, false)
		switch key {
		case 'S':
			e.gameSpeed++
		case 'F':
			e.gameSpeed--
		}
	}
}

// quickOne = O089:19C1 (char): the computer fights for it; a target on
// its own side is dropped.
func (e *Engine) quickOne(c *Character) {
	c.Auto = 1
	if t := c.Combat.Target; t != nil && t.Side == c.Side {
		c.Combat.Target = nil
	}
}

// Combat side panel cells (O151:0C8F): right of the 7-square view.
const (
	sidePanelX1    = 23
	sidePanelX2    = 38
	sidePanelTop   = 1
	sidePanelBot   = 21
	sidePanelValue = 33 // hit point column
	sidePanelAC    = 26 // armour class column
)

// combatPanel = O151:0C8F (char): the combatant's name, hit points, armour
// class, readied weapon and its state (out of action, helpless or casting)
// in the side panel.
func (e *Engine) combatPanel(c *Character) {
	if !e.panelDirty {
		return
	}
	e.panelDirty = false
	e.ext.panel = c
	e.ClearCells(sidePanelX1, sidePanelTop, sidePanelX2, sidePanelBot)
	row := sidePanelTop
	e.combatName(c, " ", row, false)
	row++
	e.DisplayString(sidePanelX1, row+1, colourGreen, "Hitpoints")
	e.displayHP(c, sidePanelValue, row+1, false, false)
	row += 2
	e.DisplayString(sidePanelX1, row+1, colourGreen, "AC")
	e.displayAC(c, sidePanelAC, row+1)
	e.msgRow = row + 1
	if w := c.Slots[0]; w != nil {
		row += 2
		e.ItemName(c, w, 0, 0, false, false)
		e.DisplayTextBlock(sidePanelX1, row+1, sidePanelX2, row+3, colourGreen, true, w.Name())
	}
	row = e.msgRow + 1 // the cursor after the weapon name
	switch {
	case c.Okay == 0:
		e.DisplayString(sidePanelX1, row+1, colourWhite, statusNames[c.Status])
	case e.isHelpless(c):
		e.DisplayString(sidePanelX1, row+1, colourWhite, "(Helpless)")
	case c.Combat.Spell > 0:
		e.DisplayString(sidePanelX1, row+1, colourWhite, "(Casting)")
	}
}

// combatName = O151:173F (char, text, row, wait): in combat the name and
// the text go in the side panel from row on; elsewhere under the view.
func (e *Engine) combatName(c *Character, text string, row int, wait bool) {
	if e.gameState == stateCombat {
		e.ClearCells(sidePanelX1, row, sidePanelX2, sidePanelBot)
		e.displayName(c, sidePanelX1, row, false)
		e.DisplayTextBlock(sidePanelX1, row+1, sidePanelX2, sidePanelBot, colourGreen, true, text)
	} else {
		top := 17
		if e.resting != 0 {
			top = 18
		}
		e.ClearCells(1, top, 38, 22)
		e.displayName(c, 1, top+1, false)
		e.DisplayTextBlock(1, top+2, 38, 22, colourGreen, true, text)
	}
	if wait {
		e.pauseForMessage()
		e.clearCombatText()
	}
}

// clearCombatText = O151:1827: clears the message part of the panel.
func (e *Engine) clearCombatText() {
	if e.gameState == stateCombat {
		e.ClearCells(sidePanelX1, 10, sidePanelX2, sidePanelBot)
		return
	}
	e.ClearCells(1, 18, 38, 22)
}

// combatMsgRow is the side panel row combat messages start on.
const combatMsgRow = 10

// combatUseHook = O151:3430: redraw the combat screen: the map frame
// (extra: full width, no panel frame), rows 23-24 cleared, the map centred on the view.
func (e *Engine) combatUseHook() {
	e.drawCombatFrame()
	e.combatCentre(e.cmap.viewX+viewW/2, e.cmap.viewY+3, 0xFF, 8)
}
