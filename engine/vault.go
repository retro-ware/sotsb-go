package engine

import (
	"bytes"
	"encoding/binary"
	"os"
)

// The vault (unit O066): coins and items the party leaves in the vault
// stay there, in vault.dat in the save directory, for any later party.

const vaultFile = "vault.dat"

// vaultMenu = O066:0060: the vault's contents become the treasure, the
// party Views, Takes and Pools; what is left behind is written back.
func (e *Engine) vaultMenu() {
	e.gameState = stateVault
	e.shopFrame = true
	e.loadPicture()
	e.shopFrame = true
	e.playerSummary(e.selected())
	e.treasureItems = nil
	e.treasureCoins = [coinKinds]int32{}
	if b, err := os.ReadFile(e.savePath(vaultFile)); err == nil {
		e.readVault(b)
	}
	for done := false; !done; {
		money, items := e.treasureLeft()
		menu := "View"
		if money || items {
			menu += " Take"
		}
		menu += " Pool Exit"
		key, extra := e.DisplayInput("", menu, colourLightMagenta, colourGreen, colourWhite, true, false)
		if extra {
			if key == scanDown || key == scanUp {
				e.selectStep(key)
			}
			continue
		}
		switch key {
		case 'P':
			e.poolMoney()
		case 'V':
			e.viewPlayer()
		case 'T':
			e.takeTreasure(&money, &items)
		case 'E':
			done = true
		}
	}
	if err := os.WriteFile(e.savePath(vaultFile), e.vaultBytes(), 0o644); err != nil {
		e.displayAndWait(colourYellow, colourBlack, "Can't save.  No room on this disk.")
	}
	e.treasureItems = nil
}

// readVault: seven coin counts, then the items (a bag followed by its
// contents).
func (e *Engine) readVault(b []byte) {
	r := bytes.NewReader(b)
	if binary.Read(r, binary.LittleEndian, &e.treasureCoins) != nil {
		e.treasureCoins = [coinKinds]int32{}
		return
	}
	rest := b[len(b)-r.Len():]
	rest = rest[:len(rest)/itemFileRecSize*itemFileRecSize]
	e.treasureItems = readItems(rest, itemFileRecSize)
}

// vaultBytes: vault.dat's contents.
func (e *Engine) vaultBytes() []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, &e.treasureCoins)
	buf.Write(writeItems(e.treasureItems))
	return buf.Bytes()
}
