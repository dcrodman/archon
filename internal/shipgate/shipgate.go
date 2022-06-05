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
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dcrodman/archon/internal/core"
	"github.com/dcrodman/archon/internal/core/data"
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

	db         *gorm.DB
	httpServer http.Server
}

func (s *Server) Start(ctx context.Context) {
	go func() {
		// Connect to the database.
		if err := s.initDatabase(); err != nil {
			s.Logger.Errorf("error initializing database connection: %v", err)
			return
		}
		s.Logger.Infof("connected to database %s:%d", s.Config.Database.Host, s.Config.Database.Port)

		// Set up and start the HTTP handler for handling the RPC requests.
		s.httpServer = http.Server{
			Addr: fmt.Sprintf(":%d", s.Config.ShipgateServer.Port),
			Handler: NewShipgateServer(&service{
				logger:         s.Logger,
				db:             s.db,
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

func (s *Server) initDatabase() error {
	var err error
	// By default only log errors but enable full SQL query prints-to-console with debug mode
	log := logger.Default.LogMode(logger.Error)
	if s.Config.Debugging.DatabaseLoggingEnabled {
		log = logger.Default.LogMode(logger.Info)
	}

	s.db, err = gorm.Open(postgres.Open(s.Config.DatabaseURL()), &gorm.Config{Logger: log})
	if err != nil {
		return fmt.Errorf("error connecting to database: %s", err)
	}

	if err = s.db.AutoMigrate(
		&data.Account{},
		&data.PlayerOptions{},
		&data.Character{},
		&data.GuildcardEntry{},
	); err != nil {
		return fmt.Errorf("error auto migrating db: %s", err)
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) {
	database, err := s.db.DB()
	if err != nil {
		s.Logger.Errorf("error while getting current connection: %v", err)
	} else {
		if err := database.Close(); err != nil {
			s.Logger.Errorf("error while closing database connection: %v", err)
		}
	}

	// Gracefully shut down the RPC server once we've received the server-wide shutdown signal.
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, time.Minute)
	_ = s.httpServer.Shutdown(shutdownCtx)
	shutdownCancel()
}
