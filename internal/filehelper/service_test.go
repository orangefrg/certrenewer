package filehelper

import (
	"fmt"
	"testing"
)

func TestServiceRestart(t *testing.T) {
	originalServiceRestart := ServiceRestart
	defer func() { ServiceRestart = originalServiceRestart }()

	mockServiceRestart := func(serviceName string) error {
		if serviceName == "fail" {
			return fmt.Errorf("mock failure")
		}
		return nil
	}

	ServiceRestart = mockServiceRestart

	tests := []struct {
		serviceName string
		expectError bool
	}{
		{"validService", false},
		{"fail", true},
	}

	for _, test := range tests {
		err := ServiceRestart(test.serviceName)
		if test.expectError && err == nil {
			t.Errorf("expected error for service %s, got nil", test.serviceName)
		}
		if !test.expectError && err != nil {
			t.Errorf("did not expect error for service %s, got %v", test.serviceName, err)
		}
	}
}
