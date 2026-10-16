"""rawcmp.py KEYS: compare /tmp/pc_port/game_NN.raw with /tmp/pc_dos/sNN.raw (what paritycmp prints)."""
import os, sys
keys = sys.argv[1]
toks = []
i = 0
while i < len(keys):
    if keys[i] == '%':
        toks.append(keys[i:i + 2]); i += 2
    else:
        toks.append(keys[i]); i += 1
out = []
for n, t in enumerate(toks, 1):
    a, b = '/tmp/pc_port/game_%02d.raw' % n, '/tmp/pc_dos/s%02d.raw' % n
    if not (os.path.exists(a) and os.path.exists(b)):
        out.append('%d:miss' % n); continue
    pa, pb = open(a, 'rb').read(), open(b, 'rb').read()
    d = sum(1 for x, y in zip(pa, pb) if x != y)
    out.append('%d%s:%s' % (n, t, d if d else 'ok'))
print(' '.join(out))
