//go:build !lognone && logslog && !logprintln

package fansiterm

import (
	"log/slog"
	"os"
)

func init() {
	LogOutput = os.Stdout
	log = slog.Default()
}
