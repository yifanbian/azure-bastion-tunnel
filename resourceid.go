package main

import (
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
)

func validateTypedResourceID(resourceID string, argumentName string, provider string, resourceTypes ...string) error {
	parsed, err := parseScopedResourceID(resourceID)
	if err != nil {
		return fmt.Errorf("invalid %s: %w", argumentName, err)
	}
	if !matchesResourceType(parsed, provider, resourceTypes...) {
		return fmt.Errorf("invalid %s: expected %s", argumentName, resourceIDPattern(provider, resourceTypes...))
	}
	return nil
}

func validateTargetResourceID(resourceID string) error {
	parsed, err := parseScopedResourceID(resourceID)
	if err != nil {
		return fmt.Errorf("invalid vmResourceId: %w", err)
	}
	if matchesResourceType(parsed, "Microsoft.Compute", "virtualMachines") ||
		matchesResourceType(parsed, "Microsoft.Compute", "virtualMachineScaleSets", "virtualMachines") {
		return nil
	}
	return fmt.Errorf(
		"invalid vmResourceId: expected %s or %s",
		resourceIDPattern("Microsoft.Compute", "virtualMachines"),
		resourceIDPattern("Microsoft.Compute", "virtualMachineScaleSets", "virtualMachines"),
	)
}

func parseScopedResourceID(resourceID string) (*arm.ResourceID, error) {
	if resourceID != strings.TrimSpace(resourceID) {
		return nil, fmt.Errorf("resource ID contains leading or trailing whitespace")
	}

	parsed, err := arm.ParseResourceID(resourceID)
	if err != nil {
		return nil, err
	}
	if parsed.SubscriptionID == "" || parsed.ResourceGroupName == "" || parsed.Name == "" {
		return nil, fmt.Errorf("resource ID must be a resource group scoped resource")
	}
	return parsed, nil
}

func matchesResourceType(resourceID *arm.ResourceID, provider string, resourceTypes ...string) bool {
	if !strings.EqualFold(resourceID.ResourceType.Namespace, provider) || len(resourceID.ResourceType.Types) != len(resourceTypes) {
		return false
	}
	for i, resourceType := range resourceTypes {
		if !strings.EqualFold(resourceID.ResourceType.Types[i], resourceType) {
			return false
		}
	}
	return true
}

func resourceIDPattern(provider string, resourceTypes ...string) string {
	var builder strings.Builder
	builder.WriteString("/subscriptions/<subscriptionId>/resourceGroups/<resourceGroupName>/providers/")
	builder.WriteString(provider)
	for _, resourceType := range resourceTypes {
		builder.WriteString("/")
		builder.WriteString(resourceType)
		builder.WriteString("/<name>")
	}
	return builder.String()
}
