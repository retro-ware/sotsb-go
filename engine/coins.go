package engine

// Money in CotAB is coins on each character plus the treasure pool of the
// current shop or treasure (unit O0BA): carrying it, pooling, sharing,
// taking and appraising gems and jewelry.

// carryBase: the weight anyone can carry before strength is counted.
const carryBase = 1500

// carryLimit = O0BA:0019 (c): the weight the character can carry.
func carryLimit(c *Character) uint16 { return uint16(int(strCarry(c)) + carryBase) }

// willOverload = O0BA:006D (c, weight): whether adding the weight
// overloads the character, and how much still fits then.
func willOverload(c *Character, weight uint16) (bool, uint16) {
	if c.Weight+weight > carryLimit(c) {
		return true, carryLimit(c) - c.Weight
	}
	return false, 0
}

// poolGold = O0BA:00C1 (coins): the pool's coins in gold pieces.
func poolGold(coins [coinKinds]int32) int32 {
	var copper int32
	for k := range coinValues {
		copper += coinValues[k] * coins[k]
	}
	return (copper + 100) / coinValues[coinGold]
}

// setGold = O0BA:0149 (gold): the selected member's coins become platinum
// and gold worth that much.
func (e *Engine) setGold(gold uint16) {
	c := e.selected()
	for k := 0; k <= coinPlatinum; k++ {
		c.Coins[k] = 0
	}
	c.Coins[coinPlatinum] = gold / platinumGold
	c.Coins[coinGold] = gold % platinumGold
}

// platinumGold: gold pieces per platinum piece.
const platinumGold = 5

// setPoolGold = O0BA:019E (gold): the pool's coins become platinum and gold.
func (e *Engine) setPoolGold(gold uint16) {
	for k := 0; k <= coinPlatinum; k++ {
		e.treasureCoins[k] = 0
	}
	e.treasureCoins[coinPlatinum] = int32(gold / platinumGold)
	e.treasureCoins[coinGold] = int32(gold % platinumGold)
}

// addPlatinum = O0BA:021C (n): platinum to the selected member; what does
// not fit goes into the pool.
func (e *Engine) addPlatinum(n uint16) {
	c := e.selected()
	if over, room := willOverload(c, n); over {
		e.printMessage("Overloaded.  Money will be put in Pool.")
		c.Coins[coinPlatinum] += room
		c.Weight += room
		e.treasureCoins[coinPlatinum] = int32(n) + e.treasureCoins[coinPlatinum] - int32(room)
		return
	}
	c.Coins[coinPlatinum] += n
	c.Weight += n
}

// askAmount = O0BA:02AD (max, prompt, colour): a number up to max typed on
// the bottom row (a digit that makes it too big is not taken); Esc is 0.
func (e *Engine) askAmount(max uint16, prompt string, colour int) uint16 {
	e.ClearRow24()
	e.displayStringBg(0, statusRow, colour, colourBlack, prompt)
	start := len(prompt)
	x := start
	text := ""
	var k byte
	for k != keyEnter && k != keyEsc {
		k = e.ReadKeyByte()
		switch {
		case k >= '0' && k <= '9':
			try := text + string(k)
			if v, ok := parseUint(try); ok && uint32(max) >= v {
				text = try
				x++
			} else {
				text = itoa(int(max))
				x = len(text) + start
			}
			e.displayStringBg(start, statusRow, colourWhite, colourBlack, text)
		case k == keyBackspace:
			if len(text) > 0 {
				text = text[:len(text)-1]
				e.DisplaySpaces(x, statusRow, colourBlack, 1)
				x--
			}
		}
	}
	e.ClearRow24()
	if k == keyEsc {
		return 0
	}
	v, _ := parseUint(text)
	return uint16(v)
}

const keyBackspace = 8

// parseUint is Turbo Pascal's Val for a number of digits.
func parseUint(s string) (uint32, bool) {
	if s == "" {
		return 0, false
	}
	var v uint32
	for i := 0; i < len(s); i++ {
		v = v*10 + uint32(s[i]-'0')
	}
	return v, true
}

// moveCoins = O0BA:046B add_object (from, to, amount, kind): hand coins
// over unless they overload the receiver.
func (e *Engine) moveCoins(from, to *Character, amount uint16, kind int) {
	if to.Weight+amount > carryLimit(to) {
		e.printMessage("Overloaded")
		return
	}
	from.Coins[kind] -= amount
	from.Weight -= amount
	to.Coins[kind] += amount
	to.Weight += amount
}

// controlFree: members the player controls (party members and the charmed).

// poolMoney = O0BA:0527: everyone the player controls puts their coins in
// the pool.
func (e *Engine) poolMoney() {
	e.poolHasMoney = true
	for _, c := range e.party {
		if c.Control == 0 || c.Control == controlCharmed {
			for k := 0; k < coinKinds; k++ {
				e.treasureCoins[k] += int32(c.Coins[k])
				c.Weight -= c.Coins[k]
			}
			c.Coins = [coinKinds]uint16{}
		}
	}
}

// sharers = O0BA:05FF: the members the pool is shared between.
func (e *Engine) sharers() byte {
	var n byte
	for _, c := range e.party {
		if c.Control == 0 || c.Control == controlCharmed {
			n++
		}
	}
	return n
}

// shareMoney = O0BA:0659: the pool is split between the members; what does
// not fit, and the remainders, go to whoever can still carry it.
func (e *Engine) shareMoney() {
	n := int32(e.sharers())
	var share, rest [coinKinds]uint16
	for k := 0; k < coinKinds; k++ {
		if e.treasureCoins[k] > 0 {
			share[k] = uint16(e.treasureCoins[k] / n)
			rest[k] = uint16(e.treasureCoins[k] % n)
		}
	}
	for _, c := range e.party {
		if c.Control >= npcControl {
			continue
		}
		for k := coinKinds - 1; k >= 0; k-- {
			if over, room := willOverload(c, share[k]); !over {
				c.Coins[k] += share[k]
				c.Weight += share[k]
				if rest[k] != 0 {
					if over, _ := willOverload(c, 1); !over {
						c.Coins[k]++
						c.Weight++
						rest[k]--
					}
				}
			} else {
				c.Coins[k] += room
				rest[k] = rest[k] + share[k] - room
				c.Weight += room
			}
		}
	}
	for k := coinKinds - 1; k >= 0; k-- {
		if rest[k] == 0 {
			continue
		}
		for _, c := range e.party {
			room := carryLimit(c) - c.Weight
			if room == 0 {
				continue
			}
			if room < rest[k] {
				c.Coins[k] += room
				c.Weight += room
				rest[k] -= room
			} else {
				c.Coins[k] += rest[k]
				c.Weight += rest[k]
				rest[k] = 0
			}
		}
	}
	e.poolHasMoney = false
	for k := 0; k < coinKinds; k++ {
		e.treasureCoins[k] = int32(rest[k])
		if e.treasureCoins[k] != 0 {
			e.poolHasMoney = true
		}
	}
}

// dropCoins = O0BA:0A19 (c, amount, kind): coins leave the character; in
// a shop or after a fight they go into the pool.
func (e *Engine) dropCoins(c *Character, amount uint16, kind int) {
	c.Coins[kind] -= amount
	c.Weight -= amount
	if e.gameState == stateAfterCombat || e.gameState == stateShop {
		e.treasureCoins[kind] += int32(amount)
	}
}

// takeCoins = O0BA:0AA0 (c, amount, kind): coins from the pool, at most
// what is there, unless they overload the character.
func (e *Engine) takeCoins(c *Character, amount uint16, kind int) {
	if over, _ := willOverload(c, amount); over {
		e.printMessage("Overloaded")
		return
	}
	if int32(amount) > e.treasureCoins[kind] {
		amount = uint16(e.treasureCoins[kind])
	}
	e.treasureCoins[kind] -= int32(amount)
	c.Coins[kind] += amount
	c.Weight += amount
}

// coinKindOf = O0BA:0BAB (line): which coin a "Name amount" line of the
// take list names, and the name with a space.
func coinKindOf(line string) (int, string) {
	i := 0
	for i < len(line) && line[i] == ' ' {
		i++
	}
	if i >= len(line) {
		return 0, ""
	}
	switch line[i] {
	case 'G':
		if i+1 < len(line) && upper(line[i+1]) == 'E' {
			return coinGems, "Gems "
		}
		return coinGold, "Gold "
	case 'P':
		return coinPlatinum, "Platinum "
	case 'E':
		return coinElectrum, "electrum "
	case 'S':
		return coinSilver, "silver "
	case 'C':
		return coinCopper, "Copper "
	case 'J':
		return coinJewelry, "Jewelry "
	}
	return 0, ""
}

// takeMoney = O0BA:0D0A takeItems: choose a kind of coin in the pool and
// how much of it the selected member takes, until the pool is empty or the
// player leaves.
func (e *Engine) takeMoney() {
	e.drawScreenFrame()
	index := 0
	for {
		redraw := true
		var list []*ListNode
		for k := coinKinds - 1; k >= 0; k-- {
			if e.treasureCoins[k] > 0 {
				list = append([]*ListNode{{Text: coinNames[k] + " " + itoa(int(e.treasureCoins[k]))}}, list...)
			}
		}
		var sel *ListNode
		key := e.ListMenu("Select type of coin ", "Select", colourLightMagenta, colourGreen, colourWhite,
			2, 2, 15, 8, list, true, &redraw, &index, &sel)
		if sel == nil || key == 0 {
			return
		}
		kind, name := coinKindOf(sel.Text)
		amount := e.askAmount(uint16(e.treasureCoins[kind]), "How much "+name+"will you take? ", colourGreen)
		e.takeCoins(e.selected(), amount, kind)
		e.poolHasMoney = false
		done := true
		for k := 0; k < coinKinds; k++ {
			if e.treasureCoins[k] > 0 {
				e.poolHasMoney = true
				done = false
			}
		}
		if done {
			return
		}
	}
}

// treasureLeft = O0BA:0F7B: whether money and whether items are left.
func (e *Engine) treasureLeft() (money, items bool) {
	for k := 0; k < coinKinds; k++ {
		if e.treasureCoins[k] != 0 {
			money = true
		}
	}
	return money, len(e.treasureItems) != 0
}

// Gem and jewel items Appraise can keep.
const (
	itemGemOrJewel = 70
	wordGemName    = 101
	wordJewelName  = 214
	maxItems       = 16
)

// appraise = O0BA:1928: gems and jewelry are valued one at a time and kept
// as items or sold for platinum. Returns false when there was nothing.
func (e *Engine) appraise() bool {
	c := e.selected()
	if c.Coins[coinGems] == 0 && c.Coins[coinJewelry] == 0 {
		e.printMessage("No Gems or Jewelry")
		return false
	}
	for {
		if c.Coins[coinGems] == 0 && c.Coins[coinJewelry] == 0 {
			return true
		}
		gems, jewels := itoa(int(c.Coins[coinGems])), itoa(int(c.Coins[coinJewelry]))
		switch c.Coins[coinGems] {
		case 0:
			gems = ""
		case 1:
			gems += " Gem"
		default:
			gems += " Gems"
		}
		switch c.Coins[coinJewelry] {
		case 0:
			jewels = ""
		case 1:
			jewels += " piece of Jewelry"
		default:
			jewels += " pieces of Jewelry"
		}
		e.ClearCells(1, 1, 38, 22)
		e.displayName(c, 1, 1, false)
		e.displayStringBg(1, 7, colourWhite, colourBlack, "You have a fine collection of:")
		e.displayStringBg(1, 9, colourWhite, colourBlack, gems)
		e.displayStringBg(1, 10, colourWhite, colourBlack, jewels)
		menu := ""
		if c.Coins[coinGems] != 0 {
			menu = "  Gems"
		}
		if c.Coins[coinJewelry] != 0 {
			menu += "  Jewelry"
		}
		menu += " Exit"
		k, _ := e.DisplayInput("Appraise : ", menu, colourLightMagenta, colourGreen, colourWhite, true, false)
		var value uint16
		var word byte
		switch k {
		case 'G':
			if c.Coins[coinGems] == 0 {
				break
			}
			c.Coins[coinGems]--
			switch r := e.Dice(1, 100); {
			case r <= 25:
				value = 10
			case r <= 50:
				value = 50
			case r <= 70:
				value = 100
			case r <= 90:
				value = 500
			case r <= 99:
				value = 1000
			default:
				value = 5000
			}
			e.displayStringBg(1, 12, colourWhite, colourBlack, "The Gem is Valued at "+itoa(int(value))+" gp.")
			word = wordGemName
		case 'J':
			if c.Coins[coinJewelry] == 0 {
				break
			}
			c.Coins[coinJewelry]--
			switch r := e.Dice(1, 100); {
			case r <= 10:
				value = uint16(e.Random(900) + 100)
			case r <= 20:
				value = uint16(e.Random(1000) + 200)
			case r <= 40:
				value = uint16(e.Random(1500) + 300)
			case r <= 50:
				value = uint16(e.Random(2500) + 500)
			case r <= 70:
				value = uint16(e.Random(5000) + 1000)
			case r <= 90:
				value = uint16(e.Random(6000) + 2000)
			default:
				value = uint16(e.Random(10000) + 2000)
			}
			e.displayStringBg(1, 12, colourWhite, colourBlack, "The Jewel is Valued at "+itoa(int(value))+" gp.")
			word = wordJewelName
		case 'E', 0:
			e.ApplyItems(c)
			return true
		}
		if word != 0 {
			full, _ := willOverload(c, 1)
			full = full || c.ItemCount >= maxItems
			choice := "Sell Keep"
			if full {
				choice = "Sell"
			}
			k, _ := e.DisplayInput("You can : ", choice, colourLightMagenta, colourGreen, colourWhite, true, false)
			if k == 'K' && !full {
				it := &Item{Type: itemGemOrJewel, Price: value, Weight: 1}
				it.Words[2] = word
				// Fixed: the original loses the item when the list is empty.
				c.Items = append(c.Items, it)
			} else {
				e.addPlatinum(value / platinumGold)
			}
		}
		e.ApplyItems(c)
	}
}
