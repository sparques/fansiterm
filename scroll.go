package fansiterm

import (
	"image"
)

func (d *Device) Scroll(rowAmount int) {
	// scrollArea Empty means scroll the whole screen--we can use more efficient algos for that
	if d.scrollArea.Empty() {
		d.Render.Scroll(rowAmount * d.Render.cell.Dy())
		// fill in scrolls section with background
		if rowAmount > 0 {
			d.Clear(0, d.rows-rowAmount, d.cols, d.rows)
		} else {
			d.Clear(0, 0, d.cols, -rowAmount)
		}
		return
	}

	// scrollArea is set; must scroll a subsection
	d.Render.RegionScroll(d.scrollArea, rowAmount*d.Render.cell.Dy())

	// fill in scrolls section with background
	if rowAmount > 0 {
		d.Clear(0, d.scrollRegion[1]-rowAmount+1, d.cols, d.scrollRegion[1]+1)
	} else {
		d.Clear(0, d.scrollRegion[0], d.cols, d.scrollRegion[0]-rowAmount)
	}
}

func (d *Device) VectorScrollCells(c1, r1, c2, r2, cn, rn int) {
}

func (d *Device) setScrollRegion(start, end int) {
	d.scrollArea.Min.X = d.Render.Bounds().Min.X
	d.scrollArea.Max.X = d.Render.Bounds().Max.X

	d.scrollRegion[0] = bound((start - 1), 0, d.rows-1)
	d.scrollRegion[1] = bound((end - 1), 0, d.rows-1)

	d.scrollArea.Min.Y = (d.scrollRegion[0] * d.Render.cell.Dy()) + d.Render.bounds.Min.Y
	// + 1 because internally we are 0-indexed, but ANSI escape codes are 1-indexed
	// + another 1 because we want the bottom of the nth cell, not the top
	d.scrollArea.Max.Y = (d.scrollRegion[1]+1)*d.Render.cell.Dy() + d.Render.bounds.Min.Y

	// if you mess up setting the scroll area, just forget the whole thing.
	if (start == 0 && end == 0) || start >= end || d.scrollArea.Eq(d.Render.Bounds()) {
		d.scrollArea = image.Rectangle{}
		d.scrollRegion = [2]int{0, d.rows - 1}
	}
}
