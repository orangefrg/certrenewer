package ychelper

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/orangefrg/certrenewer/internal/filehelper"
	"github.com/stretchr/testify/assert"
	certmgr "github.com/yandex-cloud/go-genproto/yandex/cloud/certificatemanager/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type mockCertificateService struct {
	ListFunc func(ctx context.Context, req *certmgr.ListCertificatesRequest, opts ...grpc.CallOption) (*certmgr.ListCertificatesResponse, error)
	GetFunc  func(ctx context.Context, req *certmgr.GetCertificateRequest, opts ...grpc.CallOption) (*certmgr.Certificate, error)
}

func (m *mockCertificateService) List(ctx context.Context, req *certmgr.ListCertificatesRequest, opts ...grpc.CallOption) (*certmgr.ListCertificatesResponse, error) {
	return m.ListFunc(ctx, req, opts...)
}

func (m *mockCertificateService) Get(ctx context.Context, req *certmgr.GetCertificateRequest, opts ...grpc.CallOption) (*certmgr.Certificate, error) {
	return m.GetFunc(ctx, req, opts...)
}

type mockCertificateContentService struct {
	GetFunc func(ctx context.Context, req *certmgr.GetCertificateContentRequest, opts ...grpc.CallOption) (*certmgr.GetCertificateContentResponse, error)
}

func (m *mockCertificateContentService) Get(ctx context.Context, req *certmgr.GetCertificateContentRequest, opts ...grpc.CallOption) (*certmgr.GetCertificateContentResponse, error) {
	return m.GetFunc(ctx, req, opts...)
}

func TestGetCertificate(t *testing.T) {
	tests := []struct {
		name           string
		folderId       string
		certName       string
		dueDate        time.Time
		certList       *certmgr.ListCertificatesResponse
		certInfo       *certmgr.Certificate
		certContents   *certmgr.GetCertificateContentResponse
		listErr        error
		getErr         error
		contentGetErr  error
		expectedUpdate bool
		expectedChain  []string
		expectedKey    string
		expectedErr    error
	}{
		{
			name:     "Certificate not found",
			folderId: "folder1",
			certName: "cert1",
			certList: &certmgr.ListCertificatesResponse{
				Certificates: []*certmgr.Certificate{},
			},
			expectedUpdate: false,
			expectedErr:    errors.New("could not find certificate of given name"),
		},
		{
			name:     "Certificate status not issued",
			folderId: "folder1",
			certName: "cert1",
			certList: &certmgr.ListCertificatesResponse{
				Certificates: []*certmgr.Certificate{
					{Name: "cert1", Id: "cert1-id"},
				},
			},
			certInfo: &certmgr.Certificate{
				Status: certmgr.Certificate_REVOKED,
			},
			expectedUpdate: false,
			expectedErr:    fmt.Errorf("could not renew certificate: abnormal cert status %d", int(certmgr.Certificate_REVOKED)),
		},
		{
			name:     "Certificate does not need update",
			folderId: "folder1",
			certName: "cert1",
			dueDate:  time.Now().Add(48 * time.Hour),
			certList: &certmgr.ListCertificatesResponse{
				Certificates: []*certmgr.Certificate{
					{Name: "cert1", Id: "cert1-id"},
				},
			},
			certInfo: &certmgr.Certificate{
				NotAfter: timestamppb.New(time.Now().Add(24 * time.Hour)),
				Status:   certmgr.Certificate_ISSUED,
			},
			expectedUpdate: false,
			expectedErr:    nil,
		},
		{
			name:     "Certificate needs update",
			folderId: "folder1",
			certName: "cert1",
			dueDate:  time.Now().Add(24 * time.Hour),
			certList: &certmgr.ListCertificatesResponse{
				Certificates: []*certmgr.Certificate{
					{Name: "cert1", Id: "cert1-id"},
				},
			},
			certInfo: &certmgr.Certificate{
				Status:   certmgr.Certificate_ISSUED,
				NotAfter: timestamppb.New(time.Now().Add(48 * time.Hour)),
			},
			certContents: &certmgr.GetCertificateContentResponse{
				CertificateChain: []string{"cert-chain"},
				PrivateKey:       "private-key",
			},
			expectedUpdate: true,
			expectedChain:  []string{"cert-chain"},
			expectedKey:    "private-key",
			expectedErr:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := &CertificateManager{
				Certificate: &mockCertificateService{
					ListFunc: func(ctx context.Context, req *certmgr.ListCertificatesRequest, opts ...grpc.CallOption) (*certmgr.ListCertificatesResponse, error) {
						return tt.certList, tt.listErr
					},
					GetFunc: func(ctx context.Context, req *certmgr.GetCertificateRequest, opts ...grpc.CallOption) (*certmgr.Certificate, error) {
						return tt.certInfo, tt.getErr
					},
				},
				CertificateContent: &mockCertificateContentService{
					GetFunc: func(ctx context.Context, req *certmgr.GetCertificateContentRequest, opts ...grpc.CallOption) (*certmgr.GetCertificateContentResponse, error) {
						return tt.certContents, tt.contentGetErr
					},
				},
			}

			needsUpdate, chain, privKey, err := GetCertificate(tt.folderId, tt.certName, tt.dueDate, cm)

			assert.Equal(t, tt.expectedUpdate, needsUpdate)
			assert.Equal(t, tt.expectedChain, chain)
			assert.Equal(t, tt.expectedKey, privKey)
			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
func TestRenewCertificates(t *testing.T) {
	tests := []struct {
		name          string
		folderId      string
		certs         []CertConfig
		certList      *certmgr.ListCertificatesResponse
		certInfo      *certmgr.Certificate
		certContents  *certmgr.GetCertificateContentResponse
		listErr       error
		getErr        error
		contentGetErr error
		expiryDateErr error
		expectedLogs  []string
	}{
		{
			name:     "Certificate not found",
			folderId: "folder1",
			certs: []CertConfig{
				{Name: "cert1", PrivKeyPath: "privKey1", ChainPath: "chain1", ServiceName: "service1"},
			},
			certList: &certmgr.ListCertificatesResponse{
				Certificates: []*certmgr.Certificate{},
			},
			expectedLogs: []string{
				"Updating cert1 (1/1)...",
				"could not find certificate of given name",
			},
		},
		{
			name:     "Certificate status not issued",
			folderId: "folder1",
			certs: []CertConfig{
				{Name: "cert1", PrivKeyPath: "privKey1", ChainPath: "chain1", ServiceName: "service1"},
			},
			certList: &certmgr.ListCertificatesResponse{
				Certificates: []*certmgr.Certificate{
					{Name: "cert1", Id: "cert1-id"},
				},
			},
			certInfo: &certmgr.Certificate{
				Status: certmgr.Certificate_REVOKED,
			},
			expectedLogs: []string{
				"Updating cert1 (1/1)...",
				"could not renew certificate: abnormal cert status 4",
			},
		},
		{
			name:     "Certificate does not need update",
			folderId: "folder1",
			certs: []CertConfig{
				{Name: "cert1", PrivKeyPath: "privKey1", ChainPath: "chain1", ServiceName: "service1"},
			},
			certList: &certmgr.ListCertificatesResponse{
				Certificates: []*certmgr.Certificate{
					{Name: "cert1", Id: "cert1-id"},
				},
			},
			certInfo: &certmgr.Certificate{
				Status:   certmgr.Certificate_ISSUED,
				NotAfter: timestamppb.New(time.Now().Add(12 * time.Hour)),
			},
			certContents: &certmgr.GetCertificateContentResponse{
				CertificateChain: []string{"cert-chain"},
				PrivateKey:       "private-key",
			},
			expectedLogs: []string{
				"Updating cert1 (1/1)...",
				"Certificate cert1 does not need to be updated",
			},
		},
		{
			name:     "Certificate needs update",
			folderId: "folder1",
			certs: []CertConfig{
				{Name: "cert1", PrivKeyPath: "privKey1", ChainPath: "chain1", ServiceName: "service1"},
			},
			certList: &certmgr.ListCertificatesResponse{
				Certificates: []*certmgr.Certificate{
					{Name: "cert1", Id: "cert1-id"},
				},
			},
			certInfo: &certmgr.Certificate{
				Status:   certmgr.Certificate_ISSUED,
				NotAfter: timestamppb.New(time.Now().Add(48 * time.Hour)),
			},
			certContents: &certmgr.GetCertificateContentResponse{
				CertificateChain: []string{"cert-chain"},
				PrivateKey:       "private-key",
			},
			expectedLogs: []string{
				"Updating cert1 (1/1)...",
				"Successfully written certificate cert1",
				"Restarting services...",
				"Restarting service1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock GetCertificateExpiryDate
			originalGetCertificateExpiryDate := GetCertificateExpiryDate
			defer func() { GetCertificateExpiryDate = originalGetCertificateExpiryDate }()
			GetCertificateExpiryDate = func(certPath string) (time.Time, error) {
				return time.Now().Add(24 * time.Hour), tt.expiryDateErr
			}

			cm := &CertificateManager{
				Certificate: &mockCertificateService{
					ListFunc: func(ctx context.Context, req *certmgr.ListCertificatesRequest, opts ...grpc.CallOption) (*certmgr.ListCertificatesResponse, error) {
						return tt.certList, tt.listErr
					},
					GetFunc: func(ctx context.Context, req *certmgr.GetCertificateRequest, opts ...grpc.CallOption) (*certmgr.Certificate, error) {
						return tt.certInfo, tt.getErr
					},
				},
				CertificateContent: &mockCertificateContentService{
					GetFunc: func(ctx context.Context, req *certmgr.GetCertificateContentRequest, opts ...grpc.CallOption) (*certmgr.GetCertificateContentResponse, error) {
						return tt.certContents, tt.contentGetErr
					},
				},
			}

			// Mock filehelper.WriteWithBackup
			filehelper.WriteWithBackup = func(filename string, data []byte, perm os.FileMode) error {
				return nil
			}

			// Mock filehelper.ServiceRestart
			filehelper.ServiceRestart = func(serviceName string) error {
				return nil
			}

			// Capture logs
			var logs []string
			log.SetOutput(&logWriter{logs: &logs})

			RenewCertificates(tt.folderId, cm, tt.certs)

			// Check logs
			for _, expectedLog := range tt.expectedLogs {
				var found bool
				for _, sLog := range logs {
					if strings.Contains(sLog, expectedLog) {
						found = true
						break
					}
				}
				assert.True(t, found, "expected log not found: %s\nLogs: ", expectedLog, logs)
			}
		})
	}
}

func TestGetCertificateExpiryDate(t *testing.T) {
	tests := []struct {
		name        string
		certPath    string
		certContent string
		expected    time.Time
		expectErr   bool
	}{
		{
			name:        "Valid certificate",
			certPath:    "valid_cert.pem",
			certContent: "-----BEGIN CERTIFICATE-----\nMIIOCjCCDPKgAwIBAgIQQagVgnQLepAJtGHnFyDA9DANBgkqhkiG9w0BAQsFADA7MQswCQYDVQQGEwJVUzEeMBwGA1UEChMVR29vZ2xlIFRydXN0IFNlcnZpY2VzMQwwCgYDVQQDEwNXUjIwHhcNMjQxMDIxMDgzNjU3WhcNMjUwMTEzMDgzNjU2WjAXMRUwEwYDVQQDDAwqLmdvb2dsZS5jb20wWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAARD7W/bU6abojd0puaRMYsiXqZjXddRl8yW2qlTpw+HOHg3bA183UxWTtZx+yeHSVQE3k0jMGvR7C4B3FY+n92ao4IL9zCCC/MwDgYDVR0PAQH/BAQDAgeAMBMGA1UdJQQMMAoGCCsGAQUFBwMBMAwGA1UdEwEB/wQCMAAwHQYDVR0OBBYEFIz1r4xlXzoNaxGYxiGq0aGZuvJpMB8GA1UdIwQYMBaAFN4bHu15FdQ+NyTDIbvsNDltQrIwMFgGCCsGAQUFBwEBBEwwSjAhBggrBgEFBQcwAYYVaHR0cDovL28ucGtpLmdvb2cvd3IyMCUGCCsGAQUFBzAChhlodHRwOi8vaS5wa2kuZ29vZy93cjIuY3J0MIIJzQYDVR0RBIIJxDCCCcCCDCouZ29vZ2xlLmNvbYIWKi5hcHBlbmdpbmUuZ29vZ2xlLmNvbYIJKi5iZG4uZGV2ghUqLm9yaWdpbi10ZXN0LmJkbi5kZXaCEiouY2xvdWQuZ29vZ2xlLmNvbYIYKi5jcm93ZHNvdXJjZS5nb29nbGUuY29tghgqLmRhdGFjb21wdXRlLmdvb2dsZS5jb22CCyouZ29vZ2xlLmNhggsqLmdvb2dsZS5jbIIOKi5nb29nbGUuY28uaW6CDiouZ29vZ2xlLmNvLmpwgg4qLmdvb2dsZS5jby51a4IPKi5nb29nbGUuY29tLmFygg8qLmdvb2dsZS5jb20uYXWCDyouZ29vZ2xlLmNvbS5icoIPKi5nb29nbGUuY29tLmNvgg8qLmdvb2dsZS5jb20ubXiCDyouZ29vZ2xlLmNvbS50coIPKi5nb29nbGUuY29tLnZuggsqLmdvb2dsZS5kZYILKi5nb29nbGUuZXOCCyouZ29vZ2xlLmZyggsqLmdvb2dsZS5odYILKi5nb29nbGUuaXSCCyouZ29vZ2xlLm5sggsqLmdvb2dsZS5wbIILKi5nb29nbGUucHSCDyouZ29vZ2xlYXBpcy5jboIRKi5nb29nbGV2aWRlby5jb22CDCouZ3N0YXRpYy5jboIQKi5nc3RhdGljLWNuLmNvbYIPZ29vZ2xlY25hcHBzLmNughEqLmdvb2dsZWNuYXBwcy5jboIRZ29vZ2xlYXBwcy1jbi5jb22CEyouZ29vZ2xlYXBwcy1jbi5jb22CDGdrZWNuYXBwcy5jboIOKi5na2VjbmFwcHMuY26CEmdvb2dsZWRvd25sb2Fkcy5jboIUKi5nb29nbGVkb3dubG9hZHMuY26CEHJlY2FwdGNoYS5uZXQuY26CEioucmVjYXB0Y2hhLm5ldC5jboIQcmVjYXB0Y2hhLWNuLm5ldIISKi5yZWNhcHRjaGEtY24ubmV0ggt3aWRldmluZS5jboINKi53aWRldmluZS5jboIRYW1wcHJvamVjdC5vcmcuY26CEyouYW1wcHJvamVjdC5vcmcuY26CEWFtcHByb2plY3QubmV0LmNughMqLmFtcHByb2plY3QubmV0LmNughdnb29nbGUtYW5hbHl0aWNzLWNuLmNvbYIZKi5nb29nbGUtYW5hbHl0aWNzLWNuLmNvbYIXZ29vZ2xlYWRzZXJ2aWNlcy1jbi5jb22CGSouZ29vZ2xlYWRzZXJ2aWNlcy1jbi5jb22CEWdvb2dsZXZhZHMtY24uY29tghMqLmdvb2dsZXZhZHMtY24uY29tghFnb29nbGVhcGlzLWNuLmNvbYITKi5nb29nbGVhcGlzLWNuLmNvbYIVZ29vZ2xlb3B0aW1pemUtY24uY29tghcqLmdvb2dsZW9wdGltaXplLWNuLmNvbYISZG91YmxlY2xpY2stY24ubmV0ghQqLmRvdWJsZWNsaWNrLWNuLm5ldIIYKi5mbHMuZG91YmxlY2xpY2stY24ubmV0ghYqLmcuZG91YmxlY2xpY2stY24ubmV0gg5kb3VibGVjbGljay5jboIQKi5kb3VibGVjbGljay5jboIUKi5mbHMuZG91YmxlY2xpY2suY26CEiouZy5kb3VibGVjbGljay5jboIRZGFydHNlYXJjaC1jbi5uZXSCEyouZGFydHNlYXJjaC1jbi5uZXSCHWdvb2dsZXRyYXZlbGFkc2VydmljZXMtY24uY29tgh8qLmdvb2dsZXRyYXZlbGFkc2VydmljZXMtY24uY29tghhnb29nbGV0YWdzZXJ2aWNlcy1jbi5jb22CGiouZ29vZ2xldGFnc2VydmljZXMtY24uY29tghdnb29nbGV0YWdtYW5hZ2VyLWNuLmNvbYIZKi5nb29nbGV0YWdtYW5hZ2VyLWNuLmNvbYIYZ29vZ2xlc3luZGljYXRpb24tY24uY29tghoqLmdvb2dsZXN5bmRpY2F0aW9uLWNuLmNvbYIkKi5zYWZlZnJhbWUuZ29vZ2xlc3luZGljYXRpb24tY24uY29tghZhcHAtbWVhc3VyZW1lbnQtY24uY29tghgqLmFwcC1tZWFzdXJlbWVudC1jbi5jb22CC2d2dDEtY24uY29tgg0qLmd2dDEtY24uY29tggtndnQyLWNuLmNvbYINKi5ndnQyLWNuLmNvbYILMm1kbi1jbi5uZXSCDSouMm1kbi1jbi5uZXSCFGdvb2dsZWZsaWdodHMtY24ubmV0ghYqLmdvb2dsZWZsaWdodHMtY24ubmV0ggxhZG1vYi1jbi5jb22CDiouYWRtb2ItY24uY29tghRnb29nbGVzYW5kYm94LWNuLmNvbYIWKi5nb29nbGVzYW5kYm94LWNuLmNvbYIeKi5zYWZlbnVwLmdvb2dsZXNhbmRib3gtY24uY29tgg0qLmdzdGF0aWMuY29tghQqLm1ldHJpYy5nc3RhdGljLmNvbYIKKi5ndnQxLmNvbYIRKi5nY3BjZG4uZ3Z0MS5jb22CCiouZ3Z0Mi5jb22CDiouZ2NwLmd2dDIuY29tghAqLnVybC5nb29nbGUuY29tghYqLnlvdXR1YmUtbm9jb29raWUuY29tggsqLnl0aW1nLmNvbYILYW5kcm9pZC5jb22CDSouYW5kcm9pZC5jb22CEyouZmxhc2guYW5kcm9pZC5jb22CBGcuY26CBiouZy5jboIEZy5jb4IGKi5nLmNvggZnb28uZ2yCCnd3dy5nb28uZ2yCFGdvb2dsZS1hbmFseXRpY3MuY29tghYqLmdvb2dsZS1hbmFseXRpY3MuY29tggpnb29nbGUuY29tghJnb29nbGVjb21tZXJjZS5jb22CFCouZ29vZ2xlY29tbWVyY2UuY29tgghnZ3BodC5jboIKKi5nZ3BodC5jboIKdXJjaGluLmNvbYIMKi51cmNoaW4uY29tggh5b3V0dS5iZYILeW91dHViZS5jb22CDSoueW91dHViZS5jb22CEW11c2ljLnlvdXR1YmUuY29tghMqLm11c2ljLnlvdXR1YmUuY29tghR5b3V0dWJlZWR1Y2F0aW9uLmNvbYIWKi55b3V0dWJlZWR1Y2F0aW9uLmNvbYIPeW91dHViZWtpZHMuY29tghEqLnlvdXR1YmVraWRzLmNvbYIFeXQuYmWCByoueXQuYmWCGmFuZHJvaWQuY2xpZW50cy5nb29nbGUuY29tghMqLmFuZHJvaWQuZ29vZ2xlLmNughIqLmNocm9tZS5nb29nbGUuY26CFiouZGV2ZWxvcGVycy5nb29nbGUuY24wEwYDVR0gBAwwCjAIBgZngQwBAgEwNgYDVR0fBC8wLTAroCmgJ4YlaHR0cDovL2MucGtpLmdvb2cvd3IyL29RNm55cjhGMG0wLmNybDCCAQQGCisGAQQB1nkCBAIEgfUEgfIA8AB1AH1ZHhLheCp7HGFnfF79+NCHXBSgTpWeuQMv2Q6MLnm4AAABkq5v6a4AAAQDAEYwRAIgIqLlTU1VytekLFYe6z7B9QZvvAFEtlvZ/rQAndvx4BcCIDhSWEOXYUPI6kr3vu50z40WJeAuKG8/FkmCKArIiuRJAHcAzxFW7tUufK/zh1vZaS6b6RpxZ0qwF+ysAdJbd87MOwgAAAGSrm/pugAABAMASDBGAiEA8kgns/UoNAL7/3WsQF/6ifdMBOunXlADyPxvWbpRPe0CIQCQphqwbXlfq7EkCyUnzvMJmMs9PDoMv7elVg36zqA5HjANBgkqhkiG9w0BAQsFAAOCAQEAOp1JjUVTIoBGg90DPzx1y/0N3qu5UVDGbuuPlDFzHmjjnGb20C/s6pJzAuUxMvvIClk6m12PCbS/F6ESGYf9QzIUBmU5BEiOHVSuTQrnDvUzrspB0eSe2qEqfDDvoaSb7MPX5kO7crHdBtG+DFUzUa/CZYsgbFW7or+ennD8oJT6NFCvPah7iIJwx5S1NIBPEot8IwzKBP9VrjVBhp9yXXr1hWai/Z2gOacoJsLzbEifB+hlwldNgYwdSnVkTe6n5FZIygGxGm97Dp0v54g0Qs+U4cfvwPIfx7D8oBz0EiCPlRm89TjTr0y8LqujdfiZdfbQQsUsNgsVNtIiiY0CSw==\n-----END CERTIFICATE-----",
			expected:    time.Date(2025, 1, 13, 8, 36, 56, 0, time.UTC),
			expectErr:   false,
		},
		{
			name:        "Invalid certificate",
			certPath:    "invalid_cert.pem",
			certContent: `-----BEGIN CERTIFICATE-----\nINVALID\n-----END CERTIFICATE-----`,
			expected:    time.Time{},
			expectErr:   true,
		},
		{
			name:        "No certificate found",
			certPath:    "no_cert.pem",
			certContent: ``,
			expected:    time.Time{},
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "cert_test_*.pem")
			assert.NoError(t, err)
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.Write([]byte(tt.certContent))
			assert.NoError(t, err)
			tmpFile.Close()

			expiryDate, err := GetCertificateExpiryDate(tmpFile.Name())

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.WithinDuration(t, tt.expected, expiryDate, time.Minute)
			}
		})
	}
}

type logWriter struct {
	logs *[]string
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	*w.logs = append(*w.logs, string(p))
	return len(p), nil
}
