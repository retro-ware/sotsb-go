package engine

// Spell numbers (the spell table, the spellbook and memorised lists).
const (
	spellBless                         = 1
	spellCurse                         = 2
	spellCureLightWounds               = 3
	spellCauseLightWounds              = 4
	spellDetectMagicCleric             = 5
	spellProtectFromEvilCleric         = 6
	spellProtectFromGoodCleric         = 7
	spellResistCold                    = 8
	spellBurningHands                  = 9
	spellCharmPerson                   = 10
	spellDetectMagicMU                 = 11
	spellEnlarge                       = 12
	spellReduce                        = 13
	spellFriends                       = 14
	spellMagicMissile                  = 15
	spellProtectFromEvilMU             = 16
	spellProtectFromGoodMU             = 17
	spellReadMagic                     = 18
	spellShield                        = 19
	spellShockingGrasp                 = 20
	spellSleep                         = 21
	spellFindTraps                     = 22
	spellHoldPersonCleric              = 23
	spellResistFire                    = 24
	spellSilence15Radius               = 25
	spellSlowPoison                    = 26
	spellSnakeCharm                    = 27
	spellSpiritualHammer               = 28
	spellDetectInvisibility            = 29
	spellInvisibility                  = 30
	spellKnock                         = 31
	spellMirrorImage                   = 32
	spellRayOfEnfeeblement             = 33
	spellStinkingCloud                 = 34
	spellStrength                      = 35
	spellAnimateDead                   = 36
	spellCureBlindness                 = 37
	spellCauseBlindness                = 38
	spellCureDisease                   = 39
	spellCauseDisease                  = 40
	spellDispelMagicCleric             = 41
	spellPrayer                        = 42
	spellRemoveCurse                   = 43
	spellBestowCurseCleric             = 44
	spellBlink                         = 45
	spellDispelMagicMU                 = 46
	spellFireball                      = 47
	spellHaste                         = 48
	spellHoldPersonMU                  = 49
	spellInvisibility10Radius          = 50
	spellLightningBolt                 = 51
	spellProtectFromEvil10Radius       = 52
	spellProtectFromGood10Radius       = 53
	spellProtectFromNormalMissiles     = 54
	spellSlow                          = 55
	spellRestoration                   = 56
	spellSpell57                       = 57
	spellCureSeriousWounds             = 58
	spellSpell59                       = 59
	spellSpell60                       = 60
	spellSpell61                       = 61
	spellSpell62                       = 62
	spellSpell63                       = 63
	spellSpell64                       = 64
	spellSpell65                       = 65
	spellCauseSeriousWounds            = 66
	spellNeutralizePoison              = 67
	spellPoison                        = 68
	spellProtectFromEvil10RadiusCleric = 69
	spellSticksToSnakes                = 70
	spellCureCriticalWounds            = 71
	spellCauseCriticalWounds           = 72
	spellDispelEvil                    = 73
	spellFlameStrike                   = 74
	spellRaiseDead                     = 75
	spellSlayLiving                    = 76
	spellDetectMagicDruid              = 77
	spellEntangle                      = 78
	spellFaerieFire                    = 79
	spellInvisibilityToAnimals         = 80
	spellCharmMonsters                 = 81
	spellConfusion                     = 82
	spellDimensionDoor                 = 83
	spellFear                          = 84
	spellFireShield                    = 85
	spellFumble                        = 86
	spellIceStorm                      = 87
	spellMinorGlobeOfInvulnerability   = 88
	spellSpell89                       = 89
	spellSpell90                       = 90
	spellCloudKill                     = 91
	spellConeOfCold                    = 92
	spellFeeblemind                    = 93
	spellHoldMonsters                  = 94
	spellSpell95                       = 95
	spellSpell96                       = 96
	spellSpell97                       = 97
	spellSpell98                       = 98
	spellSpell99                       = 99
	spellBestowCurseMU                 = 100
)

// Spell classes in the spell table.
const (
	spellClassCleric = iota
	spellClassDruid
	spellClassMage
)

// Spell table record fields (spellRec).
const (
	spellFieldClass         = 0
	spellFieldLevel         = 1
	spellFieldRange         = 2
	spellFieldRangePerLevel = 3
	spellFieldAmount        = 4
	spellFieldAmountPerLvl  = 5
	spellFieldLasts         = 6  // the range is at least 1
	spellFieldTargets       = 7  // 0: combat only; how the targets are chosen
	spellFieldSaveMode      = 8  // what a saving throw does
	spellFieldSaveType      = 9  // the saving throw kind
	spellFieldAffect        = 10 // the affect the spell lays
	spellEndsTurn           = 11 // casting it ends the turn
)

// spellEntry is one 16-byte record of the spell table.
type spellEntry [16]byte

func (s spellEntry) class() byte { return s[0] }
func (s spellEntry) level() byte { return s[1] }

// spellTable: the record of spell n is spellTable[n].
var spellTable = func() (t [len(spellTableData)]spellEntry) {
	for i, r := range spellTableData {
		t[i] = spellEntry(r)
	}
	return
}()
