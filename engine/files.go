package engine

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
)

// Character file I/O, ported from ovr18. Record layouts are kept raw so the
// original SAVE files load and save byte for byte.

const (
	itemRecSize   = 0x3F
	affectRecSize = 9
	gsPartyCount  = 0x67C // game-state word
	gsSymbolSet   = 0x624
)

// Item and Affect are the raw list nodes (next pointers live in the slices).
// Item is the 0x3F-byte DOS item record, in its file layout (encoding/binary).
type Item struct {
	PName   [0x29]byte // 0x00 shown name, Pascal string (built by ItemName)
	Pad29   byte       // 0x29
	_       [4]byte    // 0x2A next pointer
	Type    byte       // 0x2E base item type (index into the DS:5582 records)
	Words   [3]byte    // 0x2F name word ids into DS:10A7
	Plus    byte       // 0x32 signed
	Plus2   byte       // 0x33
	Readied byte       // 0x34
	Hidden  byte       // 0x35 bit 3-i: word i not known yet
	Cursed  byte       // 0x36
	Weight  uint16     // 0x37
	Count   byte       // 0x39
	Price   uint16     // 0x3A
	Spells  [3]byte    // 0x3C a scroll's spells, +0x80 = being scribed
}

// Affect is a spell effect or condition on a character.
type Affect struct {
	affectRecord
	Item *Item // set when an item stands in for the affect (ovr13 item handlers)
}

// affectRecord is the 9-byte DOS affect node in its file layout.
type affectRecord struct {
	Type byte   // affect number, index into the ovr13 handler table
	Time uint16 // remaining duration, 0 = permanent
	Data byte
	Call byte    // non-zero: the handler runs when the affect is removed
	_    [4]byte // next pointer
}

// readRecords reads the n-byte records in b (a partial last one is dropped).
func readRecords[T any](b []byte, n int) []*T {
	var out []*T
	for ; len(b) >= n; b = b[n:] {
		v := new(T)
		binary.Read(bytes.NewReader(b[:n]), binary.LittleEndian, v)
		out = append(out, v)
	}
	return out
}

func readAffects(b []byte) []*Affect {
	var out []*Affect
	for _, r := range readRecords[affectRecord](b, affectRecSize) {
		out = append(out, &Affect{affectRecord: *r})
	}
	return out
}

// writeRecords is the inverse of readRecords.
func writeRecords[T any](list []*T) []byte {
	var buf bytes.Buffer
	for _, v := range list {
		binary.Write(&buf, binary.LittleEndian, v)
	}
	return buf.Bytes()
}

// savePath resolves a file name inside the save directory case-insensitively
// (DOS was case-insensitive; the shipped files are upper case).
func (e *Engine) savePath(name string) string {
	dir := e.Cfg.SaveDir
	if ents, err := os.ReadDir(dir); err == nil {
		for _, en := range ents {
			if strings.EqualFold(en.Name(), name) {
				return filepath.Join(dir, en.Name())
			}
		}
	}
	return filepath.Join(dir, strings.ToUpper(name))
}

func (e *Engine) fileExists(path string) bool { // Pool 0802:02EB
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

// removeSubstr = R5BD:054B; makeFileName = R5BD:061A.
// The scan runs over the original length and does not recheck the position
// after a deletion, so adjacent copies leave every second one (original quirk).
func removeSubstr(s, sub string) string {
	if sub == "" {
		return s
	}
	for i, n := 1, len(s)-len(sub)+1; i <= n; i++ {
		if i-1+len(sub) <= len(s) && s[i-1:i-1+len(sub)] == sub {
			s = s[:i-1] + s[i-1+len(sub):]
		}
	}
	return s
}
func makeFileName(s string) string {
	for _, c := range " .*,?/\\:;|" { // DS:0D1A..0D23
		s = removeSubstr(s, string(c))
	}
	if len(s) > 8 {
		s = s[:8]
	}
	return strings.ToUpper(s)
}

// fileError reports a save file that could not be read or written, once.
// (DOS asked for the save disk here and retried until it worked.)
func (e *Engine) fileError(verb, path string) {
	s := "Cannot " + verb + " " + filepath.Base(path)
	if len(s) > 40 {
		s = s[:40]
	}
	e.MessageRow24b(0xE, s)
}

// MessageRow24b = Pool 0709:0BF5: clear row 24, show, wait for a key.
func (e *Engine) MessageRow24b(colour int, s string) byte {
	if e.Trace != nil {
		e.Trace("Pool 0709:0BF5", "MessageRow24b", colour, s)
	}
	e.ClearRow24()
	e.DisplayString(0, 24, colour, s)
	return e.ReadKeyByte()
}

// Portrait tables DS:8C7.. (gender-major).
var (
	portHeads     = [2][8]byte{{1, 2, 3, 4, 5, 8, 0xB, 0xC}, {6, 7, 9, 0xA, 0xD, 0xE, 6, 7}}
	portBodyCls0  = [2][5]byte{{2, 4, 2, 4, 2}, {2, 4, 8, 2, 4}}
	portBodyOther = [2][5]byte{{1, 3, 4, 5, 6}, {7, 9, 0xB, 7, 9}}
	portBodyCls5  = [2][5]byte{{2, 2, 2, 2, 2}, {8, 0xA, 0xC, 8, 0xA}}
	portBodyCls6  = [2][5]byte{{6, 6, 6, 6, 6}, {8, 0xA, 8, 0xA, 8}}
)

func (e *Engine) partyCount() int { return int(binary.LittleEndian.Uint16(e.game[gsPartyCount:])) }
func (e *Engine) setPartyCount(n int) {
	binary.LittleEndian.PutUint16(e.game[gsPartyCount:], uint16(n))
}
