package engine

// The AdLib music driver of R7E4. Its commands, as R7E4:0020 takes them:
const (
	musicSilence  = 0   // stop both song channels
	musicFanfare  = 1   // the short jingle on the effect channel
	musicShutdown = 255 // silence and unhook the driver
	musicLastSong = 14  // songs are 2..14 (song n plays tune n-1)
)

// musicCommand = R7E4:0020. With digitized sound or none chosen, the AdLib
// driver stays quiet.
func (e *Engine) musicCommand(command int) {
	if e.soundDevice == soundDeviceNone {
		return
	}
	switch {
	case command == musicSilence, command == musicFanfare:
		e.music.song = 0
	case command == musicShutdown:
		e.music.song = 0
		e.music.installed = false
	case command >= 2 && command <= musicLastSong:
		e.music.song = command - 1
	}
}

// musicInstall = R7E4:00CB: remember the chosen device for Ctrl-S and hook
// the driver.
func (e *Engine) musicInstall() {
	e.savedSoundDevice = e.soundDevice
	e.music.installed = true
	if e.soundDevice == soundDeviceNone {
		return
	}
	e.music.song = 0
}

// musicRestart is the pair the main loop repeats between phases: shut the
// driver down if it is running, install it again if it is not.
func (e *Engine) musicRestart() {
	if e.music.installed {
		e.musicCommand(musicShutdown)
	}
	if !e.music.installed {
		e.musicInstall()
	}
}

type musicState struct {
	installed bool // b741C
	song      int  // the tune now playing, 0 for none
}
