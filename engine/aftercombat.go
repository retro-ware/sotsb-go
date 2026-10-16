package engine

// After combat, O045: the fight's outcome, experience, the hired NPCs' share
// and the treasure screen. ECL TREASURE (and COMBAT without monsters) runs
// this directly, which is how scripts hand out treasure.

// afterCombat holds the O045 globals.
type afterCombat struct {
	monstersFound bool  // b4804: there were monsters in the fight
	fled          bool  // b4805: the party ran
	idle          int   // b8B71: members who get no experience
	exp           int32 // l8B72
	won           bool  // b8B76
}

// Area words the fight reads and leaves (word indexes into e.game).
const (
	areaWordFightResult     = 711 // 128 lost, 129 fled
	areaWordMonstersBeaten  = 712 // monsters that went down
	areaWordNoMonsterItems  = 739 // 1: the monsters' items are not taken
	areaWordNoDeath         = 742 // a duel (O04C:1B9A): the fallen get up again
	areaWordFightFlagsFirst = 880 // .. 882: cleared after every fight
	areaWordFightFlagsLast  = 882
)

const (
	fightLost = 128
	fightFled = 129
)

// Experience and treasure values (O045:0037).
const (
	duelExpPerLevel = 100
	expPerPlus      = 400 // a magic item's worth per plus
	primeScore      = 15  // a prime requisite above this earns a tenth more
)

// treasureExpDivisor / treasureExpFactor: what a coin of each kind of the
// pool adds to the experience (copper /200 .. jewelry ×2200).
var (
	treasureExpDivisor = [coinKinds]int32{200, 20, 2, 1, 1, 1, 1}
	treasureExpFactor  = [coinKinds]int32{1, 1, 1, 1, 5, 250, 2200}
)

// afterCombatAffects = DS:02A9[1..19]: the affects a fight leaves behind
// that go when it ends.
var afterCombatAffects = [...]byte{3, 11, 13, 21, 23, 27, 35, 40, 51, 52, 53, 58, 91, 136, 137, 139, 142, 144, 31}

// isMonster: the combatant is one of the encounter's monsters (combat
// block +0x13 = 1).
func isMonster(c *Character) bool { return c.Combat != nil && c.Combat.Extra == 1 }

// combatParty returns the party members before the first combat monster.
func (e *Engine) combatParty() []*Character {
	for i, c := range e.party {
		if isMonster(c) {
			return e.party[:i]
		}
	}
	return e.party
}

// computeExp = O045:0037 (var exp): the beaten monsters' worth, their
// money and items go to the treasure, and the treasure's worth counts too;
// the sum is split over the members that earn it.
func (e *Engine) computeExp() int32 {
	sel := e.selected()
	if e.duelling {
		return int32(int8(sel.MaxLevel)) * duelExpPerLevel
	}
	var total int32
	for _, c := range e.party {
		if c.Side != 1 || c.Status == statusOkay || c.Status == statusRunning {
			continue
		}
		e.ac.monstersFound = true
		for k := range e.treasureCoins {
			e.treasureCoins[k] += int32(c.Coins[k])
		}
		// the byte product is a 16-bit mul, sign-extended
		total += int32(c.ExpValue) + int32(int16(uint16(c.ExpPerHP)*uint16(c.HPRoll)))
		if wordAt(e.game, 2*areaWordNoMonsterItems) == 1 {
			continue
		}
		for _, it := range c.Items {
			e.ItemName(c, it, 0, 0, false, false)
			cp := *it
			cp.Readied = 0
			e.treasureItems = append([]*Item{&cp}, e.treasureItems...)
		}
	}
	for k, v := range e.treasureCoins {
		total += v / treasureExpDivisor[k] * treasureExpFactor[k]
	}
	for _, it := range e.treasureItems {
		if it == e.thrownItem {
			break
		}
		if p := int8(it.Plus); p > 0 {
			total += int32(int16(int(p) * expPerPlus))
		}
	}
	// the original divides by zero when no one earns any
	if d := int32(uint16(e.partyCount() - e.ac.idle)); d > 0 {
		total /= d
	}
	return total
}

// giveExp = O045:036A (exp): every member still standing gets it, with a
// tenth more for high prime requisites; multi-class characters split it.
func (e *Engine) giveExp(exp int32) {
	high := func(c *Character, stats ...int) bool {
		for _, s := range stats {
			if c.Stats[s].Value <= primeScore {
				return false
			}
		}
		return true
	}
	for _, c := range e.party {
		if c.Okay == 0 || c.Status == statusAnimated {
			continue
		}
		g := exp
		bonus := false
		switch c.Class {
		case clsCleric:
			bonus = high(c, statWis)
		case clsFighter:
			bonus = high(c, statStr)
		case clsPaladin:
			bonus = high(c, statStr, statWis)
		case clsRanger:
			bonus = high(c, statStr, statInt, statWis)
		case clsMage:
			bonus = high(c, statInt)
		case clsThief:
			bonus = high(c, statDex)
		case 8, 10, 11, 12, 13, 14, 16:
			g = exp / 2
		case 9, 15:
			g = exp / 3
		}
		if bonus {
			g += exp / 10
		}
		c.Exp = uint32(int32(c.Exp) + g)
	}
}

// combatResult = O045:0556: who won, experience, and what happens to the
// fallen. partyDead is left set when the whole party is lost.
func (e *Engine) combatResult() {
	e.ac.idle = 0
	e.partyDead = true
	e.ac.fled = false
	for _, c := range e.combatParty() {
		if c.Status == statusRunning {
			e.ac.fled = true
		}
	}
	standing := false
	for _, c := range e.party {
		if c.Okay != 0 || c.Status == statusUnconscious || c.Status == statusRunning || c.Status == statusDying {
			standing = true
			break
		}
	}
	noDeath := wordAt(e.game, 2*areaWordNoDeath) != 0
	if e.duelling || noDeath && standing {
		e.partyDead = false
	}
	e.ac.won = false
	if e.duelling && e.demo {
		for _, c := range e.party {
			if c.Okay == 1 && c.Status == statusOkay && c.Side != 1 {
				e.ac.won = true
				e.ac.exp = e.computeExp()
				e.giveExp(e.ac.exp)
			}
		}
		for _, c := range e.party {
			switch c.Status {
			case statusOkay, statusAnimated:
				c.Okay = 1
			case statusDying:
				c.Status = statusUnconscious
			}
		}
		return
	}
	for _, c := range e.combatParty() {
		st := c.Status
		if (st == statusRunning || st == statusAnimated || st == statusOkay) && c.Side == 0 && c.Control <= controlHired {
			e.partyDead = false
		}
		if st == statusAnimated || st == statusOkay {
			e.ac.won = true
			e.ac.fled = false
		}
		if c.Okay == 0 || st == statusAnimated {
			e.ac.idle++
		}
		for _, a := range afterCombatAffects {
			e.RemoveAffect(c, a, nil)
		}
	}
	if e.ac.won {
		e.ac.exp = e.computeExp()
		e.giveExp(e.ac.exp)
	}
	if e.partyDead {
		for len(e.party) > 0 && !isMonster(e.party[0]) {
			e.partySel = 0
			e.freePlayers(false, true)
		}
		putWord(e.game, gsPartyCount, 0)
		return
	}
	getUp := func(c *Character) {
		c.Status, c.Okay = statusOkay, 1
	}
	for i := 0; i < len(e.party) && !isMonster(e.party[i]); {
		c := e.party[i]
		if e.ac.fled {
			putWord(e.game, 2*areaWordFightResult, fightFled)
			if c.Status == statusRunning {
				getUp(c)
				i++
				continue
			}
			e.partySel = i
			e.freePlayers(false, true)
			continue
		}
		switch c.Status {
		case statusRunning:
			getUp(c)
		case statusDying:
			if noDeath {
				getUp(c)
				c.HP = 1
			} else {
				c.Status = statusUnconscious
			}
		case statusUnconscious:
			if c.HP > 0 {
				getUp(c)
			} else if noDeath {
				getUp(c)
				c.HP = 1
			}
		}
		i++
	}
}

// cleanupCombatants = O045:13C7: the monsters leave the party list (the
// area word counts the ones that were beaten); the members drop their
// combat blocks.
func (e *Engine) cleanupCombatants() {
	putWord(e.game, 2*areaWordMonstersBeaten, 0)
	for i := 0; i < len(e.party); {
		c := e.party[i]
		monster := isMonster(c)
		if monster || c.Side == 1 {
			e.ac.monstersFound = true
			if c.Okay != 1 {
				putWord(e.game, 2*areaWordMonstersBeaten, wordAt(e.game, 2*areaWordMonstersBeaten)+1)
			}
			e.partySel = i
			e.freePlayers(monster, true)
			continue
		}
		c.Combat = nil
		i++
	}
	e.partySel = 0
}

// hiredShare = O045:150E: hired NPCs take their shares (+0xF8 & 7) of
// every kind of coin before the party gets to the treasure.
func (e *Engine) hiredShare() {
	const shareMask = 7
	shares, parts := 0, 0
	for _, c := range e.party {
		if c.Control > controlHired && c.Status == statusOkay {
			shares += int(c.Share & shareMask)
			parts += int(c.Share & shareMask)
		} else {
			parts++
		}
	}
	taken := false
	if shares > 0 {
		for k, v := range e.treasureCoins {
			if v > 0 {
				each := byte(v / int32(byte(parts)))
				e.treasureCoins[k] -= int32(int16(uint16(byte(shares)) * uint16(each)))
				taken = true
			}
		}
	}
	if !taken {
		return
	}
	e.drawScreenFrame()
	row := 0
	for _, c := range e.party {
		if c.Control > controlHired && c.Status == statusOkay && c.Share > 0 {
			pronoun := "his"
			if c.Sex != 0 {
				pronoun = "her"
			}
			e.DisplayTextBlock(5, row+5, 34, 22, colourGreen, true, c.Name()+" takes and hides "+pronoun+" share.")
			row += 2
		}
	}
	e.DisplayInput("", "press <enter>/<return> to continue", colourWhite, colourWhite, colourWhite, true, false)
}

// freeTreasureItems drops the treasure's items.
func (e *Engine) freeTreasureItems() { e.treasureItems = nil }

// showResult = O045:0ABC (exp).
func (e *Engine) showResult(exp int32) {
	e.drawScreenFrame()
	say := func(y int, s string) { e.displayStringBg(1, y, colourGreen, 0, s) }
	switch {
	case !e.ac.monstersFound && !e.duelling:
		say(3, "The party has found Treasure!")
	case e.ac.fled:
		say(3, "The party has fled.")
		exp = 0
		e.freeTreasureItems()
		e.treasureCoins = [coinKinds]int32{}
	case !e.ac.won && (e.duelling || wordAt(e.game, 2*areaWordNoDeath) != 0):
		putWord(e.game, 2*areaWordFightResult, fightLost)
		say(3, "You have lost the fight.")
		exp = 0
	case e.duelling:
		say(3, "You have won the duel.")
	default:
		say(3, "The party has won.")
	}
	s := "Each character receives "
	if e.duelling {
		s = "The duelist receives "
	}
	say(5, s+itoa(int(exp)))
	say(7, "experience points.")
	e.DisplayInput("", "press <enter>/<return> to continue", colourWhite, colourWhite, colourWhite, true, false)
}

// lootList = O045:0D2B (var key, var item, var selection, var index): the
// treasure's items in a list to take from.
func (e *Engine) lootList(index *int) (byte, *Item) {
	e.drawScreenFrame()
	list := make([]*ListNode, len(e.treasureItems))
	for i, it := range e.treasureItems {
		e.ItemName(nil, it, 0, 0, false, false)
		list[i] = &ListNode{Text: it.Name()}
	}
	redraw := true
	var sel *ListNode
	key := e.ListMenu("Items: ", "Take", colourLightMagenta, colourGreen, colourWhite, 1, 1, 38, 22, list, true, &redraw, index, &sel)
	for i, n := range list {
		if n == sel {
			return key, e.treasureItems[i]
		}
	}
	return key, nil
}

// lootTake = O045:0DFC: take items until another key or nothing is left.
func (e *Engine) lootTake() {
	index := 0
	for {
		key, it := e.lootList(&index)
		if key != 'T' && key != keyEnter || it == nil {
			break
		}
		if e.takeItemCopy(it) {
			continue
		}
		for i, t := range e.treasureItems {
			if t == it {
				e.treasureItems = append(e.treasureItems[:i], e.treasureItems[i+1:]...)
				break
			}
		}
		if len(e.treasureItems) == 0 {
			break
		}
	}
	e.loadPicture()
}

// takeTreasure = O045:0F2E (var money, var items): Take, of money or items
// or both.
func (e *Engine) takeTreasure(money, items *bool) {
	if !*money {
		e.lootTake()
		return
	}
	if !*items {
		e.takeMoney()
		e.loadPicture()
		return
	}
	for done := false; !done; {
		key, _ := e.DisplayInput("Take: ", "Money Items Exit", colourLightMagenta, colourGreen, colourWhite, true, true)
		switch key {
		case 'M':
			e.takeMoney()
			e.loadPicture()
		case 'I':
			e.lootTake()
		case 'E', 0:
			done = true
		case 'G', 'O':
			e.cycleSelected(key)
		}
		e.playerSummary(e.selected())
		*money, *items = e.treasureLeft()
		if !*money || !*items {
			done = true
		}
	}
}

// Detection spells the treasure menu offers (memorized ids).
var treasureDetectSpells = [...]byte{5, 11, 77}

// lootMenu = O045:10C3: the treasure screen.
func (e *Engine) lootMenu() {
	e.loadPicture()
	var money, items bool
	var detect byte
	for done := false; !done; {
		money, items = e.treasureLeft()
		menu := "View Pool Exit"
		exit := " Exit"
		found := false
		if items {
			c := e.selected()
			for i := 0; i < len(c.Memorized) && !found; i++ {
				for _, s := range treasureDetectSpells {
					if c.Memorized[i] == s && c.Okay != 0 {
						found, detect = true, s
					}
				}
			}
		}
		if found {
			exit = " Detect Exit"
		}
		if money {
			menu = "View Take Pool Share" + exit
		} else if items {
			menu = "View Take Pool" + exit
		}
		key, extra := e.DisplayInput("", menu, colourLightMagenta, colourGreen, colourWhite, true, true)
		switch key {
		case 'V':
			extra = e.viewPlayer()
		case 'T':
			e.takeTreasure(&money, &items)
		case 'P':
			if !extra {
				e.poolMoney()
			}
		case 'S':
			e.shareMoney()
		case 'D':
			e.CastSpell(detect, 0, 0, &extra)
		case 'E', 0:
			if money, items = e.treasureLeft(); !money && !items {
				done = true
				break
			}
			e.DisplayTextBlock(1, 17, 38, 22, colourGreen, true, "There is still treasure left.  ")
			e.DisplayTextBlock(1, 17, 38, 22, colourWhite, false, "Do you want to go back and claim your treasure?")
			if e.scriptMenu("", "~Yes ~No", colourLightMagenta, colourGreen, colourWhite, false, false) == 1 {
				done = true
			} else {
				e.ClearBox(1, 17, 38, 22)
			}
		case 'G', 'O':
			e.cycleSelected(key)
			e.playerSummary(e.selected())
		}
	}
}

// AfterCombat = O045:17A2.
func (e *Engine) AfterCombat() {
	putWord(e.game, 2*areaWordFightResult, 0)
	e.ac.monstersFound = false
	if !e.demo {
		e.combatResult()
	}
	e.gameState = stateAfterCombat
	e.cleanupCombatants()
	if e.demo {
		return
	}
	for _, c := range e.party {
		e.ApplyItems(c)
	}
	if !e.partyDead || e.duelling {
		if e.ac.fled {
			e.freeTreasureItems()
		}
		if !e.demo {
			e.hiredShare()
			e.showResult(e.ac.exp)
			e.lootMenu()
		}
		e.freeTreasureItems()
	} else {
		putWord(e.game, 2*areaWordFightResult, fightLost)
		e.drawScreenFrame()
		e.msgCol, e.msgRow = 2, 6
		e.DisplayTextBlock(2, 5, 37, 22, colourGreen, true, "The monsters rejoice for the party has been destroyed")
		e.displayAndWait(colourLightMagenta, 0, "Press any key to continue")
	}
	// (b4FE2 is set here; nothing reads it)
	for w := areaWordFightFlagsFirst; w <= areaWordFightFlagsLast; w++ {
		putWord(e.game, 2*w, 0)
	}
	putWord(e.game, 2*areaWordNoMonsterItems, 0)
	putWord(e.game, 2*areaWordNoDeath, 0)
}
