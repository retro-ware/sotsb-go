package engine

// Items: names, lists and the START helpers around them.

func (it *Item) Name() string {
	n := int(it.PName[0])
	if n > 0x28 {
		n = 0x28
	}
	return string(it.PName[1 : n+1])
}

func (it *Item) setName(s string) {
	s = cut(s, 0x28)
	it.PName[0] = byte(len(s))
	copy(it.PName[1:41], s)
}

// AddItemCopy = O0ED:1690 (char, item): a copy of the item at the end of
// the character's list.
func (e *Engine) AddItemCopy(c *Character, it *Item) *Item {
	n := *it
	c.Items = append(c.Items, &n)
	return &n
}
