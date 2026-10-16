// Command tablefind looks for each table still read from Curse of the Azure
// Bonds in Silver Blades' data segment, by its bytes (a porting aid).
//
//	tablefind BLADES-START.EXE CURSE-START.EXE
package main

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"sotsb/exe"
	"sotsb/tablespec"
)

func main() {
	blades, err := exe.LoadBlades(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	curse, err := exe.LoadCurse(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	data := blades.Mem[blades.DS:]
	for _, t := range tablespec.Tables {
		if !t.Curse {
			continue
		}
		size := t.Len * t.Kind.Size()
		if t.Cols > 0 {
			size *= t.Cols
		}
		if t.Stride > 0 {
			size = t.Len * t.Stride
		}
		want := curse.Bytes(t.At, size)
		var found []int
		for at := 0; ; {
			i := bytes.Index(data[at:], want)
			if i < 0 {
				break
			}
			found = append(found, at+i)
			at += i + 1
		}
		fmt.Printf("%-24s curse %6d size %5d  blades %v\n", t.Name, t.At, size, found)
	}
}
