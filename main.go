package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

var programName = os.Args[0]

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(os.Stderr, usageText())
			return
		}
		fmt.Fprintf(os.Stderr, "%s: %v\n", programName, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	cfg, err := parseFlags(args)
	if err != nil {
		return err
	}

	cred, err := azidentity.NewAzureCLICredential(nil)
	if err != nil {
		return fmt.Errorf("create Azure CLI credential: %w", err)
	}

	armToken, err := getAccessToken(ctx, cred)
	if err != nil {
		return err
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
	bastion, err := getBastion(ctx, httpClient, cfg.bastionResourceID, armToken)
	if err != nil {
		return err
	}

	endpoint, err := getBastionEndpoint(ctx, httpClient, bastion, cfg.vmResourceID, cfg.resourcePort, armToken)
	if err != nil {
		return err
	}

	token, err := createBastionToken(ctx, httpClient, endpoint, cfg.vmResourceID, cfg.resourcePort, armToken, "", "")
	if err != nil {
		return err
	}
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := deleteBastionToken(cleanupCtx, httpClient, endpoint, token.AuthToken, token.NodeID); err != nil {
			fmt.Fprintf(os.Stderr, "%s: cleanup failed: %v\n", programName, err)
		}
	}()

	wsURL, err := buildWebSocketURL(endpoint, bastion.SKU.Name, token)
	if err != nil {
		return err
	}

	conn, _, err := newWebSocketDialer(cfg.insecureTLS).DialContext(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("connect websocket %s: %w", wsURL, err)
	}
	defer conn.Close()

	return proxyStdio(conn, os.Stdin, os.Stdout)
}
