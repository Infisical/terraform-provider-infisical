package infisicalclient

import (
	"fmt"
	"slices"
	"strings"
	"terraform-provider-infisical/internal/cliuser"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	Config Config
}

type AuthStrategyType string

var AuthStrategy = struct {
	SERVICE_TOKEN              AuthStrategyType
	UNIVERSAL_MACHINE_IDENTITY AuthStrategyType
	USER_PROFILE               AuthStrategyType
}{
	SERVICE_TOKEN:              "SERVICE_TOKEN",
	UNIVERSAL_MACHINE_IDENTITY: "UNIVERSAL_MACHINE_IDENTITY",
	USER_PROFILE:               "USER_PROFILE",
}

type Config struct {
	HostURL string

	AuthStrategy AuthStrategyType

	// Service Token Auth
	ServiceToken string

	// Universal Machine Identity Auth
	ClientId     string
	ClientSecret string

	Profile string

	EnvSlug     string
	SecretsPath string
	HttpClient  *resty.Client // By default a client will be created
}

func (c *Client) ValidateAuthMode(modes []AuthStrategyType) (bool, error) {
	if slices.Contains(modes, c.Config.AuthStrategy) {
		return true, nil
	}

	var authErrorString []string
	for _, mode := range modes {
		authErrorString = append(authErrorString, string(mode))
	}
	return false, fmt.Errorf("Only %s authentication is supported", strings.Join(authErrorString, ","))
}

func NewClient(cnf Config) (*Client, error) {
	if cnf.HttpClient == nil {
		cnf.HttpClient = resty.New()
		cnf.HttpClient.SetBaseURL(cnf.HostURL)
	}

	// Add more auth strategies here later
	var usingServiceToken = cnf.ServiceToken != ""
	var usingUniversalAuth = cnf.ClientId != "" && cnf.ClientSecret != ""
	var usingInfisicalProfile = cnf.Profile != ""

	if usingUniversalAuth {
		token, err := Client{cnf}.UniversalMachineIdentityAuth()

		if err != nil {
			return nil, fmt.Errorf("unable to authenticate with universal machine identity [err=%s]", err)
		}

		cnf.HttpClient.SetAuthToken(token)
		cnf.AuthStrategy = AuthStrategy.UNIVERSAL_MACHINE_IDENTITY
	} else if usingServiceToken {
		cnf.HttpClient.SetAuthToken(cnf.ServiceToken)
		cnf.AuthStrategy = AuthStrategy.SERVICE_TOKEN
	} else if usingInfisicalProfile {
		token, err := cliuser.GetCurrentLoggedInUserDetails(cnf.Profile)
		if err != nil {
			return nil, fmt.Errorf("Unable to authenticate with user profile. [err=%s]", err)
		}
		_, err = Client{cnf}.CheckJWTIsValid(token)
		if err != nil {
			return nil, fmt.Errorf("Unable to authenticate with user profile. [err=%s]", err)
		}

		cnf.HttpClient.SetAuthToken(token)
		cnf.AuthStrategy = AuthStrategy.USER_PROFILE
	} else {
		// If no auth strategy is set, then we should return an error
		return nil, fmt.Errorf("you must configure a authentication method such as service tokens or Universal Auth or infisical login before making calls")
	}

	// These two if statements were a part of an older migration.
	// And when people upgraded to the newer version, we needed a way to indicate that the EnvSlug and SecretsPath are no longer defined on a provider-level.
	if cnf.EnvSlug != "" {
		return nil, fmt.Errorf("you must set the environment before making calls")
	}

	if cnf.SecretsPath != "" {
		return nil, fmt.Errorf("you must specify the secrets path before making calls")
	}

	cnf.HttpClient.SetHeader("Accept", "application/json")

	return &Client{cnf}, nil
}
