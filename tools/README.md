# Parity tools

- `dosparity.py GAMEDIR OUTDIR STEP...` runs the original in the DOSBox-X remote-debug fork
  (headless, no host input, QMP). Steps: `wait:S`, `key:QCODE`, `text:STRING`, `vga:NAME`
  (EGA planes via the fork's `vgadump` → NAME.raw colour indices), `stack`, `status`, `peek:ADDR,N`, `dspeek:OFF,N` (decimal DS offset).
  `DOSPAR_CMD` sets the command (default `START.EXE STING`; add `Wooden` to skip the code wheel).
- `rawpng.py FILE.raw`: 320x200 colour indices to PNG.
- The engine's `cmd/snap` writes the same `.raw` next to its PNGs; diff the two byte for byte.
