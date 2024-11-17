// main_test.go
package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/orangefrg/certrenewer/internal/filehelper"
	"github.com/orangefrg/certrenewer/internal/ychelper"
)

// certConfigSample provides a sample CertConfig for testing purposes.
var certConfigSample = ychelper.CertConfig{
	Name:        "example-cert",
	PrivKeyPath: "/path/to/privkey.pem",
	ChainPath:   "/path/to/chain.pem",
	ServiceName: "example-service",
}

// TestLoadConfig tests the LoadConfig function with various scenarios.
func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		expectError bool
		expectedCfg *MainConfig
	}{
		{
			name: "Valid configuration",
			yamlContent: `
renewalPeriod: "30d10h"
heartBeatPeriod: "3d"
certs:
  - name: "example-cert"
    privKey: "/path/to/privkey.pem"
    chain: "/path/to/chain.pem"
    service: "example-service"
`,
			expectError: false,
			expectedCfg: &MainConfig{
				RenewalPeriod:   filehelper.Duration{Duration: 30*24*time.Hour + 10*time.Hour},
				HeartBeatPeriod: filehelper.Duration{Duration: 3 * 24 * time.Hour},
				Certs:           []ychelper.CertConfig{certConfigSample},
			},
		},
		{
			name: "Missing heartbeat period",
			yamlContent: `
renewalPeriod: "30d10h"
certs:
  - name: "example-cert"
    privKey: "/path/to/privkey.pem"
    chain: "/path/to/chain.pem"
    service: "example-service"
`,
			expectError: false,
			expectedCfg: &MainConfig{
				RenewalPeriod:   filehelper.Duration{Duration: 30*24*time.Hour + 10*time.Hour},
				HeartBeatPeriod: filehelper.Duration{Duration: (30*24*time.Hour + 10*time.Hour) / 10},
				Certs:           []ychelper.CertConfig{certConfigSample},
			},
		},
		{
			name:        "Missing configuration file",
			yamlContent: "",
			expectError: true,
			expectedCfg: nil,
		},
		{
			name: "Invalid YAML syntax",
			yamlContent: `
renewalPeriod: "30d10h
certs:
  - name: "example-cert"
    privKey: "/path/to/privkey.pem"
    chain: "/path/to/chain.pem"
    service: "example-service"
`,
			expectError: true,
			expectedCfg: nil,
		},
		{
			name: "Missing renewalPeriod",
			yamlContent: `
certs:
  - name: "example-cert"
    privKey: "/path/to/privkey.pem"
    chain: "/path/to/chain.pem"
    service: "example-service"
`,
			expectError: true,
			expectedCfg: nil,
		},
		{
			name: "Invalid renewalPeriod format",
			yamlContent: `
renewalPeriod: "30days10hours"
certs:
  - name: "example-cert"
    privKey: "/path/to/privkey.pem"
    chain: "/path/to/chain.pem"
    service: "example-service"
`,
			expectError: true,
			expectedCfg: nil,
		},
		{
			name: "Missing certs field",
			yamlContent: `
renewalPeriod: "30d10h"
`,
			expectError: true, // No certificates provided
			expectedCfg: nil,
		},
		{
			name: "Empty certs list",
			yamlContent: `
renewalPeriod: "30d10h"
certs: []
`,
			expectError: true, // No certificates provided
			expectedCfg: nil,
		},
		{
			name: "Missing service name",
			yamlContent: `
renewalPeriod: "30d10h"
certs:
  - name: "example-cert"
    privKey: "/path/to/privkey.pem"
    chain: "/path/to/chain.pem"
`,
			expectError: true, // serviceName is not validated in LoadConfig
			expectedCfg: nil,
		},
		{
			name: "Multiple certs entries",
			yamlContent: `
renewalPeriod: "15d5h"
heartBeatPeriod: "3d"
certs:
  - name: "cert-1"
    privKey: "/path/to/privkey1.pem"
    chain: "/path/to/chain1.pem"
    service: "service-1"
  - name: "cert-2"
    privKey: "/path/to/privkey2.pem"
    chain: "/path/to/chain2.pem"
    service: "service-2"
`,
			expectError: false,
			expectedCfg: &MainConfig{
				RenewalPeriod:   filehelper.Duration{Duration: 15*24*time.Hour + 5*time.Hour},
				HeartBeatPeriod: filehelper.Duration{Duration: 3 * 24 * time.Hour},
				Certs: []ychelper.CertConfig{
					{
						Name:        "cert-1",
						PrivKeyPath: "/path/to/privkey1.pem",
						ChainPath:   "/path/to/chain1.pem",
						ServiceName: "service-1",
					},
					{
						Name:        "cert-2",
						PrivKeyPath: "/path/to/privkey2.pem",
						ChainPath:   "/path/to/chain2.pem",
						ServiceName: "service-2",
					},
				},
			},
		},
		{
			name: "Certs with missing name",
			yamlContent: `
renewalPeriod: "10d"
certs:
  - privKey: "/path/to/privkey.pem"
    chain: "/path/to/chain.pem"
    service: "service-1"
`,
			expectError: true,
			expectedCfg: nil,
		},
		{
			name: "Certs with missing privKeyPath",
			yamlContent: `
renewalPeriod: "10d"
certs:
  - name: "cert-1"
    chain: "/path/to/chain.pem"
    service: "service-1"
`,
			expectError: true,
			expectedCfg: nil,
		},
		{
			name: "Certs with missing chainPath",
			yamlContent: `
renewalPeriod: "10d"
certs:
  - name: "cert-1"
    privKey: "/path/to/privkey.pem"
    service: "service-1"
`,
			expectError: true,
			expectedCfg: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfgPath string
			var err error

			if tt.name != "Missing configuration file" {
				// Create a temporary file with the provided YAML content
				tmpFile, err := os.CreateTemp("", "config_*.yaml")
				assert.NoError(t, err, "Failed to create temporary file")

				// Write YAML content to the temporary file
				_, err = tmpFile.WriteString(tt.yamlContent)
				assert.NoError(t, err, "Failed to write to temporary file")

				// Close the file to ensure content is written
				err = tmpFile.Close()
				assert.NoError(t, err, "Failed to close temporary file")

				cfgPath = tmpFile.Name()

				// Ensure the temporary file is removed after the test
				defer os.Remove(cfgPath)
			} else {
				// For the "Missing configuration file" test case, use a non-existent path
				cfgPath = filepath.Join(os.TempDir(), "nonexistent_config.yaml")
			}

			// Call LoadConfig
			cfg, err := LoadConfig(cfgPath)

			if tt.expectError {
				assert.Error(t, err, "Expected an error but got none")
			} else {
				assert.NoError(t, err, "Did not expect an error but got one")

				// Validate the expected configuration
				if tt.expectedCfg != nil {
					// Compare RenewalPeriod
					parsedDuration := cfg.RenewalPeriod.Duration
					expectedDuration := tt.expectedCfg.RenewalPeriod.Duration
					assert.Equal(t, expectedDuration, parsedDuration, "Mismatch in RenewalPeriod")

					// Compare Certs
					assert.Equal(t, len(tt.expectedCfg.Certs), len(cfg.Certs), "Mismatch in number of certs")

					for i, expectedCert := range tt.expectedCfg.Certs {
						if i >= len(cfg.Certs) {
							break
						}
						actualCert := cfg.Certs[i]
						assert.Equal(t, expectedCert.Name, actualCert.Name, "Mismatch in cert name")
						assert.Equal(t, expectedCert.PrivKeyPath, actualCert.PrivKeyPath, "Mismatch in privKeyPath")
						assert.Equal(t, expectedCert.ChainPath, actualCert.ChainPath, "Mismatch in chainPath")
						assert.Equal(t, expectedCert.ServiceName, actualCert.ServiceName, "Mismatch in serviceName")
					}
				} else {
					// If expectedCfg is nil, ensure cfg is also nil
					assert.Nil(t, cfg, "Expected cfg to be nil")
				}
			}
		})
	}
}
