package fansiterm

// Cursor is used to track the cursor.
type Cursor struct {
	// cols is a pointer to (*Device).cols
	cols *int
	// rows is a pointer to (*Device).rows
	rows *int
	// col is the current column. This is zero indexed.
	col int
	// row is the current row. This is zero indexed.
	row int
	// show is whether we should be showing the the cursor.
	show bool
	// visible is whether or not the cursor is currently visible. When rendering text,
	// we hide the cursor, then re-enable it when done.
	visible bool
	// prevPos is for saving cursor position; The indicies are col, row.
	prevPos [2]int

	altPos [2]int
}

// ToggleAltPos toggles between the main screen's position and
// the alt screen's position.
func (c *Cursor) ToggleAltPos() {
	c.altPos[0], c.altPos[1], c.col, c.row = c.col, c.row, c.altPos[0], c.altPos[1]
}

// ColsRemaining returns how many columns are remaining until EO
func (c *Cursor) ColsRemaining() int {
	return *c.cols - c.col
}

func (c *Cursor) MoveRel(x, y int) {
	c.col = bound(x+c.col, 0, *c.cols-1)
	c.row = bound(y+c.row, 0, *c.rows-1)
}

func (c *Cursor) MoveAbs(x, y int) {
	c.col = bound(x, 0, *c.cols-1)
	c.row = bound(y, 0, *c.rows-1)
}

func (c *Cursor) SavePos() {
	c.prevPos[0] = c.col
	c.prevPos[1] = c.row
}

func (c *Cursor) RestorePos() {
	c.col = c.prevPos[0]
	c.row = c.prevPos[1]
}
