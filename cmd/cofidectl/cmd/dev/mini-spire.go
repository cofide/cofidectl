// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package dev

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/cofide/cofidectl/internal/pkg/dev/minispire"
	"github.com/spf13/cobra"
	pb "github.com/spiffe/go-spiffe/v2/proto/spiffe/workload"
	"google.golang.org/grpc"
)

var devMiniSpireDesc = `
This command will bring up a local SPIRE workload API socket that sets up a local development CA and issues SVIDs to every client connecting to it.
THIS COMMAND SHOULD NEVER BE USED IN ANY PRODUCTION OR SERVER ENVIRONMENT.
`

type devMiniSpireOpts struct {
	socket  string
	domain  string
	keyType string
}

func devMiniSpireCmd() *cobra.Command {
	opts := devMiniSpireOpts{}

	cmd := &cobra.Command{
		Use:   "mini-spire [ARGS]",
		Short: "Sets up a SPIRE agent for local development",
		Long:  devMiniSpireDesc,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Building in-memory CA")
			var kt minispire.KeyType
			if opts.keyType == "rsa" {
				kt = minispire.KeyTypeRSA
			} else if opts.keyType == "ecdsa" {
				kt = minispire.KeyTypeECDSAP256
			} else {
				return fmt.Errorf("key type %q is unknown", opts.keyType)
			}
			ca, err := minispire.NewInMemoryCA(kt)
			if err != nil {
				return fmt.Errorf("failed to create in-memory CA: %v", err)
			}

			fmt.Println("Starting SPIRE server")
			lis, err := net.Listen("unix", opts.socket)
			if err != nil {
				return fmt.Errorf("failed to listen in %q: %v", opts.socket, err)
			}

			grpcServer := grpc.NewServer(grpc.Creds(minispire.NewCredentials()))
			wl := minispire.NewWorkloadHandler(minispire.Config{
				Domain: opts.domain,
				CA:     ca,
			})
			pb.RegisterSpiffeWorkloadAPIServer(grpcServer, wl)

			go func() {
				fmt.Println("SPIRE server listening on", opts.socket)
				err := grpcServer.Serve(lis)
				if err != nil {
					fmt.Println("SPIRE server failed", err)
				}
			}()

			// listen for signals to stop the server
			osSignals := make(chan os.Signal, 1)
			signal.Notify(osSignals, syscall.SIGINT, syscall.SIGTERM)
			<-osSignals

			fmt.Println("Shutting down server")
			lis.Close()

			return nil
		},
	}

	f := cmd.Flags()
	f.StringVarP(&opts.domain, "domain", "d", "example.com", "Trust domain to use for this trust zone")
	f.StringVarP(&opts.socket, "socket", "s", "/tmp/spire.sock", "Path to the UNIX socket to listen on")
	f.StringVarP(&opts.keyType, "key-type", "k", "rsa", "Key type to use for the CA (rsa or ecdsa)")

	return cmd
}
