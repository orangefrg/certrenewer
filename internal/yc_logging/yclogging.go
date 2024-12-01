package yc_logging

import (
	"context"
	"fmt"
	"time"

	"github.com/orangefrg/certrenewer/internal/ychelper"
	"github.com/sirupsen/logrus"
	yclog "github.com/yandex-cloud/go-genproto/yandex/cloud/logging/v1"
	ycsdk "github.com/yandex-cloud/go-sdk"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type YandexCloudHook struct {
	ctx      context.Context
	SDK      *ycsdk.SDK
	LogGroup string
}

func NewYandexCloudHook(logGroupName string, folderId string) (*YandexCloudHook, error) {
	ctx := context.Background()
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
	ycl := YandexCloudHook{
		ctx:      ctx,
		SDK:      sdk,
		LogGroup: logGroupId,
	}
	return &ycl, nil
}

func (hook *YandexCloudHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (hook *YandexCloudHook) Fire(entry *logrus.Entry) error {
	_, err := hook.SDK.LogIngestion().LogIngestion().Write(hook.ctx, &yclog.WriteRequest{
		Destination: &yclog.Destination{
			Destination: &yclog.Destination_LogGroupId{
				LogGroupId: hook.LogGroup,
			},
		},
		Entries: []*yclog.IncomingLogEntry{
			{
				Timestamp: timestamppb.New(time.Now()),
				Level:     convertLevel(entry.Level),
				Message:   entry.Message,
			},
		},
	}, nil)
	return err
}

func convertLevel(level logrus.Level) yclog.LogLevel_Level {
	switch level {
	case logrus.PanicLevel, logrus.FatalLevel:
		return yclog.LogLevel_FATAL
	case logrus.ErrorLevel:
		return yclog.LogLevel_ERROR
	case logrus.WarnLevel:
		return yclog.LogLevel_WARN
	case logrus.InfoLevel:
		return yclog.LogLevel_INFO
	case logrus.DebugLevel, logrus.TraceLevel:
		return yclog.LogLevel_DEBUG
	default:
		return yclog.LogLevel_INFO
	}
}
