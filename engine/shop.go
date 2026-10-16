package engine

// The shop (unit O048): COMBAT with the area2 shop flag set opens it; the
// goods are the treasure list, the money the pool and the buyer's coins.

// gameWordPriceFactor: the shop's prices are the item's price times this
// over 16 (1 .. 128).
const gameWordPriceFactor = 877

// shopPrice: an item's price in this shop. Fixed: the original's buying
// code has no case for the normal factor 16 and reads an unset price; the
// list shows the plain price there, so that is taken.
func (e *Engine) shopPrice(it *Item) uint16 {
	switch wordAt(e.game, 2*gameWordPriceFactor) {
	case 1:
		return it.Price / 16
	case 2:
		return it.Price / 8
	case 4:
		return it.Price >> 1 >> 1
	case 8:
		return it.Price >> 1
	case 32:
		return it.Price * 2
	case 64:
		return it.Price * 4
	case 128:
		return it.Price * 8
	}
	return it.Price
}

// Width of a shop list line, price right-aligned.
const shopLineWidth = 30

// shopList = O048:004E (var key, var item, var selection, var index): the
// goods with their prices; the item names are rebuilt afterwards.
func (e *Engine) shopList(index *int, redraw *bool) (byte, *Item) {
	list := make([]*ListNode, len(e.treasureItems))
	for i, it := range e.treasureItems {
		if it.Price == 0 {
			it.Price = 1
		}
		price := itoa(int(e.shopPrice(it)))
		if len(price) > 16 {
			price = price[:16]
		}
		line := []byte(spaces(shopLineWidth))
		copy(line[shopLineWidth-len(price):], price)
		name := it.Name()
		copy(line, name[:min(len(name), shopLineWidth)])
		list[i] = &ListNode{Text: string(line)}
	}
	e.menuSel = 0
	var sel *ListNode
	key := e.ListMenu("Items: ", "Buy", colourLightMagenta, colourGreen, colourWhite, 1, 1, 38, 22, list, true, redraw, index, &sel)
	var item *Item
	for i, n := range list {
		if n == sel {
			item = e.treasureItems[i]
		}
	}
	for _, it := range e.treasureItems {
		e.ItemName(nil, it, 0, 0, false, false)
	}
	return key, item
}

func spaces(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = ' '
	}
	return string(b)
}

// overloadedBy = O0AF:324B (c, item): sixteen items, or the item's weight
// (times its count) would overload c.
func (e *Engine) overloadedBy(c *Character, it *Item) bool {
	e.ApplyItems(c)
	over := c.ItemCount > 15
	w := it.Weight
	if it.Count > 0 {
		w = uint16(it.Count) * w
	}
	if c.Weight+w > carryLimit(c) {
		over = true
	}
	return over
}

// takeItemCopy = O048:034C overloaded (item): a copy of the item joins the
// selected member's items unless it overloads them.
func (e *Engine) takeItemCopy(it *Item) bool {
	c := e.selected()
	if e.overloadedBy(c, it) {
		e.printMessage("OverLoaded")
		return true
	}
	cp := *it
	c.Items = append(c.Items, &cp)
	e.ApplyItems(c)
	return false
}

// shopBuy = O048:0474: buy goods until another key than Buy or Enter,
// paid from the buyer's coins or else from the pool.
func (e *Engine) shopBuy() {
	index := 0
	e.drawScreenFrame()
	redraw := true
	for {
		key, it := e.shopList(&index, &redraw)
		if key != 'B' && key != keyEnter {
			return
		}
		if it == nil {
			continue
		}
		price := e.shopPrice(it)
		if gold := uint16(goldWorth(e.selected())); price <= gold {
			if !e.takeItemCopy(it) {
				e.setGold(gold - price)
			}
		} else if pool := poolGold(e.treasureCoins); int32(price) <= pool {
			if !e.takeItemCopy(it) {
				e.setPoolGold(uint16(pool - int32(price)))
			}
		} else {
			e.printMessage("Not enough Money.")
		}
	}
}

// shopMenu = O048:06E7: the shop's menu until the party leaves.
func (e *Engine) shopMenu() {
	e.gameState = stateShop
	e.shopFrame = wordAt(e.area, 2*areaWordFreeSteps) == 0
	e.loadPicture()
	e.shopFrame = true
	e.playerSummary(e.selected())
	e.treasureCoins = [coinKinds]int32{}
	e.poolHasMoney = false
	for _, it := range e.treasureItems {
		e.ItemName(nil, it, 0, 0, false, false)
	}
	for leave := false; !leave; {
		money, _ := e.treasureLeft()
		menu := "Buy View Pool Appraise Exit"
		if money {
			menu = "Buy View Take Pool Share Appraise Exit"
		}
		key, extra := e.DisplayInput("", menu, colourLightMagenta, colourGreen, colourWhite, true, false)
		appraised := false
		switch key {
		case 'B':
			e.shopBuy()
		case 'V':
			e.viewPlayer()
		case 'T':
			e.takeMoney()
		case 'P':
			if !extra {
				e.poolMoney()
			}
		case 'S':
			e.shareMoney()
		case 'A':
			appraised = e.appraise()
		case 'E':
			if money, _ := e.treasureLeft(); money {
				e.msgCol, e.msgRow = textLeft, textTop
				e.DisplayTextBlock(textLeft, textTop, textRight, textBottom, colourGreen, true,
					`As you Leave the Shopkeeper says, "Excuse me but you have Left Some Money here."  `)
				e.DisplayTextBlock(textLeft, textTop, textRight, textBottom, colourWhite, false,
					"Do you want to go back and get your Money?")
				if e.scriptMenu("", "~Yes ~No", colourLightMagenta, colourGreen, colourWhite, false, false) == 1 {
					leave = true
				} else {
					e.ClearCells(textLeft, textTop, textRight, textBottom)
				}
			} else {
				leave = true
			}
		case scanHome, scanEnd:
			e.cycleSelected(key)
		}
		switch {
		case key == 'B' || key == 'T':
			e.loadPicture()
		case key == 'A' && appraised:
			e.loadPicture()
		}
		e.playerSummary(e.selected())
	}
}
