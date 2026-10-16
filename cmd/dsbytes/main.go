// Command dsbytes prints bytes of Silver Blades' data segment in rows (a porting aid).
//
//	dsbytes START.EXE OFFSET COUNT [WIDTH]
package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"sotsb/exe"
)

func main() {
	if len(os.Args) < 4 {
		log.Fatal("usage: dsbytes START.EXE OFFSET COUNT [WIDTH]")
	}
	img, err := exe.LoadBlades(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	at, _ := strconv.Atoi(os.Args[2])
	count, _ := strconv.Atoi(os.Args[3])
	width := 16
	if len(os.Args) > 4 {
		width, _ = strconv.Atoi(os.Args[4])
	}
	raw := img.Bytes(at, count)
	for row := 0; row*width < count; row++ {
		fmt.Printf("%6d %4d:", at+row*width, row)
		for _, b := range raw[row*width : min((row+1)*width, count)] {
			fmt.Printf(" %3d", b)
		}
		fmt.Println()
	}
}
