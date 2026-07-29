package data

import (
	"testing"
)

func TestFindGuildcardEntries(t *testing.T) {
	db := setUpDatabase(t)

	testAccount := generateAccount(t)
	if err := db.Create(testAccount).Error; err != nil {
		t.Fatalf("error creating test account: %v", err)
	}

	guildcardEntries, err := FindGuildcardEntries(db, testAccount.ID)
	if err != nil {
		t.Fatalf("FindGuildcardEntries() returned an unexpected error: %v", err)
	}
	if len(guildcardEntries) > 0 {
		t.Fatalf("FindGuildcardEntries() returned guildcard entries unexpectedly: %v", guildcardEntries)
	}

	// Add tests here once this data is actually used.
}
