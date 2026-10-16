package engine

// The title sequence and the credits (unit O047 of the reconstruction).

// Title pictures in TITLE.DAX and where they go, in 8-pixel cells.
const (
	titleSSILogo       = 1 // "SSI presents an official AD&D computer product"
	titlePainting      = 2 // the full-screen painting
	titleForgotten     = 3 // the Forgotten Realms banner, over the painting
	titleSilverBlades  = 4 // the "Secret of the Silver Blades" logo, over the painting
	titleBannerColumn  = 2
	titleBannerRow     = 5
	titleLogoColumn    = 5
	titleLogoRow       = 5
	titleSoundSample   = 0 // the digitized sound played over the pictures
	titleSSIWait       = 15
	titlePictureWait   = 50
	titleLogoWait      = 60
	titleCreditsWait   = 80
	waitStepsPerSecond = 10 // O047:0000 counts its waits in tenths of a second
)

// waitOrKey = O047:0000: clear the keyboard and wait tenths/10 seconds or
// until a key is pressed. It tells whether a key cut the wait short.
func (e *Engine) waitOrKey(tenths int) bool {
	e.ClearKeyboard()
	end := e.Ticks() + int64(tenths)*ticksPerSecond/waitStepsPerSecond
	for !e.keyWaiting() && e.Ticks() < end {
		e.idle()
	}
	pressed := e.Ticks() < end
	e.ClearKeyboard()
	return pressed
}

// endTitle = O047:006F: clear the screen, stop the digitized sound and let
// the game start.
func (e *Engine) endTitle() {
	e.ClearScreen()
	e.stopDigitizedSound()
	e.ShowMouse()
	e.textWindowReady = true
	e.inTitle = false
}

// Title = O047:0571: the SSI logo, the painting with the Forgotten Realms
// banner, then with the Silver Blades logo, and the credits. A key at any
// picture ends it.
func (e *Engine) Title() {
	e.PlaySound(sndStop)
	e.restoreBackground()
	e.inTitle = true
	e.startDigitizedSound(titleSoundSample)
	e.HideMouse()
	show := func(id, col, row int) {
		e.DrawPicture(col, row, e.LoadPic("TITLE", id), 0)
	}
	show(titleSSILogo, 0, 0)
	if e.waitOrKey(titleSSIWait) {
		e.endTitle()
		return
	}
	show(titlePainting, 0, 0)
	if e.waitOrKey(titlePictureWait) {
		e.endTitle()
		return
	}
	show(titleForgotten, titleBannerColumn, titleBannerRow)
	if e.waitOrKey(titlePictureWait) {
		e.endTitle()
		return
	}
	show(titlePainting, 0, 0)
	show(titleSilverBlades, titleLogoColumn, titleLogoRow)
	if e.waitOrKey(titleLogoWait) {
		e.endTitle()
		return
	}
	e.ClearScreen()
	e.Credits()
	e.waitOrKey(titleCreditsWait)
	e.endTitle()
}

// Credits = O047:029C.
func (e *Engine) Credits() {
	e.drawCreditsFrame()
	for _, t := range creditLines {
		e.DisplayString(t.x, t.y, t.colour, t.text)
	}
}

// creditLines: column, row, colour and text of each credits line.
var creditLines = []struct {
	x, y, colour int
	text         string
}{
	{7, 0, colourMagenta, "secret of the silver blades"},
	{2, 3, colourMagenta, "game created by:    story and"},
	{2, 4, colourYellow, "ssi special"},
	{22, 4, colourMagenta, "development by:"},
	{2, 5, colourYellow, "projects team       Dave Shelley"},
	{2, 7, colourMagenta, "programming:        graphic arts:"},
	{2, 8, colourYellow, "ken nicholson       mark johnson"},
	{2, 9, colourYellow, "russ brown          laura bowen"},
	{22, 10, colourYellow, "fred butts"},
	{2, 11, colourMagenta, "Encounter Design:"},
	{22, 11, colourYellow, "cyrus lum"},
	{2, 12, colourYellow, "dave shelley        susan manley"},
	{2, 13, colourYellow, "michael mancuso     mike provenza"},
	{2, 14, colourYellow, "ken humphries       tom wahl"},
	{2, 15, colourYellow, "graeme Bayless"},
	{2, 16, colourYellow, "rick white"},
	{22, 16, colourMagenta, "Musical Score:"},
	{22, 17, colourYellow, "john halbleib"},
	{2, 19, colourMagenta, "playtest:"},
	{12, 19, colourYellow, "rick white,don mc clure,"},
	{2, 20, colourYellow, "dave lucca,cliff mann,rick wilson,"},
	{2, 21, colourYellow, "erik flom,james young,larry webber"},
}
