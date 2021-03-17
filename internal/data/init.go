package data

import (
	"fmt"

	"gorm.io/gorm/logger"

	"github.com/dcrodman/archon"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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
		return fmt.Errorf("failed to connect to database: %s", err)
	}

	err = db.AutoMigrate(&Account{}, &PlayerOptions{}, &Character{}, &GuildcardEntry{})
	if err != nil {
		return fmt.Errorf("unable to auto migrate db: %s", err)
	}

	return nil
}

func Shutdown() {
	database, err := db.DB()
	if err != nil {
		archon.Log.Error("error while getting current connection: ", err)
	}
	if err := database.Close(); err != nil {
		archon.Log.Error("error while closing database connection: ", err)
	}
}
