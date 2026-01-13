//go:build notick

package fansiterm

func (d *Device) queueHandler() {
	for {
		select {
		case <-d.done:
			// close writeQueue here??
			return
		case data := <-d.writeQueue:
			d.write(data)
		}
	}
}
