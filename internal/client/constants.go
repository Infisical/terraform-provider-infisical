package infisicalclient

import "errors"

const USER_AGENT = "terraform"

var (
	ErrNotFound = errors.New("resource not found")
)
