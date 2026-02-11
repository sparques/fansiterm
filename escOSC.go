package fansiterm

import (
	"fmt"
	"image/color"
)

func (d *Device) handleOSCSequence(seq []rune) {
	seq = trimST(seq)
	if len(seq) == 0 {
		// what does an empty OSC sequence mean?
		// Doing nothing seems safe...
		return
	}
	args := getNumericArgs(seq, 0)
	switch args[0] {
	case 0:
		// xterm set window title
		d.Config.Properties[PropertyWindowTitle] = string(seq[2:])
		d.configChange()
	case 10: // query default foreground color
		fg := color.RGBAModel.Convert(d.attrDefault.Fg).(color.RGBA)
		fmt.Fprintf(d.Output, "\x1b]10;rgb:%d/%d/%d\x1b/", fg.R, fg.G, fg.B)
	case 11: // query default background color
		bg := color.RGBAModel.Convert(d.attrDefault.Bg).(color.RGBA)
		fmt.Fprintf(d.Output, "\x1b]11;rgb:%d/%d/%d\x1b/", bg.R, bg.G, bg.B)
	default:
		if ShowUnhandled {
			log.Warn("unhandled OSC", "sequence", seqString(seq))
		}
	}
}
