package shipgate

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dcrodman/archon/internal/core"
	"github.com/dcrodman/archon/internal/core/data"
)

func NewRPCClient(cfg *core.Config) Shipgate {
	return NewShipgateProtobufClient(cfg.ShipgateAddress(), http.DefaultClient)
}

type Server struct {
	Config *core.Config
	Logger *zap.SugaredLogger

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
		s.Logger.Infof("[SHIPGATE] connected to database %s", s.db.Name())

		// Set up and start the HTTP handler for handling the RPC requests.
		s.httpServer = http.Server{
			Addr: fmt.Sprintf(":%d", s.Config.ShipgateServer.Port),
			Handler: NewShipgateServer(&service{
				logger:         s.Logger,
				db:             s.db,
				connectedShips: make(map[string]*ship),
			}),
		}

		if err := s.httpServer.ListenAndServe(); err != nil {
			s.Logger.Errorf("[SHIPGATE] error: %v", err)
		}
		s.Logger.Infof("[SHIPGATE] exited")
	}()
}

func (s *Server) initDatabase() error {
	var err error
	// By default only log errors but enable full SQL query prints-to-console with debug mode
	log := logger.Default.LogMode(logger.Silent)
	if s.Config.Debugging.DatabaseLoggingEnabled {
		log = logger.Default.LogMode(logger.Info)
	}

	var dialector gorm.Dialector
	switch strings.ToLower(s.Config.Database.Engine) {
	case "sqlite":
		dialector = sqlite.Open("archon.db")
	case "postgres":
		dialector = postgres.Open(s.Config.DatabaseURL())
	default:
		return fmt.Errorf("unsupported database engine: %s", s.Config.Database.Engine)
	}

	s.db, err = gorm.Open(dialector, &gorm.Config{Logger: log})
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
	if s.db != nil {
		database, err := s.db.DB()
		if err != nil {
			s.Logger.Errorf("error while getting current connection: %v", err)
		} else {
			if err := database.Close(); err != nil {
				s.Logger.Errorf("error while closing database connection: %v", err)
			}
		}
	}

	// Gracefully shut down the RPC server once we've received the server-wide shutdown signal.
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, time.Minute)
	_ = s.httpServer.Shutdown(shutdownCtx)
	shutdownCancel()
}
