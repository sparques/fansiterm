package fansiterm

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"golang.org/x/exp/constraints"
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

// imageTransform wraps an image.Image and transforms it. The tx method is used
// to change what pixel is returned for a given x and y coordinate. This can implement
// zooms, mirroring, and rotations or any combination thereof.
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
		tx:    func(x, y int) (int, int) { return img.Bounds().Max.X - x, y },
	}
}

func verticalMirror(img image.Image) imageTransform {
	return imageTransform{
		Image: img,
		tx:    func(x, y int) (int, int) { return x, img.Bounds().Max.Y - y },
	}
}

// wrapEdges uses modulus to make an image infinitely repeat
func wrapEdges(img image.Image) imageTransform {
	return imageTransform{
		Image: img,
		tx: func(x, y int) (int, int) {
			x = (x - img.Bounds().Min.X) % img.Bounds().Dx()
			y = (y - img.Bounds().Min.Y) % img.Bounds().Dy()
			if x < 0 {
				x += img.Bounds().Dx()
			}
			if y < 0 {
				y += img.Bounds().Dy()
			}

			return x + img.Bounds().Min.X, y + img.Bounds().Min.Y
		},
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

/*
func avgColor(a, b color.Color) color.RGBA {
	r1, g1, b1, a1 := a.RGBA()
	r2, g2, b2, a2 := b.RGBA()
	return color.RGBA{(r1 + r2) / 2, (g1 + g2) / 2, (b1 + b2) / 2, (a1 + a2) / 2}
}
*/

func weightedAvgColor(a, b color.Color, aWeight float64) color.RGBA {
	r1, g1, b1, a1 := a.RGBA()
	r2, g2, b2, a2 := b.RGBA()

	return color.RGBA{
		uint8(math.Round(float64(r1)*aWeight + float64(r2)*(1-aWeight))),
		uint8(math.Round(float64(g1)*aWeight + float64(g2)*(1-aWeight))),
		uint8(math.Round(float64(b1)*aWeight + float64(b2)*(1-aWeight))),
		uint8(math.Round(float64(a1)*aWeight + float64(a2)*(1-aWeight))),
	}
}

/*

	| 1+AB A+ABC+C |          | x |
	|  B     1+BC  |   times  | y |
*/

func rotateImageBySkew(img image.Image, degrees float64) imageTransform {
	midX := img.Bounds().Dx()/2 + img.Bounds().Min.X
	midY := img.Bounds().Dy()/2 + img.Bounds().Min.Y

	theta := degrees / 180 * math.Pi

	horizontalSkew := -math.Atan(theta / 2)
	verticalSkew := math.Sin(theta)

	return imageTransform{
		Image: img,
		tx: func(x, y int) (int, int) {
			x = x - midX
			y = y - midY
			return int(math.Round((1+horizontalSkew*verticalSkew)*float64(x)+(2*horizontalSkew+horizontalSkew*horizontalSkew*verticalSkew)*float64(y))) + midX,
				int(math.Round(verticalSkew*float64(x)+(1+horizontalSkew*verticalSkew)*float64(y))) + midY

		},
	}
}

type Number interface {
	constraints.Integer | constraints.Float
}

func abs[N Number](n N) N {
	if n < 0 {
		return -n
	}
	return n
}

// SubImage tries to use the SubImage method of img, if it has one
// otherwise, return same image wrapped so that r becomes
// the new bounds.
func SubImage(img image.Image, r image.Rectangle) image.Image {
	sb, ok := img.(interface {
		SubImage(image.Rectangle) image.Image
	})
	if ok {
		return sb.SubImage(r)
	}

	return subimage{
		Image:  img,
		bounds: r,
	}
}

type subimage struct {
	image.Image
	bounds image.Rectangle
}

func (si subimage) Bounds() image.Rectangle {
	return si.bounds
}

// drawTransform transforms *draw
type drawTransform struct {
	draw.Image
	tx func(x, y int) (int, int)
}

func (it drawTransform) Set(x, y int, c color.Color) {
	x, y = it.tx(x, y)
	it.Image.Set(x, y, c)
}

func rotateDraw(img draw.Image, degrees int) drawTransform {
	midX := img.Bounds().Dx()/2 + img.Bounds().Min.X
	midY := img.Bounds().Dy()/2 + img.Bounds().Min.Y
	rotInRadians := float64(degrees) / 180 * math.Pi

	return drawTransform{
		Image: img,
		tx: func(x, y int) (int, int) {
			newTheta := math.Atan2(float64(y-midY), float64(x-midX)) - rotInRadians
			r := math.Sqrt(math.Pow(float64(y-midY), 2) + math.Pow(float64(x-midX), 2))

			return int(math.Round(r*math.Cos(newTheta))) + midX, int(math.Round(r*math.Sin(newTheta))) + midY
		},
	}
}

type colorTransform struct {
	image.Image
	at func(x, y int) color.Color
}

func (ct *colorTransform) At(x, y int) color.Color {
	return ct.at(x, y)
}

func blurImage(img image.Image) *colorTransform {
	return &colorTransform{
		Image: img,
		at: func(x, y int) color.Color {
			var R, G, B uint32
			var i uint32
			for sx := -1; sx < 2; sx++ {
				// if x+sx < img.Bounds().Min.X || x+sx > img.Bounds().Max.X {
				// 	continue
				// }
				for sy := -1; sy < 2; sy++ {
					// if y+sy < img.Bounds().Min.Y || y+sy > img.Bounds().Max.Y {
					// 	continue
					// }
					i++
					r, g, b, _ := img.At(x+sx, y+sy).RGBA()
					R += r
					G += g
					B += b
				}
			}
			R /= i
			G /= i
			B /= i
			return color.RGBA{uint8(R / 0x101), uint8(G / 0x101), uint8(B / 0x101), 255}
		},
	}
}
