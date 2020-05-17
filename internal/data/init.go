package data

import (
	"fmt"

	"github.com/dcrodman/archon"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

var db *gorm.DB

func Initialize(dataSource string) error {
	var err error
	db, err = gorm.Open("postgres", dataSource)

	if err != nil {
		return fmt.Errorf("failed to connect to database: %s", err)
	}

	db.AutoMigrate(&Account{}, &PlayerOptions{}, &Character{}, &GuildcardEntry{})

	return nil
}

func Shutdown() {
	if err := db.Close(); err != nil {
		archon.Log.Error("error while closing database connection: ", err)
	}
}
