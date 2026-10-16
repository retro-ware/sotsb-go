"""paritycmp.py GAMEDIR SAVEDGAMEDIR KEYS [WAIT]: run the port (cmd/snap game -) and the original (dosparity.py)
with the same keys and report the pixel differences of every screen.

KEYS uses snap's notation: letters, ^ v < > arrows, | Enter, %e Esc, _ a pause. The original gets
`START.EXE STING Wooden` and the Play key first; GAMEDIR is the port's data folder, SAVEDGAMEDIR the
original's (save path C:\\). Screens: /tmp/pc_port/game_NN.raw and /tmp/pc_dos/sNN.raw."""
import os, shutil, subprocess, sys

here = os.path.dirname(os.path.abspath(__file__))
port_dir, dos_dir, keys = sys.argv[1], sys.argv[2], sys.argv[3]
wait = sys.argv[4] if len(sys.argv) > 4 else '3'

# Tokens as snap counts them (one shot each).
toks = []
i = 0
while i < len(keys):
    if keys[i] == '%':
        toks.append(keys[i:i + 2]); i += 2
    else:
        toks.append(keys[i]); i += 1

qmp = {'^': 'up', 'v': 'down', '<': 'left', '>': 'right', '|': 'ret', '%e': 'esc', ' ': 'spc'}
steps = ['wait:8', 'seed:' + os.environ.get('COTAB_SEED', '1'), 'key:p', 'wait:3']
for n, t in enumerate(toks, 1):
    if t != '_':
        steps.append('key:' + qmp.get(t, t.lower()))
    steps += ['wait:' + wait, 'vga:s%02d' % n]

steps += [x for x in os.environ.get('DOSPAR_EXTRA', '').split() if x]
for d in ('/tmp/pc_port', '/tmp/pc_dos'):
    shutil.rmtree(d, ignore_errors=True)
    os.makedirs(d)
env = dict(os.environ, DOSPAR_CMD='START.EXE STING Wooden', SDL_VIDEODRIVER='dummy', DOSBOX_NO_HOST_INPUT='1')
dos = subprocess.Popen([sys.executable, os.path.join(here, 'dosparity.py'), dos_dir, '/tmp/pc_dos'] + steps,
                       env=env, stdout=open('/tmp/pc_dos.log', 'w'), stderr=subprocess.STDOUT)
penv = dict(os.environ, SDL_VIDEODRIVER='dummy', COTAB_SEED=os.environ.get('COTAB_SEED', '1'))
port_copy = '/tmp/pc_portdata'
shutil.rmtree(port_copy, ignore_errors=True)
shutil.copytree(port_dir, port_copy)  # the game writes and deletes files
subprocess.run(['go', 'run', './cmd/snap', port_copy, '/tmp/pc_port', 'game', '-', keys],
               cwd=os.path.dirname(here), env=penv, stdout=subprocess.DEVNULL)
dos.wait()
for n, t in enumerate(toks, 1):
    a, b = '/tmp/pc_port/game_%02d.raw' % n, '/tmp/pc_dos/s%02d.raw' % n
    if not (os.path.exists(a) and os.path.exists(b)):
        print('%2d %-3s missing' % (n, t)); continue
    pa, pb = open(a, 'rb').read(), open(b, 'rb').read()
    diff = sum(1 for x, y in zip(pa, pb) if x != y)
    print('%2d %-3s %s' % (n, t, diff if diff else 'ok'))
