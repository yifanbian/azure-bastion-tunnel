package main

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

const armScope = "https://management.azure.com/.default"

func getAccessToken(ctx context.Context, cred azcore.TokenCredential) (string, error) {
	token, err := cred.GetToken(ctx, policy.TokenRequestOptions{Scopes: []string{armScope}})
	if err != nil {
		return "", fmt.Errorf("get Azure access token with AzureCliCredential: %w", err)
	}
	return token.Token, nil
}
