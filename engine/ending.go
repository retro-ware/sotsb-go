package engine

// The ending (unit O0AC): the last words of Tyranthraxus, three animated
// pictures, the Knights of Myth Drannor and fireworks over Shadowdale.

// Where the ending's text goes and the pictures it plays.
const (
	endTextX1, endTextY1 = 1, 17
	endTextX2, endTextY2 = 38, 22
	endPoolShatters      = 74
	endTyranthraxusGone  = 75
	endBondsFade         = 77
	endKnightsPortrait   = 65
	endShadowdalePic     = 122
	endFadeLoopsPerSpeed = 2
	endFadeSpeeds        = 10 // loops = (10 - gameSpeed) * 2
	animDelaySpeedBase   = 3  // a frame lasts Delay * (gameSpeed+3) ticks
)

// endGameText = O0AC:111C end_game_text.
func (e *Engine) endGameText() {
	e.lastGameState = e.gameState
	e.gameState = stateEndGame
	say := func(lines ...string) {
		for i, l := range lines {
			e.DisplayTextBlock(endTextX1, endTextY1, endTextX2, endTextY2, colourGreen, i == 0, l)
		}
	}
	cont := func() { e.displayAndWait(colourLightMagenta, colourBlack, "Press any key to continue.") }
	say("Tyranthraxus' spirit coalesces over the slain ",
		"storm giant. 'You have defeated me. Were it not for ",
		"the Amulet of Lythander, I could possess you and rob ",
		"you of your victory. Still I can escape through the pool.")
	cont()
	say("As you reach for the Pool of Radiance, he cries ",
		"out, 'Keep the Gauntlet of Moander away from there, you ",
		"will unleash dangerous energies. Stay back!' As the ",
		"gauntlet contacts the pool, it contracts and shatters it.")
	e.playPicture(viewPicX, viewPicY, endPoolShatters, 1)
	cont()
	e.ClearRow24()
	say("'I am trapped without escape, you have succeeded ",
		"where armies have not. Gloat while you may, Tyranthraxus ",
		"is slain this day.' Before your eyes he crumbles into ",
		"nothingness.")
	e.playPicture(viewPicX, viewPicY, endTyranthraxusGone, 1)
	cont()
	e.ClearRow24()
	say("You are certain he is destroyed because your ",
		"final bond fades away. The Curse of the Azure Bonds ",
		"has finally been lifted from you! You are free at ",
		"last!")
	putWord(e.area, 2*areaWordRecolour, 1)
	e.playPicture(viewPicX, viewPicY, endBondsFade, byte((endFadeSpeeds-int(e.gameSpeed))*endFadeLoopsPerSpeed))
	putWord(e.area, 2*areaWordRecolour, 0)
	e.loadPortrait(endKnightsPortrait, endKnightsPortrait)
	e.drawPortrait(viewPicX, viewPicY, true)
	say("The Knights of Myth Drannor rush in, '",
		"Congratulations, you have destroyed the Flamed One. ",
		"With the power of Elminster, let us take you from ",
		"this  foul place, to a fine feast.'")
	cont()
	e.ClearRow24()
	e.bigPicture(endShadowdalePic)
	e.wildernessPicture()
	say("You are teleported to Shadowdale, where festivities ",
		"have already begun. A huge cheer goes up at your arrival. ",
		"Gharri and Nacacia, arm in arm, yell congratulations ",
		"from the nearby stands. 'You have won!'")
	e.fireworks()
	e.gameState = e.lastGameState
	e.loadPicture()
}

// playPicture = O0AC:0B79 (x, y, block, loops): the PIC block played
// through loops times, then freed.
func (e *Engine) playPicture(x, y int, block, loops byte) {
	done := byte(0)
	since := e.Ticks()
	e.loadViewPicture("PIC", block, 2)
	a := &e.pic
	if len(a.frames) == 0 {
		return
	}
	e.drawPage(x-1, y-1, a.frames[0].Image, 0, true, false)
	e.Flip()
	a.cur = 1
	for done != loops {
		e.drawViewPicture(x, y, true, a.cur-1)
		now := e.Ticks()
		if now-since > int64(a.frames[a.cur-1].Delay)*int64(int(e.gameSpeed)+animDelaySpeedBase) {
			a.cur++
			if a.cur > len(a.frames) {
				a.cur = 1
				done++
			}
			since = now
		}
	}
	e.restoreBackground()
}
