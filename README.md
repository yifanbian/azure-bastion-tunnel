# azure-bastion-tunnel

`azure-bastion-tunnel` is an OpenSSH `ProxyCommand` helper for connecting to Azure VMs through Azure Bastion native client tunneling. It uses `AzureCliCredential`, so sign in with Azure CLI before using it:

```sh
az login
```

## Why use this instead of `az network bastion ssh`

This tool is designed to fit into normal OpenSSH workflows:

- It works as a `ProxyCommand`, so you can keep per-host settings in `~/.ssh/config` and connect with plain `ssh <host>`.
- It lets OpenSSH own the SSH session completely, including `~/.ssh/config` options, identity selection, agent forwarding, port forwarding, multiplexing, and tools that shell out to `ssh`.
- It does not need to open a separate local listening port before starting SSH. Each SSH connection is proxied directly over stdin/stdout to the Bastion WebSocket.
- It is easy to reuse from tools that understand SSH config, such as `scp`, `rsync`, Git over SSH, Ansible, and IDE remote-SSH integrations.

## SSH config

Add a `Host` entry to your OpenSSH config file, for example `~/.ssh/config`:

```sshconfig
Host my-vm-through-bastion
  User azureuser
  Port 22
  ProxyCommand /path/to/azure-bastion-tunnel -b "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-bastion/providers/Microsoft.Network/bastionHosts/my-bastion" -v "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-vm/providers/Microsoft.Compute/virtualMachines/my-vm" -p %p

Host my-vmss-instance-through-bastion
  User azureuser
  Port 22
  ProxyCommand /usr/local/bin/azure-bastion-tunnel -b "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-bastion/providers/Microsoft.Network/bastionHosts/my-bastion" -v "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-vmss/providers/Microsoft.Compute/virtualMachineScaleSets/my-vmss/virtualMachines/0" -p %p
```

`%p` is expanded by OpenSSH to the `Port` value from the host entry. You can also hard-code `-p 22` if you prefer.

Then connect normally:

```sh
ssh my-vm-through-bastion
ssh my-vmss-instance-through-bastion
```

## Supported target resource IDs

`--vm-resource-id` supports:

```text
/subscriptions/<subscriptionId>/resourceGroups/<resourceGroupName>/providers/Microsoft.Compute/virtualMachines/<vmName>
/subscriptions/<subscriptionId>/resourceGroups/<resourceGroupName>/providers/Microsoft.Compute/virtualMachineScaleSets/<vmssName>/virtualMachines/<instanceId>
```

`--bastion-resource-id` must be:

```text
/subscriptions/<subscriptionId>/resourceGroups/<resourceGroupName>/providers/Microsoft.Network/bastionHosts/<bastionName>
```

## Credit

This project references the implementation from [Azure/azure-cli-extensions/bastion](https://github.com/Azure/azure-cli-extensions/tree/main/src/bastion/azext_bastion).
