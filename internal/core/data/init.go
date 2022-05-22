package data

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

func Initialize(dataSource string, debug bool) error {
	var err error
	// By default only log errors but enable full SQL query prints-to-console with debug mode
	log := logger.Default.LogMode(logger.Error)
	if debug {
		log = logger.Default.LogMode(logger.Info)
	}
	db, err = gorm.Open(postgres.Open(dataSource), &gorm.Config{Logger: log})

	if err != nil {
		return fmt.Errorf("error connecting to database: %s", err)
	}

	err = db.AutoMigrate(&Account{}, &PlayerOptions{}, &Character{}, &GuildcardEntry{})
	if err != nil {
		return fmt.Errorf("error auto migrating db: %s", err)
	}

	return nil
}

func Shutdown() error {
	database, err := db.DB()
	if err != nil {
		return fmt.Errorf("error while getting current connection: %w", err)
	}
	if err := database.Close(); err != nil {
		return fmt.Errorf("error while closing database connection: %w", err)
	}
	return nil
}
