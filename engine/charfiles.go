package engine

import (
	"os"
	"strings"
)

// Saved characters (unit O0A1 of the reconstruction): NAME.GUY holds the
// record, NAME.SWG the items (63 bytes each), NAME.FX the affects (9 bytes
// each). Saved games keep their characters as .SAV files. The original
// writes its memory pointers into the files; the port writes zeros there
// (they are never read back).

const (
	extCharacter     = ".guy"
	extItems         = ".swg"
	extAffects       = ".fx"
	extGameCharacter = ".sav"
	maxFileName      = 8
)

// SaveCharacter = O0A1:0DFC (c, name): name "" saves to the roster under the
// character's name (asking before overwriting), else name.sav for a saved game.
func (e *Engine) SaveCharacter(c *Character, name string) {
	if !e.saveDirReady() {
		return
	}
	base, ext := makeFileName(removeSubstr(c.Name(), ".")), extCharacter
	if name != "" {
		base, ext = name, extGameCharacter
	}
	for name == "" && e.fileExists(e.savePath(base+ext)) {
		if e.YesNo(colourYellow, colourGreen, colourWhite, "Overwrite "+base+"? ") == 'Y' {
			break
		}
		base = ""
		for base == "" {
			base = e.InputString("New file name: ", colourGreen, maxFileName)
		}
	}
	if err := os.WriteFile(e.savePath(base+ext), c.record(), 0o644); err != nil {
		e.fileError("write", e.savePath(base+ext))
		return
	}
	os.Remove(e.savePath(base + extItems))
	if len(c.Items) > 0 {
		os.WriteFile(e.savePath(base+extItems), writeRecords(c.Items), 0o644)
	}
	os.Remove(e.savePath(base + extAffects))
	if len(c.Affects) > 0 {
		var recs []*affectRecord
		for _, a := range c.Affects {
			recs = append(recs, &a.affectRecord)
		}
		os.WriteFile(e.savePath(base+extAffects), writeRecords(recs), 0o644)
	}
}

// saveDirReady = O0A1:0655 save: make sure the save directory exists (the
// original asks for the save disk until it does).
func (e *Engine) saveDirReady() bool {
	if err := os.MkdirAll(e.Cfg.SaveDir, 0o755); err != nil {
		e.fileError("write", e.Cfg.SaveDir)
		return false
	}
	return true
}

// LoadCharacter (name, c, fromGame): read NAME.GUY (or NAME.SAV from a saved
// game) with its items and affects. False when the file is missing.
func (e *Engine) LoadCharacter(name string, c *Character, fromGame bool) bool {
	nm := strings.TrimSuffix(makeFileName(name), "")
	ext := extCharacter
	if fromGame {
		ext = extGameCharacter
	}
	b, err := os.ReadFile(e.savePath(nm + ext))
	if err != nil {
		e.fileError("read", e.savePath(nm+ext))
		return false
	}
	c.setRecord(b)
	c.Items, c.Affects = nil, nil
	if b, err := os.ReadFile(e.savePath(nm + extItems)); err == nil {
		c.Items = readRecords[Item](b, itemRecSize)
	}
	if b, err := os.ReadFile(e.savePath(nm + extAffects)); err == nil {
		c.Affects = readAffects(b)
	}
	e.ClearKeyboard()
	return true
}
