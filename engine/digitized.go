package engine

// The digitized sounds (unit O04B of the reconstruction): SOUND.DAX block 1 is
// a small player program, block 2 the samples it plays through the PC speaker.

// startDigitizedSound = O04B:0088 + 0246 + 02B9: load the samples and start
// sample n, unless the sound is off.
func (e *Engine) startDigitizedSound(n int) {
	if e.soundDevice == soundDeviceNone {
		return
	}
	e.digitizedSample = n
	e.digitizedPlaying = true
}

// stopDigitizedSound = O04B:02D3 + 001F: stop the player and free the samples.
func (e *Engine) stopDigitizedSound() {
	if e.soundDevice == soundDeviceNone {
		return
	}
	e.digitizedPlaying = false
}

// soundDeviceNone is the BLADES.CFG sound setting for no sound (b741A = 5).
const soundDeviceNone = 5

// The sound devices (b741A), from the BLADES.CFG sound letter.
const (
	soundDeviceAdLib   = 2
	soundDeviceTandy   = 3
	soundDeviceSpeaker = 4
)

// soundDeviceFor = R210's initialisation: the device for a sound letter.
func soundDeviceFor(letter byte) byte {
	switch letter {
	case 'A':
		return soundDeviceAdLib
	case 'T':
		return soundDeviceTandy
	case 'P':
		return soundDeviceSpeaker
	}
	return soundDeviceNone
}
