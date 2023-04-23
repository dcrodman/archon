package data

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gorm.io/gorm"
)

func TestFindCharacter(t *testing.T) {
	db := setUpDatabase(t)

	testAccount := generateAccount(t)
	if err := db.Create(testAccount).Error; err != nil {
		t.Fatalf("error creating test account: %v", err)
	}
	testCharacter := &Character{
		Account:   testAccount,
		Slot:      1,
		Guildcard: 12345,
		Level:     1,
	}
	tests := []struct {
		name     string
		seedData func(db *gorm.DB)
		want     *Character
		wantErr  bool
	}{
		{
			name:     "character does not exist",
			seedData: func(db *gorm.DB) {},
			want:     nil,
			wantErr:  false,
		},
		{
			name: "character exists",
			seedData: func(db *gorm.DB) {
				if err := db.Create(testCharacter).Error; err != nil {
					t.Fatalf("error creating character: %v", err)
				}
			},
			want:    testCharacter,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.seedData(db)

			character, err := FindCharacter(db, uint(testAccount.ID), testCharacter.Slot)
			if (err != nil) != tt.wantErr {
				t.Fatalf("FindAccountByID() wantErr = %v, error = %v", tt.wantErr, err)
			}

			if diff := cmp.Diff(tt.want, character); diff != "" {
				t.Errorf("account did not match expected; diff:\n%s", diff)
			}
		})
	}
}

func TestUpsertCharacter(t *testing.T) {
	db := setUpDatabase(t)

	testAccount := generateAccount(t)
	if err := db.Create(testAccount).Error; err != nil {
		t.Fatalf("error creating test account: %v", err)
	}
	testCharacter := &Character{
		Account:   testAccount,
		Slot:      1,
		Guildcard: 12345,
		Level:     1,
	}

	if err := db.Create(testCharacter).Error; err != nil {
		t.Fatalf("error creating character: %v", err)
	}

	if err := UpsertCharacter(db, testCharacter); err != nil {
		t.Fatalf("UpsertCharacter() returned an unexpected error: %s", err)
	}

	// Ensure the upsert applied the change.
	character, err := FindCharacter(db, uint(testAccount.ID), testCharacter.Slot)
	if err != nil {
		t.Fatalf("FindCharacter() returned an unexpected error: %s", err)
	}

	// Ignore this field for comparison.
	if !character.UpdatedAt.After(testAccount.UpdatedAt) {
		t.Fatalf("character was not updated on upsert")
	}
	character.UpdatedAt = testCharacter.UpdatedAt
	if diff := cmp.Diff(testCharacter, character); diff != "" {
		t.Fatalf("account did not match expected; diff:\n%s", diff)
	}
}

func TestDeleteCharacter(t *testing.T) {
	db := setUpDatabase(t)

	testAccount := generateAccount(t)
	if err := db.Create(testAccount).Error; err != nil {
		t.Fatalf("error creating test account: %v", err)
	}
	testCharacter := &Character{
		Account:   testAccount,
		Slot:      1,
		Guildcard: 12345,
		Level:     1,
	}

	if err := db.Create(testCharacter).Error; err != nil {
		t.Fatalf("error creating character: %v", err)
	}

	if err := DeleteCharacter(db, uint(testAccount.ID), testCharacter.Slot); err != nil {
		t.Fatalf("DeleteCharacter() returned an unexpected error: %s", err)
	}

	// Once we've deleted it, make sure it's not returned by FindCharacter.
	character, err := FindCharacter(db, uint(testAccount.ID), testCharacter.Slot)
	if err != nil {
		t.Fatalf("FindCharacter() returned an unexpected error: %s", err)
	}
	if character != nil {
		t.Fatalf("DeleteCharacter() did not delete the character:\n%v", character)
	}

	// Ensure the character was soft deleted.
	err = db.Unscoped().Where("id = ?", testCharacter.ID).Preload("Account").First(&character).Error
	if err != nil {
		t.Fatalf("querying for deleted character returned an unexpected error: %v", err)
	}

	if !character.DeletedAt.Valid {
		t.Fatalf("character's DeletedAt was not set:\n%v", character)
	}
	character.DeletedAt = gorm.DeletedAt{}
	if diff := cmp.Diff(testCharacter, character); diff != "" {
		t.Fatalf("account did not match expected; diff:\n%s", diff)
	}
}

func TestPermanentlyDeleteCharacter(t *testing.T) {
	db := setUpDatabase(t)

	testAccount := generateAccount(t)
	if err := db.Create(testAccount).Error; err != nil {
		t.Fatalf("error creating test account: %v", err)
	}
	testCharacter := &Character{
		Account:   testAccount,
		Slot:      1,
		Guildcard: 12345,
		Level:     1,
	}

	if err := db.Create(testCharacter).Error; err != nil {
		t.Fatalf("error creating character: %v", err)
	}

	if err := PermanentlyDeleteCharacter(db, testCharacter); err != nil {
		t.Fatalf("PermanentlyDeleteCharacter() returned an unexpected error: %s", err)
	}

	// Ensure the character was hard deleted.
	var character Character
	err := db.Unscoped().Where("id = ?", testCharacter.ID).First(&character).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("querying for deleted character returned an unexpected error: %v", err)
	}
}
