package xform

import (
	"image"
	"image/color"
	"math"
)

// InvertColors invert colors.
func InvertColors(img image.Image) *invertColors {
	return &invertColors{img}
}

type invertColors struct {
	image.Image
}

func (ic *invertColors) At(x, y int) color.Color {
	r, g, b, a := ic.Image.At(x, y).RGBA()
	return color.RGBA{255 - uint8(r/0x101), 255 - uint8(g/0x101), 255 - uint8(b/0x101), uint8(a / 0x101)}
}

// Translate shifts pixels around by 'by'. The bounds are also shifted.
func Translate(img image.Image, by image.Point) *translate {
	return &translate{
		Image: img,
		by:    by,
	}
}

type translate struct {
	image.Image
	by image.Point
}

func (t *translate) At(x, y int) color.Color {
	return t.Image.At(x+t.by.X, y+t.by.Y)
}

func (t *translate) Bounds() image.Rectangle {
	return t.Image.Bounds().Sub(t.by)
}

// MirrorHorizontal flips an image along its X-axis.
func MirrorHorizontal(img image.Image) *mirrorHorizontal {
	return &mirrorHorizontal{img}
}

type mirrorHorizontal struct {
	image.Image
}

func (mh *mirrorHorizontal) At(x, y int) color.Color {
	return mh.Image.At(mh.Image.Bounds().Min.X+mh.Image.Bounds().Dx()-x, y)
}

// MirrorVertical flips an image along its Y-axis.
func MirrorVertical(img image.Image) *mirrorVertical {
	return &mirrorVertical{img}
}

type mirrorVertical struct {
	image.Image
}

func (mv *mirrorVertical) At(x, y int) color.Color {
	return mv.Image.At(x, mv.Image.Bounds().Min.Y+mv.Image.Bounds().Dy()-y)
}

// Rotate180 efficiently (compared to Rotate) rotates an image 180 degrees.
func Rotate180(img image.Image) *rotate180 {
	return &rotate180{img}
}

type rotate180 struct {
	image.Image
}

func (r *rotate180) At(x, y int) color.Color {
	return r.Image.At(r.Image.Bounds().Min.X+r.Image.Bounds().Dx()-x, r.Image.Bounds().Min.Y+r.Image.Bounds().Dy()-y)
}

// Rotate90 efficiently (compared to Rotate) rotates an image 90 degrees counter-clockwise.
// Not currently working and I CBA.
func Rotate90(img image.Image) *rotate90 {
	return &rotate90{img}
}

type rotate90 struct {
	image.Image
}

func (r *rotate90) At(x, y int) color.Color {
	// x -= r.Image.Bounds().Min.X
	// y -= r.Image.Bounds().Min.Y

	// x += r.Image.Bounds().Min.Y
	// y += r.Image.Bounds().Min.X
	return r.Image.At(r.Image.Bounds().Min.Y+r.Image.Bounds().Dy()-y, x)

}

func (r *rotate90) Bounds() image.Rectangle {
	bounds := r.Image.Bounds()
	bounds.Min.X, bounds.Min.Y = bounds.Min.Y, bounds.Min.X
	bounds.Max.X, bounds.Max.Y = bounds.Max.Y, bounds.Max.X
	return bounds.Sub(image.Pt(0, r.Image.Bounds().Dx()-r.Image.Bounds().Dy()-r.Image.Bounds().Min.Y))
}

// Rotate rotates img by degrees amount about center. Positive degrees result in a counter-clockwise
// rotation. keepBounds specifies if the original bounds of img should be kept (true) or if new
// bounds should be calculated based on the rotated image (false). If center is nil, the center
// of img is used.
// If you repeatedly rotate an image.Image with keepBounds false, the image bounds will continue to grow.
// For example, doing Rotate(Rotate(img, 45, nil, false), -45, nil false) will get you an image that
// appears in the same orientation as img, but will have bounds nearly 3x the original.
func Rotate(img image.Image, degrees float64, center *image.Point, keepBounds bool) *rotate {
	if center == nil {
		center = new(image.Point)
		center.X = img.Bounds().Dx()/2 + img.Bounds().Min.X
		center.Y = img.Bounds().Dy()/2 + img.Bounds().Min.Y
	}

	theta := degrees * math.Pi / 180
	// calculate positions of corners to figure out new bounds
	x1, y1 := rot(img.Bounds().Min.X-center.X, img.Bounds().Min.Y-center.Y, theta)
	x2, y2 := rot(img.Bounds().Max.X-center.X, img.Bounds().Max.Y-center.Y, theta)
	x3, y3 := rot(img.Bounds().Min.X-center.X, img.Bounds().Max.Y-center.Y, theta)
	x4, y4 := rot(img.Bounds().Max.X-center.X, img.Bounds().Min.Y-center.Y, theta)

	bounds := image.Rectangle{
		Min: image.Pt(min(x1, x2, x3, x4)+center.X, min(y1, y2, y3, y4)+center.Y),
		Max: image.Pt(max(x1, x2, x3, x4)+center.X, max(y1, y2, y3, y4)+center.Y),
	}

	if keepBounds {
		bounds = img.Bounds()
	}

	return &rotate{
		Image:  img,
		theta:  theta,
		center: *center,
		bounds: bounds,
	}
}

type rotate struct {
	image.Image
	theta  float64
	center image.Point
	bounds image.Rectangle
}

func (r *rotate) At(x, y int) color.Color {
	x, y = rot(x-r.center.X, y-r.center.Y, r.theta)
	return r.Image.At(x+r.center.X, y+r.center.Y)
}

func (r *rotate) Bounds() image.Rectangle {
	return r.bounds
}

// Blur is a simple blur. Every pixel is averaged with its 8 neighbors.
func Blur(img image.Image) *blur {
	return &blur{img}
}

type blur struct {
	image.Image
}

func (b *blur) At(x, y int) color.Color {
	var R, G, B uint32
	var i uint32
	for sx := -1; sx < 2; sx++ {
		for sy := -1; sy < 2; sy++ {
			r, g, b, a := b.Image.At(x+sx, y+sy).RGBA()
			if a == 0 {
				continue
			}
			i++
			R += r
			G += g
			B += b
		}
	}
	if i == 0 {
		return color.RGBA{}
	}
	R /= i
	G /= i
	B /= i
	return color.RGBA{uint8(R / 0x101), uint8(G / 0x101), uint8(B / 0x101), 255}
}

// WrapEdges uses modulus to make an image infinitely repeat. The boundaries
// are kept the same.
func WrapEdges(img image.Image) *wrapEdges {
	return &wrapEdges{img}
}

type wrapEdges struct {
	image.Image
}

func (we *wrapEdges) At(x, y int) color.Color {
	x = (x - we.Image.Bounds().Min.X) % we.Image.Bounds().Dx()
	y = (y - we.Image.Bounds().Min.Y) % we.Image.Bounds().Dy()
	if x < 0 {
		x += we.Image.Bounds().Dx()
	}
	if y < 0 {
		y += we.Image.Bounds().Dy()
	}
	return we.Image.At(x+we.Image.Bounds().Min.X, y+we.Image.Bounds().Min.Y)
}
