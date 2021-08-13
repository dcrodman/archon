package auth

import (
	"fmt"
	"testing"

	"github.com/dcrodman/archon/internal/core/data"
)

func TestCreateAccount(t *testing.T) {
	type args struct {
		username string
		password string
		email    string
	}
	tests := map[string]struct {
		dbCreateFn func(account *data.Account) error
		args       args
		wantedErr  error
	}{
		"database_error": {
			dbCreateFn: func(account *data.Account) error { return fmt.Errorf("database error") },
			args:       args{username: "test", password: "test", email: "test"},
			wantedErr:  fmt.Errorf("database error"),
		},
		"happy_path": {
			dbCreateFn: func(account *data.Account) error { return nil },
			args:       args{username: "test", password: "test", email: "a@b.c"},
			wantedErr:  nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			originalCreateAccount := createAccount
			defer func() {
				createAccount = originalCreateAccount
			}()
			createAccount = tt.dbCreateFn

			account, err := CreateAccount(tt.args.username, tt.args.password, tt.args.email)
			if err != nil && err.Error() != tt.wantedErr.Error() {
				t.Fatalf("expected error to = %s, got = %s", tt.wantedErr, err)
			}

			if err == nil {
				if account.Username != tt.args.username {
					t.Errorf("expected account username = %s, got = %s", tt.args.username, account.Username)
				}
				if account.Password != HashPassword(tt.args.username) {
					t.Error("expected account password to equal hashed password")
				}
				if account.Email != tt.args.email {
					t.Errorf("expected account emmail = %s, got = %s", tt.args.email, account.Email)
				}
			}
		})
	}
}

func TestHashPassword(t *testing.T) {
	password := "password"
	hashed := HashPassword(password)

	if password == hashed {
		t.Fatalf("expected hashed password not to equal password")
	}

	for i := 0; i < 10; i++ {
		if h := HashPassword(password); hashed != h {
			t.Fatalf("password hashing is non-deterministic (expected %s, got %s)", hashed, h)
		}
	}
}

func TestVerifyAccount(t *testing.T) {
	type context struct {
		account *data.Account
		err     error
	}
	type args struct {
		username string
		password string
	}
	type expected struct {
		account *data.Account
		err     error
	}

	happyPathAccount := &data.Account{Username: "test", Password: HashPassword("test")}

	tests := map[string]struct {
		context context
		args    args
		result  expected
	}{
		"database_error": {
			context{account: nil, err: fmt.Errorf("something exploded")},
			args{username: "test", password: "test"},
			expected{account: nil, err: ErrUnknown},
		},
		"no_account": {
			context{account: nil, err: nil},
			args{username: "test", password: "test"},
			expected{account: nil, err: ErrInvalidCredentials},
		},
		"invalid_password": {
			context{account: &data.Account{Username: "test", Password: "x"}, err: nil},
			args{username: "test", password: "test"},
			expected{account: nil, err: ErrInvalidCredentials},
		},
		"banned": {
			context{account: &data.Account{Username: "test", Password: HashPassword("test"), Banned: true}, err: nil},
			args{username: "test", password: "test"},
			expected{account: nil, err: ErrAccountBanned},
		},
		"happy": {
			context{account: happyPathAccount, err: nil},
			args{username: "test", password: "test"},
			expected{account: happyPathAccount, err: nil},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			originalFindAccount := findAccount

			findAccount = func(username string) (*data.Account, error) {
				return tt.context.account, tt.context.err
			}

			_, err := VerifyAccount(tt.args.username, tt.args.password)

			if err != tt.result.err {
				t.Errorf("expected wantedErr = %s, got = %s", tt.result.err, err)
			}

			findAccount = originalFindAccount
		})
	}
}

func TestSoftDeleteAccount(t *testing.T) {
	type args struct {
		username string
	}
	tests := map[string]struct {
		dbDeleteFunc func(username string) error
		args         args
		wantedErr    error
	}{
		"database_error": {
			dbDeleteFunc: func(username string) error { return fmt.Errorf("database error") },
			args:         args{username: "test"},
			wantedErr:    fmt.Errorf("database error"),
		},
		"happy_path": {
			dbDeleteFunc: func(username string) error { return nil },
			args:         args{username: "test"},
			wantedErr:    nil,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			originalDeleteAccount := softDeleteAccount
			softDeleteAccount = tt.dbDeleteFunc

			if err := DeleteAccount(tt.args.username); err != nil && err.Error() != tt.wantedErr.Error() {
				t.Errorf("expected error to = %s, got = %s", tt.wantedErr, err)
			}

			softDeleteAccount = originalDeleteAccount
		})
	}
}

func TestPermanentlyDeleteAccount(t *testing.T) {
	type args struct {
		username string
	}
	tests := map[string]struct {
		dbDeleteFunc func(username string) error
		args         args
		wantedErr    error
	}{
		"database_error": {
			dbDeleteFunc: func(username string) error { return fmt.Errorf("database error") },
			args:         args{username: "test"},
			wantedErr:    fmt.Errorf("database error"),
		},
		"happy_path": {
			dbDeleteFunc: func(username string) error { return nil },
			args:         args{username: "test"},
			wantedErr:    nil,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			originalDeleteAccount := permanentlyDeleteAccount
			permanentlyDeleteAccount = tt.dbDeleteFunc

			if err := PermanentlyDeleteAccount(tt.args.username); err != nil && err.Error() != tt.wantedErr.Error() {
				t.Errorf("expected error to = %s, got = %s", tt.wantedErr, err)
			}

			permanentlyDeleteAccount = originalDeleteAccount
		})
	}
}

func Test_stripPadding(t *testing.T) {
	testSlice := []byte{0, 1, 2, 3, 0, 0, 0}
	trimmed := stripPadding(testSlice)

	if len(trimmed) != 4 {
		t.Errorf("expected trimmed to have len = 4, got %d", len(trimmed))
	}

	for i := 0; i < 4; i++ {
		if trimmed[i] != testSlice[i] {
			t.Errorf("expected trimmed[%d] (%d) = testSlice[%d] (%d)", i, i, trimmed[i], testSlice[i])
		}
	}
}
