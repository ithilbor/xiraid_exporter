package xiraid_grpc

import (
	// Go
	"context"
	"log/slog"
	"os"
	"sync"
	"time"
	// Xiraid exporter
	xrprotos "github.com/ironcub3/xiraid_exporter/protos"
	// Kingpin
	"github.com/alecthomas/kingpin/v2"
	// Grpc
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Variable used inside the function xrConnection
var (
	clientOnce sync.Once
	xrClient   xrprotos.XRAIDServiceClient
)

// Cmd variables to init the connection with the Xinnor Server
var (
	xrSrvHostname = kingpin.Flag(
		"xiraid-srv-hostname",
		"The hostname of the server where xiraid runs.",
	).Default("localhost").String()
	xrSrvPort = kingpin.Flag(
		"xiraid-srv-port",
		"The port of the xiraid server.",
	).Default("6066").String()
	xrCertPath = kingpin.Flag(
		"xiraid-cert-path",
		"Path under which to find the xiraid tls certificate.",
	).Default("/etc/xraid/crt/server-cert.crt").String()
)

// Start a connection for the Xinnor client only one time
func NewXiraidClient (logger *slog.Logger) (xrprotos.XRAIDServiceClient) {
	clientOnce.Do(func() {
		creds, err := credentials.NewClientTLSFromFile(*xrCertPath, "")
		if err != nil {
			logger.Error("Failed to create TLS credentials: %v", "error", err.Error())
			os.Exit(1)
		}
		grpcClient, err := grpc.NewClient(*xrSrvHostname + ":" + *xrSrvPort, grpc.WithTransportCredentials(creds))
		if err != nil {
			logger.Error("Failed to connect to xiraid grpc server: %v", "error", err.Error())
			os.Exit(1)
		}
		xrClient = xrprotos.NewXRAIDServiceClient(grpcClient)
		// Making a call to the xiraid client to make sure
		// that the connection is well configured
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		configShowRequest := &xrprotos.LicenseShow{}
		_, err = xrClient.LicenseShow(ctx, configShowRequest)
		target := grpcClient.Target()
		if err != nil {
			logger.Error("Failed to connect to xiraid grpc server!", "error", err.Error())
			logger.Error(
				"Check that you can reach the xiraid server target and " +
				"you're using the correct xiraid hostname and port.")
			logger.Info("The used target is:", "target", target)
			os.Exit(1)
		} else {
			logger.Info("Successfully connected to the Xiraid gRPC server at", "address", target)
		}
	})
	return xrClient
}