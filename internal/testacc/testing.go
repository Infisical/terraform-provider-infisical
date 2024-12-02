package testAcc

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/joho/godotenv"

	infisicalclient "terraform-provider-infisical/internal/client"
	provider "terraform-provider-infisical/internal/provider"
)

var infisicalApiClient *infisicalclient.Client

const IDENTITY_TEST_ORG_ID_ENV = "INFISICAL_TEST_ORG_ID"

var testEnvironmentVariables = []string{infisicalclient.INFISICAL_HOST_NAME, infisicalclient.INFISICAL_UNIVERSAL_AUTH_CLIENT_ID_NAME, infisicalclient.INFISICAL_UNIVERSAL_AUTH_CLIENT_SECRET_NAME, IDENTITY_TEST_ORG_ID_ENV}

func init() {
	log.SetOutput(os.Stderr)
	rootDir, err := findProjectRoot()
	if err != nil {
		log.Fatalf("Could not find root directory: %v", err)
		os.Exit(1)
	}

	if err := loadEnvFile(filepath.Join(rootDir, ".env")); err != nil {
		log.Fatalf("Could not load .env file: %v", err)
	}

	env_values := []string{}
	for _, env := range testEnvironmentVariables {
		if v := os.Getenv(env); v != "" {
			env_values = append(env_values, v)
		} else {
			log.Fatalf("%s must be set for acceptance tests", env)
		}
	}

	infisicalApiClient, err = infisicalclient.NewClient(infisicalclient.Config{
		HostURL:      env_values[0],
		AuthStrategy: infisicalclient.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY,
		ClientId:     env_values[1],
		ClientSecret: env_values[2],
	})
	if err != nil {
		log.Fatalf("Failed to initialize infisical api client: %v", err)
	}
}

func findProjectRoot() (string, error) {
	// Start from the current working directory
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Traverse up the directory tree until we find a go.mod file
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			// Found the go.mod file, this is the project root
			return dir, nil
		}

		// Move up one directory
		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			// Reached the root of the filesystem
			break
		}
		dir = parentDir
	}

	return "", fmt.Errorf("project root not found")
}

func protoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	infisicalProvider := provider.New("test")()

	return map[string]func() (tfprotov6.ProviderServer, error){
		"infisical": providerserver.NewProtocol6WithError(infisicalProvider),
	}
}

// preCheck checks if all conditions for an acceptance test are.
func preCheck(t *testing.T) func() {
	return func() {
		rootDir, err := findProjectRoot()
		if err != nil {
			t.Fatalf("Could not find root directory: %v", err)
		}
		if err := loadEnvFile(filepath.Join(rootDir, ".env")); err != nil {
			t.Fatalf("Could not load .env file: %v", err)
		}

		for _, env := range testEnvironmentVariables {
			if v := os.Getenv(env); v == "" {
				t.Fatalf("%s must be set for acceptance tests", env)
			}
		}
	}
}

type expectExternalResource struct {
	resourceAddress    string
	getResourceFromApi func(resource *tfjson.StateResource) error
}

func (e expectExternalResource) CheckState(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	var resource *tfjson.StateResource

	if req.State == nil {
		resp.Error = fmt.Errorf("state is nil")
	}

	if req.State.Values == nil {
		resp.Error = fmt.Errorf("state does not contain any state values")
	}

	if req.State.Values.RootModule == nil {
		resp.Error = fmt.Errorf("state does not contain a root module")
	}

	for _, r := range req.State.Values.RootModule.Resources {
		if e.resourceAddress == r.Address {
			resource = r
			break
		}
	}

	if resource == nil {
		resp.Error = fmt.Errorf("%s - Resource not found in state", e.resourceAddress)
		return
	}

	err := e.getResourceFromApi(resource)
	if err != nil {
		resp.Error = fmt.Errorf("Infisical API call failed for %s: %v", e.resourceAddress, err)
		return
	}
}

func ExpectExternalResource(
	resourceAddress string,
	getResourceFromApi func(resource *tfjson.StateResource) error,
) statecheck.StateCheck {
	return expectExternalResource{
		resourceAddress,
		getResourceFromApi,
	}
}

func loadEnvFile(filename string) error {
	if _, err := os.Stat(filename); errors.Is(err, os.ErrNotExist) {
		return nil
	}

	return godotenv.Load(filename)
}

// func getEnv(key, fallback string) string {
// 	if value, ok := os.LookupEnv(key); ok {
// 		return value
// 	}
// 	return fallback
// }

func getIdentityOrgId() string {
	if v := os.Getenv(IDENTITY_TEST_ORG_ID_ENV); v != "" {
		return v
	}
	log.Fatalf("Missing org id environment variable")
	return ""
}
