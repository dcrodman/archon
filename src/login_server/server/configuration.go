/*
* Archon Login Server
* Copyright (C) 2014 Andrew Rodman
*
* This program is free software: you can redistribute it and/or modify
* it under the terms of the GNU General Public License as published by
* the Free Software Foundation, either version 3 of the License, or
* (at your option) any later version.
*
* This program is distributed in the hope that it will be useful,
* but WITHOUT ANY WARRANTY; without even the implied warranty of
* MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
* GNU General Public License for more details.
*
* You should have received a copy of the GNU General Public License
* along with this program.  If not, see <http://www.gnu.org/licenses/>.
* ---------------------------------------------------------------------
*
* Singleton package for handling the login and character server config.
 */
package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
)

const loginConfigFile = "login_config.json"

type configuration struct {
	Hostname      string
	LoginPort     string
	CharacterPort string
	DBHost        string
	DBPort        string
	DBName        string
	DBUsername    string
	DBPassword    string

	database *sql.DB
}

var loginConfig *configuration = nil

func GetConfig() *configuration {
	if loginConfig == nil {
		loginConfig = new(configuration)
	}
	return loginConfig
}

// Populate config with the contents of a JSON file at path fileName. Config parameters
// in the file must match the above fields exactly in order to be read.
func (config *configuration) InitFromFile(fileName string) error {
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	json.Unmarshal(data, config)
	return nil
}

/*
func (config *configuration) InitFromMap(configMap map[string]string) {
	config.Hostname = configMap["hostname"]
	config.LoginPort = configMap["loginPort"]
	config.CharacterPort = configMap["characterPort"]
}
*/

// Establish a connection to the database and ping it to verify.
func (config *configuration) InitDb() error {
	dbName := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", config.DBUsername,
		config.DBPassword, config.DBHost, config.DBPort, config.DBName)
	var err error
	config.database, err = sql.Open("mysql", dbName)
	if err != nil || config.database.Ping() != nil {
		return err
	}
	return nil
}

func (config *configuration) CloseDb() {
	config.database.Close()
}

func (config *configuration) Database() *sql.DB { return config.database }

func (config *configuration) String() string {
	return "Hostname: " + config.Hostname + "\n" +
		"Login Port: " + config.LoginPort + "\n" +
		"Character Port: " + config.CharacterPort + "\n" +
		"Database Host: " + config.DBHost + "\n" +
		"Database Port: " + config.DBPort + "\n" +
		"Database Name: " + config.DBName + "\n" +
		"Database Username: " + config.DBUsername + "\n" +
		"Database Password: " + config.DBPassword
}
