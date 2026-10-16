"""rawpng.py FILE.raw [PAL]: render 320x200 EGA colour indices (from dosparity vga dumps or the
port's snap .raw) to FILE.png with the default EGA palette."""
import sys
from PIL import Image
EGA = [(0,0,0),(0,0,170),(0,170,0),(0,170,170),(170,0,0),(170,0,170),(170,85,0),(170,170,170),
       (85,85,85),(85,85,255),(85,255,85),(85,255,255),(255,85,85),(255,85,255),(255,255,85),(255,255,255)]
for f in sys.argv[1:]:
    idx = open(f, 'rb').read()
    im = Image.new('RGB', (320, 200))
    im.putdata([EGA[c & 15] for c in idx])
    im.save(f[:-4] + '.png')
