package ychelper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

var (
	MetadataURL = "http://169.254.169.254"
	IamTokenURL = "/computeMetadata/v1/instance/service-accounts/default/token"
	IdURL       = "/computeMetadata/v1/instance/?recursive=true"
	httpClient  = &http.Client{}
)

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

type ParsedVMMeta struct {
	Id       string       `json:"id"`
	Hostname string       `json:"hostname"`
	Name     string       `json:"name"`
	Vendor   ParsedVendor `json:"vendor"`
}

type ParsedVendor struct {
	CloudId  string `json:"cloudId"`
	FolderId string `json:"folderId"`
}

type VMIdentity struct {
	VMId     string
	CloudId  string
	FolderId string
	Name     string
	Hostname string
}

func GetMeta() (*ParsedVMMeta, error) {
	url := MetadataURL + IdURL
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("could not make a request to metadata: %w", err)
	}
	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not fetch metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("abnormal error code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read metadata response: %w", err)
	}

	var parsed ParsedVMMeta
	err = json.Unmarshal(body, &parsed)
	if err != nil {
		return nil, fmt.Errorf("could not parse JSON response: %w", err)
	}

	return &parsed, nil

}

func GetIamToken() (*TokenResponse, error) {
	url := MetadataURL + IamTokenURL
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("could not make a request to metadata: %w", err)
	}
	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not fetch metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("abnormal error code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read metadata response: %w", err)
	}

	var tokenResponse TokenResponse
	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		return nil, fmt.Errorf("could not parse JSON response: %w", err)
	}

	return &tokenResponse, nil
}
