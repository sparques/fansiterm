package tiles

import (
	"image"
	"image/color"
	"image/draw"
)

// Italics wraps a TileSet, adding a slight shear to each character.
type Italics struct {
	Tiler
}

func (i Italics) DrawTile(r rune, dst draw.Image, pt image.Point, fg color.Color, bg color.Color) {
	g, ok := i.GetTile(r)
	if !ok {
		drawTile(dst, pt, EmptyTile, fg, bg)
		return
	}

	drawTile(dst, pt, g, fg, bg)
}

func (i Italics) GetTile(r rune) (image.Image, bool) {
	g, ok := i.Tiler.GetTile(r)
	if !ok {
		return nil, false
	}
	return italicize(g), true
}

// Bold remains as a compatibility alias for older call sites.
type Bold struct {
	*FontTileSet
}

type imageTransform struct {
	image.Image
	tx func(x, y int) (int, int)
}

func italicize(img image.Image) imageTransform {
	return imageTransform{
		Image: img,
		tx: func(x, y int) (int, int) {
			switch {
			case y < 6:
				return max(x-1, 0), y
			case y < 10:
				return x, y
			default:
				return min(x+1, 7), y
			}
		},
	}
}

func (it imageTransform) At(x, y int) color.Color {
	x, y = it.tx(x, y)
	return it.Image.At(x, y)
}
