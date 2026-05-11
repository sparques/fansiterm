package tiles

import (
	"cmp"
	"image"
	"image/color"
	"image/draw"
	"maps"
	"slices"
)

// RuneIndex maps a rune to a 1-based glyph ordinal in packed storage.
type RuneIndex struct {
	Rune  rune
	Index uint16
}

type FontTileSet struct {
	image.Rectangle

	// Glyphs is retained for mutable / compatibility-oriented construction.
	// Generated sets should prefer the packed fields below.
	Glyphs map[rune][]uint8

	// Packed dense/sparse storage for generated sets.
	First  rune
	Count  int
	Index  []uint16
	Sparse []RuneIndex
	Pix    []uint8

	// glyph is reused to avoid per-call allocations in Glyph/GetTile.
	glyph image.Alpha
}

func NewFontTileSet() *FontTileSet {
	return &FontTileSet{
		Glyphs: make(map[rune][]uint8),
	}
}

// Merge copies code points / glyphs into fts, displacing any overlapping code points.
func (fts *FontTileSet) Merge(src *FontTileSet) {
	if src == nil {
		return
	}
	fts.materializeGlyphMap()
	srcMap := src.glyphMap()
	maps.Copy(fts.Glyphs, srcMap)
}

func (fts *FontTileSet) Glyph(r rune) *image.Alpha {
	pix, ok := fts.lookupGlyph(r)
	if !ok {
		return nil
	}
	fts.glyph.Pix = pix
	fts.glyph.Rect = fts.Rectangle
	fts.glyph.Stride = fts.Dx()
	return &fts.glyph
}

func (fts *FontTileSet) GetTile(r rune) (image.Image, bool) {
	glyph := fts.Glyph(r)
	if glyph == nil {
		return nil, false
	}
	return glyph, true
}

func (fts *FontTileSet) SetTile(r rune, img image.Image) {
	fts.materializeGlyphMap()
	fts.Glyphs[r] = extractAlpha(img)
	fts.clearPacked()
}

func (fts *FontTileSet) DrawTile(r rune, dst draw.Image, pt image.Point, fg color.Color, bg color.Color) {
	pix, ok := fts.lookupGlyph(r)
	if !ok {
		if shouldUseFallback(fts) {
			Fallback.DrawTile(r, dst, pt, fg, bg)
			return
		}
		pix = EmptyTile.Pix
	}

	drawAlphaPixels(dst, pt, pix, fts.Dx(), fts.Dy(), fg, bg)
}

func (fts *FontTileSet) Len() int {
	switch {
	case fts.Count != 0:
		return fts.Count
	case fts.Glyphs != nil:
		return len(fts.Glyphs)
	default:
		return 0
	}
}

func (fts *FontTileSet) Runes() []rune {
	if fts.Glyphs != nil {
		rr := make([]rune, 0, len(fts.Glyphs))
		for r := range fts.Glyphs {
			rr = append(rr, r)
		}
		slices.Sort(rr)
		return rr
	}

	rr := make([]rune, 0, fts.Len())
	if len(fts.Index) > 0 {
		for i, ord := range fts.Index {
			if ord != 0 {
				rr = append(rr, fts.First+rune(i))
			}
		}
		return rr
	}

	for _, entry := range fts.Sparse {
		rr = append(rr, entry.Rune)
	}
	return rr
}

func (fts *FontTileSet) drawTileImage(img image.Image, dst draw.Image, pt image.Point, fg color.Color, bg color.Color) {
	alphaImg, ok := img.(*image.Alpha)
	if !ok {
		drawTile(dst, pt, img, fg, bg)
		return
	}
	drawAlphaPixels(dst, pt, alphaImg.Pix, alphaImg.Rect.Dx(), alphaImg.Rect.Dy(), fg, bg)
}

func drawAlphaPixels(dst draw.Image, pt image.Point, pix []uint8, width int, height int, fg color.Color, bg color.Color) {
	bgr, bgg, bgb, _ := bg.RGBA()
	fgr, fgg, fgb, _ := fg.RGBA()

	for y := 0; y < height; y++ {
		row := y * width
		for x := 0; x < width; x++ {
			alpha8 := pix[row+x]
			switch alpha8 {
			case 0x00:
				dst.Set(pt.X+x, pt.Y+y, bg)
			case 0xFF:
				dst.Set(pt.X+x, pt.Y+y, fg)
			default:
				alpha := uint32(alpha8) * 0x101
				dst.Set(pt.X+x, pt.Y+y, color.RGBA{
					R: alphaBlend(bgr, fgr, alpha),
					G: alphaBlend(bgg, fgg, alpha),
					B: alphaBlend(bgb, fgb, alpha),
					A: 0xFF,
				})
			}
		}
	}
}

// AlphaCell is a 1-bit-depth image.Image that is always 8x16.
type AlphaCell struct {
	Pix [16]uint8
}

func (ac *AlphaCell) At(x, y int) color.Color {
	return BitAlpha((ac.Pix[y]<<x)&0x80 == 0x80)
}

func (ac *AlphaCell) Set(x, y int, c color.Color) {
	native := bitAlphaModel(c).(BitAlpha)
	if native {
		ac.Pix[y] |= 0x80 >> x
	} else {
		ac.Pix[y] &= ^(0x80 >> x)
	}
}

func (ac *AlphaCell) Bounds() image.Rectangle {
	return image.Rect(0, 0, 8, 16)
}

func (ac *AlphaCell) ColorModel() color.Model {
	return BitAlphaModel
}

type AlphaCellTileSet struct {
	image.Rectangle

	// Glyphs is retained for mutable / compatibility-oriented construction.
	Glyphs map[rune][16]uint8

	First  rune
	Count  int
	Index  []uint16
	Sparse []RuneIndex
	Cells  [][16]uint8

	glyph AlphaCell
}

// Alpha1TileSet is a packed one-bit tile set with fixed tile dimensions.
//
// Each glyph occupies bytesPerRow(width) * height bytes in Pix. Rune lookup uses
// the same dense Index or sorted Sparse tables as FontTileSet and
// AlphaCellTileSet.
type Alpha1TileSet struct {
	image.Rectangle

	First  rune
	Count  int
	Index  []uint16
	Sparse []RuneIndex
	Pix    []uint8

	glyph Alpha1
}

func NewAlphaCellTileSet() *AlphaCellTileSet {
	return &AlphaCellTileSet{
		Rectangle: image.Rect(0, 0, 8, 16),
		Glyphs:    make(map[rune][16]uint8),
	}
}

func NewAlpha1TileSet(width int, height int) *Alpha1TileSet {
	return &Alpha1TileSet{
		Rectangle: image.Rect(0, 0, width, height),
	}
}

func (ats *AlphaCellTileSet) Glyph(r rune) *AlphaCell {
	pix, ok := ats.lookupGlyph(r)
	if !ok {
		return nil
	}
	ats.glyph.Pix = pix
	return &ats.glyph
}

func (ats *AlphaCellTileSet) GetTile(r rune) (image.Image, bool) {
	glyph := ats.Glyph(r)
	if glyph == nil {
		return nil, false
	}
	return glyph, true
}

func (ats *Alpha1TileSet) Glyph(r rune) *Alpha1 {
	pix, ok := ats.lookupGlyph(r)
	if !ok {
		return nil
	}
	ats.glyph.Pix = pix
	ats.glyph.Stride = bytesPerRow(ats.Dx())
	ats.glyph.Rect = ats.Rectangle
	return &ats.glyph
}

func (ats *Alpha1TileSet) GetTile(r rune) (image.Image, bool) {
	glyph := ats.Glyph(r)
	if glyph == nil {
		return nil, false
	}
	return glyph, true
}

func (ats *AlphaCellTileSet) DrawTile(r rune, dst draw.Image, pt image.Point, fg color.Color, bg color.Color) {
	pix, ok := ats.lookupGlyph(r)
	if !ok {
		if shouldUseFallback(ats) {
			Fallback.DrawTile(r, dst, pt, fg, bg)
			return
		}
		pix = [16]uint8{}
	}

	drawAlphaCell(dst, pt, pix, fg, bg)
}

func (ats *Alpha1TileSet) DrawTile(r rune, dst draw.Image, pt image.Point, fg color.Color, bg color.Color) {
	pix, ok := ats.lookupGlyph(r)
	if !ok {
		if shouldUseFallback(ats) {
			Fallback.DrawTile(r, dst, pt, fg, bg)
			return
		}
		return
	}

	drawAlpha1Pixels(dst, pt, pix, ats.Dx(), ats.Dy(), bytesPerRow(ats.Dx()), fg, bg)
}

func (ats *AlphaCellTileSet) Len() int {
	switch {
	case ats.Count != 0:
		return ats.Count
	case ats.Glyphs != nil:
		return len(ats.Glyphs)
	default:
		return 0
	}
}

func (ats *AlphaCellTileSet) Runes() []rune {
	if ats.Glyphs != nil {
		rr := make([]rune, 0, len(ats.Glyphs))
		for r := range ats.Glyphs {
			rr = append(rr, r)
		}
		slices.Sort(rr)
		return rr
	}

	rr := make([]rune, 0, ats.Len())
	if len(ats.Index) > 0 {
		for i, ord := range ats.Index {
			if ord != 0 {
				rr = append(rr, ats.First+rune(i))
			}
		}
		return rr
	}

	for _, entry := range ats.Sparse {
		rr = append(rr, entry.Rune)
	}
	return rr
}

func (ats *Alpha1TileSet) Len() int {
	return ats.Count
}

func (ats *Alpha1TileSet) Runes() []rune {
	rr := make([]rune, 0, ats.Len())
	if len(ats.Index) > 0 {
		for i, ord := range ats.Index {
			if ord != 0 {
				rr = append(rr, ats.First+rune(i))
			}
		}
		return rr
	}

	for _, entry := range ats.Sparse {
		rr = append(rr, entry.Rune)
	}
	return rr
}

func (ats *AlphaCellTileSet) drawTileImage(img image.Image, dst draw.Image, pt image.Point, fg color.Color, bg color.Color) {
	cell, ok := img.(*AlphaCell)
	if !ok {
		drawTile(dst, pt, img, fg, bg)
		return
	}
	drawAlphaCell(dst, pt, cell.Pix, fg, bg)
}

func (ats *Alpha1TileSet) drawTileImage(img image.Image, dst draw.Image, pt image.Point, fg color.Color, bg color.Color) {
	alpha1, ok := img.(*Alpha1)
	if !ok {
		drawTile(dst, pt, img, fg, bg)
		return
	}
	drawAlpha1Pixels(dst, pt, alpha1.Pix, alpha1.Rect.Dx(), alpha1.Rect.Dy(), alpha1.Stride, fg, bg)
}

func drawAlphaCell(dst draw.Image, pt image.Point, pix [16]uint8, fg color.Color, bg color.Color) {
	_, _, _, bgAlpha := bg.RGBA()
	drawBG := bgAlpha >= m/2

	for y := 0; y < len(pix); y++ {
		row := pix[y]
		for x := 0; x < 8; x++ {
			if (row>>(7-x))&1 == 1 {
				dst.Set(pt.X+x, pt.Y+y, fg)
			} else if drawBG {
				dst.Set(pt.X+x, pt.Y+y, bg)
			}
		}
	}
}

func drawAlpha1Pixels(dst draw.Image, pt image.Point, pix []uint8, width int, height int, stride int, fg color.Color, bg color.Color) {
	_, _, _, bgAlpha := bg.RGBA()
	drawBG := bgAlpha >= m/2

	for y := 0; y < height; y++ {
		row := y * stride
		for x := 0; x < width; x++ {
			if (pix[row+x/8]>>(7-(x%8)))&1 == 1 {
				dst.Set(pt.X+x, pt.Y+y, fg)
			} else if drawBG {
				dst.Set(pt.X+x, pt.Y+y, bg)
			}
		}
	}
}

func extractAlpha(img image.Image) []uint8 {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		return nil
	}

	if alphaImg, ok := img.(*image.Alpha); ok {
		if alphaImg.Rect.Eq(bounds) && alphaImg.Stride == width && len(alphaImg.Pix) == width*height {
			return alphaImg.Pix
		}

		pix := make([]uint8, width*height)
		for y := 0; y < height; y++ {
			src := alphaImg.PixOffset(bounds.Min.X, bounds.Min.Y+y)
			dst := y * width
			copy(pix[dst:dst+width], alphaImg.Pix[src:src+width])
		}
		return pix
	}

	pix := make([]uint8, width*height)
	for y := 0; y < height; y++ {
		srcY := bounds.Min.Y + y
		row := y * width
		for x := 0; x < width; x++ {
			_, _, _, alpha := img.At(bounds.Min.X+x, srcY).RGBA()
			pix[row+x] = uint8(alpha / 0x101)
		}
	}
	return pix
}

func (fts *FontTileSet) glyphArea() int {
	return fts.Dx() * fts.Dy()
}

func (fts *FontTileSet) lookupGlyph(r rune) ([]uint8, bool) {
	if ord, ok := fts.lookupOrdinal(r); ok {
		stride := fts.glyphArea()
		start := ord * stride
		end := start + stride
		if start >= 0 && end <= len(fts.Pix) {
			return fts.Pix[start:end], true
		}
	}
	if fts.Glyphs == nil {
		return nil, false
	}
	pix, ok := fts.Glyphs[r]
	return pix, ok
}

func (fts *FontTileSet) lookupOrdinal(r rune) (int, bool) {
	if len(fts.Index) > 0 {
		i := int(r - fts.First)
		if 0 <= i && i < len(fts.Index) {
			if ord := fts.Index[i]; ord != 0 {
				return int(ord - 1), true
			}
		}
	}
	if len(fts.Sparse) > 0 {
		i, ok := slices.BinarySearchFunc(fts.Sparse, r, func(entry RuneIndex, target rune) int {
			return cmp.Compare(entry.Rune, target)
		})
		if ok {
			return int(fts.Sparse[i].Index - 1), true
		}
	}
	return 0, false
}

func (fts *FontTileSet) glyphMap() map[rune][]uint8 {
	if fts.Glyphs != nil {
		return fts.Glyphs
	}
	ret := make(map[rune][]uint8, fts.Len())
	for _, r := range fts.Runes() {
		if pix, ok := fts.lookupGlyph(r); ok {
			buf := make([]byte, len(pix))
			copy(buf, pix)
			ret[r] = buf
		}
	}
	return ret
}

func (fts *FontTileSet) materializeGlyphMap() {
	if fts.Glyphs == nil {
		fts.Glyphs = fts.glyphMap()
	}
}

func (fts *FontTileSet) clearPacked() {
	fts.First = 0
	fts.Count = 0
	fts.Index = nil
	fts.Sparse = nil
	fts.Pix = nil
}

func (ats *Alpha1TileSet) glyphStride() int {
	return bytesPerRow(ats.Dx()) * ats.Dy()
}

func (ats *Alpha1TileSet) lookupGlyph(r rune) ([]uint8, bool) {
	if ord, ok := ats.lookupOrdinal(r); ok {
		stride := ats.glyphStride()
		start := ord * stride
		end := start + stride
		if start >= 0 && end <= len(ats.Pix) {
			return ats.Pix[start:end], true
		}
	}
	return nil, false
}

func (ats *Alpha1TileSet) lookupOrdinal(r rune) (int, bool) {
	if len(ats.Index) > 0 {
		i := int(r - ats.First)
		if 0 <= i && i < len(ats.Index) {
			if ord := ats.Index[i]; ord != 0 {
				return int(ord - 1), true
			}
		}
	}
	if len(ats.Sparse) > 0 {
		i, ok := slices.BinarySearchFunc(ats.Sparse, r, func(entry RuneIndex, target rune) int {
			return cmp.Compare(entry.Rune, target)
		})
		if ok {
			return int(ats.Sparse[i].Index - 1), true
		}
	}
	return 0, false
}

func (ats *AlphaCellTileSet) lookupGlyph(r rune) ([16]uint8, bool) {
	if ord, ok := ats.lookupOrdinal(r); ok && ord < len(ats.Cells) {
		return ats.Cells[ord], true
	}
	if ats.Glyphs == nil {
		return [16]uint8{}, false
	}
	pix, ok := ats.Glyphs[r]
	return pix, ok
}

func (ats *AlphaCellTileSet) lookupOrdinal(r rune) (int, bool) {
	if len(ats.Index) > 0 {
		i := int(r - ats.First)
		if 0 <= i && i < len(ats.Index) {
			if ord := ats.Index[i]; ord != 0 {
				return int(ord - 1), true
			}
		}
	}
	if len(ats.Sparse) > 0 {
		i, ok := slices.BinarySearchFunc(ats.Sparse, r, func(entry RuneIndex, target rune) int {
			return cmp.Compare(entry.Rune, target)
		})
		if ok {
			return int(ats.Sparse[i].Index - 1), true
		}
	}
	return 0, false
}
