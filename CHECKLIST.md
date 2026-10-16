# sotsb-go — Secret of the Silver Blades on Go/Ebiten

Source of truth: the byte-perfect Turbo Pascal reconstruction in ~/Projects/sotsb-re (894/894 procedures,
START.EXE and GAME.OVR rebuilt byte-identical). Infrastructure and the Gold Box engine shape come from
cotab-go (itself from por-go); every function is checked against its SotSB counterpart and changed to it.
Rules: ~/notes/c-to-go.md and goplay INTEGRATION.md: idiomatic Go, no CPU/memory emulation, no low-level
memory manipulation, no far pointers, no hex or magic numbers, meaningful names everywhere.

Porting aids (in sotsb-re): `tools/procmatch.py` pairs SotSB procedures with CotAB ones (data/cotab_map.tsv:
628 of 840 have a counterpart, 117 identical in shape); `tools/dsmap.py` maps CotAB data addresses to SotSB
(data/ds_map.tsv); `tools/xref.py` gives the work list per Go function (data/port_worklist.tsv) and the SotSB
procedures no Go function covers yet (data/port_uncovered.tsv). `cmd/tablefind` finds CotAB tables in SotSB's
data segment by content.

- [x] M0 repo: cotab-go infrastructure copied, module `sotsb`, `cmd/sotsb`
- [x] M1 `-import` via internal/gameimport (folders, zip, ISO/BIN/CUE, GOG installer, loose files): markers
  BLADES.CFG + TITLE.DAX + GAME.OVR; COVER.PNG = TITLE painting 2 with the Silver Blades logo 4 at (5,5)
  (O047:0571). Tested: GOG folder, zip in a subfolder, loose files, setup_secret_of_the_silver_blades_1.30_(28187).exe
- [ ] M2 tables: every data table from SotSB's START.EXE (`exe.LoadBlades`, DS paragraph 3554), named, by tablegen
- [ ] M3 boot: title (O047), credits (O047:029C), main menu, frames (R19D)
- [ ] M4 characters: create, load/save (SotSB record), party screens, import from Curse
- [ ] M5 world: 3D view, area map, ECL interpreter (SotSB opcode set), adventuring loop
- [ ] M6 camp, shops, temples, training
- [ ] M7 combat
- [ ] M8 sound
- [ ] M9 goplay: games.js, deploy.sh build rule, README tables, wasm build
- [ ] M10 publish (public deploy: only with the user's go-ahead)

## Parity method
As cotab-go: `tools/dosparity.py` drives the original in DOSBox-X (QMP vgadump/memwrite), `cmd/snap` drives the
port with the same keys; screens compared as EGA colour indices.

## Deliberate deviations
