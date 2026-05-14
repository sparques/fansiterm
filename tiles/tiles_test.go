package tiles

import (
	"image"
	"image/color"
	"testing"
)

func TestFontTileSetSetTileCopiesCompactAlphaFromSubImage(t *testing.T) {
	base := image.NewAlpha(image.Rect(0, 0, 4, 2))
	base.Pix = []uint8{
		1, 2, 3, 4,
		5, 6, 7, 8,
	}

	sub := base.SubImage(image.Rect(1, 0, 3, 2))

	fts := &FontTileSet{
		Rectangle: image.Rect(0, 0, 2, 2),
	}
	fts.SetTile('x', sub)

	glyph := fts.Glyph('x')
	if glyph == nil {
		t.Fatalf("expected glyph lookup to succeed")
	}
	got := glyph.Pix
	want := []uint8{2, 3, 6, 7}
	if len(got) != len(want) {
		t.Fatalf("glyph length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("glyph[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestFontTileSetSetTileUpdatesExistingPackedGlyph(t *testing.T) {
	fts := &FontTileSet{
		Rectangle: image.Rect(0, 0, 2, 1),
		First:     'a',
		Count:     2,
		Index:     []uint16{1, 2},
		Pix:       []uint8{1, 2, 3, 4},
	}
	img := image.NewAlpha(image.Rect(0, 0, 2, 1))
	copy(img.Pix, []uint8{8, 9})

	fts.SetTile('b', img)

	if got := fts.Pix; len(got) != 4 || got[0] != 1 || got[1] != 2 || got[2] != 8 || got[3] != 9 {
		t.Fatalf("packed pix = %#v, want existing second glyph replaced", got)
	}
	if fts.Count != 2 {
		t.Fatalf("count = %d, want 2", fts.Count)
	}
}

func TestFontTileSetSetTileConvertsImageToAlpha(t *testing.T) {
	fts := &FontTileSet{
		Rectangle: image.Rect(0, 0, 2, 1),
	}
	img := image.NewRGBA(image.Rect(0, 0, 2, 1))
	img.SetRGBA(0, 0, color.RGBA{A: 0x7f})
	img.SetRGBA(1, 0, color.RGBA{A: 0xff})

	fts.SetTile('x', img)

	glyph := fts.Glyph('x')
	if glyph == nil {
		t.Fatalf("expected glyph lookup to succeed")
	}
	if got := glyph.Pix; len(got) != 2 || got[0] != 0x7f || got[1] != 0xff {
		t.Fatalf("glyph pix = %#v, want []uint8{0x7f, 0xff}", got)
	}
}

func TestFontTileSetSetTileConvertsDenseToSparseForOutOfOrderRune(t *testing.T) {
	fts := &FontTileSet{
		Rectangle: image.Rect(0, 0, 1, 1),
		First:     'b',
		Count:     1,
		Index:     []uint16{1},
		Pix:       []uint8{2},
	}
	img := image.NewAlpha(image.Rect(0, 0, 1, 1))
	img.Pix[0] = 1

	fts.SetTile('a', img)

	if len(fts.Index) != 0 {
		t.Fatalf("index = %#v, want sparse storage", fts.Index)
	}
	if got := fts.Runes(); len(got) != 2 || got[0] != 'a' || got[1] != 'b' {
		t.Fatalf("runes = %#v, want []rune{'a', 'b'}", got)
	}
	glyph := fts.Glyph('a')
	if glyph == nil || len(glyph.Pix) != 1 || glyph.Pix[0] != 1 {
		t.Fatalf("glyph a = %#v, want pixel 1", glyph)
	}
}

func TestAlphaCellTileSetSetTilePopulatesNewTileSet(t *testing.T) {
	ats := NewAlphaCellTileSet()
	img := image.NewAlpha(image.Rect(0, 0, 8, 16))
	img.SetAlpha(0, 0, color.Alpha{A: 0xFF})
	img.SetAlpha(7, 15, color.Alpha{A: 0xFF})

	ats.SetTile('x', img)

	glyph := ats.Glyph('x')
	if glyph == nil {
		t.Fatalf("expected glyph lookup to succeed")
	}
	if glyph.Pix[0] != 0x80 || glyph.Pix[15] != 0x01 {
		t.Fatalf("glyph pix = %#v, want high bit in row 0 and low bit in row 15", glyph.Pix)
	}
	if ats.First != 'x' || ats.Count != 1 || len(ats.Index) != 1 || ats.Index[0] != 1 || len(ats.Cells) != 1 {
		t.Fatalf("packed fields = first %q count %d index %#v cells %d, want one packed glyph", ats.First, ats.Count, ats.Index, len(ats.Cells))
	}
}

func TestAlphaCellTileSetSetTileUpdatesExistingPackedGlyph(t *testing.T) {
	ats := &AlphaCellTileSet{
		Rectangle: image.Rect(0, 0, 8, 16),
		First:     'a',
		Count:     2,
		Index:     []uint16{1, 2},
		Cells: [][16]uint8{
			{0x80},
			{0x40},
		},
	}
	img := image.NewAlpha(image.Rect(0, 0, 8, 16))
	img.SetAlpha(7, 0, color.Alpha{A: 0xFF})

	ats.SetTile('b', img)

	if ats.Cells[0][0] != 0x80 {
		t.Fatalf("first packed glyph changed to %#x", ats.Cells[0][0])
	}
	if ats.Cells[1][0] != 0x01 {
		t.Fatalf("second packed glyph row = %#x, want %#x", ats.Cells[1][0], 0x01)
	}
	if ats.Count != 2 {
		t.Fatalf("count = %d, want 2", ats.Count)
	}
}

func TestAlpha1TileSetSetTileConvertsDenseToSparseForOutOfOrderRune(t *testing.T) {
	ats := &Alpha1TileSet{
		Rectangle: image.Rect(0, 0, 9, 2),
		First:     'b',
		Count:     1,
		Index:     []uint16{1},
		Pix: []uint8{
			0x80, 0x00,
			0x00, 0x80,
		},
	}
	img := image.NewAlpha(image.Rect(0, 0, 9, 2))
	img.SetAlpha(8, 0, color.Alpha{A: 0xFF})
	img.SetAlpha(1, 1, color.Alpha{A: 0xFF})

	ats.SetTile('a', img)

	if len(ats.Index) != 0 {
		t.Fatalf("index = %#v, want sparse storage", ats.Index)
	}
	if got := ats.Runes(); len(got) != 2 || got[0] != 'a' || got[1] != 'b' {
		t.Fatalf("runes = %#v, want []rune{'a', 'b'}", got)
	}
	glyph := ats.Glyph('a')
	if glyph == nil {
		t.Fatalf("expected glyph lookup to succeed")
	}
	if got := glyph.Pix; len(got) != 4 || got[0] != 0x00 || got[1] != 0x80 || got[2] != 0x40 || got[3] != 0x00 {
		t.Fatalf("glyph pix = %#v, want bit-packed alpha1 glyph", got)
	}
}

func TestAlpha1HonorsStrideAndOrigin(t *testing.T) {
	a := NewAlpha1(image.Rect(4, 5, 13, 7))
	if a.Stride != 2 {
		t.Fatalf("stride = %d, want 2", a.Stride)
	}
	if len(a.Pix) != 4 {
		t.Fatalf("pix len = %d, want 4", len(a.Pix))
	}

	a.Set(4, 5, BitAlpha(true))
	a.Set(12, 6, BitAlpha(true))

	if got := a.At(4, 5).(BitAlpha); !bool(got) {
		t.Fatalf("pixel (4,5) was not set")
	}
	if got := a.At(12, 6).(BitAlpha); !bool(got) {
		t.Fatalf("pixel (12,6) was not set")
	}
	if got := a.At(5, 5).(BitAlpha); bool(got) {
		t.Fatalf("neighbor pixel unexpectedly set")
	}
}

func TestAlpha1FillAndScroll(t *testing.T) {
	a := NewAlpha1(image.Rect(0, 0, 8, 4))
	a.Fill(image.Rect(0, 1, 8, 3), color.Alpha{A: 0xFF})

	for y := 0; y < 4; y++ {
		for x := 0; x < 8; x++ {
			want := y == 1 || y == 2
			got := bool(a.At(x, y).(BitAlpha))
			if got != want {
				t.Fatalf("before scroll pixel (%d,%d) = %v, want %v", x, y, got, want)
			}
		}
	}

	a.Scroll(1)

	for y := 0; y < 4; y++ {
		for x := 0; x < 8; x++ {
			want := y == 0 || y == 1
			got := bool(a.At(x, y).(BitAlpha))
			if got != want {
				t.Fatalf("after scroll pixel (%d,%d) = %v, want %v", x, y, got, want)
			}
		}
	}
}

func TestFontTileSetPackedDenseLookup(t *testing.T) {
	fts := &FontTileSet{
		Rectangle: image.Rect(0, 0, 2, 1),
		First:     'a',
		Count:     2,
		Index:     []uint16{1, 2},
		Pix:       []uint8{1, 2, 3, 4},
	}

	glyph := fts.Glyph('b')
	if glyph == nil {
		t.Fatalf("expected packed glyph lookup to succeed")
	}
	if got := glyph.Pix; len(got) != 2 || got[0] != 3 || got[1] != 4 {
		t.Fatalf("packed glyph = %#v, want []uint8{3, 4}", got)
	}
}

func TestAlphaCellTileSetPackedSparseLookup(t *testing.T) {
	ats := &AlphaCellTileSet{
		Rectangle: image.Rect(0, 0, 8, 16),
		Count:     2,
		Sparse: []RuneIndex{
			{Rune: 'x', Index: 1},
			{Rune: 'z', Index: 2},
		},
		Cells: [][16]uint8{
			{0xAA},
			{0x55},
		},
	}

	glyph := ats.Glyph('z')
	if glyph == nil {
		t.Fatalf("expected packed sparse glyph lookup to succeed")
	}
	if glyph.Pix[0] != 0x55 {
		t.Fatalf("packed sparse glyph[0] = %#x, want %#x", glyph.Pix[0], 0x55)
	}
}

func TestAlpha1TileSetPackedDenseLookup(t *testing.T) {
	ats := &Alpha1TileSet{
		Rectangle: image.Rect(0, 0, 9, 2),
		First:     'a',
		Count:     2,
		Index:     []uint16{1, 2},
		Pix: []uint8{
			0x80, 0x80,
			0x40, 0x00,
			0x20, 0x00,
			0x10, 0x80,
		},
	}

	glyph := ats.Glyph('b')
	if glyph == nil {
		t.Fatalf("expected packed glyph lookup to succeed")
	}
	if glyph.Stride != 2 {
		t.Fatalf("glyph stride = %d, want 2", glyph.Stride)
	}
	if !glyph.Bounds().Eq(image.Rect(0, 0, 9, 2)) {
		t.Fatalf("glyph bounds = %v, want %v", glyph.Bounds(), image.Rect(0, 0, 9, 2))
	}
	if got := glyph.Pix; len(got) != 4 || got[0] != 0x20 || got[3] != 0x80 {
		t.Fatalf("packed glyph = %#v, want second 4-byte glyph", got)
	}
}

func TestAlpha1TileSetDrawTile(t *testing.T) {
	ats := &Alpha1TileSet{
		Rectangle: image.Rect(0, 0, 9, 2),
		First:     'x',
		Count:     1,
		Index:     []uint16{1},
		Pix: []uint8{
			0x80, 0x80,
			0x40, 0x00,
		},
	}
	dst := image.NewRGBA(image.Rect(0, 0, 9, 2))
	fg := color.RGBA{R: 0xFF, A: 0xFF}
	bg := color.RGBA{B: 0xFF, A: 0xFF}

	ats.DrawTile('x', dst, image.Point{}, fg, bg)

	for y := 0; y < 2; y++ {
		for x := 0; x < 9; x++ {
			wantFG := (y == 0 && (x == 0 || x == 8)) || (y == 1 && x == 1)
			got := dst.RGBAAt(x, y)
			if wantFG && got != fg {
				t.Fatalf("pixel (%d,%d) = %#v, want fg", x, y, got)
			}
			if !wantFG && got != bg {
				t.Fatalf("pixel (%d,%d) = %#v, want bg", x, y, got)
			}
		}
	}
}
