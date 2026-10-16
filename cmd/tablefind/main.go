// Command tablefind looks for each tablespec table of Curse of the Azure Bonds in
// Silver Blades' data segment by content (a porting aid, not part of the game).
//
//	tablefind CURSE-START.EXE BLADES-START.EXE ds_map.tsv
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"sotsb/exe"
	"sotsb/tablespec"
)

func main() {
	curse, err := exe.LoadCurse(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	blades, err := exe.LoadBlades(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	moved := map[int]int{}
	if f, err := os.Open(os.Args[3]); err == nil {
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			p := strings.Fields(sc.Text())
			c, _ := strconv.ParseInt(p[0], 16, 32)
			s, _ := strconv.ParseInt(p[1], 16, 32)
			moved[int(c)] = int(s)
		}
	}
	segment := blades.Mem[blades.DS:]
	for _, t := range tablespec.Tables {
		n := t.Len
		if t.Cols > 0 {
			n *= t.Cols
		}
		want := curse.Bytes(t.At, n*t.Kind.Size())
		var hits []string
		for i := 0; i+len(want) <= len(segment) && i < 1<<16; i++ {
			if bytes.Equal(segment[i:i+len(want)], want) {
				hits = append(hits, strconv.Itoa(i))
			}
		}
		guess := "-"
		if s, ok := moved[t.At]; ok {
			guess = strconv.Itoa(s)
		}
		fmt.Printf("%s\t%d\tmap=%s\tcontent=%s\n", t.Name, t.At, guess, strings.Join(hits, ","))
	}
}
