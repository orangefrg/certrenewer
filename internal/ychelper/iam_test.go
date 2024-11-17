// iam_test.go

package ychelper

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMeta_Success(t *testing.T) {
	// Mock server to simulate successful response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Google", r.Header.Get("Metadata-Flavor"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
            "id": "test-id",
            "hostname": "test-hostname",
            "name": "test-name",
            "vendor": {
                "cloudId": "test-cloudId",
                "folderId": "test-folderId"
            }
        }`))
	}))
	defer server.Close()

	// Override MetadataURL and IdURL for testing
	originalMetadataURL := MetadataURL
	originalIdURL := IdURL
	defer func() {
		MetadataURL = originalMetadataURL
		IdURL = originalIdURL
	}()
	MetadataURL = server.URL
	IdURL = ""

	parsed, err := GetMeta()
	assert.NoError(t, err)
	assert.NotNil(t, parsed)
	assert.Equal(t, "test-id", parsed.Id)
	assert.Equal(t, "test-hostname", parsed.Hostname)
	assert.Equal(t, "test-name", parsed.Name)
	assert.Equal(t, "test-cloudId", parsed.Vendor.CloudId)
	assert.Equal(t, "test-folderId", parsed.Vendor.FolderId)
}

func TestGetMeta_NewRequestError(t *testing.T) {
	// Set invalid URL to cause http.NewRequest to fail
	originalMetadataURL := MetadataURL
	defer func() { MetadataURL = originalMetadataURL }()
	MetadataURL = "http://\x7f"

	parsed, err := GetMeta()
	assert.Nil(t, parsed)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not make a request to metadata")
}

func TestGetMeta_DoRequestError(t *testing.T) {
	httpClient = &http.Client{
		Transport: &mockTransport{err: errors.New("mock client.Do error")},
	}
	defer func() { httpClient = &http.Client{} }()

	parsed, err := GetMeta()
	assert.Nil(t, parsed)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not fetch metadata")
}

func TestGetMeta_NonOKStatus(t *testing.T) {
	// Mock server to return non-OK status
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Override MetadataURL and IdURL for testing
	originalMetadataURL := MetadataURL
	originalIdURL := IdURL
	defer func() {
		MetadataURL = originalMetadataURL
		IdURL = originalIdURL
	}()
	MetadataURL = server.URL
	IdURL = ""

	parsed, err := GetMeta()
	assert.Nil(t, parsed)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "abnormal error code")
}

func TestGetMeta_ReadAllError(t *testing.T) {
	// Mock server to force ReadAll error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		buf.WriteString("{}")
		w.Header().Set("Content-Length", "500")
		io.Copy(w, &buf)
	}))
	defer server.Close()

	// Override MetadataURL and IdURL for testing
	originalMetadataURL := MetadataURL
	originalIdURL := IdURL
	defer func() {
		MetadataURL = originalMetadataURL
		IdURL = originalIdURL
	}()
	MetadataURL = server.URL
	IdURL = ""

	parsed, err := GetMeta()
	assert.Nil(t, parsed)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not read metadata response")
}

func TestGetMeta_JSONUnmarshalError(t *testing.T) {
	// Mock server to return invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	// Override MetadataURL and IdURL for testing
	originalMetadataURL := MetadataURL
	originalIdURL := IdURL
	defer func() {
		MetadataURL = originalMetadataURL
		IdURL = originalIdURL
	}()
	MetadataURL = server.URL
	IdURL = ""

	parsed, err := GetMeta()
	assert.Nil(t, parsed)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not parse JSON response")
}

func TestGetIamToken_Success(t *testing.T) {
	// Mock server to simulate successful token response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Google", r.Header.Get("Metadata-Flavor"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
            "access_token": "test-access-token",
            "expires_in": 3600,
            "token_type": "Bearer"
        }`))
	}))
	defer server.Close()

	// Override MetadataURL and IamTokenURL for testing
	originalMetadataURL := MetadataURL
	originalIamTokenURL := IamTokenURL
	defer func() {
		MetadataURL = originalMetadataURL
		IamTokenURL = originalIamTokenURL
	}()
	MetadataURL = server.URL
	IamTokenURL = ""

	tokenResponse, err := GetIamToken()
	assert.NoError(t, err)
	assert.NotNil(t, tokenResponse)
	assert.Equal(t, "test-access-token", tokenResponse.AccessToken)
	assert.Equal(t, 3600, tokenResponse.ExpiresIn)
	assert.Equal(t, "Bearer", tokenResponse.TokenType)
}

func TestGetIamToken_NewRequestError(t *testing.T) {
	// Set invalid URL to cause http.NewRequest to fail
	originalMetadataURL := MetadataURL
	defer func() { MetadataURL = originalMetadataURL }()
	MetadataURL = "http://\x7f"

	tokenResponse, err := GetIamToken()
	assert.Nil(t, tokenResponse)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not make a request to metadata")
}

func TestGetIamToken_DoRequestError(t *testing.T) {
	// Override httpClient to use a client that returns an error
	httpClient = &http.Client{
		Transport: &mockTransport{err: errors.New("mock client.Do error")},
	}
	defer func() { httpClient = &http.Client{} }()

	tokenResponse, err := GetIamToken()
	assert.Nil(t, tokenResponse)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not fetch metadata")
}

func TestGetIamToken_NonOKStatus(t *testing.T) {
	// Mock server to return non-OK status
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Override MetadataURL and IamTokenURL for testing
	originalMetadataURL := MetadataURL
	originalIamTokenURL := IamTokenURL
	defer func() {
		MetadataURL = originalMetadataURL
		IamTokenURL = originalIamTokenURL
	}()
	MetadataURL = server.URL
	IamTokenURL = ""

	tokenResponse, err := GetIamToken()
	assert.Nil(t, tokenResponse)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "abnormal error code")
}

func TestGetIamToken_ReadAllError(t *testing.T) {
	// Mock server to force ReadAll error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		buf.WriteString("{}")
		w.Header().Set("Content-Length", "500")
		io.Copy(w, &buf)
	}))
	defer server.Close()

	// Override MetadataURL and IamTokenURL for testing
	originalMetadataURL := MetadataURL
	originalIamTokenURL := IamTokenURL
	defer func() {
		MetadataURL = originalMetadataURL
		IamTokenURL = originalIamTokenURL
	}()
	MetadataURL = server.URL
	IamTokenURL = ""

	tokenResponse, err := GetIamToken()
	assert.Nil(t, tokenResponse)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not read metadata response")
}

func TestGetIamToken_JSONUnmarshalError(t *testing.T) {
	// Mock server to return invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	// Override MetadataURL and IamTokenURL for testing
	originalMetadataURL := MetadataURL
	originalIamTokenURL := IamTokenURL
	defer func() {
		MetadataURL = originalMetadataURL
		IamTokenURL = originalIamTokenURL
	}()
	MetadataURL = server.URL
	IamTokenURL = ""

	tokenResponse, err := GetIamToken()
	assert.Nil(t, tokenResponse)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not parse JSON response")
}

// Mock transport to simulate client.Do error
type mockTransport struct {
	err error
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, m.err
}
