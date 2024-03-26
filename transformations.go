package fansiterm

import (
	"image"
	"image/color"
	"image/draw"
	"math"
)

// invertColors composites an image.Image, overriding the At() method
// so that colors returned are inverted. This is primarily for drawing
// the cursor.
type invertColors struct {
	image.Image
}

func (ic invertColors) At(x, y int) color.Color {
	r, g, b, a := ic.Image.At(x, y).RGBA()
	return color.RGBA{255 - uint8(r), 255 - uint8(g), 255 - uint8(b), uint8(a)}
}

// imageTranslate works a bit like the Subimage() method on various image package
// objects. However, it wraps a draw.Image allowing both calls to Set() and At().
// This doesn't restrict any pixel operations, so the margins still remain
// accesible.
type imageTranslate struct {
	draw.Image
	offset image.Point
}

// NewImageTranslate returns an imageTranslate object which offsets all operations
// to img by offset.
func NewImageTranslate(offset image.Point, img draw.Image) *imageTranslate {
	return &imageTranslate{
		offset: offset,
		Image:  img,
	}
}

func (it *imageTranslate) Set(x, y int, c color.Color) {
	it.Image.Set(x+it.offset.X, y+it.offset.Y, c)
}

func (it *imageTranslate) At(x, y int) color.Color {
	//return it.Image.At(x+it.offset.X, y+it.offset.Y)
	return it.Image.At(x+it.offset.X, y+it.offset.Y)
}

func (it imageTranslate) Bounds() image.Rectangle {
	return it.Image.Bounds().Sub(it.offset)
}

// neat, but I'd have to override the Glyph method of Render.fontDraw to use it for
// individual characters/tiles. It does work as a means to manipulate the whole
// screen, though.
type imageTransform struct {
	image.Image
	tx func(x, y int) (int, int)
}

func (it imageTransform) At(x, y int) color.Color {
	x, y = it.tx(x, y)
	return it.Image.At(x, y)
}

func horizontalMirror(img image.Image) imageTransform {
	return imageTransform{
		Image: img,
		tx:    func(x, y int) (int, int) { return img.Bounds().Dx() - x, y },
	}
}

func verticalMirror(img image.Image) imageTransform {
	return imageTransform{
		Image: img,
		tx:    func(x, y int) (int, int) { return x, img.Bounds().Dy() - y },
	}
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
