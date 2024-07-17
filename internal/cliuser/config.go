package cliuser

import (
	"encoding/json"
	"fmt"
	"os"
)

const (
	CONFIG_FOLDER_NAME = ".infisical"
	CONFIG_FILE_NAME   = "infisical-config.json"
)

type LoggedInUser struct {
	Email  string `json:"email"`
	Domain string `json:"domain"`
}

// The file struct for Infisical config file.
type ConfigFile struct {
	LoggedInUserEmail  string         `json:"loggedInUserEmail"`
	LoggedInUserDomain string         `json:"LoggedInUserDomain,omitempty"`
	LoggedInUsers      []LoggedInUser `json:"loggedInUsers,omitempty"`
	VaultBackendType   string         `json:"vaultBackendType,omitempty"`
}

func GetHomeDir() (string, error) {
	directory, err := os.UserHomeDir()
	return directory, err
}

func GetFullConfigFilePath() (fullPathToFile string, fullPathToDirectory string, err error) {
	homeDir, err := GetHomeDir()
	if err != nil {
		return "", "", err
	}

	fullPath := fmt.Sprintf("%s/%s/%s", homeDir, CONFIG_FOLDER_NAME, CONFIG_FILE_NAME)
	fullDirPath := fmt.Sprintf("%s/%s", homeDir, CONFIG_FOLDER_NAME)
	return fullPath, fullDirPath, err
}

func GetConfigFile() (ConfigFile, error) {
	fullConfigFilePath, _, err := GetFullConfigFilePath()
	if err != nil {
		return ConfigFile{}, err
	}

	configFileAsBytes, err := os.ReadFile(fullConfigFilePath)
	if err != nil {
		if err, ok := err.(*os.PathError); !ok {
			return ConfigFile{}, err
		}
		return ConfigFile{}, nil
	}

	var configFile ConfigFile
	err = json.Unmarshal(configFileAsBytes, &configFile)
	if err != nil {
		return ConfigFile{}, err
	}

	return configFile, nil
}

func ConfigFileExists() bool {
	fullConfigFileURI, _, err := GetFullConfigFilePath()
	if err != nil {
		return false
	}

	if _, err := os.Stat(fullConfigFileURI); err == nil {
		return true
	} else {
		return false
	}
}
