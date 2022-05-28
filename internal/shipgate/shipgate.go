package shipgate

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/dcrodman/archon/internal/core"
)

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
