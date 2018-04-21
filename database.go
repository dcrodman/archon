package main

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	// Constants for collection names.
	accounts   = "accounts"
	options    = "player_options"
	characters = "characters"
	guildcards = "guildcards"
)

var database *Database

// dbFunc is an alias for the method signature expected by Database.op. All methods that
// leverage the session boilerplate define this for the actual database operations.
type dbFunc func(c *mgo.Collection) (interface{}, error)

// Database is our namespace for any methods that access our datastore.
type Database struct {
	session *mgo.Session
}

// Initialize the base Mongo session that we'll copy for all of our work.
func InitializeDatabase() (*Database, error) {
	session, err := mgo.Dial("mongodb://archonadmin:psoadminpassword@127.0.0.1:27017/archondb")
	if err != nil {
		return nil, err
	}
	return &Database{session: session}, nil
}

func (db *Database) Close() {
	db.session.Close()
}

// FindAccount will return the account data corresponding to username, or nil if none exists.
func (db *Database) FindAccount(username string) (*Account, error) {
	account, err := db.op(accounts, func(c *mgo.Collection) (interface{}, error) {
		var account Account
		err := c.Find(bson.M{}).One(&account)
		return &account, err
	})
	if account == nil {
		return nil, err
	}
	return account.(*Account), err
}

// FindPlayerOptions returns the PlayerOptions struct for the account identified by
// guildcard, or nil if there is no record for the account.
func (db *Database) FindPlayerOptions(guildcard uint32) (*PlayerOptions, error) {
	dbFn := func(c *mgo.Collection) (interface{}, error) {
		var playerOptions PlayerOptions
		err := c.Find(bson.M{}).One(&playerOptions)
		return &playerOptions, err
	}
	options, err := db.op(options, dbFn)
	if options == nil {
		return nil, err
	}
	return options.(*PlayerOptions), err
}

// UpdatePlayerOptions will update the persisted PlayerOptions or create a new entity
// if one does not already exist.
func (db *Database) UpdatePlayerOptions(playerOptions *PlayerOptions) error {
	dbFn := func(c *mgo.Collection) (interface{}, error) {
		err := c.Update(bson.M{"guildcard": playerOptions.Guildcard}, &playerOptions)
		if err == mgo.ErrNotFound {
			err = c.Insert(playerOptions)
		}
		return nil, err
	}
	_, err := db.op(options, dbFn)
	return err
}

// Create a character in the specified slot. Note that this method does not make any
// attempt to delete an existing character; use DeleteCharacter to do so.
func (db *Database) CreateCharacter(guildcard uint32, slotNum uint32, character *Character) error {
	_, err := db.op(characters, func(c *mgo.Collection) (interface{}, error) {
		return nil, c.Insert(character)
	})
	return err
}

// FindCharacter queries an account's characters for a particular slot to fetch
// its Character data. If no character exists for that slot, this function returns nil.
func (db *Database) FindCharacter(guildcard uint32, slotNum uint32) (*Character, error) {
	dbFn := func(c *mgo.Collection) (interface{}, error) {
		var character Character
		err := c.Find(bson.M{"guildcard": guildcard, "slot": slotNum}).One(&character)
		return &character, err
	}
	character, err := db.op(characters, dbFn)
	if character == nil {
		return nil, err
	}
	return character.(*Character), err
}

// UpdateCharacter will overwrite the character data for the character in slotNum for
// the account identified by guildcard.
func (db *Database) UpdateCharacter(guildcard uint32, slotNum uint32, character *Character) error {
	_, err := db.op(characters, func(c *mgo.Collection) (interface{}, error) {
		return nil, c.Update(bson.M{"guildcard": guildcard, "slot": slotNum}, &character)
	})
	return err
}

// DeleteCharacter wipes the character data in slotNum for the account identified by guildcard.
func (db *Database) DeleteCharacter(guildcard uint32, slotNum uint32) error {
	_, err := db.op(characters, func(c *mgo.Collection) (interface{}, error) {
		return nil, c.Remove(bson.M{"guildcard": guildcard, "slot": slotNum})
	})
	return err
}

// FindGuildcardData returns all guildcards that a user has added to their friends list.
func (db *Database) FindGuildcardData(guildcard uint32) ([]GuildcardEntry, error) {
	dbFn := func(c *mgo.Collection) (interface{}, error) {
		var guildcards []GuildcardEntry
		err := c.Find(bson.M{"guildcard": guildcard}).All(&guildcards)
		return guildcards, err
	}
	guildcards, err := db.op(guildcards, dbFn)
	return guildcards.([]GuildcardEntry), err
}

// Internal utility method for performing a database operation within the specified
// collection. Makes sure the copied session is correctly closed each time and
// transforms the ErrNotFound errors into nil for convenience.
func (db *Database) op(collection string, dbfn dbFunc) (interface{}, error) {
	s := db.session.Copy()
	defer s.Close()

	c := s.DB(config.DBName).C(collection)
	result, err := dbfn(c)

	if err == mgo.ErrNotFound {
		return nil, nil
	}
	return result, err
}
