package infisicalclient

import (
	"fmt"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	Config Config
}

type Config struct {
	HostURL      string
	ServiceToken string
	EnvSlug      string
	SecretsPath  string
	HttpClient   *resty.Client // By default a client will be created
}

func NewClient(cnf Config) (*Client, error) {
	if cnf.HttpClient == nil {
		cnf.HttpClient = resty.New()
		cnf.HttpClient.SetBaseURL(cnf.HostURL)
	}

	if cnf.ServiceToken == "" {
		return nil, fmt.Errorf("you must set the service token for the client before making calls")
	}

	if cnf.ServiceToken != "" {
		cnf.HttpClient.SetAuthToken(cnf.ServiceToken)
	}

	if cnf.EnvSlug != "" {
		return nil, fmt.Errorf("you must set the environment before making calls")
	}

	if cnf.SecretsPath != "" {
		return nil, fmt.Errorf("you must specify the secrets path before making calls")
	}

	cnf.HttpClient.SetHeader("Accept", "application/json")

	return &Client{cnf}, nil
}
