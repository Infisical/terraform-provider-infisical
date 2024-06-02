package cliuser

import (
	"fmt"
	"github.com/zalando/go-keyring"
)

const MAIN_KEYRING_SERVICE = "infisical-cli"

func GetCurrentVaultBackend() (string, error) {
	configFile, err := GetConfigFile()
	if err != nil {
		return "", fmt.Errorf("getCurrentVaultBackend: unable to get config file [err=%s]", err)
	}

	if configFile.VaultBackendType == "" {
		return "auto", nil
	}

	if configFile.VaultBackendType != "auto" && configFile.VaultBackendType != "file" {
		return "auto", nil
	}

	return configFile.VaultBackendType, nil
}

func GetValueInKeyring(key string) (string, error) {
	currentVaultBackend, err := GetCurrentVaultBackend()
	if err != nil {
		return "", fmt.Errorf("Unable to get current vault. Tip: run [infisical reset] then try again. %w", err)
	}

	return keyring.Get(currentVaultBackend, MAIN_KEYRING_SERVICE, key)
}
