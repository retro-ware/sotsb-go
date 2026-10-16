package engine

// Affect numbers: the type byte of an affect node, which is also the index
// of its handler in the O064 table (affects.go). Named after what the
// handler does. The racial and class ones are in create.go:
// affectGnomeVsManSized 18, affectDwarfVsOrc 26, affectVsGiants 47,
// affectGnomeExtra 48, affectConSavingBonus 97, affectElfResistSleep 107,
// affectHalfElfResistance 124, affectRangerVsGiants 134.
const (
	affectBless              = 1
	affectCurse              = 2
	affectSticksToSnakes     = 3
	affectWardEnchanted      = 4 // attackers with creature flag 1 (enchanted) are hampered
	affectKindBane           = 6 // to hit and damage by the target's kind
	affectArmourBonus        = 7
	affectProtFromEvil       = 8
	affectProtFromGood       = 9
	affectResistCold         = 10
	affectCharmed            = 11
	affectEnlarge            = 12 // magical strength; its data keeps the strength it replaced
	affectSuffocating        = 13
	affectCharisma           = 14 // data: charisma bonus
	affectPoisonDamage       = 15 // a point of damage now and then
	affectShield             = 17
	affectResistFire         = 20
	affectSilenced           = 21
	affectSlowPoison         = 22
	affectSpiritualHammer    = 23
	affectDetectInvisibility = 24
	affectInvisible          = 25
	affectStunned            = 27
	affectMirrorImage        = 28 // data: images
	affectQuarterLessDamage  = 29
	affectCoughing           = 30
	affectNausea             = 31
	affectCollapse           = 32 // an animated creature: falls when the spell ends
	affectDazzled            = 33
	affectWeakAndWasting     = 34
	affectConfused           = 35
	affectHampered           = 36
	affectStumbling          = 37
	affectStrengthSpell      = 38
	affectHaste              = 39
	affectStinkingCloud      = 40 // the caster's cloud; ending it clears the cloud
	affectAvoidPlainWeapons  = 41
	affectSlow               = 42
	affectWeakened           = 43
	affectWasting            = 44
	affectProtFromEvil10     = 45
	affectProtFromGood10     = 46
	affectPrayer             = 49
	affectFireBody           = 50 // cold saves +2, unsaved fire doubles
	affectSleep              = 51
	affectParalyzed          = 52
	affectHeld               = 53
	affectIceBody            = 54 // fire saves +2, cold doubles
	affectPoisoned           = 55
	affectFlicker            = 56
	affectEngulfs            = 57
	affectEngulfed           = 58
	affectRegenerating       = 59
	affectMagicWeaponsOnly   = 60
	affectFireDamageReduced  = 61
	affectHealingOverTime    = 62
	affectMinorGlobe         = 63
	affectPoisonSting        = 64
	affectPoisonSting4       = 65
	affectPoisonSting2       = 66
	affectParalysingTouch    = 67
	affectFeebleminded       = 68
	affectHiddenFromKind19   = 69
	affectPoisonStingMinus2  = 70
	affectBlur               = 71
	affectFlicker95          = 72
	affectUnaffectedByType32 = 73
	affectMiscasting         = 74 // half its spells fail; no free attacks as foes step away
	affectVsKind3            = 75
	affectVsKind8            = 76
	affectBerserk            = 77
	affectRecovering         = 78
	affectBurningTouch       = 79
	affectAcidTouch          = 80
	affectHalfDamage         = 81
	affectSaveVsElements     = 82
	affectStoningGaze        = 83 // O0C2 handlers from here (setup_spells)
	affectElectricityHeals   = 84
	affectArrowsScratch      = 85
	affectSpitAcid           = 86
	affectRays               = 87
	affectBreathLightning    = 88
	affectFirstStrike        = 89
	affectBreathAcid         = 90
	affectCloudkill          = 91 // the caster's cloud; ending it clears the cloud
	affectFireHalved         = 93
	affectRangedHalved       = 94
	affectFallsDead          = 95
	affectHugs               = 96
	affectRegrowth           = 98
	affectBacksUp            = 99
	affectRegrows            = 100
	affectRegenerates        = 101
	affectRegrowing          = 102
	affectBurns              = 103
	affectAvoidRanged60      = 104
	affectMagicResist50      = 105
	affectMagicResist15      = 106
	affectNoCharmHold        = 108
	affectNoParalysis        = 109
	affectColdImmune         = 110
	affectNoPoisonParalysis  = 111
	affectFireImmune         = 112
	affectFireReduced        = 113
	affectElectricityHalved  = 114
	affectEdgedHalved        = 115
	affectMagicWeaponsHalved = 116
	affectSlingStones        = 117
	affectColdHalved         = 118
	affectNeedsMagicWeapon   = 119
	affectAvoidsBows         = 120
	affectSpitsAcid          = 121
	affectParalysingGaze     = 122
	affectFireTouch          = 123
	affectNoCharmHoldSleep   = 125
	affectGazeParalyse       = 126
	affectGazeReflectable    = 127 // a mirror readied by the target turns the stoning gaze
	affectBreathFire         = 128
	affectMagicImmune        = 129
	affectSlainByArrow       = 130
	affectBreathFireClose    = 131
	affectThrowLightning     = 132
	affectNoMindAffects      = 133
	affectElectricityImmune  = 135
	affectRooted             = 136
	affectBerserkFoe         = 137
	affectAlwaysInvisible    = 138
	affectEngulfing          = 139
	affectPaladinCures       = 141
	affectRunningAway        = 142
	affectZapsAttackers      = 143
	affectHugging            = 144
	affectDispelEvil         = 145
	affectGiantStrength      = 146 // magical strength like affectEnlarge
	itemAffectHandler        = 147 // the table's last entry: a readied item's affect
	affectHandlerCount       = 148
)

// Damage types (the dmgType bits).
const (
	dmgFire        = 1
	dmgCold        = 2
	dmgElectricity = 4
	dmgMagic       = 8
	dmgAcid        = 16
	dmgType32      = 32
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

// Creature flags (Pad331).
const (
	creatureEnchanted = 1
	creatureGoblinoid = 2
	creatureOrc       = 4
	creatureGiant     = 8
)

// Control values.
const (
	controlCharmed = 179
	controlBerserk = 178
	controlHired   = 127 // above: a hired NPC
)

// Character status values beyond damage.go's.
const statusDisintegrated = 8
