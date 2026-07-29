package shipgate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dcrodman/archon/internal/core"
	"github.com/dcrodman/archon/internal/core/data"
	"github.com/dcrodman/archon/internal/core/proto"
	"github.com/glebarez/sqlite"
)

// There should only ever be one instance of the shipgate, so treat it as a global with
// package-level lifecycle functions to manage it.
var (
	shipgateService *service
	shipgate        *http.Server
)

// NewClient returns an RPC client for the shipgate, connected to the address defined in
// the config file.
//
// Since Archon does not (yet) support running independent ship servers and thus the shipgate
// listen address will always be the same its connect address, this function assumes as
// much and reuses the same address/port.
//
// If and/or when Archon *does* support that, this connection will need to be updated with
// mTLS or some other authentication mechanism.
func NewClient(cfg *core.Config) Shipgate {
	return NewShipgateProtobufClient(cfg.ShipgateAddress(), http.DefaultClient)
}

// Start starts the shipgate at the address defined in the config file.
func Start(cfg *core.Config, logger *zap.SugaredLogger) {
	if shipgate != nil {
		panic("shipgate has already been initialized")
	}

	// Connect to the database.
	db, err := initDatabase(cfg)
	if err != nil {
		logger.Errorf("[SHIPGATE] error initializing database connection: %v", err)
		return
	}
	logger.Infof("[SHIPGATE] connected to database %s", db.Name())

	// Set up the underlying service.
	shipgateService = &service{
		logger:         logger,
		db:             db,
		connectedShips: make(map[string]*ship),
	}
	// Set up and start the HTTP handler for handling the RPC requests.
	shipgate = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.ShipgateServer.Port),
		Handler: NewShipgateServer(shipgateService),
	}
	go func() {
		if err := shipgate.ListenAndServe(); err != nil {
			shipgateService.logger.Errorf("[SHIPGATE] error: %v", err)
		}
		shipgateService.logger.Infof("[SHIPGATE] exited")
	}()
}

func initDatabase(cfg *core.Config) (*gorm.DB, error) {
	var err error
	// By default only log errors but enable full SQL query prints-to-console with debug mode
	log := logger.Default.LogMode(logger.Silent)
	if cfg.Debugging.DatabaseLoggingEnabled {
		log = logger.Default.LogMode(logger.Info)
	}

	var dialector gorm.Dialector
	switch strings.ToLower(cfg.Database.Engine) {
	case "sqlite":
		dialector = sqlite.Open(cfg.QualifiedPath(cfg.Database.Filename))
	case "postgres":
		dialector = postgres.Open(cfg.DatabaseURL())
	default:
		return nil, fmt.Errorf("unsupported database engine: %s", cfg.Database.Engine)
	}

	db, err := gorm.Open(dialector, &gorm.Config{Logger: log})
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %s", err)
	}

	if err = db.AutoMigrate(
		&data.Account{},
		&data.PlayerOptions{},
		&data.Character{},
		&data.GuildcardEntry{},
	); err != nil {
		return nil, fmt.Errorf("error auto migrating db: %s", err)
	}
	return db, nil
}

// Shutdown gracefully closes all connections to the shipgate, shuts the server down, and
// closes any external connections.
func Shutdown(ctx context.Context) {
	if shipgate == nil {
		return
	}

	// Gracefully shut down the RPC server once we've received the server-wide shutdown signal.
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, time.Minute)
	_ = shipgate.Shutdown(shutdownCtx)
	shutdownCancel()

	// Close the DB connection once we're no longer handling requests.
	if db, _ := shipgateService.db.DB(); db != nil {
		db.Close()
	}
}

// Service implements the shipgate server logic, which acts as the data and coordination
// layer between the other server components. It never directly interacts with the client,
// only handling RPC requests from other trusted servers.
type service struct {
	config *core.Config
	logger *zap.SugaredLogger

	db                  *gorm.DB
	connectedShips      map[string]*ship
	connectedShipsMutex sync.RWMutex
}

var (
	ErrUnknown            = errors.New("an unexpected error occurred, please contact your server administrator")
	ErrInvalidCredentials = errors.New("username/combination password not found")
	ErrAccountBanned      = errors.New("this account has been suspended")
)

func (s *service) AuthenticateAccount(ctx context.Context, req *AuthenticateAccountRequest) (*proto.Account, error) {
	s.logger.Debug("AuthenticateAccount")
	account, err := data.FindAccountByUsername(s.db, req.Username)
	if err != nil {
		return nil, ErrUnknown
	}

	if account == nil || account.Password != HashPassword(req.Password) {
		return nil, ErrInvalidCredentials
	} else if account.Banned {
		return nil, ErrAccountBanned
	}

	return &proto.Account{
		Id:               uint64(account.ID),
		Username:         account.Username,
		Email:            account.Email,
		RegistrationDate: account.RegistrationDate.Format(time.RFC3339),
		Guildcard:        uint64(account.Guildcard),
		Gm:               account.GM,
		Banned:           account.Banned,
		Active:           account.Active,
		TeamId:           int64(account.TeamID),
		PrivilegeLevel:   []byte{account.PrivilegeLevel},
	}, nil
}

// HashPassword returns a version of password with Archon's chosen hashing strategy.
func HashPassword(password string) string {
	hash := sha256.New()
	if _, err := hash.Write(stripPadding([]byte(password))); err != nil {
		panic(fmt.Errorf("error generating password hash: %v", err))
	}
	return hex.EncodeToString(hash.Sum(nil)[:])
}

func stripPadding(b []byte) []byte {
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] != 0 {
			return b[:i+1]
		}
	}
	return b
}

func (s *service) GetPlayerOptions(ctx context.Context, req *GetPlayerOptionsRequest) (*GetPlayerOptionsResponse, error) {
	s.logger.Debug("GetPlayerOptions")

	playerOptions, err := data.FindPlayerOptions(s.db, req.AccountId)
	if err != nil {
		return nil, fmt.Errorf("error retrieving player options for account %d: %w", req.AccountId, err)
	}

	resp := &GetPlayerOptionsResponse{
		Exists:        false,
		PlayerOptions: &proto.PlayerOptions{},
	}
	if playerOptions != nil {
		resp.Exists = true
		resp.PlayerOptions = playerOptionsToProto(playerOptions)
	}
	return resp, nil
}

func (s *service) UpsertPlayerOptions(ctx context.Context, req *UpsertPlayerOptionsRequest) (*emptypb.Empty, error) {
	s.logger.Debug("UpsertPlayerOptions")

	playerOptions := playerOptionsFromProto(req.PlayerOptions)
	playerOptions.Account = &data.Account{
		ID: req.AccountId,
	}
	if err := data.CreatePlayerOptions(s.db, playerOptions); err != nil {
		return nil, fmt.Errorf("error creating player options: %v", err)
	}
	return &emptypb.Empty{}, nil
}
