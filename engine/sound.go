package engine

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"sync"

	"sotsb/amiga"
)

//go:generate sh -c "go run ../cmd/sounddata $$COTAB_EXE > sounddata.go"

// The sound driver of START.EXE (its segment R699): a sequencer run from the timer
// interrupt, reprogrammed to 236.7 Hz, that steps two sets of four channels every tick,
// the PC speaker's and the Tandy/PCjr's (three tones and the noise of its SN76489). A song
// starts a script on each channel of one set; a script sets the channel's fields (its
// period and level and their slides, a vibrato) and waits on its timer. The songs, their
// scripts and the vibrato waveforms are in sounddata.go, generated from the exe by
// cmd/sounddata.
//
// The driver also plays notes with an envelope, but no song of the game does, so that
// part is not ported (cmd/sounddata refuses data that would need it). Op 0 of the
// program ends a script: the Tandy's song 18 (sound 19, which the game never plays)
// returns without a call, which in the original runs on into the driver's own code.
//
// Nothing is audible, and nothing waits, until SoundReader is called, so the
// headless tools stay silent and fast.

const (
	sndRate     = 44100
	pitClock    = 1193182.0       // the PC's timer chip
	sndTimerHz  = pitClock / 5041 // the driver's tick: timer 0's divisor 5041
	snClock     = 3579545.0       // SN76489 input clock on the Tandy 1000 / PCjr
	sndChannels = 4
	sndSongs    = 21
	speakerAmp  = 4000
)

// The sound numbers the game passes to PlaySound (the table at DS:2612).
const (
	sndStop       = 255 // stop the song
	sndMute       = 0   // stop and mute
	sndUnmute     = 1
	sndSpell      = 2 // a spell flies (the spells but 47 and 51)
	sndHit        = 3 // a weapon hits
	sndMagicHit   = 4 // a magical weapon hits
	sndFall       = 5 // a combatant falls
	sndMissile    = 6 // a missile flies
	sndBump       = 7 // walking into a wall
	sndSpell51    = 8
	sndMiss       = 9 // an attack hits nobody
	sndStep       = 10
	sndSpell47    = 11 // also the script call 0x3201 in areas of kind 10
	sndMissileEnd = 12 // a missile arrives
	sndTitle      = 13 // the Forgotten Realms banner
	sndCountdown  = 15
)

// The game's sound card (DS:5C04).
const (
	sndSpeaker = iota // DS:5C04, from the config file's 'P'
	sndTandy          // 'T'
	sndNone
)

// sndOp is a command of a song's script.
type sndOp struct {
	kind  sndKind
	field sndField
	arg   int
}

type sndKind uint8

const (
	sndSet    sndKind = iota // set field to arg; setting the timer ends the event, 0 the script
	sndCall                  // go to op arg, back at the next sndReturn
	sndReturn                //
	sndSwitch                // work on channel arg of the set from here on, its sound cleared
	sndLoop                  // count field down and go to op arg until it reaches 0 (from 0: always)
)

// sndField is a channel field a script sets.
type sndField uint8

const (
	sndTimer sndField = iota
	sndPeriod
	sndSlide
	sndLevel
	sndLevelSlide
	sndVibrato
	sndVibratoSpeed
	sndVibratoDepth
	sndVibratoLength
	sndCounter
	sndCounter2
)

// sndChannel is one channel of a set. The words wrap around as the 16-bit driver's did.
type sndChannel struct {
	timer  uint16 // ticks to the script's next event; 0: the channel is idle
	script int    // its next op, 0 when it has ended
	period uint16 // a timer divisor (speaker) or the SN76489 divider times 64 (Tandy)
	slide  uint16 // added to the period every tick
	out    uint16 // the period with the vibrato: what the sound chip gets
	// the speaker sounds while bits 0 and 1 are set; the Tandy's attenuation is the top
	// four bits of its complement
	level, levelSlide uint16
	vibrato           int    // the waveform (sndVibratos), 0 none
	vibratoPos        uint16 // in 16ths of an entry
	vibratoSpeed      uint16
	vibratoDepth      uint16
	vibratoLength     uint16
	counter, counter2 uint16
}

func (c *sndChannel) field(f sndField) *uint16 {
	switch f {
	case sndTimer:
		return &c.timer
	case sndPeriod:
		return &c.period
	case sndSlide:
		return &c.slide
	case sndLevel:
		return &c.level
	case sndLevelSlide:
		return &c.levelSlide
	case sndVibratoSpeed:
		return &c.vibratoSpeed
	case sndVibratoDepth:
		return &c.vibratoDepth
	case sndVibratoLength:
		return &c.vibratoLength
	case sndCounter:
		return &c.counter
	case sndCounter2:
		return &c.counter2
	}
	panic("sound: no such field")
}

func (c *sndChannel) set(f sndField, v int) {
	if f == sndVibrato {
		c.vibrato = v
		return
	}
	*c.field(f) = uint16(v)
}

// clear silences a channel a script switches to; its timer, script and counters stay.
func (c *sndChannel) clear() {
	*c = sndChannel{timer: c.timer, script: c.script, counter: c.counter, counter2: c.counter2}
}

// sndChannelSet is the four channels of a sound chip and the songs for them.
type sndChannelSet struct {
	ch    [sndChannels]sndChannel
	songs *[sndSongs][sndChannels]int
}

// sound is the driver state.
type sound struct {
	mu        sync.Mutex
	speaker   sndChannelSet
	tandySet  sndChannelSet
	on        bool // a sink is attached (SoundReader called)
	isr       bool // the timer interrupt is hooked
	typ       int  // DS:5C04: sndSpeaker, sndTandy or sndNone
	saved     int  // DS:5C05: typ before Ctrl-S turned the sound off
	installed bool // DS:5C06: soundInit ran, PlaySound(sndStop) not since
	song      int  // the song playing, 0 none
	enabled   bool // PlaySound(sndUnmute) and (sndMute)
	ret       int  // where sndReturn goes
	period    int  // timer 2 divisor sent to the speaker
	tone      bool // speaker gate and data bits both on
	phase     float64
	tickLeft  float64
	tandy     bool // Tandy sound chip present (config 'T'): render the SN76489
	sn        sn76489
	am        *amiga.Player // DISK2/SOUND.BIN imported: the Amiga players instead
}

// amigaFx maps PlaySound 2..12 to Amiga effects: the Amiga's effect table parallels the
// DOS sound numbers.
var amigaFx = [...]int{5, 9, 8, 10, 12, 11, 6, 11, 4, 7, 0}

// sn76489 is the Tandy/PCjr sound chip: three square wave tones and a noise
// channel, each with a 4-bit attenuation, programmed through port C0.
type sn76489 struct {
	latch int        // register selected by the last latch byte
	tone  [3]int     // 10-bit dividers
	noise int        // noise control (feedback, rate)
	att   [4]int     // 0 loudest .. 15 off
	phase [4]float64 // tone phases; [3] is the noise shift clock
	lfsr  uint16
}

// The SN76489's port: a latch byte (bit 7 set) selects a register (bits 4-6: the channel
// and whether it is the attenuation) and writes its low four bits; a data byte writes the
// high six bits of the selected tone divider.
const (
	snLatch     = 128
	snSilent    = 15
	snNoiseSeed = 1 << 14
)

// out = a byte written to port C0.
func (c *sn76489) out(b byte) {
	if b&snLatch != 0 {
		c.latch = int(b>>4) % 8
		ch := c.latch / 2
		switch {
		case c.latch%2 != 0:
			c.att[ch] = int(b % 16)
		case ch == 3:
			c.noise, c.lfsr = int(b%8), snNoiseSeed // a noise write resets the shift register
		default:
			c.tone[ch] = c.tone[ch]/16*16 + int(b%16)
		}
		return
	}
	if ch := c.latch / 2; c.latch%2 == 0 && ch < 3 {
		c.tone[ch] = c.tone[ch]%16 + int(b%64)*16
	}
}

// sample renders one output sample.
func (c *sn76489) sample() int {
	v := 0
	for ch := range 4 {
		if c.att[ch] == snSilent {
			continue
		}
		amp := int(2000 * math.Pow(10, -float64(c.att[ch])/10)) // 2 dB per step
		n := 0
		if ch < 3 {
			n = c.tone[ch]
		} else if c.noise%4 == 3 {
			n = c.tone[2]
		} else {
			n = 16 << uint(c.noise%4)
		}
		if n == 0 {
			n = 1024
		}
		f := snClock / 32 / float64(n)
		if f >= sndRate/2 {
			continue // beyond Nyquist: the chip's output is a flat level
		}
		c.phase[ch] += f / sndRate
		if ch < 3 {
			c.phase[ch] -= math.Floor(c.phase[ch])
			if c.phase[ch] < 0.5 {
				v += amp
			} else {
				v -= amp
			}
			continue
		}
		for ; c.phase[3] >= 1; c.phase[3]-- {
			// ponytail: 15-bit LFSR with TI taps; the Tandy's NCR 8496 may differ.
			fb := c.lfsr & 1
			if c.noise&4 != 0 {
				fb ^= c.lfsr >> 1 & 1
			}
			c.lfsr = c.lfsr>>1 | fb<<14
		}
		if c.lfsr&1 != 0 {
			v += amp
		} else {
			v -= amp
		}
	}
	return v
}

// snSilence turns the four channels' attenuation to off.
func (s *sound) snSilence() {
	for ch := range byte(4) {
		s.sn.out(snLatch | ch<<5 | 1<<4 | snSilent)
	}
}

// SoundReader attaches an audio sink and returns its source: signed 16-bit
// little endian stereo at 44100 Hz, to be fed to an audio player.
func (e *Engine) SoundReader() *soundReader {
	s := &e.snd
	s.mu.Lock()
	defer s.mu.Unlock()
	s.speaker = sndChannelSet{songs: &sndSpeakerSongs}
	s.tandySet = sndChannelSet{songs: &sndTandySongs}
	s.enabled, s.ret = true, 0
	s.sn.att = [4]int{snSilent, snSilent, snSilent, snSilent}
	s.tandy = e.Cfg.Sound == 'T'
	switch e.Cfg.Sound { // main 0x4574
	case 'T':
		s.typ = sndTandy
	case 'P':
		s.typ = sndSpeaker
	default:
		s.typ = sndNone
	}
	s.on = true
	if img, err := os.ReadFile(filepath.Join(e.L.Dir, "DISK2", "SOUND.BIN")); err == nil && len(img) == amiga.End-amiga.Base {
		s.am = amiga.New(img)
	}
	return &soundReader{s}
}

type soundReader struct{ s *sound }

// Read renders the speaker as a square wave (plus the SN76489 when the
// Tandy chip is present), running the sequencer once per timer tick while
// the interrupt is hooked.
func (r *soundReader) Read(p []byte) (int, error) {
	s := r.s
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(p) / 4 * 4
	if s.am != nil {
		if s.typ == sndNone {
			clear(p[:n])
		} else {
			s.am.Render(p[:n], sndRate)
			kind := "sfx"
			if s.am.Playing() {
				kind = "music"
			}
			scale(p[:n], volume(kind))
		}
		return n, nil
	}
	for i := 0; i < n; i += 4 {
		v := 0
		if s.isr {
			if s.tickLeft--; s.tickLeft < 0 {
				s.tickLeft += sndRate / sndTimerHz
				s.tick()
			}
			if s.tone {
				per := s.period
				if per == 0 {
					per = 1 << 16 // the PIT counts a 0 divisor as 65536
				}
				s.phase += pitClock / float64(per) / sndRate
				s.phase -= math.Floor(s.phase)
				v = speakerAmp
				if s.phase >= 0.5 {
					v = -speakerAmp
				}
			}
		}
		if s.tandy {
			v += s.sn.sample()
		}
		p[i], p[i+1], p[i+2], p[i+3] = byte(v), byte(v>>8), byte(v), byte(v>>8)
	}
	scale(p[:n], volume("sfx")) // the DOS game has no music
	return n, nil
}

// scale turns s16 samples down by g.
func scale(p []byte, g float64) {
	if g >= 1 {
		return
	}
	for i := 0; i+1 < len(p); i += 2 {
		v := float64(int16(binary.LittleEndian.Uint16(p[i:])))
		binary.LittleEndian.PutUint16(p[i:], uint16(int16(v*g)))
	}
}

// tick = the timer interrupt: the speaker's set, then the Tandy's.
func (s *sound) tick() {
	s.tickSpeaker()
	s.tickTandy()
}

// tickTandy steps the Tandy set, then writes each channel's divider (its period / 64) and
// attenuation (the top four bits of the level's complement) to the chip. The noise
// channel gets only the divider's low bits as its control byte.
func (s *sound) tickTandy() {
	if s.song == 0 {
		s.snSilence()
		return
	}
	for c := range s.tandySet.ch {
		if s.tandySet.ch[c].timer != 0 {
			s.step(&s.tandySet, c)
		}
	}
	if !s.enabled {
		s.snSilence()
		return
	}
	for c := range s.tandySet.ch {
		ch := &s.tandySet.ch[c]
		latch := byte(snLatch + 32*c)
		divider := ch.out / 64
		s.sn.out(byte(divider%16) | latch)
		if c != 3 {
			s.sn.out(byte(divider / 16))
		}
		s.sn.out(byte((math.MaxUint16-ch.level)>>12) | latch | 1<<4)
	}
}

// tickSpeaker steps the four speaker channels, then gives the first sounding one to the
// speaker.
func (s *sound) tickSpeaker() {
	s.tone = false
	if s.song == 0 {
		return
	}
	chosen := -1
	for c := range s.speaker.ch {
		ch := &s.speaker.ch[c]
		if ch.timer == 0 {
			continue
		}
		s.step(&s.speaker, c)
		if chosen < 0 && ch.level != 0 && ch.timer != 0 {
			chosen = c
		}
	}
	if !s.enabled || chosen < 0 {
		return
	}
	ch := &s.speaker.ch[chosen]
	s.period = int(ch.out)
	s.tone = ch.level%4 == 3 // bit 0 gates timer 2, bit 1 enables the speaker
}

// step runs one tick of channel c: the level and pitch slides, the vibrato, and the
// timer, the script's next event when it runs out.
func (s *sound) step(set *sndChannelSet, c int) {
	ch := &set.ch[c]
	ch.level += ch.levelSlide
	ch.period += ch.slide
	var bend uint16
	if pos := ch.vibratoPos + ch.vibratoSpeed; pos != 0 {
		if pos >= ch.vibratoLength {
			pos -= ch.vibratoLength
		}
		ch.vibratoPos = pos
		wave := sndVibratos[ch.vibrato]
		if i := int(int16(pos)) >> 4; i >= 0 && i < len(wave) { // cmd/sounddata checks it is
			sample := int32(wave[i]) * 256
			bend = uint16(uint32(sample*int32(int16(ch.vibratoDepth))) >> 16) // the high word of the product
		}
	}
	ch.out = ch.period + bend
	ch.timer--
	if ch.timer == 0 {
		s.fetch(set, c)
	}
}

// fetch runs channel c's script up to its next event: a timer set. A script may switch
// to another channel of the set and set that one; the timer set then ends the event of
// the channel it started on, which is idle when its own timer is still 0.
func (s *sound) fetch(set *sndChannelSet, c int) {
	cur := c
	pc := set.ch[cur].script
	if pc == 0 {
		return
	}
	for range 1 << 16 {
		op := sndProgram[pc]
		pc++
		switch op.kind {
		case sndReturn:
			pc = s.ret
		case sndCall:
			s.ret, pc = pc, op.arg
		case sndSwitch:
			c = op.arg
			set.ch[c].clear()
		case sndLoop:
			n := set.ch[c].field(op.field)
			if *n != 0 {
				*n--
				if *n == 0 {
					continue
				}
			}
			pc = op.arg
		case sndSet:
			set.ch[c].set(op.field, op.arg)
			if op.field == sndTimer {
				if set.ch[cur].timer == 0 {
					pc = 0
				}
				set.ch[cur].script = pc
				return
			}
		}
	}
	set.ch[cur].script = 0 // runaway script
}

// start loads song's four channel scripts into a set and makes it the song playing.
func (s *sound) start(set *sndChannelSet, song int) {
	s.song = song
	for c := range set.ch {
		if p := set.songs[song][c]; p != 0 {
			set.ch[c] = sndChannel{timer: 1, script: p}
		}
	}
}

// stop loads song 0, which ends every channel.
func (s *sound) stop() {
	s.start(&s.tandySet, 0)
	s.start(&s.speaker, 0)
}

// PlaySound = R699:0020 (n): only 0..15 and 255 are taken, and only with a sound
// card. sndMute starts the empty song 0 on both sets, sndUnmute lets the speaker
// sound, sndStop silences everything and unhooks the timer interrupt; 2..14 start
// song n-1 on the card's set. 15 arms the driver's copy-protection countdown
// (R699:1338: after 5000 ticks it prints a message and exits); no caller passes it.
func (e *Engine) PlaySound(n int) {
	s := &e.snd
	if !(n >= sndMute && n <= sndCountdown || n == sndStop) {
		return
	}
	if !s.on || s.typ == sndNone {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case n == sndMute:
		s.stop()
	case n == sndUnmute:
		s.enabled = true
	case n == sndStop:
		s.stop()
		s.unhook()
		s.installed = false
	case n >= sndSpell && n < sndCountdown:
		if s.am != nil && n-sndSpell < len(amigaFx) {
			s.am.Effect(amigaFx[n-sndSpell])
		} else if s.typ == sndSpeaker {
			s.start(&s.speaker, n-1)
		} else {
			s.start(&s.tandySet, n-1)
		}
	}
}

// soundInit = R699:00D4: remember the card (DS:5C05), mark the driver
// installed (DS:5C06) and, with a card, hook the timer interrupt (R699:12BF)
// and start song 0 on both sets.
func (e *Engine) soundInit() {
	s := &e.snd
	s.mu.Lock()
	defer s.mu.Unlock()
	s.saved = s.typ
	s.installed = true
	if !s.on || s.typ == sndNone {
		return
	}
	s.isr = true
	s.start(&s.speaker, 0)
	s.start(&s.tandySet, 0)
}

// unhook restores the timer interrupt: speaker off, all Tandy voices off.
func (s *sound) unhook() {
	s.isr, s.tone = false, false
	s.snSilence()
}

// soundToggle = R670 debug_txt's Ctrl-S: with a card, remember it (DS:5C05),
// PlaySound(sndMute) and take sndNone; without, the remembered card comes back
// (when it is the speaker or the Tandy) with PlaySound(sndUnmute).
func (e *Engine) soundToggle() {
	s := &e.snd
	if s.typ != sndNone {
		s.saved = s.typ
		e.PlaySound(sndMute)
		s.mu.Lock()
		s.typ = sndNone
		s.mu.Unlock()
		return
	}
	if s.saved == sndSpeaker || s.saved == sndTandy {
		s.mu.Lock()
		s.typ = s.saved
		s.mu.Unlock()
		e.PlaySound(sndUnmute)
	}
}

// daxBlock = R5BD load_decode_dax: a .DAX block; the song playing is cut
// (PlaySound(sndMute)) and the speaker let sound again after.
func (e *Engine) daxBlock(name string, id int) []byte {
	e.PlaySound(sndMute)
	b := e.L.Block(name, id)
	e.PlaySound(sndUnmute)
	return b
}
