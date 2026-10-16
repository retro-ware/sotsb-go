package engine

import (
	"encoding/binary"
	"os"
	"strconv"
)

// Saved games (O0F8:2813 and 2FAB): savgamA..J.dat with the area, game
// and aux records, the position and states and the wall sets, and the
// party as CHRDATxN.sav files.

// The saved game's layout: area, game state, aux, then the position and
// states, the wall slots and the party's file names.
const (
	saveNameSize  = 41 // string[40]
	saveNameSlots = 8
	savePosition  = 5 // x, y, facing, the wall and the cell there
	saveWallSlots = 3
	saveGameSize  = 1 + 2048 + 2048 + 1024 + savePosition + 2 + saveWallSlots*4 + 1 + saveNameSize*saveNameSlots
)

// Area and game bytes a save carries the settings in.
const (
	areaWordSpeedByte = 504 // byte offset: the game speed
	areaWordPics      = 255 // pictures on * 2 + animation
	gameAreaByte      = 1572
)

// gameFileName: savgamX.dat in the save directory.
func (e *Engine) gameFileName(letter byte) string {
	return e.savePath("savgam" + string(letter) + ".dat")
}

// partyFileName: the saved game's file of party member n (1..).
func partyFileName(letter byte, n int) string {
	return "CHRDAT" + string(letter) + strconv.Itoa(n)
}

// SaveGame = O0F8:2FAB: write savgamX.dat and the party's .sav files.
func (e *Engine) SaveGame() {
	var key byte
	for {
		key, _ = e.menuChoice("Save Which Game: ", "A B C D E F G H I J", colourLightMagenta, colourGreen, colourWhite, false, e.gameState == stateCamp)
		if key == 0 || key >= 'A' && key <= 'J' {
			break
		}
	}
	if key == 0 {
		return
	}
	e.addSource = fromSilverBlades
	if !e.saveDirReady() {
		return
	}
	e.ClearRow24()
	e.displayStringBg(0, statusRow, colourGreen, colourBlack, "Saving...Please Wait")
	e.area[areaWordSpeedByte] = e.gameSpeed
	e.area[areaWordSpeedByte+1] = 0
	pics := uint16(0)
	if e.picturesOn {
		pics = 2
	}
	if e.loadingMessage {
		pics++
	}
	putWord(e.area, 2*areaWordPics, pics)
	putWord(e.game, gameAreaByte, uint16(e.gameArea))
	s := &e.sc
	b := make([]byte, 0, saveGameSize)
	b = append(b, e.gameArea)
	b = append(b, e.area...)
	b = append(b, e.game...)
	b = append(b, e.aux...)
	b = append(b, s.posX, s.posY, s.facing, s.wallHere, s.cellHere)
	b = append(b, e.lastGameState, e.gameState)
	for k := 0; k < saveWallSlots; k++ {
		b = binary.LittleEndian.AppendUint16(b, uint16(int16(e.mapPairA[k])))
		b = binary.LittleEndian.AppendUint16(b, uint16(int16(e.mapPairB[k])))
	}
	if len(e.party) == 0 {
		e.displayAndWait(colourYellow, colourBlack, "WARNING: Problem Saving Characters")
	}
	b = append(b, byte(len(e.party)))
	var names [saveNameSlots * saveNameSize]byte
	for i := range e.party {
		if i >= saveNameSlots {
			break
		}
		n := partyFileName(key, i+1)
		names[i*saveNameSize] = byte(len(n))
		copy(names[i*saveNameSize+1:], n)
	}
	b = append(b, names[:]...)
	if err := os.WriteFile(e.gameFileName(key), b, 0o644); err != nil {
		e.displayAndWait(colourYellow, colourBlack, "Unexpected error during save: "+err.Error())
		return
	}
	for i, c := range e.party {
		e.SaveCharacter(c, partyFileName(key, i+1))
		e.removeCharacterFiles(c)
	}
	e.saved = true
	e.ClearRow24()
}

// LoadGame = O0F8:2813: pick a saved game and read it back.
func (e *Engine) LoadGame() {
	e.addSource = fromSilverBlades
	if !e.saveDirReady() {
		return
	}
	menu := ""
	for l := byte('A'); l <= 'J'; l++ {
		if e.fileExists(e.gameFileName(l)) {
			menu += string(l) + " "
		}
	}
	if menu == "" {
		return
	}
	menu = menu[:len(menu)-1]
	var key byte
	for {
		key, _ = e.menuChoice("Load Which Game: ", menu, colourLightMagenta, colourGreen, colourWhite, false, false)
		if key >= 'A' && key <= 'J' && e.fileExists(e.gameFileName(key)) || key == 0 {
			break
		}
	}
	if key == 0 {
		return
	}
	b, err := os.ReadFile(e.gameFileName(key))
	if err != nil || len(b) < saveGameSize {
		e.fileError("read", e.gameFileName(key))
		return
	}
	e.ClearRow24()
	e.displayStringBg(0, statusRow, colourGreen, colourBlack, "Loading...Please Wait")
	e.loadedGame = true
	s := &e.sc
	p := 0
	read := func(n int) []byte { r := b[p : p+n]; p += n; return r }
	e.gameArea = read(1)[0]
	copy(e.area, read(len(e.area)))
	copy(e.game, read(len(e.game)))
	copy(e.aux, read(len(e.aux)))
	pos := read(savePosition)
	s.posX, s.posY, s.facing, s.wallHere, s.cellHere = pos[0], pos[1], pos[2], pos[3], pos[4]
	e.lastGameState = read(1)[0]
	e.gameState = read(1)[0]
	for k := 0; k < saveWallSlots; k++ {
		e.mapPairA[k] = int(int16(binary.LittleEndian.Uint16(read(2))))
		e.mapPairB[k] = int(int16(binary.LittleEndian.Uint16(read(2))))
	}
	count := int(read(1)[0])
	names := read(saveNameSlots * saveNameSize)
	pics := wordAt(e.area, 2*areaWordPics)
	e.picturesOn = pics/2 != 0
	e.loadingMessage = pics%2 != 0
	e.gameSpeed = e.area[areaWordSpeedByte]
	e.setPartyCount(0)
	e.party, e.partySel = nil, 0
	for i := 0; i < count && i < saveNameSlots; i++ {
		entry := names[i*saveNameSize:]
		n := min(int(entry[0]), saveNameSize-1)
		name := makeFileName(removeSubstr(string(entry[1:1+n]), "."))
		if e.fileExists(e.savePath(name + extGameCharacter)) {
			c := &Character{}
			if e.LoadCharacter(name+extGameCharacter, c) {
				e.AddToParty(c)
			}
		}
	}
	for _, c := range e.party {
		e.removeCharacterFiles(c)
	}
	for i, c := range e.party {
		e.partySel = i
		if c.Control < npcControl {
			e.BuildCharIcon(c, true)
		} else {
			e.LoadIconPic("CPIC", int(c.RandomID), int(c.Order))
		}
	}
	e.partySel = 0
	e.gameArea = e.game[gameAreaByte]
	if wordAt(e.area, 2*areaWordFreeSteps) != 0 {
		if e.mapPairA[0] > 0 {
			e.loadGeo(e.area[2*areaWordGeo])
		}
		for k := 0; k < saveWallSlots; k++ {
			if e.mapPairA[k] > 0 {
				e.LoadWalldef(int(byte(e.mapPairA[k])), int(byte(e.mapPairB[k])))
			}
		}
	} else {
		e.bigPicture(bigPicWild)
	}
	e.ClearKeyboard()
	e.ClearRow24()
	e.lastGameState = e.gameState
	e.gameState = stateStartMenu
}
