package fansiterm

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"

	"github.com/sparques/fansiterm/tiles"
)

func Test_RenderScreenshot(t *testing.T) {
	// Screen is 240x135px; so if we're using an 8x16 font that means 40 columns and nearly 8.5 rows. We'll round down to 8 rows and use the extra 7 pixels for things like the battery meter.
	term := NewAtResolution(240, 135, nil)

	term.Write([]byte(" \x1b[34m\x0e(\x0f\x1b[44;97;1mFANSITERM™\x0e\x1b[34;41;22m)\x0f \x1b[37mTX v1.0\x1b[40;31m\x0e>\x0f\x1b[m\n\n"))
	term.Write([]byte("  Freq:\t\t\x0e{\x1b[7m433\x0f MHz\x0e\x1b[27m}\x0f\n\n"))
	term.Write([]byte("  Bandwidth:\t\x0e{\x1b[7m005\x0f KHz\x0e\x1b[27m}\x0f\n\n"))

	// generate a horizontal gradient tile
	gradientTile := image.NewAlpha(image.Rect(0, 0, 8, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 8; x++ {
			gradientTile.Set(x, y, color.Alpha{uint8(256 / 8 * x)})
		}
	}
	// add gradient tile to altCharSet as the '#' char
	term.Render.altCharSet.(*tiles.FontTileSet).Glyphs['#'] = gradientTile.Pix

	// generate true-color gradient
	for i := 0; i < term.cols-1; i++ {
		term.Write([]byte(fmt.Sprintf("\x0e\x1b[48;2;65;127;%d;38;2;65;127;%dm#", (i)*256/term.cols, (i+1)*256/term.cols)))
	}

	term.Write([]byte(" "))

	fh, err := os.Create("screenshot.png")
	if err != nil {
		panic(err)
	}

	png.Encode(fh, term.Render)
	fh.Close()
}
