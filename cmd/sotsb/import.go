package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path"
	"path/filepath"
	"strings"

	"sotsb/dax"
	"sotsb/ega"
	"sotsb/internal/gameimport"
)

// markerFiles: a folder holding all of these is a Curse of the Azure Bonds install.
// Pool of Radiance has neither the big pictures nor the sky archive.
var markerFiles = []string{"BIGPIC1.DAX", "SKY.DAX", "GAME.OVR"}

// isMarker tells whether name is one of markerFiles.
func isMarker(name string) bool {
	base := strings.ToUpper(path.Base(name))
	for _, m := range markerFiles {
		if base == m {
			return true
		}
	}
	return false
}

// gameFile tells which files the engine reads: the .DAX archives, the base item
// table, the configuration and saved games.
func gameFile(name string) bool {
	base := strings.ToUpper(path.Base(name))
	switch {
	case strings.HasSuffix(base, ".DAX"):
		return true
	case base == "ITEMS" || base == "CURSE.CFG":
		return true
	case strings.HasPrefix(base, "SAVGAM") || strings.HasPrefix(base, "CHRDAT"):
		return true // saved games and saved characters
	}
	return false
}

// importGame (-import, for goplay) finds a Curse of the Azure Bonds install in
// whatever src is (folders, zips, CD images, the GOG installer), copies the files
// the engine reads into dst and writes dst/COVER.PNG from the title picture.
func importGame(src, dst string) error {
	files, err := gameimport.Find(src, func(name string) bool { return gameFile(name) || isMarker(name) })
	if err != nil {
		return err
	}
	// Group by folder and take the one holding every marker.
	byDir := map[string]map[string][]byte{}
	for name, b := range files {
		d := path.Dir(name)
		if byDir[d] == nil {
			byDir[d] = map[string][]byte{}
		}
		byDir[d][strings.ToUpper(path.Base(name))] = b
	}
	var game map[string][]byte
	for _, fs := range byDir {
		ok := true
		for _, m := range markerFiles {
			if _, has := fs[m]; !has {
				ok = false
			}
		}
		if ok {
			game = fs
			break
		}
	}
	if game == nil {
		return fmt.Errorf("no Curse of the Azure Bonds found in %s", src)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	for name, b := range game {
		if !gameFile(name) {
			continue // a marker only
		}
		if err := os.WriteFile(filepath.Join(dst, name), b, 0o644); err != nil {
			return err
		}
	}
	return writeCover(dst)
}

// The cover is the title screen as the game first shows it (engine/title.go):
// the painting with the Curse logo over it.
const (
	coverPainting  = 2 // TITLE.DAX's full-screen painting
	coverLogo      = 3 // the "Curse of the Azure Bonds" logo
	coverLogoCol   = 6 // where the logo goes, in 8-pixel cells
	coverLogoRow   = 11
	coverCellPixel = 8
)

// writeCover renders the title painting with its logo.
func writeCover(dst string) error {
	l, err := dax.NewLoader(dst)
	if err != nil {
		return err
	}
	pal := make(color.Palette, 16)
	for i, c := range ega.Palette {
		pal[i] = color.RGBA{c[0], c[1], c[2], 255}
	}
	img := image.NewPaletted(image.Rect(0, 0, ega.W, ega.H), pal)
	draw := func(id, col, row int) error {
		b := l.Block("TITLE", id)
		if b == nil {
			return fmt.Errorf("TITLE.DAX has no picture %d", id)
		}
		im := dax.ParseImage(b, false, 0)
		d, w := im.Frame(0), im.Width*coverCellPixel
		x0, y0 := col*coverCellPixel, row*coverCellPixel
		for y := 0; y < im.Height && y0+y < ega.H; y++ {
			for x := 0; x < w && x0+x < ega.W; x++ {
				img.SetColorIndex(x0+x, y0+y, d[y*w+x]&15)
			}
		}
		return nil
	}
	if err := draw(coverPainting, 0, 0); err != nil {
		return err
	}
	if err := draw(coverLogo, coverLogoCol, coverLogoRow); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(dst, "COVER.PNG"))
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
