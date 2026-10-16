package engine

// The combat icon editor (unit O0A7 of the reconstruction).

// iconMenus = DS 1827 + 41*state: the editor's menus (state 3 gets "Hair"
// or "Face" for xxxx).
var iconMenus = [6]string{"", "Parts 1st-color 2nd-color Size Exit", "Head Weapon Exit",
	"Weapon Body xxxx Shield Arm Leg Exit", " Keep Exit", "Next Prev Keep Exit"}

// iconWorkSlot: the icon slot the editor builds its working copy in.
const iconWorkSlot = 12

// iconBuilder = O0A7:3EC6 icon_builder: the combat icon editor (called
// from the character creation and the camp). States (menus iconMenus):
// 1 Parts/color-1/color-2/Size/Exit, 2 Head/Weapon, 3 the part to colour,
// 4 Size, 5 the Next/Prev/Keep/Exit stepping for the choice made before.
// The old icon is shown above the new one; Is this icon ok? ends it.
func (e *Engine) iconBuilder() {
	c := e.selected()
	e.drawScreenFrame()
	e.combatViewLeave() // O121:00F2
	menu3 := iconMenus[3]
	var state, parent, part int // parent and part are uninitialised until set
	var sub, key byte
	var face bool
	for {
		e.BuildCharIcon(c, false) // O0A1:0A90
		state = 1
		var colours [6]byte
		copy(colours[:], c.IconColour[:])
		order := c.Order
		c.Order = iconWorkSlot // the working copy is built into slot 12
		e.BuildCharIcon(c, false)
		c.Order = order
		head, weapon, size := c.IconHead, c.IconBody, c.IconSize
		e.iconCopy(order, iconWorkSlot, true)
		e.iconPane(1, 2)
		e.DisplayString(8, 6, colourWhite, "old")
		e.DisplayString(3, 10, colourWhite, "ready   action")
		e.DisplayString(8, 12, colourWhite, "new")
		e.DisplayString(3, 16, colourWhite, "ready   action")
		for {
			e.iconPane(1, 4)
			sizeStr := ""
			if state == 4 {
				sizeStr = "Large"
				if c.IconSize == 2 {
					sizeStr = "Small" // as in the original
				}
			}
			m := iconMenus[state]
			if state == 3 {
				m = menu3
			}
			k, extra := e.DisplayInput("", sizeStr+m, colourLightMagenta, colourGreen, colourWhite, false, false)
			key = k
			if !extra {
				done := key == 0 || key == 'E'
				switch state {
				case 1:
					parent = 1
					switch key {
					case 'P':
						state = 2
					case '1', '2':
						if e.Cfg.Graphics == 'C' {
							e.statusMessage("Sorry, not in CGA", colourYellow, colourBlack)
							break
						}
						state, face = 3, key == '2'
						label := "Hair"
						if face {
							label = "Face"
						}
						// Delete(s, 13, 4); Insert(label, s, 13) on DS:83A.
						s := menu3[:12] + menu3[16:]
						menu3 = cut(s[:12]+label+s[12:], 0x28)
					case 'S':
						state = 4
					}
				case 2:
					parent = 2
					if done {
						state = 1
					} else {
						sub, state = key, 5
					}
				case 3:
					parent = 3
					part = 1
					switch key {
					case 'W':
						part = 6
					case 'B':
						part = 1
					case 'H', 'F':
						part = 4
					case 'S':
						part = 5
					case 'A':
						part = 2
					case 'L':
						part = 3
					}
					if done {
						state = 1
					} else {
						state = 5
					}
				case 4:
					switch {
					case key == 'L':
						c.IconSize = 2
						e.BuildCharIcon(c, false)
					case key == 'S':
						c.IconSize = 1
						e.BuildCharIcon(c, false)
					case key == 'K':
						size, state, key = c.IconSize, 1, ' '
					case done:
						c.IconSize = size
						state, key = 1, ' '
					}
					e.BuildCharIcon(c, false)
				case 5:
					switch parent {
					case 2:
						// Step head (0..13) or weapon (0..31) pictures.
						pic, saved, top := &c.IconHead, &head, byte(0xD)
						if sub == 'W' {
							pic, saved, top = &c.IconBody, &weapon, 0x1F
						}
						if sub == 'H' || sub == 'W' {
							switch {
							case key == 'P':
								if *pic > 0 {
									*pic--
								} else {
									*pic = top
								}
							case key == 'N':
								if *pic < top {
									*pic++
								} else {
									*pic = 0
								}
							case key == 'K':
								*saved, state, key = *pic, parent, ' '
							case done:
								*pic = *saved
								state, key = parent, ' '
							}
							e.BuildCharIcon(c, false)
						}
					case 3:
						lo, hi := c.IconColour[part-1]&0xF, c.IconColour[part-1]>>4
						switch {
						case key == 'N', key == 'P':
							d := byte(1)
							if key == 'P' {
								d = 0xF // (n-1) and 15
							}
							if face {
								hi = (hi + d) & 0xF
							} else {
								lo = (lo + d) & 0xF
							}
							c.IconColour[part-1] = hi<<4 + lo
						case key == 'K':
							copy(colours[:], c.IconColour[:])
							state, key = parent, ' '
						case done:
							copy(c.IconColour[:], colours[:])
							state, key = parent, ' '
						}
					}
				}
			}
			e.iconCopy(c.Order, iconWorkSlot, true)
			if parent == 1 && (key == 0 || key == 'E') {
				break
			}
		}
		c.IconHead, c.IconBody, c.IconSize = head, weapon, size
		copy(c.IconColour[:], colours[:])
		e.iconCopy(c.Order, iconWorkSlot, true)
		e.iconCopy(iconWorkSlot, c.Order, false)
		e.ClearRow24()
		e.freeIcon(iconWorkSlot) // O12A:0156
		if e.YesNo(colourLightMagenta, colourGreen, colourWhite, "Is this icon ok? ") == 'Y' {
			break
		}
	}
	e.combatViewEnter() // O121:0130
}
