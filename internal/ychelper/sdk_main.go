package ychelper

import (
	"context"
	"fmt"

	ycsdk "github.com/yandex-cloud/go-sdk"
)

func MakeSDK() (*ycsdk.SDK, error) {
	iamToken, err := GetIamToken()
	if err != nil {
		return nil, fmt.Errorf("could not get IAM key: %w", err)
	}
	sdk, err := ycsdk.Build(context.Background(), ycsdk.Config{
		Credentials: ycsdk.NewIAMTokenCredentials(iamToken.AccessToken),
	})
	if err != nil {
		return nil, fmt.Errorf("could not initialize SDK: %w", err)
	}
	return sdk, nil
}
