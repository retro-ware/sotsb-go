# cotab-go — Curse of the Azure Bonds on Go/Ebiten

Source of truth: the byte-perfect Turbo Pascal reconstruction (github.com/memmaker/cotab-re).
Infrastructure (EGA screen, DAX reader, Ebiten host) from por-go; game logic ported per procedure
from the Pascal. Rules: ~/notes/c-to-go.md and goplay INTEGRATION.md (no exe reads at runtime,
no hex/magic numbers, readable names).

- [x] M1 `-import` via internal/gameimport (folders, zip, ISO/BIN/CUE, GOG installer, loose files); COVER.PNG
- [x] M2 `tables` package: every data table the engine reads, named, generated from START.EXE by `cmd/tablegen` (go:generate); no `e.X` at runtime
- [x] M3 boot: title screen, main menu (START main, title_screen, startGameMenu) — party menu pixel-identical to DOSBox
- [x] M4 characters: create, load/save, party screens
  - [x] creation (race..alignment, stat roll, silent training, icon editor, save) — every screen pixel-identical, .GUY byte-identical (pointer bytes aside), seeded RNG
  - [x] Add character (Curse .GUY files) with the party rules
  - [x] Pool of Radiance / Hillsfar import (O0A1 0465 roster, 22A0 import_char01, 14B3 load_player, 1F35 Hillsfar merge, 13AE): Pool import checked with a Pool save; Hillsfar untested (no .HIL file at hand)
  - [x] Drop / Modify / View / Remove screens and the party menu with a loaded character: pixel-identical to DOSBox (Change Class only for eligible humans: not offered to the test cleric, as in the original)
  - [x] Load / Save game (O0A1 loadGame/SaveGame): savgamX.dat byte-identical but for stack garbage in unused name slots; loading pixel-identical
- [x] M5 world: 3D view (WALLDEF/8X8D/SKY), area map, ECL interpreter (CotAB opcode set)
  - [x] ECL VM (O04C): operands, packed text, text variables, getVar/setVar with the character values, compares, calls
  - [x] adventuring loop (O03C:3607/3677/3758), command bar, steps, doors (bash/pick/knock), status line, party summary
  - [x] wilderness map cursor (O109, places from DS:0A4A/0A6A by area word 417, blink in displayInput O104)
  - [x] view: adventure frame, EGA sky (bands, sun/moon, SKY 252 floor), 3D view and overhead map, pictures (PIC/SPRIT/HEAD/BODY/bigpic)
  - [x] commands 0..35, 37..64 except COMBAT (36) and PROGRAM (56); first scene .. overhead map pixel-identical to DOSBox
  - [x] COMBAT (36: shops O048, temple O042, after-combat O045, combat O057) and PROGRAM (56: party menu, ending, camp, quit)
  - [x] after-combat O045 (outcome, experience, hired shares, treasure menu, take money/items): ported from CotAB, replacing Pool ovr06 and the party purse
  - [x] the duel (O04C:1B9A)
  - [x] the ending O0AC (text, animated pictures, fireworks)
  - [x] the sound calls (R699, M8)
  - [x] PARLAY (O03C:27B7 talk_style): its char prompt becomes the one-character string " ", as in the port
- [x] M6 camp, shops, temples, training (paladin/ranger etc.)
  - [x] shop (O048) with money (O0BA: pool, share, take, appraise) and View (O0AF viewPlayer, items menu, trade/drop coins, paladin heal/cure): pixel-identical
  - [x] camp (O09D) and resting (O0B6): menus, magic, memorize, scribe, display, rest, alter, fix: pixel-identical but for campfire frames
  - [x] temple (O042)
  - [x] item functions (O0AF): ready with the item effects (1B04), trade, halve, join, sell (coins + pool), identify (coins or pool)
  - [x] O0AF items menu (1583) and Use (2478: item spell & 127, scroll class check with the thief chance, "oops!")
  - [x] training halls: game word 680 (set by the hall's script, 255 after the ending) offers Train and Change Class in the party menu (O0A7:01CD), train_player O0A7:4D07
- [x] M7 combat
  - [x] rules (O0E3): affect groups, saving throw, damage, add/kill/heal/revive, poison and stinking clouds
  - [x] affect handlers (O064 setup_spells2, 148 entries incl. O08D's engulf/hug/rays and the item call)
  - [x] clouds (O0C2 noxious/poisonous cloud, O064 clearing)
  - [x] O05D combat setup: CotAB wilderness map (cliffs, trees, scatter), dungeon map with rubble, placement, blocks
  - [x] O057 combat loop: turn order, camp_menu/combat_menu (extended keys step at once), move/attack, delay menu, speed, quick; battle01 with the 15-round limit (reset by every attack)
  - [x] O08D aiming (manual aim, target picking by hit dice/area, ranged rule), casterLevel (paladin/ranger offsets), god_intervene (debug word "Wooden")
  - [x] O0C2 spells 1-100 (setup_spells) and the breath/gaze/spit handlers it adds (affects 83, 86, 88, 90, 126, 128, 131, 132); cone (17CF), bolt (3986/3A44), hold (1B24), dispel with clouds (31E4)
  - [x] O05A computer turn: every procedure checked; canCast reads the spell table and saves area victims, the
    step check (073B) uses O121:0D04's cloud flags, attack loop bandages first, DS:75FE kept as aiEngaged
  - [x] No START.EXE reads at runtime: the last Pool-address reads (status names, facing names, sky colour,
    item words, class names, line directions) use the generated CotAB tables; Pool-only code removed (area map,
    portrait chooser, endgame fade, Pool RandomItem); Modify Character is CotAB's O0A7:2830 (name editing, HP)
  - [x] O121 combat map: audited and relabelled; death flicker (9 frames, icon slot 0x18 = DS:66B0), redrawChar only while b7600
  - [x] O11C line walker (CotAB DDA, not Bresenham), nearSort tie-break fixed, LOS/arc/buildNear verified
  - [x] O12A icons/tiles: Load24x24Set counts (DungCom 25, RandCom 6) and n honoured
  - [x] O08D turn code and O0C2 spell bodies renamed and audited; O045 after-combat ported
- [x] M8 sound: R699 (Pool's 08AE driver moved 189 bytes) as Go data from CotAB's START.EXE; PlaySound/soundInit/Ctrl-S per R699:0020, 00D4, R670; DAX loads cut the song (R5BD)
- [x] M9 goplay: games.js, deploy.sh, README tables, hardware cursor n/a (no pointer), wasm build, box art
- [x] M10 publish (GitHub + deploy)

## Parity method
`tools/dosparity.py` drives the original in the DOSBox-X remote-debug fork (added QMP `vgadump`,
`memwrite`; `status` reports CS:IP); `cmd/snap game - KEYS` drives the port with the same keys;
`COTAB_SEED` / `seed:N` set Turbo Pascal's RandSeed (DS 18416) on both sides. Screens are compared as
raw EGA colour indices.

## Known placeholders

## Deliberate deviations (obvious bugs fixed, per c-to-go.md)
- load_player (O0A1:14B3) raises a Pool character's strength to the race's exceptional strength maximum (it compares that
  maximum with the strength); the port caps the exceptional strength at it instead.
- The overlay loader's driver stop/restart around overlay calls (O03C:17E4, START) is not modelled: the port has no overlays.
- Fireworks sparks (O0AC:04F7) wait 15 ms per step: the original was paced only by its slow BIOS pixel calls.
- maxHitPoints / setRolledHP (O0A7:4793, modifyPlayer): a character with no class level returns instead of a divide error.
- aiPickWeapon (O05A:1673): ready_Item is not called with a nil weapon when the off hand is empty or cursed.
- Shops with the normal price factor (16): the original's Buy (O048:0474) has no case for it and charges an
  unset local, which holds the menu colour pushed by the shop menu's displayInput (15) — every item costs
  15 gold. The port charges the price the list shows.
- Character sheet movement (O0AF): the original doubles for affect 42 (slow) and halves for 39 (haste), the
  reverse of combat (O064:0BA5/101B, 39 ages and doubles) and encounters (O04C:20E5); the port's sheet
  doubles for haste (39) and halves for slow (42).
- Ice body (O064:1243) tests DS:5D0B (unrelated) where fire body tests the saving throw; the port tests the
  saving throw (DS:6FC2) for both.
- Constitution save bonus (O064:1E42) below 4 adds an unset local; the port adds nothing.
- ENCOUNTER with a digit that is no menu item read past its reaction table; the port asks again.
- Cones (O0C2:17CF) hand lineOfSight an unset local as the range; the port gives no range limit (walls still
  cut the cone).
- Kept as in the original (not bugs to the player): Dispel Magic looks for Cloudkill's squares in Stinking
  Cloud's table run on (DS:27D9); Entangle's duration comes from bytes past the spell table (6 + 3/level);
  once Fear's cone meets a combatant twice, later ones on the line are left out.
- The shop's "Excuse me" text: unchanged; DIVIDE by zero in scripts gives 0 instead of a runtime error.
- Ring of wizardry put away (O0AF:1B04): the original looks up each memorized spell's class in the spell table
  without masking bit 7 (a spell being memorized); the port masks it, so such spells count too.
- Sell (O0AF:27C2) compares a byte local read as a word (the room left) with the platinum; the port uses the byte.
- Experience (O045:0037) divides by the party count less the idle members; when that is 0 the original
  stops with a divide error, the port does not divide.
- Taking loot (O045:0DFC) with nothing selected handed a nil item to the shop's overloaded check; the port stops.
