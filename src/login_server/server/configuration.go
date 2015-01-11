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
	"encoding/json"
	"io/ioutil"
)

const loginConfigFile = "login_config.json"

type configuration struct {
	Hostname      string
	LoginPort     string
	CharacterPort string
}

var loginConfig *configuration = nil

func GetConfig() *configuration {
	if loginConfig == nil {
		loginConfig = new(configuration)
	}
	return loginConfig
}

func (config *configuration) GetHostname() string {
	return config.Hostname
}

func (config *configuration) GetLoginPort() string {
	return config.LoginPort
}

func (config *configuration) GetCharacterPort() string {
	return config.CharacterPort
}

func (config *configuration) InitFromMap(configMap map[string]string) {
	config.Hostname = configMap["hostname"]
	config.LoginPort = configMap["loginPort"]
	config.CharacterPort = configMap["characterPort"]
}

func (config *configuration) InitFromFile(fileName string) error {
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	json.Unmarshal(data, config)
	return nil
}

func (config *configuration) String() string {
	return "Hostname: " + config.GetHostname() + "\n" +
		"Login Port: " + config.GetLoginPort() + "\n" +
		"Character Port: " + config.GetCharacterPort()
}
