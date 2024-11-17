package logging

type Logger interface {
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	Crit(format string, args ...interface{})
	Enable()
	Disable()
	IsEnabled() bool
}
