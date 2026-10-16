package engine

// The title sequence (unit O039 of the reconstruction).

// Title pictures in TITLE.DAX.
const (
	titleSSILogo     = 1 // "SSI presents an official AD&D computer product"
	titleHeroine     = 2 // the full-screen painting
	titleCurseLogo   = 3 // "Curse of the Azure Bonds", over the painting
	titleForgotten   = 4 // the Forgotten Realms banner, over the painting
	titleLogoColumn  = 6 // where the Curse logo goes, in 8-pixel columns
	titleBannerRow   = 11
	titleFirstWait   = 5 // seconds
	titleSecondWait  = 10
	soundTitleBanner = titleSound // the tune started with the banner
)

// waitSecondsOrKey = O039:0000: clear the keyboard, wait n seconds or until a
// key is pressed, clear the keyboard again.
func (e *Engine) waitSecondsOrKey(n int) {
	e.ClearKeyboard()
	end := e.Ticks() + int64(n)*ticksPerSecond
	for !e.keyWaiting() && e.Ticks() < end {
		e.idle()
	}
	e.ClearKeyboard()
}

// Title = O039:0632 title_screen.
func (e *Engine) Title() {
	e.DrawPicture(0, 0, e.LoadPic("TITLE", titleSSILogo), 0)
	e.waitSecondsOrKey(titleFirstWait)
	if !e.Cfg.FullIntro {
		return
	}
	e.DrawPicture(0, 0, e.LoadPic("TITLE", titleHeroine), 0)
	e.DrawPicture(titleLogoColumn, titleBannerRow, e.LoadPic("TITLE", titleCurseLogo), 0)
	e.waitSecondsOrKey(titleSecondWait)
	banner := e.LoadPic("TITLE", titleForgotten)
	e.PlaySound(soundTitleBanner)
	e.DrawPicture(0, titleBannerRow, banner, 0)
	e.waitSecondsOrKey(titleSecondWait)
	e.ClearScreen()
	e.Credits()
	e.waitSecondsOrKey(titleSecondWait)
	e.ClearScreen()
}

// Credits = O039:01DD.
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
	{2, 1, colourGreen, `based on the tsr novel "azure bonds"`},
	{6, 2, colourGreen, "by:"},
	{9, 2, colourCyan, "kate novak"},
	{20, 2, colourGreen, "and"},
	{24, 2, colourCyan, "jeff grubb"},
	{10, 4, colourGreen, "scenario created by:"},
	{11, 5, colourYellow, "tsr, inc."},
	{21, 5, colourGreen, "and"},
	{25, 5, colourYellow, "ssi"},
	{14, 6, colourCyan, "jeff grubb"},
	{11, 7, colourCyan, "george mac donald"},
	{1, 9, colourGreen, "game created by:"},
	{18, 9, colourYellow, "ssi special projects"},
	{2, 11, colourYellow, "project leader:"},
	{2, 12, colourCyan, "george mac donald"},
	{2, 14, colourYellow, "programming:"},
	{2, 15, colourCyan, "scot bayless"},
	{2, 16, colourCyan, "russ brown"},
	{2, 17, colourCyan, "michael mancuso"},
	{2, 19, colourYellow, "development:"},
	{2, 20, colourCyan, "david shelley"},
	{2, 21, colourCyan, "michael mancuso"},
	{2, 22, colourCyan, "oran kangas"},
	{22, 11, colourYellow, "graphic arts:"},
	{22, 12, colourCyan, "tom wahl"},
	{22, 13, colourCyan, "fred butts"},
	{22, 14, colourCyan, "susan manley"},
	{22, 15, colourCyan, "mark johnson"},
	{22, 16, colourCyan, "cyrus lum"},
	{22, 18, colourYellow, "playtesting:"},
	{22, 19, colourCyan, "jim jennings"},
	{22, 20, colourCyan, "james kucera"},
	{22, 21, colourCyan, "rick white"},
	{22, 22, colourCyan, "robert daly"},
}
