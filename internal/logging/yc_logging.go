package logging

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/orangefrg/certrenewer/internal/ychelper"
	yclog "github.com/yandex-cloud/go-genproto/yandex/cloud/logging/v1"
	ycsdk "github.com/yandex-cloud/go-sdk"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type YCWriter struct {
	logger *YCLogger
}

type YCLogger struct {
	enabled   bool
	SDK       *ycsdk.SDK
	LogGroup  string
	ctx       context.Context
	canceller context.CancelFunc
	writer    *YCWriter
}

func (w *YCWriter) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, errors.New("empty log message")
	}
	if !w.logger.IsEnabled() {
		return 0, errors.New("logger is disabled")
	}
	switch p[0] {
	case 'D':
		w.logger.Debug(string(p[1:]))
	case 'I':
		w.logger.Info(string(p[1:]))
	case 'W':
		w.logger.Warn(string(p[1:]))
	case 'E':
		w.logger.Error(string(p[1:]))
	case 'C':
		w.logger.Crit(string(p[1:]))
	default:
		w.logger.Info(string(p))
	}
	return len(p), nil
}

func (l *YCLogger) Enable() {
	l.ctx, l.canceller = context.WithCancel(context.Background())
	l.enabled = true
}

func (l *YCLogger) Disable() {
	l.canceller()
	l.enabled = false
}

func (l *YCLogger) IsEnabled() bool {
	return l.enabled
}

func (l *YCLogger) Debug(format string, args ...interface{}) {
	l.log(yclog.LogLevel_DEBUG, format, args...)
}

func (l *YCLogger) Info(format string, args ...interface{}) {
	l.log(yclog.LogLevel_INFO, format, args...)
}

func (l *YCLogger) Warn(format string, args ...interface{}) {
	l.log(yclog.LogLevel_WARN, format, args...)
}

func (l *YCLogger) Error(format string, args ...interface{}) {
	l.log(yclog.LogLevel_ERROR, format, args...)
}

func (l *YCLogger) Crit(format string, args ...interface{}) {
	l.log(yclog.LogLevel_FATAL, format, args...)
}

func (l *YCLogger) log(level yclog.LogLevel_Level, format string, args ...interface{}) {
	if !l.enabled {
		return
	}
	l.SDK.LogIngestion().LogIngestion().Write(l.ctx, &yclog.WriteRequest{
		Destination: &yclog.Destination{
			Destination: &yclog.Destination_LogGroupId{
				LogGroupId: l.LogGroup,
			},
		},
		Entries: []*yclog.IncomingLogEntry{
			{
				Timestamp: timestamppb.New(time.Now()),
				Level:     level,
				Message:   fmt.Sprintf(format, args...),
			},
		},
	}, nil)
}

func (l *YCLogger) GetWriter() io.Writer {
	return l.writer
}

func NewYCLogger(logGroupName string, folderId string) (Logger, error) {
	sdk, err := ychelper.MakeSDKForInstanceSA()
	if err != nil {
		return nil, fmt.Errorf("could not initialize SDK: %w", err)
	}
	logResp, err := sdk.Logging().LogGroup().List(context.Background(), &yclog.ListLogGroupsRequest{
		FolderId: folderId,
		Filter:   fmt.Sprintf("name=%s", logGroupName),
	})
	if err != nil {
		return nil, fmt.Errorf("could not list filtered log groups with name %s: %w", logGroupName, err)
	}
	logGroupId := ""
	for _, grp := range logResp.Groups {
		if grp.Name == logGroupName {
			logGroupId = grp.Id
			break
		}
	}
	if logGroupId == "" {
		return nil, fmt.Errorf("could not find log group with name %s", logGroupName)
	}
	ycl := YCLogger{
		SDK:      sdk,
		enabled:  true,
		LogGroup: logGroupId,
	}
	ycl.writer = &YCWriter{logger: &ycl}
	ycl.Enable()
	return &ycl, nil
}
