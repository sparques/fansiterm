package fansiterm

import "io"

var (
	LogOutput io.Writer
	log       Logger
)

type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}
