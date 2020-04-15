package shipgate

import (
	"crypto/tls"
	"fmt"
	"github.com/dcrodman/archon/server/shipgate/api"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
	"net"
)

// StartAPIServer starts the gRPC API server on the specified address.
func StartAPIServer(addr string) error {
	cert, err := loadX509Certificate()
	if err != nil {
		return err
	}

	creds := credentials.NewServerTLSFromCert(cert)
	opts := []grpc.ServerOption{grpc.Creds(creds)}
	grpcServer := grpc.NewServer(opts...)

	s := shipServiceServer{}
	api.RegisterShipServiceServer(grpcServer, &s)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start shigate on %s: %s\n", addr, err)
	}

	if err := grpcServer.Serve(l); err != nil {
		return fmt.Errorf("failed to start shipgate on %s: %s\n", addr, err)
	}

	return nil
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
