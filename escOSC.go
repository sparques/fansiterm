package fansiterm

import "fmt"

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
		d.Properties[PropertyWindowTitle] = string(seq[2:])

	default:
		if ShowUnhandled {
			fmt.Println("Unhandled OSC:", seqString(seq))
		}
	}
}
