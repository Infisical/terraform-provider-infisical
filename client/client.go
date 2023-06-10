package infisicalclient

import (
	"fmt"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	cnf Config
}

type Config struct {
	HostURL      string
	ServiceToken string
	ApiKey       string
	HttpClient   *resty.Client // By default a client will be created
}

func NewClient(cnf Config) (*Client, error) {
	if cnf.ApiKey == "" && cnf.ServiceToken == "" {
		return nil, fmt.Errorf("You must enter either a API Key or Service token for authentication with Infisical API")
	}

	if cnf.HttpClient == nil {
		cnf.HttpClient = resty.New()
		cnf.HttpClient.SetBaseURL(cnf.HostURL)
	}

	if cnf.ServiceToken != "" {
		cnf.HttpClient.SetAuthToken(cnf.ServiceToken)
	}

	if cnf.ApiKey != "" {
		cnf.HttpClient.SetHeader("X-API-KEY", cnf.ApiKey)
	}

	cnf.HttpClient.SetHeader("Accept", "application/json")

	return &Client{cnf}, nil
}
