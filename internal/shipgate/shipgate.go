package shipgate

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/dcrodman/archon/internal/shipgate/api"
)

// Start starts the gRPC API servers listening on addr.
func Start(ctx context.Context, logger *logrus.Logger, addr string, readyChan chan bool, errChan chan error) {
	cert, err := loadX509Certificate()
	if err != nil {
		errChan <- err
		return
	}

	creds := credentials.NewServerTLSFromCert(cert)
	opts := []grpc.ServerOption{grpc.Creds(creds)}
	grpcServer := grpc.NewServer(opts...)

	api.RegisterShipgateServiceServer(grpcServer, &shipgateServiceServer{
		logger:         logger,
		connectedShips: make(map[string]*ship),
	})

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		errChan <- fmt.Errorf("error starting ship info service listener on %s: %s", addr, err)
		return
	}

	// Spin off the listener in its own goroutine since we need to listen for context cancellations.
	go func() {
		logger.Printf("SHIPGATE waiting for requests on %s", addr)

		if err := grpcServer.Serve(listener); err != nil {
			errChan <- fmt.Errorf("error starting ship info service on %s: %s", addr, err)
			return
		}

		close(errChan)
	}()

	readyChan <- true
	<-ctx.Done()

	grpcServer.GracefulStop()
	logger.Printf("SHIPGATE server exited")
}

func loadX509Certificate() (*tls.Certificate, error) {
	certFile, err := ioutil.ReadFile(viper.GetString("shipgate_certificate_file"))
	if err != nil {
		return nil, fmt.Errorf("error loading certificate file: %s", err)
	}

	keyFile, err := ioutil.ReadFile(viper.GetString("shipgate_server.ssl_key_file"))
	if err != nil {
		return nil, fmt.Errorf("error loading key file: %s", err)
	}

	cert, err := tls.X509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("error loading X.509 certificate: %s", err)
	}

	return &cert, nil
}
