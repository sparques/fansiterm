package fansiterm

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func Test_RenderScreenshot(t *testing.T) {
	// Screen is 240x135px; so if we're using an 8x16 font that means 40 columns and nearly 8.5 rows. We'll round down to 8 rows and use the extra 7 pixels for things like the battery meter.
	term := NewAtResolution(240, 135, nil)

	// load 7-segment numbers and block shapes into altCharSet
	files, _ := filepath.Glob("tiles/*.png")
	for _, file := range files {
		runes := []rune(filepath.Base(file))
		term.Render.altCharSet.LoadTileFromFile(runes[0], file)
	}

	//fmt.Printf(" \x1b[34m\x0e(\x0f\x1b[44;97;1mFANSITERM™\x0e\x1b[34;41;22m)\x0f \x1b[37mTX v1.0\x1b[40;31m\x0e>\x0f\x1b[m\n\n  Freq:\t\t\x0e{\x1b[7m433\x0f MHz\x0e\x1b[27m}\x0f\n\n  Bandwidth:\t\x0e{\x1b[7m005\x0f KHz\x0e\x1b[27m}\x0f\n\n")

	term.Write([]byte(" \x1b[34m\x0e(\x0f\x1b[44;97;1mFANSITERM™\x0e\x1b[34;41;22m)\x0f \x1b[37mTX v1.0\x1b[40;31m\x0e>\x0f\x1b[m\n\n"))
	term.Write([]byte("  Freq:\t\t\x0e{\x1b[7m433\x0f MHz\x0e\x1b[27m}\x0f\n\n"))
	term.Write([]byte("  Bandwidth:\t\x0e{\x1b[7m005\x0f KHz\x0e\x1b[27m}\x0f\n\n"))

	// generate true-color gradient
	for i := 0; i < term.cols; i++ {
		term.Write([]byte(fmt.Sprintf("\x1b[48;2;%d;65;127m•", i*256/term.cols)))
	}

	fh, err := os.Create("screenshot.png")
	if err != nil {
		panic(err)
	}

	png.Encode(fh, term.Render)
	fh.Close()
}
