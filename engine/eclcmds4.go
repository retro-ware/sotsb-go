package engine

import "math"

// More script commands (O03C): ROB, CLOCK, ADD NPC, WHO, PROTECTION and
// CLEAR BOX.

// cmdRob = O03C:1F76 (who, percent, item chance): money is taken (percent)
// and items may go; who 0 is the selected member, otherwise everyone.
func (e *Engine) cmdRob() {
	e.readArgs(3)
	who := byte(e.arg(1))
	keep := real48Div(float64(100-int(byte(e.arg(2)))), 100)
	chance := byte(e.arg(3))
	if who == 0 {
		e.robMoney(keep, e.selected())
		e.robItems(chance, e.selected())
		return
	}
	for _, c := range e.party {
		e.robMoney(keep, c)
		e.robItems(chance, c)
	}
}

// robMoney = O04C:1DEF (factor, c): each kind of coin times the factor.
func (e *Engine) robMoney(keep float64, c *Character) {
	for k := range c.Coins {
		c.Coins[k] = uint16(math.Trunc(real48Mul(float64(c.Coins[k]), keep)))
	}
}

// robItems = O04C:1F1C (chance, c): each item goes with the chance in
// percent; heavy items lower the chance for the rest of the list.
func (e *Engine) robItems(chance byte, c *Character) {
	for _, it := range append([]*Item(nil), c.Items...) {
		if it.Weight > 255 {
			if chance > 90 {
				chance -= 90
			} else {
				chance = 0
			}
		} else if it.Weight > 24 {
			if chance > 50 {
				chance -= 50
			} else {
				chance = 0
			}
		}
		if byte(e.Dice(1, 100)) <= chance {
			e.loseItem(c, it)
		}
	}
}

// loseItem = O0ED:159C (c, item): the item leaves the character's list.
func (e *Engine) loseItem(c *Character, it *Item) {
	if it == nil {
		e.displayAndWait(colourYellow, colourBlack, "Nil Item pointer...")
		return
	}
	for i, x := range c.Items {
		if x == it {
			c.Items = append(c.Items[:i], c.Items[i+1:]...)
			return
		}
	}
	e.displayAndWait(colourYellow, colourBlack, "Tried to Lose item & couldn't find it!")
}

// real48 rounds to Turbo Pascal's 6-byte Real (40 significant bits).
func real48(x float64) float64 {
	if x == 0 {
		return 0
	}
	m, exp := math.Frexp(x)
	return math.Ldexp(math.Round(math.Ldexp(m, real48Bits)), exp-real48Bits)
}

const real48Bits = 40

func real48Div(a, b float64) float64 { return real48(real48(a) / real48(b)) }
func real48Mul(a, b float64) float64 { return real48(real48(a) * real48(b)) }

// cmdClock = O03C:2CDA (count, unit).
func (e *Engine) cmdClock() {
	e.readArgs(2)
	count, unit := byte(e.arg(1)), byte(e.arg(2))
	e.AdvanceClock(int(count), int(int8(unit)))
}

// npcControl: ADD NPC's morale operand is halved into the control byte.
const npcControl = 128

// cmdAddNPC = O03C:2DCA (monster, morale): a monster joins as a hired NPC.
func (e *Engine) cmdAddNPC() {
	e.readArgs(2)
	block := byte(e.arg(1))
	e.addNPC(block)
	morale := byte(e.arg(2))
	e.selected().Control = morale>>1 | npcControl
	e.ApplyItems(e.selected())
	e.playerSummary(e.selected())
}

// addNPC = O0A1:357D (monster): load it, join the party and take its
// combat icon.
func (e *Engine) addNPC(block byte) {
	if wordAt(e.game, 2*cvPartyCount) > 7 {
		return
	}
	c := e.loadMonsterRecord(block)
	c.RandomID = block
	e.AddToParty(c)
	e.LoadIconPic("CPIC", int(block), int(c.Order))
}

// cmdWho = O03C:2D7F (prompt): the player picks the member.
func (e *Engine) cmdWho() {
	e.readArgs(1)
	e.ClearCells(textLeft, textTop, textRight, textBottom)
	e.selectMember(e.selectMemberMenu(e.sc.text[1], false, e.selected()))
}

// selectMemberMenu = O0ED:2E2E selectAPlayer (prompt, allowExit, start):
// Select (and Exit) with Home / End stepping through the party from start;
// nil on Exit.
func (e *Engine) selectMemberMenu(prompt string, allowExit bool, start *Character) *Character {
	menu := "Select"
	if allowExit {
		menu += " Exit"
	}
	cur := start
	k := byte(' ')
	for k != keyEnter && k != keyEsc && k != 'E' && k != 'S' {
		e.playerSummary(cur)
		animate := e.gameState == stateCamp || e.gameState == stateAfterCombat
		var extra bool
		k, extra = e.DisplayInput(prompt+" ", menu, colourLightMagenta, colourGreen, colourWhite, true, animate)
		if extra {
			i := e.partyIndex(cur)
			switch k {
			case scanEnd:
				if i+1 < len(e.party) {
					cur = e.party[i+1]
				} else {
					cur = e.party[0]
				}
			case scanHome:
				if i > 0 {
					cur = e.party[i-1]
				} else {
					cur = e.party[len(e.party)-1]
				}
			}
		} else if allowExit && (k == 'E' || k == 0) {
			cur = nil
		}
	}
	return cur
}

const keyEsc = 27

// cmdProtection = O03C:323F: the code wheel question (left out, as at the
// start), then the view is put back.
func (e *Engine) cmdProtection() {
	e.sc.pictureShown, e.sc.pictureBig = false, false
	e.sc.pictureOn = false
	e.readArgs(1)
	e.loadPicture()
}

// cmdClearBox = O03C:2D38: the adventuring frame and the picture again.
func (e *Engine) cmdClearBox() {
	s := &e.sc
	s.pc++
	e.drawAdventureFrame()
	e.playerSummary(e.selected())
	e.statusLine()
	e.drawViewPicture(viewPicX, viewPicY, true, 0)
	e.statusLine()
	s.firstLoad = false
	s.inSpecial = true
}
