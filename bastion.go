package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const bastionAPIVersion = "2024-01-01"

type bastionResource struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	SKU        sku    `json:"sku"`
	Properties struct {
		DNSName         string `json:"dnsName"`
		EnableTunneling *bool  `json:"enableTunneling"`
	} `json:"properties"`
}

type sku struct {
	Name string `json:"name"`
}

type tokenResponse struct {
	AuthToken      string `json:"authToken"`
	NodeID         string `json:"nodeId"`
	WebSocketToken string `json:"websocketToken"`
	Message        string `json:"message"`
}

func getBastion(ctx context.Context, client *http.Client, resourceID string, accessToken string) (bastionResource, error) {
	endpoint := "https://management.azure.com" + resourceID + "?api-version=" + url.QueryEscape(bastionAPIVersion)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return bastionResource{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	var bastion bastionResource
	if err := doJSON(client, req, &bastion); err != nil {
		return bastionResource{}, fmt.Errorf("get bastion resource: %w", err)
	}
	if bastion.Properties.DNSName == "" {
		return bastionResource{}, errors.New("bastion resource does not contain properties.dnsName")
	}
	if !isNativeClientEnabled(bastion) {
		return bastionResource{}, fmt.Errorf("bastion SKU %q must be Standard or Premium with enableTunneling=true, or Developer", bastion.SKU.Name)
	}
	return bastion, nil
}

func isNativeClientEnabled(bastion bastionResource) bool {
	if strings.EqualFold(bastion.SKU.Name, "Developer") {
		return true
	}
	if !strings.EqualFold(bastion.SKU.Name, "Standard") && !strings.EqualFold(bastion.SKU.Name, "Premium") {
		return false
	}
	return bastion.Properties.EnableTunneling != nil && *bastion.Properties.EnableTunneling
}

func getBastionEndpoint(ctx context.Context, client *http.Client, bastion bastionResource, vmResourceID string, resourcePort int, accessToken string) (string, error) {
	if strings.EqualFold(bastion.SKU.Name, "Developer") || strings.EqualFold(bastion.SKU.Name, "QuickConnect") {
		return getDataPod(ctx, client, bastion, vmResourceID, resourcePort, accessToken)
	}
	return bastion.Properties.DNSName, nil
}

func getDataPod(ctx context.Context, client *http.Client, bastion bastionResource, vmResourceID string, resourcePort int, accessToken string) (string, error) {
	body := map[string]any{
		"resourceId":        vmResourceID,
		"bastionResourceId": bastion.ID,
		"vmPort":            resourcePort,
		"azToken":           accessToken,
		"connectionType":    "nativeclient",
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://"+bastion.Properties.DNSName+"/api/connection", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	responseBody, err := doBytes(client, req)
	if err != nil {
		return "", fmt.Errorf("get bastion data pod endpoint: %w", err)
	}
	endpoint := strings.TrimSpace(string(responseBody))
	if endpoint == "" {
		return "", errors.New("empty bastion data pod endpoint")
	}
	return endpoint, nil
}

func createBastionToken(ctx context.Context, client *http.Client, endpoint string, vmResourceID string, resourcePort int, accessToken string, previousAuthToken string, nodeID string) (tokenResponse, error) {
	form := url.Values{}
	form.Set("resourceId", vmResourceID)
	form.Set("protocol", "tcptunnel")
	form.Set("workloadHostPort", strconv.Itoa(resourcePort))
	form.Set("aztoken", accessToken)
	if previousAuthToken != "" {
		form.Set("token", previousAuthToken)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://"+endpoint+"/api/tokens", strings.NewReader(form.Encode()))
	if err != nil {
		return tokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if nodeID != "" {
		req.Header.Set("X-Node-Id", nodeID)
	}

	var token tokenResponse
	if err := doJSON(client, req, &token); err != nil {
		return tokenResponse{}, fmt.Errorf("create bastion tunnel token: %w", err)
	}
	if token.AuthToken == "" || token.WebSocketToken == "" {
		return tokenResponse{}, errors.New("bastion token response missing authToken or websocketToken")
	}
	return token, nil
}

func deleteBastionToken(ctx context.Context, client *http.Client, endpoint string, authToken string, nodeID string) error {
	if authToken == "" {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, "https://"+endpoint+"/api/tokens/"+url.PathEscape(authToken), nil)
	if err != nil {
		return err
	}
	if nodeID != "" {
		req.Header.Set("X-Node-Id", nodeID)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Errorf("delete token returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
}

func buildWebSocketURL(endpoint string, skuName string, token tokenResponse) (string, error) {
	escapedToken := url.PathEscape(token.WebSocketToken)
	if strings.EqualFold(skuName, "Developer") || strings.EqualFold(skuName, "QuickConnect") {
		return "wss://" + endpoint + "/omni/webtunnel/" + escapedToken, nil
	}
	if token.NodeID == "" {
		return "", errors.New("bastion token response missing nodeId")
	}
	return "wss://" + endpoint + "/webtunnelv2/" + escapedToken + "?X-Node-Id=" + url.QueryEscape(token.NodeID), nil
}

func doJSON(client *http.Client, req *http.Request, out any) error {
	body, err := doBytes(client, req)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode JSON response: %w", err)
	}
	return nil
}

func doBytes(client *http.Client, req *http.Request) ([]byte, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s returned %s: %s", req.URL.Redacted(), resp.Status, strings.TrimSpace(string(body)))
	}
	return body, nil
}
