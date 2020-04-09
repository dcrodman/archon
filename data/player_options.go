package data

import "github.com/jinzhu/gorm"

type PlayerOptions struct {
	gorm.Model

	AccountID int
	Account   Account

	KeyConfig []byte
}

// FindPlayerOptions returns all of hte PlayerOptions associated with an Account.
func FindPlayerOptions(account *Account) (*PlayerOptions, error) {
	var playerOptions PlayerOptions
	q := db.Model(&account).Related(&playerOptions)

	if q.Error != nil {
		if gorm.IsRecordNotFoundError(q.Error) {
			return nil, nil
		}
		return nil, q.Error
	}

	return &playerOptions, nil
}

// UpdatePlayerOptions updates the PlayerOptions row with the contents in po.
func UpdatePlayerOptions(po *PlayerOptions) error {
	if db.NewRecord(po) {
		return db.Save(po).Error
	}
	return db.Update(po).Error
}
