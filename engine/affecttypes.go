package engine

// Affect numbers: the type byte of an affect node, which is also the index
// of its handler in the O0A4:3969 table (affects.go). Named after the
// affect names of DS:2227 where the game shows one, else after what the
// handler does. Numbers without a name have an empty handler.
const (
	affectBless              = 1
	affectCurse              = 2
	affectSticksToSnakes     = 3
	affectDispellingEvil     = 4 // enchanted attackers are hampered
	affectDetectMagic        = 5
	affectElectricityImmune  = 6
	affectGnomeVsFoes        = 7 // a gnome's bonus against its old foes
	affectProtFromEvil       = 8
	affectProtFromGood       = 9
	affectResistCold         = 10
	affectCharmed            = 11
	affectEnlarge            = 12 // magical strength; its data keeps the strength it replaced
	affectBarkskin           = 13
	affectFriendly           = 14 // data: charisma bonus
	affectPoisonDamage       = 15 // slowed poison: a point of damage now and then
	affectReadMagic          = 16
	affectShield             = 17
	affectHalfElfResistance  = 18
	affectFindTraps          = 19
	affectResistFire         = 20
	affectSilenced           = 21
	affectSlowPoison         = 22
	affectSpiritualHammer    = 23
	affectDetectInvisibility = 24
	affectInvisible          = 25
	affectDwarfVsFoes        = 26 // a dwarf's bonus against its old foes
	affectBefuddled          = 27
	affectMirrorImage        = 28 // data: images in the high nibble
	affectEnfeebled          = 29 // a quarter less damage
	affectCoughing           = 30
	affectNausea             = 31
	affectDispelEvil         = 32
	affectBlind              = 33
	affectDiseased           = 34
	affectConfused           = 35
	affectCursed             = 36
	affectBlinking           = 37
	affectStrengthSpell      = 38
	affectHaste              = 39
	affectStinkingCloud      = 40 // the caster's cloud; ending it clears the cloud
	affectProtNormalMissiles = 41
	affectSlow               = 42
	affectWeakened           = 43
	affectWasting            = 44
	affectProtFromEvil10     = 45
	affectProtFromGood10     = 46
	affectDwarfDodges        = 47 // giant kin hit a dwarf less
	affectGnomeDodges        = 48
	affectPrayer             = 49
	affectFireShieldHot      = 50
	affectImmobile           = 51
	affectHeld               = 52
	affectAsleep             = 53
	affectFireShieldCold     = 54
	affectPoisoned           = 55
	affectInvisibleRing      = 56
	affectGlobe              = 57
	affectStoningGaze        = 58
	affectGazeReflectable    = 59 // the stoning gaze can be turned by a mirror
	affectNeedsPlus3Weapon   = 60
	affectFireResistRing     = 61
	affectHealingOverTime    = 62
	affectMinorGlobe         = 63
	affectPoisonSting        = 64 // data on a victim: extra save bonus
	affectStoningTouch       = 65
	affectSwallowsWhole      = 66
	affectBreathFireClose    = 67
	affectFeebleminded       = 68
	affectInvisibleToAnimals = 69
	affectConfusingGaze      = 70
	affectHighlighted        = 71
	affectMirror             = 72
	affectDragonBreathImmune = 73
	affectMagicalBoots       = 74
	affectBaneWeapon         = 75
	affectDiseaseImmune      = 76
	affectBerserk            = 77
	affectMagicalStone       = 78
	affectFireHeals          = 79
	affectBreathFire         = 80
	affectSpitAcid           = 81
	affectFearAura           = 82
	affectBreathWeapon       = 83 // handler set by the magic overlay (O113)
	affectPoisonStingMinus2  = 86
	affectParalysingTouch    = 88
	affectMagicalCloak       = 89 // the first blow each round misses
	affectBoltsHarmless      = 90
	affectCloudkill          = 91 // the caster's cloud; ending it clears the cloud
	affectFearless           = 92
	affectFireImmune         = 93
	affectHardyAll           = 94 // constitution bonus against paralysis, rods and spells
	affectElfResistance      = 95
	affectNoCharmSleep       = 96
	affectConSavingBonus     = 97 // constitution bonus against rods and spells
	affectColdImmune         = 98
	affectPoisonImmune       = 99
	affectDelayedBlast       = 100
	affectPlainWeaponsImmune = 103
	affectBreathWeapon2      = 104 // handler set by the magic overlay (O113)
	affectRangerVsGiants     = 105
	affectStunned            = 106
	affectBerserkFoe         = 107
	affectAlwaysInvisible    = 108
	affectLaidOnHands        = 109
	affectPaladinCures       = 110 // the week's cures are used up
	affectFrightened         = 111
	affectZapsAttackers      = 112
	affectGiantStrength      = 113 // magical strength like affectEnlarge
	itemAffectHandler        = 120 // the table's last entry: a readied item's affect
	affectHandlerCount       = 121
)

// Damage types (the dmgType bits).
const (
	dmgFire        = 1
	dmgCold        = 2
	dmgElectricity = 4
	dmgMagic       = 8
	dmgAcid        = 16
	dmgBreath      = 32
	dmgType64      = 64
)

// Saving throw kinds (Saves index).
const (
	saveVsParalysis = 0
	saveVsPetrify   = 1
	saveVsRod       = 2
	saveVsBreath    = 3
	saveVsSpell     = 4
)

// Creature flags (CreatureFlags, byte 349): what kind of foe it is.
const (
	creatureEnchanted   = 1
	creatureCharmable   = 2  // Charm Monsters works on it
	creatureGiantKin    = 4  // dwarves dodge it
	creatureRangerFoe   = 8  // rangers do extra damage
	creatureSnake       = 16 // Snake Charm works on it
	creatureGnomeFoe    = 32 // gnomes dodge it
	creatureDwarfFoe    = 128
	creatureAnimalsSeen = 69 // the kinds invisibility to animals works for
)

// Monster flags (MonsterFlags, byte 350).
const (
	monsterBane       = 1 // a bane weapon does 6d6 against it
	monsterPersonLike = 2 // Charm Person works on it
	monsterGnomeFoe   = 4
)

// Control values.
const (
	controlCharmed = 179
	controlBerserk = 178
	controlHired   = 127 // above: a hired NPC
)

// Character status values beyond damage.go's.
const statusDisintegrated = 8

// affectNameOf: the name the game shows for the affect, "" for none.
func affectNameOf(typ byte) string {
	if int(typ) < len(affectDisplayNames) {
		return affectDisplayNames[typ]
	}
	return ""
}
