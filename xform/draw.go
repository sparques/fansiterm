package xform

import (
	"fmt"
	"image"
	"image/draw"
)

// SubImage tries to use the SubImage method of img, if it has one
// otherwise, return same image wrapped so that r becomes
// the new bounds.
func SubImage(img draw.Image, r image.Rectangle) draw.Image {
	sb, ok := img.(interface {
		SubImage(image.Rectangle) image.Image
	})
	if ok {
		if sbd, ok := sb.SubImage(r).(draw.Image); ok {
			return sbd
		}
	}
	fmt.Println("using transform")
	return &subimage{
		Image:  img,
		bounds: r,
	}
}

type subimage struct {
	draw.Image
	bounds image.Rectangle
}

func (s *subimage) Bounds() image.Rectangle {
	return s.bounds
}
