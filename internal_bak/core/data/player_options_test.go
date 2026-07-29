package data

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"gorm.io/gorm"
)

func TestFindPlayerOptions(t *testing.T) {
	db := setUpDatabase(t)

	testAccount := generateAccount(t)
	if err := db.Create(testAccount).Error; err != nil {
		t.Fatalf("error creating test account: %v", err)
	}

	testPlayerOptions := &PlayerOptions{
		Account:   testAccount,
		KeyConfig: []byte{1, 2, 3, 4},
	}

	tests := []struct {
		name     string
		seedData func(db *gorm.DB)
		want     *PlayerOptions
		wantErr  bool
	}{
		{
			name:     "playeroptions does not exist",
			seedData: func(db *gorm.DB) {},
			want:     nil,
			wantErr:  false,
		},
		{
			name: "playeroptions exists",
			seedData: func(db *gorm.DB) {
				if err := CreatePlayerOptions(db, testPlayerOptions); err != nil {
					t.Fatalf("CreatePlayerOptions() returned an error: %v", err)
				}
			},
			want:    testPlayerOptions,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.seedData(db)

			playerOptions, err := FindPlayerOptions(db, testAccount.ID)
			if (err != nil) != tt.wantErr {
				t.Fatalf("FindPlayerOptions() wantErr = %v, error = %v", tt.wantErr, err)
			}

			if diff := cmp.Diff(tt.want, playerOptions); diff != "" {
				t.Errorf("account did not match expected; diff:\n%s", diff)
			}
		})
	}
}

func TestUpdatePlayerOptions(t *testing.T) {
	db := setUpDatabase(t)

	testAccount := generateAccount(t)
	if err := db.Create(testAccount).Error; err != nil {
		t.Fatalf("error creating test account: %v", err)
	}

	testPlayerOptions := &PlayerOptions{
		Account:   testAccount,
		KeyConfig: []byte{1, 2, 3, 4},
	}

	if err := CreatePlayerOptions(db, testPlayerOptions); err != nil {
		t.Fatalf("CreatePlayerOptions() returned an unexpected error: %s", err)
	}

	testPlayerOptions.KeyConfig = []byte{5, 6, 7, 8}
	if err := UpdatePlayerOptions(db, testPlayerOptions); err != nil {
		t.Fatalf("UpdatePlayerOptions() returned an unexpected error: %s", err)
	}

	updatedPlayerOptions, err := FindPlayerOptions(db, testAccount.ID)
	if err != nil {
		t.Fatalf("FindPlayerOptions() returned an unexpected error: %s", err)
	}
	if diff := cmp.Diff(testPlayerOptions, updatedPlayerOptions); diff != "" {
		t.Errorf("player options were not updated\n%s", diff)
	}
}
