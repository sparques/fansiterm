// This was based on github.com/golang/freetype/example/genbasicfont
// It's been modified to create FontTileSet for fansiterm:
// github.com/sparques/fansiterm; original copyright noticed left intact below.

// Copyright 2016 The Freetype-Go Authors. All rights reserved.
// Use of this source code is governed by your choice of either the
// FreeType License or the GNU General Public License version 2 (or
// any later version), both of which can be found in the LICENSE file.

//go:build gentileset

// Program genbasicfont generates Go source code that imports
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"image"
	"image/draw"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"unicode"

	"github.com/golang/freetype/truetype"
	"github.com/sparques/fansiterm/tiles"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

var (
	fontfile = flag.String("fontfile", "../../testdata/luxisr.ttf", "filename or URL of the TTF font")
	hinting  = flag.String("hinting", "none", "none, vertical or full")
	pkg      = flag.String("pkg", "example", "the package name for the generated code")
	size     = flag.Float64("size", 12, "the number of pixels in 1 em")
	vr       = flag.String("var", "example", "the variable name for the generated code")
	dump     = flag.Bool("showmetrics", false, "Show font metrics and exit (doesn't generate anything).")
	startcp  = flag.Int("start", 0, "starting codepoint")
	endcp    = flag.Int("end", unicode.MaxRune, "ending codepoint")
	output   = flag.String("output", "fts", "what to output; fts generates a go source file; png generates a set of pngs; tile generates ascii tile files")
)

func loadFontFile() ([]byte, error) {
	if strings.HasPrefix(*fontfile, "http://") || strings.HasPrefix(*fontfile, "https://") {
		resp, err := http.Get(*fontfile)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		return ioutil.ReadAll(resp.Body)
	}
	return ioutil.ReadFile(*fontfile)
}

func parseHinting(h string) font.Hinting {
	switch h {
	case "full":
		return font.HintingFull
	case "vertical":
		log.Fatal("TODO: have package truetype implement vertical hinting")
		return font.HintingVertical
	}
	return font.HintingNone
}

func privateUseArea(r rune) bool {
	return 0xe000 <= r && r <= 0xf8ff ||
		0xf0000 <= r && r <= 0xffffd ||
		0x100000 <= r && r <= 0x10fffd
}

func loadRanges(f *truetype.Font) (ret [][2]rune) {
	rr := [2]rune{-1, -1}
	for r := rune(*startcp); r <= rune(*endcp); r++ {
		if f.Index(r) == 0 {
			continue
		}
		if rr[1] == r {
			rr[1] = r + 1
			continue
		}
		if rr[0] != -1 {
			ret = append(ret, rr)
		}
		rr = [2]rune{r, r + 1}
	}
	if rr[0] != -1 {
		ret = append(ret, rr)
	}
	return ret
}

func main() {
	flag.Parse()
	b, err := loadFontFile()
	if err != nil {
		log.Fatal(err)
	}
	f, err := truetype.Parse(b)
	if err != nil {
		log.Fatal(err)
	}
	face := truetype.NewFace(f, &truetype.Options{
		Size:    *size,
		Hinting: parseHinting(*hinting),
	})
	defer face.Close()

	baseline := fixed.I(16) - face.Metrics().Descent

	if *dump {
		fmt.Printf("%+v\n", face.Metrics())
		fmt.Printf("Baseline: %+v\n", baseline)
		return
	}

	alphaCellTileSet := tiles.NewAlphaCellTileSet()

	ranges := loadRanges(f)
	for _, rr := range ranges {
		for r := rr[0]; r < rr[1]; r++ {
			dr, mask, maskp, _, ok := face.Glyph(fixed.Point26_6{fixed.I(0), baseline}, r)
			if !ok {
				log.Fatalf("could not load glyph for %U", r)
			}
			dst := &tiles.AlphaCell{}
			draw.DrawMask(dst, dr, image.White, image.Point{}, mask, maskp, draw.Src)
			alphaCellTileSet.Glyphs[r] = dst.Pix
		}
	}

	if *output == "tile" {
		for k, v := range alphaCellTileSet.Glyphs {
			fn := fmt.Sprintf("0x%04X.tile", k)
			fh, err := os.Create(fn)
			if err != nil {
				panic(err)
			}
			for _, row := range v {
				fh.Write([]byte(strings.Map(func(r rune) rune {
					switch r {
					case '0':
						return ' '
					case '1':
						return '#'
					default:
						return r
					}
				},
					fmt.Sprintf("%08b!\n", row))))
			}
			fh.Close()
		}
		return
	}

	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "package %s\n", *pkg)
	fmt.Fprintf(buf, "import \"image\"\n")
	fmt.Fprintf(buf, "import \"github.com/sparques/fansiterm/tiles\"\n")
	fmt.Fprintf(buf, "var %s = &tiles.AlphaCellTileSet{\n", *vr)
	fmt.Fprintf(buf, "Rectangle: image.Rect(0,0, 8, 16),\n")
	fmt.Fprintf(buf, "Glyphs: map[rune][16]uint8{\n")

	// maps are intentionally randomized, but we want to consistently order
	// our entries; generate a slice of the keys and sort them

	rr := make([]rune, len(alphaCellTileSet.Glyphs))
	i := 0
	for r := range alphaCellTileSet.Glyphs {
		rr[i] = r
		i++
	}
	slices.Sort(rr)

	for _, r := range rr {
		// fmt.Fprintf(buf, "\t%d: %#v,\n", r, alphaCellTileSet.Glyphs[r])
		fmt.Fprintf(buf, "\t0x%02X: [16]uint8{\n", r)
		for i := range 16 {
			fmt.Fprintf(buf, "0b%08b,\n", alphaCellTileSet.Glyphs[r][i])
		}
		fmt.Fprintf(buf, "},\n")

	}
	fmt.Fprintf(buf, "}}\n")

	fmted, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatalf("format.Source: %v", err)
	}
	if err := ioutil.WriteFile(*vr+".go", fmted, 0644); err != nil {
		log.Fatalf("ioutil.WriteFile: %v", err)
	}
}
