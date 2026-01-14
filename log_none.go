//go:build lognone && !logslog && !logprintln
package fansiterm

func init() {
	LogOutput = io.Discard
	log = NilLog{}
}

type nilLog struct{}

func (nilLog) Info(string, any...) {}
func (nilLog) Warn(string, any...) {}
func (nilLog) Error(string, any...) {}
