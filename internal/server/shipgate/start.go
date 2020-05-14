package shipgate

import (
	"crypto/tls"
	"fmt"
	"github.com/dcrodman/archon/internal/server/shipgate/api"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
	"net"
)

// Start starts the gRPC API server on the specified address.
func Start(metaAddr string, shipAddr string) error {
	cert, err := loadX509Certificate()
	if err != nil {
		return err
	}

	creds := credentials.NewServerTLSFromCert(cert)
	opts := []grpc.ServerOption{grpc.Creds(creds)}

	mec := startMetadataService(metaAddr, opts)
	sec := startShipService(shipAddr, opts)

	select {
	case err := <-mec:
		return err
	case err := <-sec:
		return err
	}
}

func loadX509Certificate() (*tls.Certificate, error) {
	certFile, err := ioutil.ReadFile(viper.GetString("shipgate_certificate_file"))
	if err != nil {
		return nil, fmt.Errorf("unable to load certificate file: %s", err)
	}

	keyFile, err := ioutil.ReadFile(viper.GetString("shipgate_server.ssl_key_file"))
	if err != nil {
		return nil, fmt.Errorf("unable to load key file: %s", err)
	}

	cert, err := tls.X509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("unable to load X.509 certificate: %s", err)
	}

	return &cert, nil
}

func startMetadataService(addr string, opts []grpc.ServerOption) <-chan error {
	errChan := make(chan error)

	go func() {
		grpcServer := grpc.NewServer(opts...)
		s := shipMetadataServiceServer{}
		api.RegisterShipMetadataServiceServer(grpcServer, &s)

		l, err := net.Listen("tcp", addr)
		if err != nil {
			errChan <- fmt.Errorf("failed to start shigate metadata service on %s: %s\n", addr, err)
		}

		fmt.Printf("waiting for ShipMetadataService requests on %s\n", addr)

		if err := grpcServer.Serve(l); err != nil {
			errChan <- fmt.Errorf("failed to start shipgate metadata service on %s: %s\n", addr, err)
		}

		close(errChan)
	}()

	return errChan
}

func startShipService(addr string, opts []grpc.ServerOption) <-chan error {
	errChan := make(chan error)

	go func() {
		grpcServer := grpc.NewServer(opts...)
		s := shipgateServiceServer{}
		api.RegisterShipgateServiceServer(grpcServer, &s)

		l, err := net.Listen("tcp", addr)
		if err != nil {
			errChan <- fmt.Errorf("failed to start shigate ship service on %s: %s\n", addr, err)
		}

		fmt.Printf("waiting for ShipgateService requests on %s\n", addr)

		if err := grpcServer.Serve(l); err != nil {
			errChan <- fmt.Errorf("failed to start shipgate ship service on %s: %s\n", addr, err)
		}

		close(errChan)
	}()

	return errChan
}
