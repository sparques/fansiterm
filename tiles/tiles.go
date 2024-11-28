package tiles

import (
	"image"
	"image/color"
	"image/draw"
	_ "image/png"
	"io"
	"maps"
	"math"
	"os"
)

var EmptyTile = image.NewAlpha(image.Rect(0, 0, 8, 16))

// Fallback is used when a FontTileSet cannot find a Glyph for a rune. By default Fallback is initialized to an internal fallback
// that implements Tiler such that all runes return EmptyTile.
var Fallback = &fallback{}

type Tiler interface {
	DrawTile(r rune, dst draw.Image, pt image.Point, fg color.Color, bg color.Color)
	GetTile(r rune) image.Image
}

type FontTileSet struct {
	image.Rectangle
	// Glyphs maps a rune to a slice of alpha pixel data
	Glyphs map[rune][]uint8
}

func NewFontTileSet() *FontTileSet {
	return &FontTileSet{
		Glyphs: make(map[rune][]uint8),
	}
}

// Merge copiess code points / glyphs into fts, displacing any overlapping code points.
func (fts *FontTileSet) Merge(src *FontTileSet) {
	maps.Copy(fts.Glyphs, src.Glyphs)
}

func (fts *FontTileSet) Glyph(r rune) *image.Alpha {
	return &image.Alpha{
		Pix:    fts.Glyphs[r],
		Stride: fts.Dx(),
		Rect:   fts.Rectangle,
	}
}

func (fts *FontTileSet) GetTile(r rune) image.Image {
	return fts.Glyph(r)
}

func (fts *FontTileSet) SetTile(r rune, img image.Image) {
	fts.Glyphs[r] = getPix(img)
}

func (fts *FontTileSet) DrawTile(r rune, dst draw.Image, pt image.Point, fg color.Color, bg color.Color) {
	pix, ok := fts.Glyphs[r]
	if !ok {
		if Tiler(Fallback) != Tiler(fts) {
			Fallback.DrawTile(r, dst, pt, fg, bg)
			return
		}
		// fallback to fallback, use EmptyTile
		pix = EmptyTile.Pix

	}
	for x := 0; x < fts.Rectangle.Dx(); x++ {
		for y := 0; y < fts.Rectangle.Dy(); y++ {
			switch pix[y*fts.Dx()+x] {
			// skip all the math for the most common values: 0x00 and 0xFF
			case 0x00:
				dst.Set(pt.X+x, pt.Y+y, bg)
			case 0xFF:
				dst.Set(pt.X+x, pt.Y+y, fg)
			default:
				// alpha is stored as a uint8, but all the color values are uint16
				// multiply by 0x101 to scale uint8 to uint16
				alpha := uint32(fts.Glyphs[r][y*fts.Dx()+x]) * 0x101
				bgr, bgg, bgb, _ := bg.RGBA()
				fgr, fgg, fgb, _ := fg.RGBA()

				dst.Set(pt.X+x, pt.Y+y,
					color.RGBA{
						alphaBlend(bgr, fgr, alpha),
						alphaBlend(bgg, fgg, alpha),
						alphaBlend(bgb, fgb, alpha),
						255})
			}
		}
	}
}

type BitColor bool

func (bc BitColor) RGBA() (r, g, b, a uint32) {
	if bc {
		a = 0xFF * 0x101
	}
	return
}

var BitColorModel = color.ModelFunc(bitColorModel)

func bitColorModel(c color.Color) color.Color {
	if b, ok := c.(BitColor); ok {
		return b
	}
	_, _, _, a := c.RGBA()
	if a > 0xFF*0x101/2 {
		return BitColor(true)
	}
	return BitColor(false)
}

// AlphaCell is a 1-bit-depth image.Image that is always 8x16
type AlphaCell struct {
	Pix [16]uint8
}

func (ac *AlphaCell) At(x, y int) color.Color {
	return BitColor((ac.Pix[y]<<x)&8 == 8)
}

func (ac *AlphaCell) Set(x, y int, c color.Color) {
	native := bitColorModel(c).(BitColor)
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
	return BitColorModel
}

// Alpha1 is a single bit-depth image.Image.
type Alpha1 struct {
	Pix    []uint8
	Stride int
	Rect   image.Rectangle
}

func (a *Alpha1) At(x, y int) (c color.Color) {
	if !image.Pt(x, y).In(a.Rect) {
		return BitColor(false)
	}
	return BitColor((a.Pix[y*a.Stride+x/8]<<(x%8))&8 == 8)
}

func (a *Alpha1) Set(x, y int, c color.Color) {
	native := bitColorModel(c).(BitColor)

	if native {
		a.Pix[y*a.Stride+x/8] |= 1 << (x % 8)
	} else {
		a.Pix[y*a.Stride+x/8] &= ^(1 << (x % 8))
	}
}

type AlphaCellTileSet struct {
	image.Rectangle
	// Glyphs maps a rune to a slice of alpha pixel data
	Glyphs map[rune][16]uint8
}

func (ats *AlphaCellTileSet) Glyph(r rune) *AlphaCell {
	return &AlphaCell{
		Pix: ats.Glyphs[r],
	}
}

func (ats *AlphaCellTileSet) GetTile(r rune) image.Image {
	return ats.Glyph(r)
}

func (ats *AlphaCellTileSet) DrawTile(r rune, dst draw.Image, pt image.Point, fg color.Color, bg color.Color) {
	pix, ok := ats.Glyphs[r]
	if !ok {
		if Tiler(Fallback) != Tiler(ats) {
			Fallback.DrawTile(r, dst, pt, fg, bg)
			return
		}
		// fallback to fallback, use EmptyTile
		pix = [16]uint8{}

	}
	for y := range len(pix) {
		for x := 0; x < 8; x++ {
			if (pix[y]>>(7-x))&1 == 1 {
				dst.Set(pt.X+x, pt.Y+y, fg)
			} else {
				dst.Set(pt.X+x, pt.Y+y, bg)
			}
		}
	}
}

// Remap lets you remap runes in a FontTileSet. This is useful, for example, to make it so
// line-drawing mode can simply use the same font, but use unicode glyphs.
type Remap struct {
	*FontTileSet
	Map map[rune]rune
}

func NewRemap(base *FontTileSet) *Remap {
	return &Remap{
		FontTileSet: base,
		Map:         make(map[rune]rune),
	}
}

func (remap *Remap) DrawTile(r rune, dst draw.Image, pt image.Point, fg color.Color, bg color.Color) {
	if newr, ok := remap.Map[r]; ok {
		r = newr
	}
	remap.FontTileSet.DrawTile(r, dst, pt, fg, bg)
}

func (remap *Remap) GetTile(r rune) image.Image {
	if newr, ok := remap.Map[r]; ok {
		r = newr
	}
	return remap.FontTileSet.Glyph(r)
}

// fallback implements Tiler such that all runes return EmptyTile
type fallback struct{}

func (*fallback) GetTile(rune) image.Image {
	return EmptyTile
}

func (*fallback) DrawTile(r rune, dst draw.Image, pt image.Point, fg color.Color, bg color.Color) {
	drawTile(dst, pt, EmptyTile, fg, bg)
}

// TileSet
type TileSet map[rune]image.Image

func NewTileSet() TileSet {
	return make(TileSet)
}

func (ts TileSet) GetTile(r rune) image.Image {
	return ts[r]
}

func (ts TileSet) LoadTileFromFile(r rune, file string) {
	fh, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer fh.Close()
	ts[r], _, err = image.Decode(fh)
	if err != nil {
		panic(err)
	}
}

func (ts TileSet) LoadTileFromReader(r rune, rd io.Reader) {
	var err error
	ts[r], _, err = image.Decode(rd)
	if err != nil {
		panic(err)
	}
}

func (ts TileSet) DrawTile(r rune, dst draw.Image, pt image.Point, fg color.Color, bg color.Color) {
	if _, ok := ts[r]; !ok {
		drawTile(dst, pt, EmptyTile, fg, bg)
		return
	}
	drawTile(dst, pt, ts[r], fg, bg)
}

type FullColorTileSet TileSet

func (fc FullColorTileSet) DrawTile(r rune, dst draw.Image, pt image.Point, fg color.Color, bg color.Color) {
	src, ok := fc[r]
	if !ok {
		return
	}

	// first draw bg color then
	// image.Draw(dst, src.Bounds().Add(pt), src, src.Bounds().Min(), draw.Over)
	// image.Draw(dst, src.Bounds().Add(pt), src, src.Bounds().Min(), draw.Over)

	// Would it be better if I used draw.Draw here instead??
	for x := 0; x < src.Bounds().Dx(); x++ {
		for y := 0; y < src.Bounds().Dy(); y++ {
			r, g, b, alpha := src.At(x+src.Bounds().Min.X, y+src.Bounds().Min.Y).RGBA()
			switch alpha {
			case 0x00:
				dst.Set(pt.X+x, pt.Y+y, bg)
			case 0xFF * 0x101:
				dst.Set(pt.X+x, pt.Y+y, color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), 255})
			default:
				bgr, bgg, bgb, _ := bg.RGBA()
				dst.Set(pt.X+x, pt.Y+y,
					color.RGBA{
						alphaBlend(bgr, r, alpha),
						alphaBlend(bgg, g, alpha),
						alphaBlend(bgb, b, alpha),
						255})
			}
		}
	}
}

// Italics wraps a TileSet, adding a 10 degree rotation to each character to
// kinda sorta halfway fake an italic character set. Also makes your text-based
// drawings look drunk.
type Italics struct {
	*FontTileSet
}

func (i Italics) DrawTile(r rune, dst draw.Image, pt image.Point, fg color.Color, bg color.Color) {
	if _, ok := i.FontTileSet.Glyphs[r]; !ok {
		drawTile(dst, pt, EmptyTile, fg, bg)
		return
	}
	drawTile(dst, pt, i.GetTile(r), fg, bg)
}

func (i Italics) GetTile(r rune) image.Image {
	return rotateImage(i.FontTileSet.GetTile(r), -10)
}

type Bold struct {
	*FontTileSet
}

func (b Bold) DrawTile(r rune, dst draw.Image, pt image.Point, fg color.Color, bg color.Color) {
	if _, ok := b.FontTileSet.Glyphs[r]; !ok {
		drawTile(dst, pt, EmptyTile, fg, bg)
		return
	}
	drawTile(dst, pt, b.GetTile(r), fg, bg)
}

func (b Bold) GetTile(r rune) image.Image {
	// todo, composite the same tile with itself here shifted by one pixel to fake "bold"
	return b.FontTileSet.GetTile(r)
}

// drawTile is a broadly compatible, if not efficient, way to draw a tile.
func drawTile(dst draw.Image, pt image.Point, src image.Image, fg color.Color, bg color.Color) {
	for x := 0; x < src.Bounds().Dx(); x++ {
		for y := 0; y < src.Bounds().Dy(); y++ {
			// only use the alpha channel from ts[r]?
			// could have non-white or non-black pixels values override the foreground color.
			// performance enhancements? Considering checling if ts[r] is an image.Alpha or
			// if it supports AlphaAt?
			_, _, _, alpha := src.At(x+src.Bounds().Min.X, y+src.Bounds().Min.Y).RGBA()
			switch alpha {
			case 0x00:
				dst.Set(pt.X+x, pt.Y+y, bg)
			case m:
				dst.Set(pt.X+x, pt.Y+y, fg)
			default:
				bgr, bgg, bgb, _ := bg.RGBA()
				fgr, fgg, fgb, _ := fg.RGBA()

				dst.Set(pt.X+x, pt.Y+y,
					color.RGBA{
						alphaBlend(bgr, fgr, alpha),
						alphaBlend(bgg, fgg, alpha),
						alphaBlend(bgb, fgb, alpha),
						255})
			}
		}
	}
}

// had to copy and paste this out of fansiterm/transformations.go; probably need
// to make a dedicated transformations package.
type imageTransform struct {
	image.Image
	tx func(x, y int) (int, int)
}

func (it imageTransform) At(x, y int) color.Color {
	x, y = it.tx(x, y)
	return it.Image.At(x, y)
}

func rotateImage(img image.Image, degrees int) imageTransform {

	midX := img.Bounds().Dx()/2 + img.Bounds().Min.X
	midY := img.Bounds().Dy()/2 + img.Bounds().Min.Y
	rotInRadians := float64(degrees) / 180 * math.Pi

	return imageTransform{
		Image: img,
		tx: func(x, y int) (int, int) {
			newTheta := math.Atan2(float64(y-midY), float64(x-midX)) + rotInRadians
			r := math.Sqrt(math.Pow(float64(y-midY), 2) + math.Pow(float64(x-midX), 2))

			return int(math.Round(r*math.Cos(newTheta))) + midX, int(math.Round(r*math.Sin(newTheta))) + midY
		},
	}
}

func rectangleAt(rect image.Rectangle, pt image.Point) image.Rectangle {
	return image.Rect(pt.X, pt.Y, pt.X+rect.Dx(), pt.Y+rect.Dy())
}

// m is the maximum value for an unsigned 16bit integer
const m = 1<<16 - 1

// alphaBlend blends together two values. Fully opaque (alpha == m) means
// all fg is shown, fully transparent (alpha == 0) means only bg is shown.
// Otherwise blend the two together based on the ratio of alpha between 0 and m.
// The arguments are uint32, the values of the arguments are in the 16bit range
// and we need to return a uint8. Confused? I sure am.
//
//go:inline
func alphaBlend(bg, fg, alpha uint32) uint8 {
	return uint8(((bg*(m-alpha) + fg*alpha) / m) >> 8)
}

// getPix extracts the alpha values from an image.Image
func getPix(img image.Image) []uint8 {
	if alphaImg, ok := img.(*image.Alpha); ok {
		return alphaImg.Pix
	}
	// otherwise, just do it the dumb inefficient, but guaranteed to work way
	pix := make([]uint8, img.Bounds().Dx()*img.Bounds().Dy())
	for y := img.Bounds().Min.Y; y <= img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x <= img.Bounds().Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			pix[y*img.Bounds().Dx()+x] = uint8(a / 0x101)
		}
	}
	return pix
}
