package logging

type YCLogger struct {
	enabled bool
}

func (l *YCLogger) Enable() {
	l.enabled = true
}

func (l *YCLogger) Disable() {
	l.enabled = false
}

func (l *YCLogger) IsEnabled() bool {
	return l.enabled
}

func (l *YCLogger) Info(format string, args ...interface{}) {
	// TODO
}

func (l *YCLogger) Warn(format string, args ...interface{}) {
	// TODO
}

func (l *YCLogger) Error(format string, args ...interface{}) {
	// TODO
}

func (l *YCLogger) Crit(format string, args ...interface{}) {
	// TODO
}

func NewYCLogger() (*YCLogger, error) {
	// TODO create SDK and set up periodical IAM token renewal
	return &YCLogger{}, nil
}
