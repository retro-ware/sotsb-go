package engine

import (
	"os"
	"strings"
)

// Saved characters (unit O0F8): NAME.WHO holds the record, NAME.STF the
// items (67 bytes each, a bag followed by its contents), NAME.SFX the
// affects (9 bytes each). A saved game keeps its characters as .SAV files
// with the same .STF and .SFX companions. The original writes its memory
// pointers into the files; the port writes zeros there (they are never
// read back).

const (
	extCharacter     = ".who"
	extItems         = ".stf"
	extAffects       = ".sfx"
	extGameCharacter = ".sav"
	maxFileName      = 8
)

// removeCharacterFiles = O0F8:0C33 (c): the roster files of a character
// now kept in the saved game.
func (e *Engine) removeCharacterFiles(c *Character) {
	base := makeFileName(c.Name())
	for _, ext := range []string{extCharacter, extItems, extAffects} {
		os.Remove(e.savePath(base + ext))
	}
}

// SaveCharacter = O0F8:0DA0 (c, name): name "" saves to the roster under
// the character's name (asking before overwriting; a new file name also
// becomes the character's name), else name.sav for a saved game.
func (e *Engine) SaveCharacter(c *Character, name string) {
	e.addSource = fromSilverBlades
	if !e.saveDirReady() {
		return
	}
	base, ext := makeFileName(removeSubstr(c.Name(), ".")), extCharacter
	if name != "" {
		base, ext = name, extGameCharacter
	}
	for name == "" && e.fileExists(e.savePath(base+ext)) {
		if e.YesNo(colourYellow, colourGreen, colourWhite, "Overwrite "+base+"? ") != 'N' {
			break
		}
		base = ""
		for base == "" {
			base = e.InputString("New file name: ", colourGreen, maxFileName)
		}
		c.SetName(base)
	}
	if err := os.WriteFile(e.savePath(base+ext), c.record(), 0o644); err != nil {
		e.fileError("write", e.savePath(base+ext))
		return
	}
	os.Remove(e.savePath(base + extItems))
	if len(c.Items) > 0 {
		os.WriteFile(e.savePath(base+extItems), writeItems(c.Items), 0o644)
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

// saveDirReady = O0F8:04F3: make sure the save directory exists (the
// original asks for the save disk until it does).
func (e *Engine) saveDirReady() bool {
	if err := os.MkdirAll(e.Cfg.SaveDir, 0o755); err != nil {
		e.fileError("write", e.Cfg.SaveDir)
		return false
	}
	return true
}

// LoadCharacter = O0F8:1CEB (file, c): a character from a Silver Blades
// file (.who, or .sav of a saved game) or, when adding from Curse of the
// Azure Bonds, its .guy record made into a Silver Blades one; then its
// items (Silver Blades files only) and affects, and the values the items
// and classes give are worked out again. A paladin may cure disease once
// more; a paladin from Curse also gets the protection from evil and a
// ranger the bonus against giants that Silver Blades keeps as affects.
func (e *Engine) LoadCharacter(file string, c *Character) bool {
	e.DisplayString(0, statusRow, colourGreen, "Loading...Please Wait")
	b, err := os.ReadFile(e.savePath(file))
	if err != nil {
		e.fileError("read", e.savePath(file))
		return false
	}
	*c = Character{}
	paladin := func() bool {
		return int8(c.Levels[clsPaladin]) > 0 || c.OldClass == clsPaladin && oldClassUsable(c)
	}
	ranger := func() bool {
		return int8(c.Levels[clsRanger]) > 0 || c.OldClass == clsRanger && oldClassUsable(c)
	}
	if e.addSource == fromSilverBlades {
		c.setRecord(b)
		if paladin() {
			c.PaladinCures = 1
		}
	} else {
		c.fromCurse(readCurseRecord(b))
		if ranger() {
			e.AddAffect(c, affectRangerVsGiants, 0, permanent, 0)
		}
		if paladin() {
			c.PaladinCures = 1
			e.AddAffect(c, affectProtFromEvil, 0, permanent, 0)
		}
	}
	base := strings.TrimSuffix(file, filepathExt(file))
	if e.addSource == fromSilverBlades {
		if b, err := os.ReadFile(e.savePath(base + extItems)); err == nil {
			c.Items = readItems(b, itemFileRecSize)
		}
	}
	// The original links the .sfx affects after the ones added above
	// through a pointer it never set; the port simply appends them.
	if b, err := os.ReadFile(e.savePath(base + extAffects)); err == nil {
		c.Affects = append(c.Affects, readAffects(b)...)
	}
	e.ClearKeyboard()
	e.ApplyItems(c)
	e.recalcClassBonuses(c)
	return true
}
