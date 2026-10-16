package engine

// The mouse (unit R73D of the reconstruction, INT 33h). The host draws the
// cursor; the game only shows and hides it around its drawing.

// HideMouse = R73D:0055: hide the cursor while the screen is drawn.
func (e *Engine) HideMouse() { e.mouseShown = false }

// ShowMouse = R73D:0032.
func (e *Engine) ShowMouse() { e.mouseShown = true }

// MouseShown tells the host whether to draw the cursor.
func (e *Engine) MouseShown() bool { return e.mouseShown }
