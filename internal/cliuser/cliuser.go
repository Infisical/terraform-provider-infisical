package cliuser

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/zalando/go-keyring"
)

type UserCredentials struct {
	Email        string `json:"email"`
	PrivateKey   string `json:"privateKey"`
	JTWToken     string `json:"JTWToken"`
	RefreshToken string `json:"RefreshToken"`
}

func GetCurrentLoggedInUserDetails(profile string) (string, error) {
	if ConfigFileExists() {
		configFile, err := GetConfigFile()
		if err != nil {
			return "", fmt.Errorf("getCurrentLoggedInUserDetails: unable to get logged in user from config file [err=%s]", err)
		}

		if configFile.LoggedInUserEmail == "" {
			return "", errors.New("Login user not found. Try infisical login.")
		}

		if configFile.LoggedInUserEmail != profile {
			return "", errors.New("User profile not found. Try infisical login.")
		}

		userCreds, err := GetUserCredsFromKeyRing(configFile.LoggedInUserEmail)
		if err != nil {
			if strings.Contains(err.Error(), "credentials not found in system keyring") {
				return "", errors.New("we couldn't find your logged in details, try running [infisical login] then try again")
			} else {
				return "", fmt.Errorf("failed to fetch credentials from keyring because [err=%s]", err)
			}
		}
		return userCreds.JTWToken, nil
	}

	return "", errors.New("Config file not found. Try infisical login.")
}

func GetUserCredsFromKeyRing(userEmail string) (credentials UserCredentials, err error) {
	credentialsValue, err := GetValueInKeyring(userEmail)
	if err != nil {
		if err == keyring.ErrUnsupportedPlatform {
			return UserCredentials{}, errors.New("your OS does not support keyring. Consider using a service token https://infisical.com/docs/documentation/platform/token")
		} else if err == keyring.ErrNotFound {
			return UserCredentials{}, errors.New("credentials not found in system keyring")
		} else {
			return UserCredentials{}, fmt.Errorf("something went wrong, failed to retrieve value from system keyring [error=%v]", err)
		}
	}

	var userCredentials UserCredentials

	err = json.Unmarshal([]byte(credentialsValue), &userCredentials)
	if err != nil {
		return UserCredentials{}, fmt.Errorf("getUserCredsFromKeyRing: Something went wrong when unmarshalling user creds [err=%s]", err)
	}

	return userCredentials, err
}
