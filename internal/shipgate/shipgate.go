package shipgate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dcrodman/archon/internal/data"
)

var (
	ErrUnknown            = errors.New("an unexpected error occurred, please contact your server administrator")
	ErrInvalidCredentials = errors.New("username/combination password not found")
	ErrAccountBanned      = errors.New("this account has been suspended")
)

// Shipgate is a singleton instance of ShipgateService.
var Shipgate *ShipgateService

type DBConfig struct {
	Engine         string
	Filename       string
	URL            string
	LoggingEnabled bool
}

// Init starts the shipgate at the address defined in the config file.
func Init(cfg DBConfig, logger *zap.SugaredLogger) {
	if Shipgate != nil {
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
	Shipgate = &ShipgateService{
		logger: logger,
		db:     db,
	}
}

func initDatabase(cfg DBConfig) (*gorm.DB, error) {
	var err error
	// By default only log errors but enable full SQL query prints-to-console with debug mode
	log := logger.Default.LogMode(logger.Silent)
	if cfg.LoggingEnabled {
		log = logger.Default.LogMode(logger.Info)
	}

	var dialector gorm.Dialector
	switch strings.ToLower(cfg.Engine) {
	case "sqlite":
		dialector = sqlite.Open(cfg.Filename)
	case "postgres":
		dialector = postgres.Open(cfg.URL)
	default:
		return nil, fmt.Errorf("unsupported database engine: %s", cfg.Engine)
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
	if Shipgate == nil {
		return
	}

	// Close the DB connection once we're no longer handling requests.
	if db, _ := Shipgate.db.DB(); db != nil {
		db.Close()
	}
}

// ShipgateService implements the shipgate server logic, which acts as the data and
// coordination layer between the other server components. It never directly interacts
// with the client, only handling RPC requests from other trusted servers.
type ShipgateService struct {
	logger *zap.SugaredLogger
	db     *gorm.DB

	ships    []data.Ship
	shipsMtx sync.RWMutex
}

// AuthenticateAccount verifies an account. A password should be provided
// via the rpc call metadata.
func (s *ShipgateService) AuthenticateAccount(ctx context.Context, username, password string) (*data.Account, error) {
	s.logger.Debug("AuthenticateAccount")
	account, err := findAccountByUsername(s.db, username)
	if err != nil {
		return nil, ErrUnknown
	}

	if account == nil || account.Password != HashPassword(password) {
		return nil, ErrInvalidCredentials
	} else if account.Banned {
		return nil, ErrAccountBanned
	}

	return account, nil
}

// HashPassword returns a version of password with Archon's chosen hashing strategy.
func HashPassword(password string) string {
	hash := sha256.New()
	if _, err := hash.Write(stripBytePadding([]byte(password))); err != nil {
		panic(fmt.Errorf("error generating password hash: %v", err))
	}
	return hex.EncodeToString(hash.Sum(nil)[:])
}

func stripBytePadding(b []byte) []byte {
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] != 0 {
			return b[:i+1]
		}
	}
	return b
}

// FindCharacter looks up character in a slot on an account.
func (s *ShipgateService) FindCharacter(ctx context.Context, accountID uint64, slot uint32) (*data.Character, error) {
	s.logger.Debug("FindCharacter")

	character, err := findCharacter(s.db, accountID, slot)
	if err != nil {
		return nil, fmt.Errorf("error retrieving character for account %d slot %d: %w", accountID, slot, err)
	}
	return character, nil
}

// UpsertCharacter creates a new character in a slot on an account.
func (s *ShipgateService) UpsertCharacter(ctx context.Context, accountID uint64, c *data.Character) error {
	s.logger.Debug("UpsertCharacter")

	c.AccountID = uint64(accountID)
	if err := upsertCharacter(s.db, c); err != nil {
		return fmt.Errorf("error updating character for account %d slot %d: %w", accountID, c.Slot, err)
	}
	return nil
}

// DeleteCharacter deletes the character data in a slot on an account.
func (s *ShipgateService) DeleteCharacter(ctx context.Context, accountID uint64, slot uint32) error {
	s.logger.Debug("DeleteCharacter")

	if err := deleteCharacter(s.db, accountID, slot); err != nil {
		return fmt.Errorf("error deleting character for account %d slot %d: %w", accountID, slot, err)
	}
	return nil
}

// GetGuildcardEntires returns the list of guildcards on an account.
func (s *ShipgateService) GetGuildcardEntries(ctx context.Context, accountID uint64) ([]data.GuildcardEntry, error) {
	s.logger.Debug("GetGuildcardEntries")

	entries, err := findGuildcardEntries(s.db, uint64(accountID))
	if err != nil {
		return nil, fmt.Errorf("error retrieving guildcard entries for account %d: %w", accountID, err)
	}
	return entries, nil
}

// GetPlayerOptions returns the player options tied to an account.
func (s *ShipgateService) GetPlayerOptions(ctx context.Context, accountID uint64) (*data.PlayerOptions, error) {
	s.logger.Debug("GetPlayerOptions")

	playerOptions, err := findPlayerOptions(s.db, accountID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving player options for account %d: %w", accountID, err)
	}
	return playerOptions, nil
}

// GetPlayerOptions updates or creates the player options tied to an account.
func (s *ShipgateService) UpsertPlayerOptions(ctx context.Context, accountID uint64, po *data.PlayerOptions) error {
	s.logger.Debug("UpsertPlayerOptions")

	po.Account = &data.Account{ID: accountID}
	if err := createPlayerOptions(s.db, po); err != nil {
		return fmt.Errorf("error creating player options: %v", err)
	}
	return nil
}

// RegisterShip adds a new Ship (i.e. game server) to its registry to indicate that players may join it.
func (s *ShipgateService) RegisterShip(ctx context.Context, ship data.Ship) {
	s.logger.Debug("RegisterShip")

	s.shipsMtx.Lock()
	defer s.shipsMtx.Unlock()
	s.ships = append(s.ships, ship)
}

// GetAvailableShips returns a view of all known ships currently registered and active.
func (s *ShipgateService) GetAvailableShips(ctx context.Context) []data.Ship {
	s.shipsMtx.RLock()
	defer s.shipsMtx.RUnlock()
	return s.ships
}
