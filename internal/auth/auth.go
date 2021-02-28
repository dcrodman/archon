package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/dcrodman/archon/internal/data"
)

var (
	ErrUnknown            = errors.New("an unexpected error occurred, please contact your server administrator")
	ErrInvalidCredentials = errors.New("username/combination password not found")
	ErrAccountBanned      = errors.New("this account has been suspended")
)

// VerifyAccount checks the Accounts table for the specified credentials
// combination and validates that the account is accessible.
func VerifyAccount(username, password string) (*data.Account, error) {
	account, err := findAccount(username)
	if err != nil {
		return nil, ErrUnknown
	}

	if account == nil || account.Password != HashPassword(password) {
		return nil, ErrInvalidCredentials
	} else if account.Banned {
		return nil, ErrAccountBanned
	}

	return account, nil
}

var findAccount = func(username string) (*data.Account, error) {
	return data.FindAccount(username)
}

// CreateAccount takes the specified credentials and creates a new record in
// the database, returning either the expected or any errors encountered.
func CreateAccount(username, password, email string) (*data.Account, error) {
	account := &data.Account{
		Username: username,
		Password: HashPassword(password),
		Email:    email,
	}

	if err := createAccount(account); err != nil {
		return nil, err
	}

	return account, nil
}

var createAccount = func(account *data.Account) error {
	return data.CreateAccount(account)
}

// DeleteAccount takes the specified credentials and soft-deletes a record in
// the database, returning any errors encountered.
func DeleteAccount(username string) error {
	return softDeleteAccount(username)
}

var softDeleteAccount = func(username string) error {
	a, err := data.FindAccount(username)
	if err != nil {
		return err
	}
	return data.DeleteAccount(a)
}

// PermanentlyDeleteAccount takes the specified credentials and deletes a record in
// the database, returning any errors encountered.
func PermanentlyDeleteAccount(username string) error {
	return permanentlyDeleteAccount(username)
}

var permanentlyDeleteAccount = func(username string) error {
	a, err := data.FindUnscopedAccount(username)
	if err != nil {
		return err
	}
	return data.PermanentlyDeleteAccount(a)
}

// HashPassword returns a version of password with Archon's chosen hashing strategy.
func HashPassword(password string) string {
	hash := sha256.New()
	if _, err := hash.Write(stripPadding([]byte(password))); err != nil {
		panic(fmt.Errorf("error generating password hash: %v", err))
	}
	return hex.EncodeToString(hash.Sum(nil)[:])
}

func stripPadding(b []byte) []byte {
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] != 0 {
			return b[:i+1]
		}
	}
	return b
}
