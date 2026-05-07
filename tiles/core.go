package tiles

import (
	"image"
	"image/color"
	"image/draw"
)

var EmptyTile = image.NewAlpha(image.Rect(0, 0, 8, 16))

// Fallback is used when a Tiler cannot find a glyph for a rune.
// By default it is an internal implementation that always returns EmptyTile.
var Fallback Tiler = &fallback{}

type Tiler interface {
	DrawTile(r rune, dst draw.Image, pt image.Point, fg color.Color, bg color.Color)
	GetTile(r rune) (image.Image, bool)
}

type tileImageDrawer interface {
	drawTileImage(img image.Image, dst draw.Image, pt image.Point, fg color.Color, bg color.Color)
}

// DrawTile draws src onto dst using fg and bg as the source colors.
func DrawTile(dst draw.Image, pt image.Point, src image.Image, fg color.Color, bg color.Color) {
	drawTile(dst, pt, src, fg, bg)
}

// drawTile is a broadly compatible, if not efficient, way to draw a tile.
func drawTile(dst draw.Image, pt image.Point, src image.Image, fg color.Color, bg color.Color) {
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	bgr, bgg, bgb, _ := bg.RGBA()
	fgr, fgg, fgb, _ := fg.RGBA()

	for y := 0; y < height; y++ {
		srcY := bounds.Min.Y + y
		for x := 0; x < width; x++ {
			srcX := bounds.Min.X + x
			_, _, _, alpha := src.At(srcX, srcY).RGBA()
			switch alpha {
			case 0x00:
				dst.Set(pt.X+x, pt.Y+y, bg)
			case m:
				dst.Set(pt.X+x, pt.Y+y, fg)
			default:
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

type fallback struct{}

func (*fallback) GetTile(rune) (image.Image, bool) {
	return EmptyTile, true
}

func (*fallback) DrawTile(r rune, dst draw.Image, pt image.Point, fg color.Color, bg color.Color) {
	drawTile(dst, pt, EmptyTile, fg, bg)
}

// m is the maximum value for an unsigned 16-bit integer.
const m = 0xFFFF

// alphaBlend blends together two 16-bit channels and returns an 8-bit result.
//
//go:inline
func alphaBlend(bg, fg, alpha uint32) uint8 {
	return uint8(((bg*(m-alpha) + fg*alpha) / m) >> 8)
}

func shouldUseFallback(current Tiler) bool {
	return current != nil && current != Fallback
}
