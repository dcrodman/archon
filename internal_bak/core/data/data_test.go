package data

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// Creates a database for testing. For the sake of simplicity, this only uses the
// SQLite engine and creates a new database on every invocation since it is relatively
// cheap to do so (especially given the low number of tests). If this ever becomes
// prohibitive due to performance, this approach will need to be reevaluated.
func setUpDatabase(t *testing.T) *gorm.DB {
	testDBFile := filepath.Join(t.TempDir(), "test.db")
	db, err := gorm.Open(sqlite.Open(testDBFile))
	if err != nil {
		t.Fatalf("error initializing test database: %s", err)
	}

	if err = db.AutoMigrate(
		&Account{},
		&PlayerOptions{},
		&Character{},
		&GuildcardEntry{},
	); err != nil {
		t.Fatalf("error auto migrating db: %s", err)
	}
	return db
}
