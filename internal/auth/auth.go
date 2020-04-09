package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/dcrodman/archon"
	"github.com/dcrodman/archon/data"
)

var (
	ErrUnknown            = errors.New("an unexpected error occurred, please contact your server administrator")
	ErrInvalidCredentials = errors.New("username/combination password not found")
	ErrAccountBanned      = errors.New("this account has been suspended")
)

// VerifyAccount checks the Accounts table for the specified credentials
// combination and validates that the account is accessible.
func VerifyAccount(username, password string) (*data.Account, error) {
	account, err := data.FindAccount(username)
	if err != nil {
		archon.Log.Warn("error in FindAccount: ", err)
		return nil, ErrUnknown
	}

	if account == nil || account.Password != HashPassword(password) {
		return nil, ErrInvalidCredentials
	} else if account.Banned {
		return nil, ErrAccountBanned
	}

	return account, nil
}

// CreateAccount takes the specified credentials and creates a new record in
// the database, returning either the result or any errors encountered.
func CreateAccount(username, password, email string) (*data.Account, error) {
	account := &data.Account{
		Username: username,
		Password: HashPassword(password),
		Email:    email,
	}

	if err := data.CreateAccount(account); err != nil {
		return nil, err
	}

	return account, nil
}

// HashPassword returns a version of password with Archon's chosen hashing strategy.
func HashPassword(password string) string {
	hash := sha256.New()
	hash.Write(stripPadding([]byte(password)))
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
