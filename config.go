package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
)

type config struct {
	bastionResourceID string
	vmResourceID      string
	resourcePort      int
	insecureTLS       bool
}

func parseFlags(args []string) (config, error) {
	var cfg config

	fs := flag.NewFlagSet(programName, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&cfg.bastionResourceID, "bastion-resource-id", "", "Azure Bastion resource ID")
	fs.StringVar(&cfg.bastionResourceID, "b", "", "Azure Bastion resource ID")
	fs.StringVar(&cfg.vmResourceID, "vm-resource-id", "", "target VM resource ID")
	fs.StringVar(&cfg.vmResourceID, "v", "", "target VM resource ID")
	fs.IntVar(&cfg.resourcePort, "resource-port", 22, "target resource port")
	fs.IntVar(&cfg.resourcePort, "p", 22, "target resource port")
	fs.BoolVar(&cfg.insecureTLS, "insecure-skip-verify", false, "skip TLS certificate verification")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return config{}, err
		}
		return config{}, usageError(err)
	}
	if cfg.bastionResourceID == "" {
		return config{}, usageError(errors.New("missing --bastion-resource-id/-b"))
	}
	if cfg.vmResourceID == "" {
		return config{}, usageError(errors.New("missing --vm-resource-id/-v"))
	}
	if err := validateTypedResourceID(cfg.bastionResourceID, "bastionResourceId", "Microsoft.Network", "bastionHosts"); err != nil {
		return config{}, usageError(err)
	}
	if err := validateTargetResourceID(cfg.vmResourceID); err != nil {
		return config{}, usageError(err)
	}
	if cfg.resourcePort <= 0 || cfg.resourcePort > 65535 {
		return config{}, usageError(fmt.Errorf("invalid --resource-port/-p %d", cfg.resourcePort))
	}

	return cfg, nil
}

func usageText() string {
	return fmt.Sprintf("usage: %s -b <bastionResourceId> -v <vmResourceId> -p <resourcePort>", programName)
}

func usageError(err error) error {
	return fmt.Errorf("%w\n%s", err, usageText())
}
