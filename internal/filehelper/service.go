package filehelper

import (
	"fmt"
	"os/exec"
)

var ServiceRestart = func(serviceName string) error {
	cmd := exec.Command("systemctl", "restart", serviceName)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to restart service %s: %w", serviceName, err)
	}
	return nil
}
