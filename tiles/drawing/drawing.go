//go:build generate

//go:generate go run -tags=generate drawing.go
package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/sparques/fansiterm/tiles"
)

/*
This is a utility for turning a txt file into an Alpha1 or AlphaCell.

Similar to gentileset, this will generate a go source file that will contain an TileSet.
*/

const (
	packageName  = "drawing"
	variableName = "TileSet"
)

func main() {
	ts := tiles.NewAlphaCellTileSet()

	files, _ := filepath.Glob("*.tile")
	for _, file := range files {
		data, err := open(file)
		if err != nil {
			panic("could not open " + file + ": " + err.Error())
		}
		img, err := parse(data)
		if err != nil {
			panic("could not parse " + file + ": " + err.Error())
		}
		ts.Glyphs[getRuneFromName(file)] = img.Pix
	}

	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "package %s\n", packageName)
	fmt.Fprintf(buf, "import \"github.com/sparques/fansiterm/tiles\"\n")
	fmt.Fprintf(buf, "var %s = &tiles.AlphaCellTileSet{\n", variableName)
	fmt.Fprintf(buf, "Glyphs: map[rune][16]uint8{\n")

	rr := make([]rune, len(ts.Glyphs))
	i := 0
	for r := range ts.Glyphs {
		rr[i] = r
		i++
	}
	slices.Sort(rr)

	for _, r := range rr {
		fmt.Fprintf(buf, "\t0x%04X: %#v,\n", r, ts.Glyphs[r])
	}
	fmt.Fprintf(buf, "}}\n")

	fmted, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatalf("format.Source: %v", err)
	}
	if err := ioutil.WriteFile(fmt.Sprintf("%s.go", variableName), fmted, 0644); err != nil {
		log.Fatalf("ioutil.WriteFile: %v", err)
	}
}

// Figures out what rune a file name
func getRuneFromName(filename string) rune {
	filename = strings.TrimSuffix(filename, ".tile")
	if i, err := strconv.ParseInt(filename, 0, 64); err == nil {
		return rune(i)
	}

	panic("could not determine how to map image data to a rune!")
}

func open(fn string) ([]byte, error) {
	fh, err := os.Open(fn)
	if err != nil {
		return nil, fmt.Errorf("could not open %s: %w", fn, err)
	}
	defer fh.Close()

	data, err := io.ReadAll(fh)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %w", fn, err)
	}
	return data, nil
}

func parse(data []byte) (*tiles.AlphaCell, error) {

	if bytes.Index(data, []byte{'\n'}) == -1 {
		return nil, errors.New("no newline found; there must be at least one")
	}

	pix := make([]byte, 0, len(data))
	lines := 0
	j := 0
	b := byte(0)
	width := 0
	for i := range data {
		switch data[i] {
		case ' ', '.', ':':
			// zero
			// b |= 0
		case '!': // used for marking borders
			continue
		case '\n':
			if width == 0 {
				width = j
			}
			if j%8 != 0 {
				b <<= 7 - (j % 8)
				pix = append(pix, b)
				b = 0
				j = 0
			}
			lines++
			continue
		default:
			// one
			b |= 1
		}
		j++
		if j%8 == 0 {
			pix = append(pix, b)
			b = 0
			continue
		}
		b <<= 1
	}

	stride := width / 8
	if width%8 != 0 {
		stride++
	}

	ac := tiles.AlphaCell{}
	for i := range 16 {
		ac.Pix[i] = pix[i]
	}
	return &ac, nil

}
