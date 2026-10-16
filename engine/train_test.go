package engine

import "testing"

// Modify Character keeps abilities within raceAbilityLimits, indexed by this
// game's race ids (an elf may reach dexterity 19; Pool's table row 1 is a dwarf).
func TestModifyUsesRaceAbilityLimits(t *testing.T) {
	e := &Engine{}
	c := &Character{}
	c.Stats[statDex].Value = 18
	e.modifyRaise(c, statDex, raceAbilityLimits[1], 0) // 1 Elf
	if c.Stats[statDex].Value != 19 {
		t.Fatalf("elf dex = %d, want 19", c.Stats[statDex].Value)
	}
	c.Stats[statInt].Value = 9
	e.modifyLower(c, statInt, raceAbilityLimits[1], clsFighter, 0)
	if c.Stats[statInt].Value != 8 { // elf int minimum 8
		t.Fatalf("elf int = %d, want 8", c.Stats[statInt].Value)
	}
}
