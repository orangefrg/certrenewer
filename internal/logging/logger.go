package logging

import "io"

type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	Crit(format string, args ...interface{})
	GetWriter() io.Writer
	Enable()
	Disable()
	IsEnabled() bool
}
