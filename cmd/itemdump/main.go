// Command itemdump lists the item records (63 bytes each) in item DAX
// files with their base type and name words (a porting aid).
//
//	itemdump START.EXE ITEM1.DAX ...
package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"sotsb/dax"
	"sotsb/exe"
)

const (
	itemSize    = 63
	typeAt      = 46
	wordsAt     = 47
	wordsStart  = 3853
	wordSlot    = 21
)

func main() {
	img, err := exe.LoadBlades(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	data := img.Mem[img.DS:]
	word := func(n byte) string {
		p := wordsStart + int(n)*wordSlot
		return string(data[p+1 : p+1+int(data[p])])
	}
	names := map[byte]map[string]bool{}
	for _, path := range os.Args[2:] {
		f, err := dax.Open(path)
		if err != nil {
			log.Fatal(err)
		}
		for _, en := range f.Entries {
			b := f.Block(en.ID)
			for at := 0; at+itemSize <= len(b); at += itemSize {
				r := b[at : at+itemSize]
				var parts []string
				for _, w := range r[wordsAt : wordsAt+3] {
					if w != 0 {
						parts = append(parts, word(w))
					}
				}
				t := r[typeAt]
				if names[t] == nil {
					names[t] = map[string]bool{}
				}
				names[t][strings.Join(parts, " ")] = true
			}
		}
	}
	var types []int
	for t := range names {
		types = append(types, int(t))
	}
	sort.Ints(types)
	for _, t := range types {
		var list []string
		for n := range names[byte(t)] {
			list = append(list, n)
		}
		sort.Strings(list)
		if len(list) > 6 {
			list = list[:6]
		}
		fmt.Printf("%3d %s\n", t, strings.Join(list, " | "))
	}
}
