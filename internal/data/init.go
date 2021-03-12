package data

import (
	"fmt"

	"github.com/dcrodman/archon"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func Initialize(dataSource string) error {
	var err error
	db, err = gorm.Open(postgres.Open(dataSource))

	if err != nil {
		return fmt.Errorf("failed to connect to database: %s", err)
	}

	db.AutoMigrate(&Account{}, &PlayerOptions{}, &Character{}, &GuildcardEntry{})

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
