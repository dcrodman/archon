package server

/* Singleton package for handling the login and character server config. */

import ()

type configuration struct {
	host          string
	loginPort     string
	characterPort string
}

var loginConfig *configuration = nil

func (config *configuration) Hostname() string {
	return config.host
}

func (config *configuration) LoginPort() string {
	return config.loginPort
}

func (config *configuration) CharacterPort() string {
	return config.characterPort
}

func (config *configuration) InitFromMap(configMap map[string]string) {
	config.host = configMap["hostname"]
	config.loginPort = configMap["loginPort"]
	config.characterPort = configMap["characterPort"]
}

/*
func InitFromFile(fileName string) {
}
*/

func GetConfig() *configuration {
	if loginConfig == nil {
		loginConfig = new(configuration)
	}
	return loginConfig
}

// TODO: Other fields

// TODO: Load other files and resources
