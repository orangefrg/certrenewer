package ychelper

import (
	"context"
	"fmt"

	ycsdk "github.com/yandex-cloud/go-sdk"
)

func MakeSDKForInstanceSA() (*ycsdk.SDK, error) {
	sdk, err := ycsdk.Build(context.Background(), ycsdk.Config{
		Credentials: ycsdk.InstanceServiceAccount(),
	})
	if err != nil {
		return nil, fmt.Errorf("could not initialize SDK: %w", err)
	}
	return sdk, nil
}
