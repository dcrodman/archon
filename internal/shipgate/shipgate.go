package shipgate

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	ioutil "io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/dcrodman/archon/internal/core"
)

// NewRPCClient returns a Shipgate client initialized with the configured certificates.
func NewRPCClient(cfg *core.Config) Shipgate {
	// Load client cert
	cert, err := tls.LoadX509KeyPair(cfg.ShipgateCertFile, cfg.ShipgateServer.SSLKeyFile)
	if err != nil {
		log.Fatal(err)
	}

	// Load CA cert
	caCert, err := ioutil.ReadFile(cfg.ShipgateCertFile)
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Setup HTTPS client
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}
	tlsConfig.BuildNameToCertificate()
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	return NewShipgateProtobufClient(cfg.ShipgateAddress(), httpClient)
}

type Server struct {
	Config *core.Config
	Logger *logrus.Logger

	httpServer http.Server
}

func (s *Server) Start(ctx context.Context) {
	go func() {
		s.httpServer = http.Server{
			Addr: fmt.Sprintf(":%d", s.Config.ShipgateServer.Port),
			Handler: NewShipgateServer(&service{
				logger:         s.Logger,
				connectedShips: make(map[string]*ship),
			}),
		}

		if err := s.httpServer.ListenAndServeTLS(
			s.Config.ShipgateCertFile,
			s.Config.ShipgateServer.SSLKeyFile,
		); err != nil {
			s.Logger.Errorf("[SHIPGATE] error: %v", err)
		}

		s.Logger.Printf("[SHIPGATE] exited")
	}()
}

func (s *Server) Shutdown(ctx context.Context) {
	// Gracefully shut down the RPC server once we've received the server-wide shutdown signal.
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, time.Minute)
	_ = s.httpServer.Shutdown(shutdownCtx)
	shutdownCancel()
}
