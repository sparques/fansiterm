//go:build generate

//go:generate go run -tags=generate fansi.go
package main

import (
	"bytes"
	"fmt"
	"go/format"
	"image"
	"image/draw"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"slices"

	"github.com/sparques/fansiterm/tiles"
)

func main() {
	ts := tiles.NewFontTileSet()

	files, _ := filepath.Glob("*.png")
	for _, file := range files {
		runes := []rune(file)
		img := LoadTileFromFile(file)
		ts.Glyphs[runes[0]] = img.Pix
		if ts.Rectangle.Eq(image.Rectangle{}) {
			ts.Rectangle = img.Bounds()
		}
	}

	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "package %s\n", "fansi")
	fmt.Fprintf(buf, "import \"image\"\n")
	fmt.Fprintf(buf, "import \"github.com/sparques/fansiterm/tiles\"\n")
	fmt.Fprintf(buf, "var %s = &tiles.FontTileSet{\n", "AltCharSet")
	fmt.Fprintf(buf, "Rectangle: image.Rect(0,0, %d, %d),\n", 8, 16)
	fmt.Fprintf(buf, "Glyphs: map[rune][]uint8{\n")

	rr := make([]rune, len(ts.Glyphs))
	i := 0
	for r := range ts.Glyphs {
		rr[i] = r
		i++
	}
	slices.Sort(rr)

	for _, r := range rr {
		fmt.Fprintf(buf, "\t%d: %#v,\n", r, ts.Glyphs[r])
	}
	fmt.Fprintf(buf, "}}\n")

	fmted, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatalf("format.Source: %v", err)
	}
	if err := ioutil.WriteFile("altCharSet.go", fmted, 0644); err != nil {
		log.Fatalf("ioutil.WriteFile: %v", err)
	}

}

func LoadTileFromFile(file string) *image.Alpha {
	fh, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer fh.Close()
	img, _, err := image.Decode(fh)
	if err != nil {
		panic(err)
	}

	alpha := image.NewAlpha(img.Bounds())
	draw.Draw(alpha, alpha.Bounds(), img, image.Point{}, draw.Src)
	return alpha
}
