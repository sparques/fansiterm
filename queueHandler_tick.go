//go:build !notick

package fansiterm

import (
	"time"
)

func (d *Device) queueHandler() {
	// since I have to have this background goroutine, I could add a tick here
	// to run periodic tasks... like blinking a curosr
	tick := time.NewTicker(time.Second / 2)
	for {
		select {
		case <-d.done:
			// close writeQueue here??
			return
			/*		case buf := <-d.bufChan:
					// disable cursor
					d.preUpdate()
					d.useBuf(buf)
					d.postUpdate()*/
		case <-tick.C:
			if d.cursor.show {
				d.BlinkCursor()
			}
		case data := <-d.writeQueue:
			d.write(data)
		}
	}
}
