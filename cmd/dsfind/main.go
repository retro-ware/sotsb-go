// Command dsfind prints where text occurs in Silver Blades' data segment,
// or with -strings N AT COUNT the COUNT Pascal strings of N characters from AT (a porting aid).
package main

import (
	"bytes"
	"fmt"
	"os"
	"strconv"

	"sotsb/exe"
)

func main() {
	img, err := exe.LoadBlades(os.Args[1])
	if err != nil {
		fmt.Println(err)
		return
	}
	data := img.Mem[img.DS:]
	if os.Args[2] == "-strings" {
		size, _ := strconv.Atoi(os.Args[3])
		at, _ := strconv.Atoi(os.Args[4])
		count, _ := strconv.Atoi(os.Args[5])
		for i := 0; i < count; i++ {
			p := at + i*(size+1)
			fmt.Printf("%d %q\n", i, string(data[p+1:p+1+int(data[p])]))
		}
		return
	}
	at := 0
	for {
		i := bytes.Index(data[at:], []byte(os.Args[2]))
		if i < 0 {
			return
		}
		fmt.Println(at + i)
		at += i + 1
	}
}
