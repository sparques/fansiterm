//go:build !lognone && !logslog && logprintln

package fansiterm

import (
	"fmt"
	"os"
	"time"
)

func init() {
	LogOutput = os.Stdout
	log = printlnLog{}
}

type printlnLog struct{}

func (p printlnLog) Info(msg string, args ...any) {
	p.log("INFO", msg, args...)
}
func (p printlnLog) Warn(msg string, args ...any) {
	p.log("WARN", msg, args...)
}
func (p printlnLog) Error(msg string, args ...any) {
	p.log("ERROR", msg, args...)
}

func (printlnLog) log(lvl, msg string, args ...any) {
	fmt.Fprintf(LogOutput, "%d [%s] %s", time.Now().Unix(), lvl, msg)
	for i := 0; i < len(args)-2; i += 2 {
		fmt.Fprintf(LogOutput, " %s=%v", args[i], args[i+1])
	}
	fmt.Fprintf(LogOutput, "\r\n")
}
