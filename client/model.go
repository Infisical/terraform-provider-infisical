package infisicalclient

import "time"

type GetEncryptedSecretsV2Request struct {
	Environment string `json:"environment"`
	WorkspaceId string `json:"workspaceId"`
	TagSlugs    string `json:"tagSlugs"`
	SecretPath  string `json:"secretPath"`
}

type GetEncryptedSecretsV2Response struct {
	Secrets []struct {
		ID                      string    `json:"_id"`
		Version                 int       `json:"version"`
		Workspace               string    `json:"workspace"`
		Type                    string    `json:"type"`
		Environment             string    `json:"environment"`
		SecretKeyCiphertext     string    `json:"secretKeyCiphertext"`
		SecretKeyIV             string    `json:"secretKeyIV"`
		SecretKeyTag            string    `json:"secretKeyTag"`
		SecretValueCiphertext   string    `json:"secretValueCiphertext"`
		SecretValueIV           string    `json:"secretValueIV"`
		SecretValueTag          string    `json:"secretValueTag"`
		SecretCommentCiphertext string    `json:"secretCommentCiphertext"`
		SecretCommentIV         string    `json:"secretCommentIV"`
		SecretCommentTag        string    `json:"secretCommentTag"`
		V                       int       `json:"__v"`
		CreatedAt               time.Time `json:"createdAt"`
		UpdatedAt               time.Time `json:"updatedAt"`
		User                    string    `json:"user,omitempty"`
		Tags                    []struct {
			ID        string `json:"_id"`
			Name      string `json:"name"`
			Slug      string `json:"slug"`
			Workspace string `json:"workspace"`
		} `json:"tags"`
	} `json:"secrets"`
}

type GetServiceTokenDetailsResponse struct {
	ID           string    `json:"_id"`
	Name         string    `json:"name"`
	Workspace    string    `json:"workspace"`
	Environment  string    `json:"environment"`
	ExpiresAt    time.Time `json:"expiresAt"`
	EncryptedKey string    `json:"encryptedKey"`
	Iv           string    `json:"iv"`
	Tag          string    `json:"tag"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	V            int       `json:"__v"`
}
