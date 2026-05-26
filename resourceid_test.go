package main

import "testing"

func TestValidateTypedResourceID(t *testing.T) {
	tests := []struct {
		name         string
		resourceID   string
		provider     string
		resourceType string
		wantErr      bool
	}{
		{
			name:         "valid bastion ID",
			resourceID:   "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Network/bastionHosts/bastion1",
			provider:     "Microsoft.Network",
			resourceType: "bastionHosts",
		},
		{
			name:         "valid VM ID case insensitive",
			resourceID:   "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/rg/providers/microsoft.compute/virtualmachines/vm1",
			provider:     "Microsoft.Compute",
			resourceType: "virtualMachines",
		},
		{
			name:         "wrong provider",
			resourceID:   "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Network/virtualMachines/vm1",
			provider:     "Microsoft.Compute",
			resourceType: "virtualMachines",
			wantErr:      true,
		},
		{
			name:         "missing leading slash",
			resourceID:   "subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/vm1",
			provider:     "Microsoft.Compute",
			resourceType: "virtualMachines",
			wantErr:      true,
		},
		{
			name:         "extra segment",
			resourceID:   "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/vm1/extensions/ext1",
			provider:     "Microsoft.Compute",
			resourceType: "virtualMachines",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTypedResourceID(tt.resourceID, "resourceId", tt.provider, tt.resourceType)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}

func TestValidateTargetResourceID(t *testing.T) {
	tests := []struct {
		name       string
		resourceID string
		wantErr    bool
	}{
		{
			name:       "valid standalone VM",
			resourceID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/vm1",
		},
		{
			name:       "valid VMSS virtualMachines instance",
			resourceID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Compute/virtualMachineScaleSets/vmss1/virtualMachines/0",
		},
		{
			name:       "VMSS instances form is not supported",
			resourceID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Compute/virtualMachineScaleSets/vmss1/instances/0",
			wantErr:    true,
		},
		{
			name:       "wrong provider",
			resourceID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Network/virtualMachines/vm1",
			wantErr:    true,
		},
		{
			name:       "VM child resource is not target",
			resourceID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/vm1/extensions/ext1",
			wantErr:    true,
		},
		{
			name:       "VMSS parent is not target instance",
			resourceID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Compute/virtualMachineScaleSets/vmss1",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTargetResourceID(tt.resourceID)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}
