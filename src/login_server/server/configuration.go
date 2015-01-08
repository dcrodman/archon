// Singleton package for handling the login and character server config.
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
