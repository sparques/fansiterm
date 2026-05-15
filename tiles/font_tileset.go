package tiles

import (
	"image"
	"image/color"
	"image/draw"
	"slices"
)

type FontTileSet struct {
	image.Rectangle

	// Packed dense/sparse storage for generated sets.
	First  rune
	Count  int
	Index  []uint16
	Sparse []rune
	Pix    []uint8

	// glyph is reused to avoid per-call allocations in Glyph/GetTile.
	glyph image.Alpha
}

func NewFontTileSet() *FontTileSet {
	return &FontTileSet{}
}

// Merge copies code points / glyphs into fts, displacing any overlapping code
// points.
func (fts *FontTileSet) Merge(src *FontTileSet) {
	if src == nil {
		return
	}
	for _, r := range src.Runes() {
		if glyph := src.Glyph(r); glyph != nil {
			fts.SetTile(r, glyph)
		}
	}
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
	if img == nil {
		return
	}
	if fts.Rectangle.Empty() {
		bounds := img.Bounds()
		fts.Rectangle = image.Rect(0, 0, bounds.Dx(), bounds.Dy())
	}
	pix := extractAlphaCell(img, fts.Dx(), fts.Dy())
	if len(pix) == 0 {
		return
	}
	stride := fts.glyphArea()

	// does it already exist? If so we can just displace it.
	if ord, ok := fts.lookupOrdinal(r); ok {
		start := ord * stride
		end := start + stride
		if start >= 0 && end <= len(fts.Pix) {
			copy(fts.Pix[start:end], pix)
		}
		return
	}

	// doesn't already exist, must add new entry

	// Are we a sparse setup? That's pretty easy
	if len(fts.Sparse) > 0 {
		fts.appendSparse(r, pix)
		return
	}

	// this is the complicated part... if the rune is in order, we can just append it, otherwise
	// we have to convert from simple Index to Sparse
	if len(fts.Index) == 0 && fts.Count == 0 {
		fts.First = r
		fts.Count = 1
		fts.Index = []uint16{1}
		fts.Pix = append(fts.Pix, pix...)
		return
	}

	if len(fts.Index) > 0 {
		i := int(r - fts.First)
		if i == len(fts.Index) {
			fts.Index = append(fts.Index, uint16(fts.Count+1))
			fts.Count++
			fts.Pix = append(fts.Pix, pix...)
			return
		}
		if 0 <= i && i < len(fts.Index) && fts.Index[i] == 0 {
			fts.Index[i] = uint16(fts.Count + 1)
			fts.Count++
			fts.Pix = append(fts.Pix, pix...)
			return
		}
	}

	fts.convertToSparse()
	fts.appendSparse(r, pix)
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
	default:
		return 0
	}
}

func (fts *FontTileSet) Runes() []rune {
	rr := make([]rune, 0, fts.Len())
	if len(fts.Index) > 0 {
		for i, ord := range fts.Index {
			if ord != 0 {
				rr = append(rr, fts.First+rune(i))
			}
		}
		return rr
	}

	rr = append(rr, fts.Sparse...)
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

	First  rune
	Count  int
	Index  []uint16
	Sparse []rune
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
	Sparse []rune
	Pix    []uint8

	glyph Alpha1
}

func NewAlphaCellTileSet() *AlphaCellTileSet {
	return &AlphaCellTileSet{
		Rectangle: image.Rect(0, 0, 8, 16),
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

func (ats *AlphaCellTileSet) SetTile(r rune, img image.Image) {
	if img == nil {
		return
	}
	if ats.Rectangle.Empty() {
		ats.Rectangle = image.Rect(0, 0, 8, 16)
	}

	pix := extractAlphaCellBits(img)
	if ord, ok := ats.lookupOrdinal(r); ok {
		if ord < len(ats.Cells) {
			ats.Cells[ord] = pix
		}
		return
	}

	if len(ats.Sparse) > 0 {
		ats.appendSparse(r, pix)
		return
	}

	if len(ats.Index) == 0 && ats.Count == 0 {
		ats.First = r
		ats.Count = 1
		ats.Index = []uint16{1}
		ats.Cells = append(ats.Cells, pix)
		return
	}

	if len(ats.Index) > 0 {
		i := int(r - ats.First)
		if i == len(ats.Index) {
			ats.Index = append(ats.Index, uint16(ats.Count+1))
			ats.Count++
			ats.Cells = append(ats.Cells, pix)
			return
		}
		if 0 <= i && i < len(ats.Index) && ats.Index[i] == 0 {
			ats.Index[i] = uint16(ats.Count + 1)
			ats.Count++
			ats.Cells = append(ats.Cells, pix)
			return
		}
	}

	ats.convertToSparse()
	ats.appendSparse(r, pix)
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

func (ats *Alpha1TileSet) SetTile(r rune, img image.Image) {
	if img == nil {
		return
	}
	if ats.Rectangle.Empty() {
		bounds := img.Bounds()
		ats.Rectangle = image.Rect(0, 0, bounds.Dx(), bounds.Dy())
	}
	pix := extractAlpha1Bits(img, ats.Dx(), ats.Dy())
	if len(pix) == 0 {
		return
	}
	stride := ats.glyphStride()

	if ord, ok := ats.lookupOrdinal(r); ok {
		start := ord * stride
		end := start + stride
		if start >= 0 && end <= len(ats.Pix) {
			copy(ats.Pix[start:end], pix)
		}
		return
	}

	if len(ats.Sparse) > 0 {
		ats.appendSparse(r, pix)
		return
	}

	if len(ats.Index) == 0 && ats.Count == 0 {
		ats.First = r
		ats.Count = 1
		ats.Index = []uint16{1}
		ats.Pix = append(ats.Pix, pix...)
		return
	}

	if len(ats.Index) > 0 {
		i := int(r - ats.First)
		if i == len(ats.Index) {
			ats.Index = append(ats.Index, uint16(ats.Count+1))
			ats.Count++
			ats.Pix = append(ats.Pix, pix...)
			return
		}
		if 0 <= i && i < len(ats.Index) && ats.Index[i] == 0 {
			ats.Index[i] = uint16(ats.Count + 1)
			ats.Count++
			ats.Pix = append(ats.Pix, pix...)
			return
		}
	}

	ats.convertToSparse()
	ats.appendSparse(r, pix)
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
	default:
		return 0
	}
}

func (ats *AlphaCellTileSet) Runes() []rune {
	rr := make([]rune, 0, ats.Len())
	if len(ats.Index) > 0 {
		for i, ord := range ats.Index {
			if ord != 0 {
				rr = append(rr, ats.First+rune(i))
			}
		}
		return rr
	}

	rr = append(rr, ats.Sparse...)
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

	rr = append(rr, ats.Sparse...)
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

func extractAlphaCell(img image.Image, width int, height int) []uint8 {
	if width <= 0 || height <= 0 {
		return nil
	}

	bounds := img.Bounds()
	pix := make([]uint8, width*height)
	maxY := min(height, bounds.Dy())
	maxX := min(width, bounds.Dx())
	if maxX <= 0 || maxY <= 0 {
		return pix
	}

	if alphaImg, ok := img.(*image.Alpha); ok {
		for y := 0; y < maxY; y++ {
			src := alphaImg.PixOffset(bounds.Min.X, bounds.Min.Y+y)
			dst := y * width
			copy(pix[dst:dst+maxX], alphaImg.Pix[src:src+maxX])
		}
		return pix
	}

	for y := 0; y < maxY; y++ {
		srcY := bounds.Min.Y + y
		row := y * width
		for x := 0; x < maxX; x++ {
			_, _, _, alpha := img.At(bounds.Min.X+x, srcY).RGBA()
			pix[row+x] = uint8(alpha / 0x101)
		}
	}
	return pix
}

func extractAlphaCellBits(img image.Image) [16]uint8 {
	var pix [16]uint8
	if img == nil {
		return pix
	}

	bounds := img.Bounds()
	maxY := min(len(pix), bounds.Dy())
	maxX := min(8, bounds.Dx())
	for y := 0; y < maxY; y++ {
		srcY := bounds.Min.Y + y
		for x := 0; x < maxX; x++ {
			if BitAlphaModel.Convert(img.At(bounds.Min.X+x, srcY)).(BitAlpha) {
				pix[y] |= 0x80 >> x
			}
		}
	}
	return pix
}

func extractAlpha1Bits(img image.Image, width int, height int) []uint8 {
	stride := bytesPerRow(width)
	if stride <= 0 || height <= 0 {
		return nil
	}

	bounds := img.Bounds()
	pix := make([]uint8, stride*height)
	maxY := min(height, bounds.Dy())
	maxX := min(width, bounds.Dx())
	for y := 0; y < maxY; y++ {
		srcY := bounds.Min.Y + y
		row := y * stride
		for x := 0; x < maxX; x++ {
			if BitAlphaModel.Convert(img.At(bounds.Min.X+x, srcY)).(BitAlpha) {
				pix[row+x/8] |= 0x80 >> (x % 8)
			}
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
	return nil, false
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
		return slices.BinarySearch(fts.Sparse, r)
	}
	return 0, false
}

func (fts *FontTileSet) appendSparse(r rune, pix []uint8) {
	i, ok := slices.BinarySearch(fts.Sparse, r)
	if ok {
		return
	}
	stride := fts.glyphArea()
	fts.Count++
	fts.Sparse = slices.Insert(fts.Sparse, i, r)
	fts.Pix = slices.Insert(fts.Pix, i*stride, pix...)
}

func (fts *FontTileSet) convertToSparse() {
	if len(fts.Sparse) > 0 {
		return
	}
	if len(fts.Index) > 0 {
		stride := fts.glyphArea()
		oldPix := fts.Pix
		fts.Sparse = make([]rune, 0, fts.Count+1)
		fts.Pix = make([]uint8, 0, len(oldPix))
		for i, ord := range fts.Index {
			if ord != 0 {
				start := int(ord-1) * stride
				end := start + stride
				if start >= 0 && end <= len(oldPix) {
					fts.Sparse = append(fts.Sparse, fts.First+rune(i))
					fts.Pix = append(fts.Pix, oldPix[start:end]...)
				}
			}
		}
		fts.Count = len(fts.Sparse)
		fts.Index = nil
		fts.First = 0
		return
	}

	fts.Sparse = make([]rune, 0, fts.Count+1)
	for i := 0; i < fts.Count; i++ {
		fts.Sparse = append(fts.Sparse, fts.First+rune(i))
	}
	fts.First = 0
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
		return slices.BinarySearch(ats.Sparse, r)
	}
	return 0, false
}

func (ats *Alpha1TileSet) appendSparse(r rune, pix []uint8) {
	i, ok := slices.BinarySearch(ats.Sparse, r)
	if ok {
		return
	}
	stride := ats.glyphStride()
	ats.Count++
	ats.Sparse = slices.Insert(ats.Sparse, i, r)
	ats.Pix = slices.Insert(ats.Pix, i*stride, pix...)
}

func (ats *Alpha1TileSet) convertToSparse() {
	if len(ats.Sparse) > 0 {
		return
	}
	if len(ats.Index) > 0 {
		stride := ats.glyphStride()
		oldPix := ats.Pix
		ats.Sparse = make([]rune, 0, ats.Count+1)
		ats.Pix = make([]uint8, 0, len(oldPix))
		for i, ord := range ats.Index {
			if ord != 0 {
				start := int(ord-1) * stride
				end := start + stride
				if start >= 0 && end <= len(oldPix) {
					ats.Sparse = append(ats.Sparse, ats.First+rune(i))
					ats.Pix = append(ats.Pix, oldPix[start:end]...)
				}
			}
		}
		ats.Count = len(ats.Sparse)
		ats.Index = nil
		ats.First = 0
		return
	}

	ats.Sparse = make([]rune, 0, ats.Count+1)
	for i := 0; i < ats.Count; i++ {
		ats.Sparse = append(ats.Sparse, ats.First+rune(i))
	}
	ats.First = 0
}

func (ats *AlphaCellTileSet) lookupGlyph(r rune) ([16]uint8, bool) {
	if ord, ok := ats.lookupOrdinal(r); ok && ord < len(ats.Cells) {
		return ats.Cells[ord], true
	}
	return [16]uint8{}, false
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
		return slices.BinarySearch(ats.Sparse, r)
	}
	return 0, false
}

func (ats *AlphaCellTileSet) appendSparse(r rune, pix [16]uint8) {
	i, ok := slices.BinarySearch(ats.Sparse, r)
	if ok {
		return
	}
	ats.Count++
	ats.Sparse = slices.Insert(ats.Sparse, i, r)
	ats.Cells = slices.Insert(ats.Cells, i, pix)
}

func (ats *AlphaCellTileSet) convertToSparse() {
	if len(ats.Sparse) > 0 {
		return
	}
	if len(ats.Index) > 0 {
		oldCells := ats.Cells
		ats.Sparse = make([]rune, 0, ats.Count+1)
		ats.Cells = make([][16]uint8, 0, len(oldCells))
		for i, ord := range ats.Index {
			if ord != 0 {
				idx := int(ord - 1)
				if idx >= 0 && idx < len(oldCells) {
					ats.Sparse = append(ats.Sparse, ats.First+rune(i))
					ats.Cells = append(ats.Cells, oldCells[idx])
				}
			}
		}
		ats.Count = len(ats.Sparse)
		ats.Index = nil
		ats.First = 0
		return
	}

	ats.Sparse = make([]rune, 0, ats.Count+1)
	for i := 0; i < ats.Count; i++ {
		ats.Sparse = append(ats.Sparse, ats.First+rune(i))
	}
	ats.First = 0
}
