package engine

import "testing"

// Every song runs through the sequencer, and the songs with notes put a period of
// the note table (some octave of it) on a channel.
func TestSongsPlayNotes(t *testing.T) {
	notes := 0
	for song := 1; song < sndSongs; song++ {
		for _, set := range []*sndChannelSet{
			{songs: &sndSpeakerSongs, periods: &sndSpeakerPeriods},
			{songs: &sndTandySongs, periods: &sndTandyPeriods},
		} {
			s := &sound{song: song}
			s.start(set, song)
			for range 20000 {
				for c := range set.ch {
					if set.ch[c].timer != 0 {
						s.step(set, c)
					}
				}
				if p := set.ch[0].period; p != 0 && set.ch[0].release != 0 && inOctaves(p, set.periods) {
					notes++
					break
				}
			}
		}
	}
	if notes == 0 {
		t.Fatal("no song played a note")
	}
}

func inOctaves(p uint16, periods *[notesPerOctave]uint16) bool {
	for _, q := range periods {
		for o := range 16 {
			if q>>o == p {
				return true
			}
		}
	}
	return false
}
