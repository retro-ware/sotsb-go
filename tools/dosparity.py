"""dosparity.py GAMEDIR OUTDIR STEP...: run the original game in DOSBox-X (headless, no host
input, QMP on 127.0.0.1:4444), apply steps and save screen dumps for parity checks.

Steps: wait:SECONDS  key:NAME (QMP key name, pressed ~150 ms)  text:STRING  shot:NAME
The game runs from a temporary copy of GAMEDIR, started as START.EXE STING (no title)."""
import os, shutil, subprocess, sys, tempfile, time
sys.path.insert(0, os.path.expanduser('~/tools/dos-mcp/src'))
from dos_mcp.qmp import QmpClient
from dos_mcp import tools
from dos_mcp.errors import QmpTransportError

DOSBOX = os.path.expanduser('~/tools/dosbox-x-remotedebug/src/dosbox-x')


# EGA palette (the 64-colour rgbRGB values of the attribute palette entries).
def ega_rgb(v):
    r = 0x55 * (((v >> 2) & 1) * 2 + ((v >> 5) & 1))
    g = 0x55 * (((v >> 1) & 1) * 2 + ((v >> 4) & 1))
    b = 0x55 * ((v & 1) * 2 + ((v >> 3) & 1))
    return r, g, b


def save_ega(dumpfile, base):
    """Decode a 320x200 16-colour EGA screen (mode 0Dh) from a vgadump to base.png and base.raw."""
    data = open(dumpfile, 'rb').read()
    nl = data.index(b'\n')
    hdr = dict(kv.split('=') for kv in data[:nl].decode().split())
    mem = data[nl + 1:]
    start, offset = int(hdr['start']), int(hdr['offset'])
    pal = [int(x) for x in hdr['pal'].split(',')]
    stride = offset * 2 if offset else 40
    idx = bytearray(320 * 200)
    for y in range(200):
        for xb in range(40):
            a = (start + y * stride + xb) * 4
            p0, p1, p2, p3 = mem[a], mem[a + 1], mem[a + 2], mem[a + 3]
            for bit in range(8):
                m = 0x80 >> bit
                c = (p0 & m and 1) | (p1 & m and 2) | (p2 & m and 4) | (p3 & m and 8)
                idx[y * 320 + xb * 8 + bit] = c
    open(base + '.raw', 'wb').write(bytes(idx))
    try:
        from PIL import Image
        im = Image.new('RGB', (320, 200))
        im.putdata([ega_rgb(pal[c]) for c in idx])
        im.save(base + '.png')
    except ImportError:
        pass


_ds = [None]


def game_ds(client):
    """The game's data segment: found by the camp menu text at DS:042A."""
    if _ds[0] is None:
        mem = b''
        for a in range(0, 0xA0000, 0x1000):
            mem += bytes.fromhex(tools.read_memory(client, a, 0x1000, allow_running=True)['data_hex'])
        sig = b'\x23Save View Magic Rest Alter Fix Exit'
        i = mem.find(sig)
        while i >= 0 and (i - 0x042A) % 16 != 0:
            i = mem.find(sig, i + 1)
        _ds[0] = (i - 0x042A) // 16 if i >= 0 else tools.status(client)['cpu']['ds']
    return _ds[0]


def main():
    game, out, steps = sys.argv[1], sys.argv[2], sys.argv[3:]
    os.makedirs(out, exist_ok=True)
    tmp = tempfile.mkdtemp(prefix='cotab-dos-')
    for f in os.listdir(game):
        p = os.path.join(game, f)
        if os.path.isfile(p):
            shutil.copy(p, tmp)
            os.chmod(os.path.join(tmp, f), 0o644)  # TP opens data files read/write
        elif os.path.isdir(p):
            shutil.copytree(p, os.path.join(tmp, f))
            for q in os.listdir(os.path.join(tmp, f)):
                os.chmod(os.path.join(tmp, f, q), 0o644)
    conf = os.path.join(tmp, 'dosbox.conf')
    with open(conf, 'w') as f:
        f.write('[sdl]\noutput=surface\n[dosbox]\nmachine=svga_s3\nmemsize=16\nqmpserver=true\nqmpport=4444\n'
                '[cpu]\ncore=normal\ncycles=max\n[dos]\nxms=true\nems=true\n[autoexec]\nmount c "%s"\nc:\n%s\n' % (tmp, os.environ.get('DOSPAR_CMD', 'START.EXE STING')))
    env = dict(os.environ, SDL_VIDEODRIVER=os.environ.get('DOSPAR_VIDEO', 'dummy'), DOSBOX_NO_HOST_INPUT='1')
    proc = subprocess.Popen([DOSBOX, '-conf', conf], env=env, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    try:
        client = QmpClient(port=4444, timeout=35)
        for _ in range(100):
            try:
                tools.status(client)
                break
            except Exception:
                time.sleep(0.2)
        def run(s):
            kind, _, arg = s.partition(':')
            if kind == 'wait':
                time.sleep(float(arg))
            elif kind == 'key':
                tools.key_event(client, arg, down=True)
                time.sleep(0.15)
                tools.key_event(client, arg, down=False)
                time.sleep(0.15)
            elif kind == 'text':
                for ch in arg:
                    k = ch.lower() if ch != ' ' else 'spc'
                    tools.key_event(client, k, down=True)
                    time.sleep(0.12)
                    tools.key_event(client, k, down=False)
                    time.sleep(0.12)
            elif kind == 'text80':
                tools.pause(client)
                r = tools.read_memory(client, 0xB8000, 4000)
                tools.resume(client)
                b = bytes.fromhex(r['data_hex'])
                for row in range(25):
                    line = ''.join(chr(c) if 32 <= c < 127 else ' ' for c in b[row * 160:row * 160 + 160:2]).rstrip()
                    if line:
                        print(line)
            elif kind == 'peek':
                a, n = arg.split(',')
                r = tools.read_memory(client, int(a, 16), int(n), allow_running=True)
                print('peek', a, r['data_hex'])
            elif kind == 'dspeek':
                # dspeek:OFF,N: N bytes at DS:OFF (decimal)
                off, n = (int(x) for x in arg.split(','))
                ds = game_ds(client)
                r = tools.read_memory(client, ds * 16 + off, n, allow_running=True)
                print('dspeek', arg, r['data_hex'], flush=True)
            elif kind == 'dsptr':
                # dsptr:OFF,DELTA,N: the far pointer at DS:OFF (decimal), then N bytes at pointer+DELTA
                off, delta, n = (int(x) for x in arg.split(','))
                ds = game_ds(client)
                p = bytes.fromhex(tools.read_memory(client, ds * 16 + off, 4, allow_running=True)['data_hex'])
                lin = (p[2] | p[3] << 8) * 16 + (p[0] | p[1] << 8) + delta
                r = tools.read_memory(client, lin, n, allow_running=True)
                print('dsptr', arg, 'ds=%04X' % ds, r['data_hex'])
            elif kind == 'seed':
                # Turbo Pascal's RandSeed: DS offset 18416 in Curse of the Azure Bonds
                ds = tools.status(client)['cpu']['ds']
                with client.session() as session:
                    session.execute('memwrite', {'address': ds * 16 + 18416, 'data': int(arg).to_bytes(4, 'little').hex()})
                print('seed', arg)
            elif kind == 'saves':
                # copy the game's SAVE folder (as the game wrote it) to OUTDIR/arg
                src = os.path.join(tmp, 'SAVE')
                dst = os.path.join(out, arg)
                shutil.rmtree(dst, ignore_errors=True)
                if os.path.isdir(src):
                    shutil.copytree(src, dst)
                print('saves', arg)
            elif kind == 'status':
                st = tools.status(client)
                print(st)
            elif kind == 'stack':
                st = tools.status(client)['cpu']
                a = st['ss'] * 16 + st['sp']
                r = tools.read_memory(client, a, 64, allow_running=True)
                b = bytes.fromhex(r['data_hex'])
                print('cs:ip %04X:%04X' % (st['cs'], st['ip']), ' '.join('%04X' % (b[i] | b[i + 1] << 8) for i in range(0, 64, 2)))
            elif kind == 'mode':
                r = tools.read_memory(client, 0x449, 1, allow_running=True)
                print('video mode', r['data_hex'])
            elif kind == 'vga':
                dumpfile = os.path.join(tmp, 'vga.bin')
                with client.session() as session:
                    session.execute('vgadump', {'file': dumpfile})
                shutil.copy(dumpfile, os.path.join(out, arg + '.bin'))
                save_ega(dumpfile, os.path.join(out, arg))
                print('vga', arg)
            elif kind == 'shot':
                open(os.path.join(out, arg + '.png'), 'wb').write(tools.capture_screen(client))
                print('shot', arg)

        for s in steps:
            # a stuck QMP greeting fails before the command is sent: try again
            for attempt in range(3):
                try:
                    run(s)
                    break
                except QmpTransportError:
                    if attempt == 2:
                        raise
                    print('retry', s, flush=True)
                    time.sleep(2)
    finally:
        # SIGKILL: SIGTERM opens DOSBox-X's quit dialog (an osascript that
        # inherits the QMP socket and keeps the port taken)
        proc.kill()
        proc.wait()
        shutil.rmtree(tmp, ignore_errors=True)


if __name__ == '__main__':
    main()
