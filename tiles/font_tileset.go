package tiles

import (
	"image"
	"image/color"
	"image/draw"
	"maps"
)

type FontTileSet struct {
	image.Rectangle

	// Glyphs maps a rune to compact alpha pixel data.
	Glyphs map[rune][]uint8

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
	maps.Copy(fts.Glyphs, src.Glyphs)
}

func (fts *FontTileSet) Glyph(r rune) *image.Alpha {
	pix, ok := fts.Glyphs[r]
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
	fts.Glyphs[r] = extractAlpha(img)
}

func (fts *FontTileSet) DrawTile(r rune, dst draw.Image, pt image.Point, fg color.Color, bg color.Color) {
	pix, ok := fts.Glyphs[r]
	if !ok {
		if shouldUseFallback(fts) {
			Fallback.DrawTile(r, dst, pt, fg, bg)
			return
		}
		pix = EmptyTile.Pix
	}

	drawAlphaPixels(dst, pt, pix, fts.Dx(), fts.Dy(), fg, bg)
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

	// Glyphs maps a rune to 8x16 1-bit pixel data.
	Glyphs map[rune][16]uint8

	glyph AlphaCell
}

func NewAlphaCellTileSet() *AlphaCellTileSet {
	return &AlphaCellTileSet{
		Rectangle: image.Rect(0, 0, 8, 16),
		Glyphs:    make(map[rune][16]uint8),
	}
}

func (ats *AlphaCellTileSet) Glyph(r rune) *AlphaCell {
	pix, ok := ats.Glyphs[r]
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

func (ats *AlphaCellTileSet) DrawTile(r rune, dst draw.Image, pt image.Point, fg color.Color, bg color.Color) {
	pix, ok := ats.Glyphs[r]
	if !ok {
		if shouldUseFallback(ats) {
			Fallback.DrawTile(r, dst, pt, fg, bg)
			return
		}
		pix = [16]uint8{}
	}

	drawAlphaCell(dst, pt, pix, fg, bg)
}

func (ats *AlphaCellTileSet) drawTileImage(img image.Image, dst draw.Image, pt image.Point, fg color.Color, bg color.Color) {
	cell, ok := img.(*AlphaCell)
	if !ok {
		drawTile(dst, pt, img, fg, bg)
		return
	}
	drawAlphaCell(dst, pt, cell.Pix, fg, bg)
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
