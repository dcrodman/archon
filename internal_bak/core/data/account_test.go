package data

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gorm.io/gorm"
)

func seedRandomAccounts(t *testing.T, db *gorm.DB) {
	t.Helper()
	for i := 0; i < 10; i++ {
		if err := CreateAccount(db, generateAccount(t)); err != nil {
			t.Fatalf("error seeding test account: %v", err)
		}
	}
}

func generateAccount(t *testing.T) *Account {
	t.Helper()
	return &Account{
		Username: strconv.Itoa(rand.Int()),
		Password: strconv.Itoa(rand.Int()),
		Email:    fmt.Sprintf("%d@%d.c", rand.Int(), rand.Int()),
	}
}

func assertAccountsMatch(t *testing.T, expected *Account, got *Account) {
	if expected == nil && got == nil {
		return
	}

	if got != nil {
		got.DeletedAt = gorm.DeletedAt{}
	}
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("account did not match expected; diff:\n%s", diff)
	}
}

func TestFindAccountByID(t *testing.T) {
	db := setUpDatabase(t)
	seedRandomAccounts(t, db)

	testAccount := generateAccount(t)
	tests := []struct {
		name     string
		seedData func(db *gorm.DB)
		want     *Account
		wantErr  bool
	}{
		{
			name:     "account does not exist",
			seedData: func(db *gorm.DB) {},
			want:     nil,
			wantErr:  false,
		},
		{
			name: "account exists",
			seedData: func(db *gorm.DB) {
				if err := CreateAccount(db, testAccount); err != nil {
					t.Fatalf("error creating test account data: %s", err)
				}
			},
			want:    testAccount,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.seedData(db)

			// gorm assigns IDs back to the struct on creation.
			account, err := FindAccountByID(db, uint(testAccount.ID))
			if (err != nil) != tt.wantErr {
				t.Fatalf("FindAccountByID() wantErr = %v, error = %v", tt.wantErr, err)
			}
			assertAccountsMatch(t, tt.want, account)
		})
	}
}

func TestFindAccountByUsername(t *testing.T) {
	db := setUpDatabase(t)
	seedRandomAccounts(t, db)

	testAccount := generateAccount(t)
	tests := []struct {
		name     string
		seedData func(db *gorm.DB)
		want     *Account
		wantErr  bool
	}{
		{
			name:     "account does not exist",
			seedData: func(db *gorm.DB) {},
			want:     nil,
			wantErr:  false,
		},
		{
			name: "account exists",
			seedData: func(db *gorm.DB) {
				if err := CreateAccount(db, testAccount); err != nil {
					t.Fatalf("error creating test account data: %s", err)
				}
			},
			want:    testAccount,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.seedData(db)

			account, err := FindAccountByUsername(db, testAccount.Username)
			if (err != nil) != tt.wantErr {
				t.Fatalf("FindAccountByUsername() wantErr = %v, error = %v", tt.wantErr, err)
			}
			assertAccountsMatch(t, tt.want, account)
		})
	}
}

func TestFindUnscopedAccount(t *testing.T) {
	db := setUpDatabase(t)

	testAccount := generateAccount(t)
	if err := db.Create(testAccount).Error; err != nil {
		t.Fatalf("error creating test account: %v", err)
	}
	account, err := FindUnscopedAccount(db, testAccount.Username)
	if err != nil {
		t.Fatalf("FindUnscopedAccount() returned an unexpected error: %v", err)
	}
	assertAccountsMatch(t, testAccount, account)

	// Account exists, but has been soft deleted.
	if err := DeleteAccount(db, account); err != nil {
		t.Fatalf("error creating test account data: %s", err)
	}
	account, err = FindUnscopedAccount(db, testAccount.Username)
	if err != nil {
		t.Fatalf("FindUnscopedAccount() returned an unexpected error: %v", err)
	}
	assertAccountsMatch(t, testAccount, account)

	// Account has been hard deleted.
	if err := PermanentlyDeleteAccount(db, account); err != nil {
		t.Fatalf("error creating test account data: %s", err)
	}
	account, err = FindUnscopedAccount(db, testAccount.Username)
	if err != nil {
		t.Fatalf("FindUnscopedAccount() returned an unexpected error: %v", err)
	}
	if account != nil {
		t.Fatalf("FindUnscopedAccount() returned an account unexpectedly: %v", account)
	}
}
