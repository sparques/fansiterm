package fansiterm

import "github.com/sparques/fansiterm/tiles"

// back in the day, we had alternate character sets that we could
// switch between using escape sequences. Nowadays, unicode lets
// use tuck all those code points into a single font.
// altToUnicode maps the alternate charset (G1) to the unicode
// codepoint.
//
// By and large, it's better to simply use the unicode codepoints,
// but we must support the legacy way of getting graphical
// characters too.
var altToUnicode = map[rune]rune{
	'0':  9608, // block
	0x61: 9618, // 50% block
	0x68: 9617, // 25% block
	0x6a: 9496, // bottom right corner
	0x6b: 9488, // top right corner
	0x6c: 9484, // top left corner
	0x6d: 9492, // bottom left corner
	0x6e: 9532, // cross
	'q':  9472, // horizontal
	'r':  9472, // horizontal

	0x74: 9500,   // T right
	0x75: 9508,   // T left
	0x76: 9524,   // T up
	0x77: 9516,   // T down
	0x78: 0x2502, // vertical

	// nonstandard
	'(': 0x25D6,
	')': 0x25D7,

	// mappings to fansiterm specific Private Use Area U+E000..U+F8FF
	'{': 0xe000,
	'}': 0xe001,
	'<': 0xe002,
	'>': 0xe003,
}

// altCharsetViaUnicode takes a tiles.Tiler and remaps code points
// for graphical symbols.
func altCharsetViaUnicode(ts tiles.Tiler) (rm *tiles.Remap) {
	rm = tiles.NewRemap(ts)
	rm.Map = altToUnicode
	return
}
