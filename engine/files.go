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
	itemRecSize     = 63 // an item in ITEM*.DAX and MON*ITM
	itemFileRecSize = 67 // an item in a character's .stf file: the record and a pointer
	affectRecSize   = 9
	gsPartyCount    = 1660 // game-state word
	gsSymbolSet     = 1572
)

// Item is a carried item: its record and, for a bag, what is in it.
type Item struct {
	itemRecord
	Contents []*Item // a bag's items (O100: the list at +63)
}

// itemRecord is the 63-byte DOS item record, in its file layout
// (encoding/binary).
type itemRecord struct {
	PName   [41]byte // 0 shown name, Pascal string (built by ItemName)
	Pad41   byte     // 41
	_       [4]byte  // 42 next pointer
	Type    byte     // 46 base item type (index into the item type records)
	Words   [3]byte  // 47 name word ids into itemWords
	Plus    byte     // 50 signed
	Plus2   byte     // 51
	Readied byte     // 52
	Hidden  byte     // 53 bit 3-i: word i not known yet
	Cursed  byte     // 54
	Weight  uint16   // 55
	Count   byte     // 57 a bag: the items in it
	Price   uint16   // 58
	Spells  [3]byte  // 60 a scroll's spells, +128 = being scribed
}

// itemTypeBag: the base item type that holds other items.
const itemTypeBag = 73

// readItems reads items of n bytes each (63 or 67); a bag's Count items
// follow it in a character's .stf file.
func readItems(b []byte, n int) []*Item {
	var out []*Item
	for _, r := range readRecords[itemRecord](b, n) {
		out = append(out, &Item{itemRecord: *r})
	}
	if n != itemFileRecSize {
		return out
	}
	var list []*Item
	for i := 0; i < len(out); i++ {
		it := out[i]
		list = append(list, it)
		if it.Type == itemTypeBag {
			end := min(len(out), i+1+int(it.Count))
			it.Contents = out[i+1 : end]
			i = end - 1
		}
	}
	return list
}

// writeItems = the item part of O0F8:0DA0: each item as a 67-byte record,
// a bag followed by its contents.
func writeItems(list []*Item) []byte {
	var buf bytes.Buffer
	pointer := make([]byte, itemFileRecSize-itemRecSize)
	write := func(it *Item) {
		binary.Write(&buf, binary.LittleEndian, &it.itemRecord)
		buf.Write(pointer)
	}
	for _, it := range list {
		write(it)
		if it.Type == itemTypeBag {
			for _, in := range it.Contents {
				write(in)
			}
		}
	}
	return buf.Bytes()
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
