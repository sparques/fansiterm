package fansiterm

import (
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"strings"
)

func (d *Device) handleFansiSequence(seq []rune) {
	seq = trimST(seq)
	if len(seq) <= 1 {
		// Doing nothing seems safe...
		return
	}
	params := splitParams(seq[1:])
	switch seq[0] {
	case 'B': // B for Blit
		// ESC/B<pixdata>ESC\
		// Display image defined by pixdata at cursor location; no scalling is done

		// ESC/Bx,y;<pixdata>ESC\
		// display pixdata at x,y

		// ESC/Bx1,y1;x2,y2;<pixdata>ESC\
		// Display at x1,y1 and scale it to fit (if necessary) the rectangle (x1,y1)-(x2,y2)

		var (
			img        image.Image
			err        error
			loc        image.Point
			targetRect image.Rectangle
		)
		switch len(params) {
		case 0: // nothing
		case 1: // just pix data, display at cursor
			img, err = DecodeImageData(params[0])
			if err != nil {
				return
			}
			targetRect = img.Bounds().Add(d.cursorPt())
		case 2: // show at specific pixel offset
			fmt.Sscanf(string(params[0]), "%d,%d;", &loc.X, &loc.Y)
			img, err = DecodeImageData(params[1])
			if err != nil {
				return
			}
			targetRect = img.Bounds().Add(d.Render.bounds.Min).Add(loc)
		case 3: // show within a limited area
			n, _ := fmt.Sscanf(string(seq[1:len(seq)-len(params[2])]), "%d,%d;%d,%d;", &targetRect.Min.X, &targetRect.Min.Y, &targetRect.Max.X, &targetRect.Max.Y)
			if n != 4 {
				return
			}
			img, err = DecodeImageData(params[2])
			if err != nil {
				return
			}
			targetRect = targetRect.Canon().Add(d.Render.bounds.Min)
		}

		draw.Draw(d.Render, targetRect, img, image.Point{}, draw.Over)
		x := targetRect.Dx() / d.Render.cell.Dx()
		if targetRect.Dx()%d.Render.cell.Dx() != 0 {
			x++
		}
		d.MoveCursorRel(x, 0)

	case 'C': // C for Cell
		// ESC/C<pixdata>ESC\
		// ESC/C receives pixel data and puts it in the cursor's current position.
		// The data is serialized binary pixel values, rgb, one byte per channel, base64 encoded.
		// ESC/Cx,y;<pixdata>ESC\
		// shift the point referenced in the image so you display a different portion of the image
		var pt image.Point
		if len(params) == 2 {
			n, _ := fmt.Sscanf(string(params[0]), "%d,%d;", &pt.X, &pt.Y)
			if n != 2 {
				return
			}
		}

		img, err := DecodeImageData(params[len(params)-1])
		if err == nil {
			draw.Draw(d.Render, d.Render.cell.Add(d.cursorPt()), img, pt, draw.Over)
			return
		}

		seq = seq[1:]
		pixData, err := base64.StdEncoding.DecodeString(string(seq))
		if err != nil {
			return
		}

		img = &RGBImage{
			Pix:       pixData,
			Rectangle: d.Render.cell,
		}
		draw.Draw(d.Render, d.Render.cell.Add(d.cursorPt()), img, pt, draw.Over)
		d.MoveCursorRel(1, 0)
	case 'F': // F for Fill
		var (
			rect image.Rectangle
			c    color.RGBA
		)
		c.A = 255

		n, _ := fmt.Sscanf(string(seq), "F%d,%d;%d,%d;#%2x%2x%2x", &rect.Min.X, &rect.Min.Y, &rect.Max.X, &rect.Max.Y, &c.R, &c.G, &c.B)
		if n != 7 {
			n, _ = fmt.Sscanf(string(seq), "F%d,%d;%d,%d;%d,%d,%d", &rect.Min.X, &rect.Min.Y, &rect.Max.X, &rect.Max.Y, &c.R, &c.G, &c.B)
		}
		if n != 7 {
			return
		}

		rect = rect.Canon()
		d.Render.Fill(rect, c)
	case 'L': // L for line
		var (
			pt1, pt2 image.Point
			c        color.Color
			r, g, b  int
			swap     bool
		)
		n, _ := fmt.Sscanf(string(seq), "L%d,%d;%d,%d;#%2x%2x%2x", &pt1.X, &pt1.Y, &pt2.X, &pt2.Y, &r, &g, &b)
		if n == 4 {
			n, _ = fmt.Sscanf(string(seq), "L%d,%d;%d,%d;%d,%d,%d", &pt1.X, &pt1.Y, &pt2.X, &pt2.Y, &r, &g, &b)
		}
		if n == 4 {
			c = d.Render.active.fg
		} else {
			c = d.Render.colorSystem.Convert(color.RGBA{uint8(r), uint8(g), uint8(b), 255})
		}

		pt1, pt2 = pt1.Add(d.Render.bounds.Min), pt2.Add(d.Render.bounds.Min)

		dx := pt1.X - pt2.X
		dy := pt1.Y - pt2.Y

		var x_step, y_step int

		if dx < 0 {
			dx *= -1
		}
		if dy < 0 {
			dy *= -1
		}

		if dy > dx {
			dx, dy = dy, dx
			pt1.X, pt1.Y = pt1.Y, pt1.X
			pt2.X, pt2.Y = pt2.Y, pt2.X
			swap = true
		}

		if pt1.X < pt2.X {
			x_step = 1
		} else {
			x_step = -1
		}
		if pt1.Y < pt2.Y {
			y_step = 1
		} else {
			y_step = -1
		}
		p := 2*dy - dx

		x, y := pt1.X, pt1.Y
		for range dx {
			if swap {
				d.Render.Set(y, x, c)
			} else {
				d.Render.Set(x, y, c)
			}
			if p >= 0 {
				y += y_step
				p -= 2 * dx
			}
			x += x_step
			p += 2 * dy
		}
	case 'P': // P for Palette
		var (
			t  byte
			id int
			c  color.RGBA
		)
		n, _ := fmt.Sscanf(string(seq), "P%c%d;#%2x%2x%2x", &t, &id, &c.R, &c.G, &c.B)
		if n != 5 {
			n, _ = fmt.Sscanf(string(seq), "P%c%d;#%2x%2x%2x", &t, &id, &c.R, &c.G, &c.B)
		}
		if n != 5 {
			return
		}
		switch t {
		case 'a': // a for ANSI
			if id < 0 || id > 15 {
				return
			}
			d.Render.colorSystem.PaletteANSI[id] = d.Render.colorSystem.NewRGB(c.R, c.G, c.B)
		case 'p': // p for 256-palette
			d.Render.colorSystem.Palette256[id] = d.Render.colorSystem.NewRGB(c.R, c.G, c.B)
		default:
			return
		}
	case 'R': // R for radius (to make circles)
		var (
			x, y, r int
			rect    image.Rectangle
			c       color.RGBA
			nc      color.Color
			n       int
		)
		c.A = 255

		if strings.Contains(string(seq), "#") {
			n, _ = fmt.Sscanf(string(seq), "R%d,%d,%d;#%2x%2x%2x", &x, &y, &r, &c.R, &c.G, &c.B)
		} else {
			n, _ = fmt.Sscanf(string(seq), "R%d,%d,%d;%d,%d,%d", &x, &y, &r, &c.R, &c.G, &c.B)
		}
		switch n {
		case 3:
			nc = d.Render.active.fg
		case 6:
			nc = d.Render.colorSystem.Convert(c)
		default:
			return
		}

		rect.Min.X = x - r
		rect.Max.X = x + r
		rect.Min.Y = y - r
		rect.Max.Y = y + r

		rect = rect.Canon()
		for yp := rect.Min.Y; yp <= rect.Max.Y; yp++ {
			for xp := rect.Min.X; xp <= rect.Max.X; xp++ {
				pt := image.Pt(xp, yp).Add(d.Render.bounds.Min)
				if r*r >= (xp-x)*(xp-x)+(yp-y)*(yp-y) {
					d.Render.Set(pt.X, pt.Y, nc)
				}
			}
		}
	case 'b': // b for box to draw non-filled rectangles
		var (
			rect image.Rectangle
			c    color.RGBA
			nc   color.Color
		)
		c.A = 255

		n, _ := fmt.Sscanf(string(seq), "b%d,%d;%d,%d;#%2x%2x%2x", &rect.Min.X, &rect.Min.Y, &rect.Max.X, &rect.Max.Y, &c.R, &c.G, &c.B)
		if n != 7 {
			n, _ = fmt.Sscanf(string(seq), "b%d,%d;%d,%d;%d,%d,%d", &rect.Min.X, &rect.Min.Y, &rect.Max.X, &rect.Max.Y, &c.R, &c.G, &c.B)
		}

		switch n {
		case 4:
			nc = d.Render.active.fg
		case 7:
			nc = d.Render.colorSystem.Convert(c)
		default:
			return
		}

		rect = rect.Canon().Add(d.Render.bounds.Min)

		for x := rect.Min.X; x <= rect.Max.X; x++ {
			d.Render.Set(x, rect.Min.Y, nc)
			d.Render.Set(x, rect.Max.Y, nc)
		}
		for y := rect.Min.Y; y <= rect.Max.Y; y++ {
			d.Render.Set(rect.Min.X, y, nc)
			d.Render.Set(rect.Max.X, y, nc)
		}
	case 'r': // r for radius to make non-filled circles
		var (
			x, y, r int
			c       color.RGBA
			nc      color.Color
			n       int
		)
		c.A = 255

		if strings.Contains(string(seq), "#") {
			n, _ = fmt.Sscanf(string(seq), "r%d,%d,%d;#%2x%2x%2x", &x, &y, &r, &c.R, &c.G, &c.B)
		} else {
			n, _ = fmt.Sscanf(string(seq), "r%d,%d,%d;%d,%d,%d", &x, &y, &r, &c.R, &c.G, &c.B)
		}
		switch n {
		case 3:
			nc = d.Render.active.fg
		case 6:
			nc = d.Render.colorSystem.Convert(c)
		default:
			return
		}

		x += d.Render.bounds.Min.X
		y += d.Render.bounds.Min.Y

		xp := 0
		yp := r
		de := 3 - 2*r
		for xp <= yp {
			d.Render.Set(xp+x, yp+y, nc)
			d.Render.Set(xp+x, -yp+y, nc)
			d.Render.Set(-xp+x, yp+y, nc)
			d.Render.Set(-xp+x, -yp+y, nc)
			d.Render.Set(yp+x, xp+y, nc)
			d.Render.Set(yp+x, -xp+y, nc)
			d.Render.Set(-yp+x, xp+y, nc)
			d.Render.Set(-yp+x, -xp+y, nc)
			if de < 0 {
				de = de + 4*xp + 6
			} else {
				de = de + 4*(xp-yp) + 10
				yp--
			}
			xp++
		}

		// draw a circle using polar coordinates. Only calculate 1/8 the circle and use
		// symmetry to plot the rest.
		// I came up with this algo on my own. Bresenham's method is better :(

		// determine steps size by finding the inverse of the circumference.
		/*
			step := 1 / (r * 2 * math.Pi)
			for theta := 0.0; theta <= math.Pi/4; theta += step {
				xp := int(math.Round(r * math.Cos(theta)))
				yp := int(math.Round(r * math.Sin(theta)))
				d.Render.Set(xp+x, yp+y, c)
				d.Render.Set(xp+x, -yp+y, c)
				d.Render.Set(-xp+x, yp+y, c)
				d.Render.Set(-xp+x, -yp+y, c)
				d.Render.Set(yp+x, xp+y, c)
				d.Render.Set(yp+x, -xp+y, c)
				d.Render.Set(-yp+x, xp+y, c)
				d.Render.Set(-yp+x, -xp+y, c)
			}
		*/

	case 'S': // S for Set
		var (
			pt image.Point
			c  color.RGBA
			n  int
		)
		if strings.Contains(string(seq), "#") {
			n, _ = fmt.Sscanf(string(seq), "S%d,%d;#%2x%2x%2x", &pt.X, &pt.Y, &c.R, &c.G, &c.B)
		} else {
			n, _ = fmt.Sscanf(string(seq), "S%d,%d;%d,%d,%d", &pt.X, &pt.Y, &c.R, &c.G, &c.B)
		}
		pt = pt.Add(d.Render.bounds.Min).Mod(d.Render.bounds)
		switch n {
		case 2:
			d.Render.Set(pt.X, pt.Y, d.Render.active.fg)
		case 5:
			d.Render.Set(pt.X, pt.Y, c)
		default:
		}
	case 'V': // V for vectorScroll
		var (
			region image.Rectangle
			vector image.Point
		)
		n, _ := fmt.Sscanf(string(seq), "V%d,%d;%d,%d;%d,%d", &region.Min.X, &region.Min.Y, &region.Max.X, &region.Max.Y, &vector.X, &vector.Y)
		if n != 6 {
			return
		}
		region = region.Canon().Add(d.Render.bounds.Min).Intersect(d.Render.bounds)
		d.Render.VectorScroll(region, vector)
	}

}
