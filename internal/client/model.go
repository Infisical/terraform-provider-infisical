package infisicalclient

import (
	"time"
)

type GetEncryptedSecretsV3Request struct {
	Environment string `json:"environment"`
	WorkspaceId string `json:"workspaceId"`
	SecretPath  string `json:"secretPath"`
}

type EncryptedSecretV3 struct {
	ID        string `json:"_id"`
	Version   int    `json:"version"`
	Workspace string `json:"workspace"`
	Type      string `json:"type"`
	Tags      []struct {
		ID        string `json:"_id"`
		Name      string `json:"name"`
		Slug      string `json:"slug"`
		Workspace string `json:"workspace"`
	} `json:"tags"`
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
	Algorithm               string    `json:"algorithm"`
	KeyEncoding             string    `json:"keyEncoding"`
	Folder                  string    `json:"folder"`
	V                       int       `json:"__v"`
	CreatedAt               time.Time `json:"createdAt"`
	UpdatedAt               time.Time `json:"updatedAt"`
}

type Project struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Description        string    `json:"description"`
	Slug               string    `json:"slug"`
	AutoCapitalization bool      `json:"autoCapitalization"`
	OrgID              string    `json:"orgId"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
	Version            int       `json:"version"`

	UpgradeStatus string `json:"upgradeStatus"` // can be null. if its null it will be converted to an empty string.
}

type ProjectTag struct {
	ID    string `json:"id"`
	Slug  string `json:"slug"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

type ProjectUser struct {
	ID     string `json:"id"`
	UserID string `json:"userId"`
	User   struct {
		Email     string `json:"email"`
		ID        string `json:"id"`
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		PublicKey string `json:"publicKey"`
	} `json:"user"`
	Roles []ProjectMemberRole
}

type ProjectIdentity struct {
	ID         string `json:"id"`
	IdentityID string `json:"identityId"`
	Roles      []ProjectMemberRole
	Identity   struct {
		Name                string   `json:"name"`
		HasDeleteProtection bool     `json:"hasDeleteProtection"`
		Id                  string   `json:"id"`
		AuthMethods         []string `json:"authMethods"`
	} `json:"identity"`
	Project struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"project"`
}

type ProjectMemberRole struct {
	ID                       string    `json:"id"`
	Role                     string    `json:"role"`
	CustomRoleSlug           string    `json:"customRoleSlug"`
	ProjectMembershipId      string    `json:"projectMembershipId"`
	CustomRoleId             string    `json:"customRoleId"`
	IsTemporary              bool      `json:"isTemporary"`
	TemporaryMode            string    `json:"temporaryMode"`
	TemporaryRange           string    `json:"temporaryRange"`
	TemporaryAccessStartTime time.Time `json:"temporaryAccessStartTime"`
	TemporaryAccessEndTime   time.Time `json:"temporaryAccessEndTime"`
	CreatedAt                time.Time `json:"createdAt"`
	UpdatedAt                time.Time `json:"updatedAt"`
}

type OrgIdentity struct {
	Identity   Identity `json:"identity"`
	Role       string   `json:"role"`
	OrgID      string   `json:"orgId"`
	CustomRole *struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		Description string `json:"description"`
	} `json:"customRole,omitempty"`
	Metadata []MetaEntry `json:"metadata"`
}

type Identity struct {
	Name                string    `json:"name"`
	HasDeleteProtection bool      `json:"hasDeleteProtection"`
	ID                  string    `json:"id"`
	AuthMethods         []string  `json:"authMethods"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

type MetaEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type IdentityUniversalAuth struct {
	ID                      string                  `json:"id"`
	ClientID                string                  `json:"clientId"`
	AccessTokenTTL          int64                   `json:"accessTokenTTL"`
	AccessTokenMaxTTL       int64                   `json:"accessTokenMaxTTL"`
	AccessTokenNumUsesLimit int64                   `json:"accessTokenNumUsesLimit"`
	CreatedAt               string                  `json:"createdAt"`
	UpdatedAt               string                  `json:"updatedAt"`
	IdentityID              string                  `json:"identityId"`
	ClientSecretTrustedIps  []IdentityAuthTrustedIp `json:"clientSecretTrustedIps"`
	AccessTokenTrustedIps   []IdentityAuthTrustedIp `json:"accessTokenTrustedIps"`
}

type IdentityAwsAuth struct {
	ID                      string                  `json:"id"`
	AccessTokenTTL          int64                   `json:"accessTokenTTL"`
	AccessTokenMaxTTL       int64                   `json:"accessTokenMaxTTL"`
	AccessTokenNumUsesLimit int64                   `json:"accessTokenNumUsesLimit"`
	AccessTokenTrustedIPS   []IdentityAuthTrustedIp `json:"accessTokenTrustedIps"`
	CreatedAt               string                  `json:"createdAt"`
	UpdatedAt               string                  `json:"updatedAt"`
	IdentityID              string                  `json:"identityId"`
	Type                    string                  `json:"type"`
	StsEndpoint             string                  `json:"stsEndpoint"`
	AllowedPrincipalArns    string                  `json:"allowedPrincipalArns"`
	AllowedAccountIDS       string                  `json:"allowedAccountIds"`
}

type IdentityAzureAuth struct {
	ID                         string                  `json:"id"`
	AccessTokenTTL             int64                   `json:"accessTokenTTL"`
	AccessTokenMaxTTL          int64                   `json:"accessTokenMaxTTL"`
	AccessTokenNumUsesLimit    int64                   `json:"accessTokenNumUsesLimit"`
	AccessTokenTrustedIPS      []IdentityAuthTrustedIp `json:"accessTokenTrustedIps"`
	CreatedAt                  string                  `json:"createdAt"`
	UpdatedAt                  string                  `json:"updatedAt"`
	IdentityID                 string                  `json:"identityId"`
	TenantID                   string                  `json:"tenantId"`
	Resource                   string                  `json:"resource"`
	AllowedServicePrincipalIDS string                  `json:"allowedServicePrincipalIds"`
}

type IdentityGcpAuth struct {
	ID                      string                  `json:"id"`
	AccessTokenTTL          int64                   `json:"accessTokenTTL"`
	AccessTokenMaxTTL       int64                   `json:"accessTokenMaxTTL"`
	AccessTokenNumUsesLimit int64                   `json:"accessTokenNumUsesLimit"`
	AccessTokenTrustedIPS   []IdentityAuthTrustedIp `json:"accessTokenTrustedIps"`
	CreatedAt               string                  `json:"createdAt"`
	UpdatedAt               string                  `json:"updatedAt"`
	IdentityID              string                  `json:"identityId"`
	Type                    string                  `json:"type"`
	AllowedServiceAccounts  string                  `json:"allowedServiceAccounts"`
	AllowedProjects         string                  `json:"allowedProjects"`
	AllowedZones            string                  `json:"allowedZones"`
}

type IdentityKubernetesAuth struct {
	ID                         string                  `json:"id"`
	AccessTokenTTL             int64                   `json:"accessTokenTTL"`
	AccessTokenMaxTTL          int64                   `json:"accessTokenMaxTTL"`
	AccessTokenNumUsesLimit    int64                   `json:"accessTokenNumUsesLimit"`
	AccessTokenTrustedIPS      []IdentityAuthTrustedIp `json:"accessTokenTrustedIps"`
	CreatedAt                  string                  `json:"createdAt"`
	UpdatedAt                  string                  `json:"updatedAt"`
	IdentityID                 string                  `json:"identityId"`
	KubernetesHost             string                  `json:"kubernetesHost"`
	AllowedNamespaces          string                  `json:"allowedNamespaces"`
	AllowedServiceAccountNames string                  `json:"allowedNames"`
	AllowedAudience            string                  `json:"allowedAudience"`
	CACERT                     string                  `json:"caCert"`
	TokenReviewerJwt           string                  `json:"tokenReviewerJwt"`
}

type IdentityOidcAuth struct {
	ID                      string                  `json:"id"`
	AccessTokenTTL          int64                   `json:"accessTokenTTL"`
	AccessTokenMaxTTL       int64                   `json:"accessTokenMaxTTL"`
	AccessTokenNumUsesLimit int64                   `json:"accessTokenNumUsesLimit"`
	AccessTokenTrustedIPS   []IdentityAuthTrustedIp `json:"accessTokenTrustedIps"`
	CreatedAt               string                  `json:"createdAt"`
	UpdatedAt               string                  `json:"updatedAt"`
	IdentityID              string                  `json:"identityId"`
	OidcDiscoveryUrl        string                  `json:"oidcDiscoveryUrl"`
	BoundIssuer             string                  `json:"boundIssuer"`
	BoundAudiences          string                  `json:"boundAudiences"`
	BoundClaims             map[string]string       `json:"boundClaims"`
	ClaimMetadataMapping    map[string]string       `json:"claimMetadataMapping"`
	BoundSubject            string                  `json:"boundSubject"`
	CACERT                  string                  `json:"caCert"`
}

type IdentityAuthTrustedIp struct {
	Type      string `json:"type"`
	Prefix    *int   `json:"prefix,omitempty"`
	IpAddress string `json:"ipAddress"`
}

type IdentityUniversalAuthClientSecret struct {
	ID                       string `json:"id"`
	CreatedAt                string `json:"createdAt"`
	UpdatedAt                string `json:"updatedAt"`
	Description              string `json:"description"`
	ClientSecretPrefix       string `json:"clientSecretPrefix"`
	ClientSecretNumUses      int64  `json:"clientSecretNumUses"`
	ClientSecretNumUsesLimit int64  `json:"clientSecretNumUsesLimit"`
	ClientSecretTTL          int64  `json:"clientSecretTTL"`
	IdentityUAID             string `json:"identityUAId"`
	IsClientSecretRevoked    bool   `json:"isClientSecretRevoked"`
}

type ProjectIdentitySpecificPrivilege struct {
	ID                       string    `json:"id"`
	Slug                     string    `json:"slug"`
	ProjectMembershipId      string    `json:"projectMembershipId"`
	IsTemporary              bool      `json:"isTemporary"`
	TemporaryMode            string    `json:"temporaryMode"`
	TemporaryRange           string    `json:"temporaryRange"`
	TemporaryAccessStartTime time.Time `json:"temporaryAccessStartTime"`
	TemporaryAccessEndTime   time.Time `json:"temporaryAccessEndTime"`
	// because permission can have multiple structure.
	Permissions []map[string]any
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type ProjectRole struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	// because permission can have multiple structure.
	Permissions []map[string]any
}

type ProjectWithEnvironments struct {
	ID                    string               `json:"id"`
	Name                  string               `json:"name"`
	Description           string               `json:"description"`
	Slug                  string               `json:"slug"`
	AutoCapitalization    bool                 `json:"autoCapitalization"`
	OrgID                 string               `json:"orgId"`
	CreatedAt             time.Time            `json:"createdAt"`
	UpdatedAt             time.Time            `json:"updatedAt"`
	KmsSecretManagerKeyId string               `json:"kmsSecretManagerKeyId"`
	HasDeleteProtection   bool                 `json:"hasDeleteProtection"`
	Version               int64                `json:"version"`
	UpgradeStatus         string               `json:"upgradeStatus"`
	Environments          []ProjectEnvironment `json:"environments"`
	AuditLogRetentionDays int64                `json:"auditLogsRetentionDays"`
}

type GetProjectByIdResponse struct {
	Workspace ProjectWithEnvironments `json:"workspace"`
}

type ProjectMemberships struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	UserID    string    `json:"userId"`
	ProjectID string    `json:"projectId"`
}

type ProjectEnvironment struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	ID   string `json:"id"`
}

type ProjectEnvironmentWithPosition struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Position  int64  `json:"position"`
	ProjectID string `json:"projectId"`
}

type SecretFolder struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	EnvID string `json:"envId"`
	Path  string `json:"path"`
}

type SecretFolderByID struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	EnvID       string `json:"envId"`
	ProjectID   string `json:"projectId"`
	Path        string `json:"path"`
	Environment struct {
		EnvID   string `json:"envId"`
		EnvName string `json:"envName"`
		EnvSlug string `json:"envSlug"`
	} `json:"environment"`
}

type SecretImport struct {
	ID         string `json:"id"`
	SecretPath string `json:"secretPath"`
	ImportPath string `json:"importPath"`
}

type SecretImportByID struct {
	ID            string `json:"id"`
	SecretPath    string `json:"secretPath"`
	ImportPath    string `json:"importPath"`
	ProjectID     string `json:"projectId"`
	IsReplication bool   `json:"isReplication"`

	// Imported from
	ImportEnvironment struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Slug string `json:"slug"`
	} `json:"importEnv"`

	// Imported into
	Environment struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Slug string `json:"slug"`
	} `json:"environment"`
}

type CreateProjectResponse struct {
	Project Project `json:"project"`
}

type InviteUsersToProjectResponse struct {
	Members []ProjectMemberships `json:"memberships"`
}

type DeleteProjectResponse struct {
	Project Project `json:"workspace"`
}

type UpdateProjectResponse Project

type GetEncryptedSecretsV3Response struct {
	Secrets []EncryptedSecretV3 `json:"secrets"`
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

type MachineIdentityAuthResponse struct {
	AccessToken       string `json:"accessToken"`
	ExpiresIn         int    `json:"expiresIn"`
	AccessTokenMaxTTL int    `json:"accessTokenMaxTTL"`
	TokenType         string `json:"tokenType"`
}

type SingleEnvironmentVariable struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type"`
	ID    string `json:"_id"`
	Tags  []struct {
		ID        string `json:"_id"`
		Name      string `json:"name"`
		Slug      string `json:"slug"`
		Workspace string `json:"workspace"`
	} `json:"tags"`
	Comment string `json:"comment"`
}

// Workspace key request.
type GetEncryptedWorkspaceKeyRequest struct {
	WorkspaceId string `json:"workspaceId"`
}

// Workspace key response.
type GetEncryptedWorkspaceKeyResponse struct {
	ID           string `json:"_id"`
	EncryptedKey string `json:"encryptedKey"`
	Nonce        string `json:"nonce"`
	Sender       struct {
		ID             string    `json:"_id"`
		Email          string    `json:"email"`
		RefreshVersion int       `json:"refreshVersion"`
		CreatedAt      time.Time `json:"createdAt"`
		UpdatedAt      time.Time `json:"updatedAt"`
		V              int       `json:"__v"`
		FirstName      string    `json:"firstName"`
		LastName       string    `json:"lastName"`
		PublicKey      string    `json:"publicKey"`
	} `json:"sender"`
	Receiver  string    `json:"receiver"`
	Workspace string    `json:"workspace"`
	V         int       `json:"__v"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// encrypted secret.
type EncryptedSecret struct {
	ID                      string `json:"id"`
	SecretName              string `json:"secretName"`
	WorkspaceID             string `json:"workspaceId"`
	Type                    string `json:"type"`
	Environment             string `json:"environment"`
	SecretKeyCiphertext     string `json:"secretKeyCiphertext"`
	SecretKeyIV             string `json:"secretKeyIV"`
	SecretKeyTag            string `json:"secretKeyTag"`
	SecretValueCiphertext   string `json:"secretValueCiphertext"`
	SecretValueIV           string `json:"secretValueIV"`
	SecretValueTag          string `json:"secretValueTag"`
	SecretCommentCiphertext string `json:"secretCommentCiphertext"`
	SecretCommentIV         string `json:"secretCommentIV"`
	SecretCommentTag        string `json:"secretCommentTag"`
	SecretPath              string `json:"secretPath"`
}

// create secrets.
type CreateSecretV3Request struct {
	SecretName              string   `json:"secretName"`
	WorkspaceID             string   `json:"workspaceId"`
	Type                    string   `json:"type"`
	Environment             string   `json:"environment"`
	SecretKeyCiphertext     string   `json:"secretKeyCiphertext"`
	SecretKeyIV             string   `json:"secretKeyIV"`
	SecretKeyTag            string   `json:"secretKeyTag"`
	SecretValueCiphertext   string   `json:"secretValueCiphertext"`
	SecretValueIV           string   `json:"secretValueIV"`
	SecretValueTag          string   `json:"secretValueTag"`
	SecretCommentCiphertext string   `json:"secretCommentCiphertext"`
	SecretCommentIV         string   `json:"secretCommentIV"`
	SecretCommentTag        string   `json:"secretCommentTag"`
	SecretPath              string   `json:"secretPath"`
	TagIDs                  []string `json:"tags"`
}

// delete secret by name api.
type DeleteSecretV3Request struct {
	SecretName  string `json:"secretName"`
	WorkspaceId string `json:"workspaceId"`
	Environment string `json:"environment"`
	Type        string `json:"type"`
	SecretPath  string `json:"secretPath"`
}

// update secret by name api.
type UpdateSecretByNameV3Request struct {
	SecretName            string   `json:"secretName"`
	WorkspaceID           string   `json:"workspaceId"`
	Environment           string   `json:"environment"`
	Type                  string   `json:"type"`
	SecretPath            string   `json:"secretPath"`
	SecretValueCiphertext string   `json:"secretValueCiphertext"`
	SecretValueIV         string   `json:"secretValueIV"`
	SecretValueTag        string   `json:"secretValueTag"`
	TagIDs                []string `json:"tags,omitempty"`
}

// get secret by name api.
type GetSingleSecretByNameV3Request struct {
	SecretName  string `json:"secretName"`
	WorkspaceId string `json:"workspaceId"`
	Environment string `json:"environment"`
	Type        string `json:"type"`
	SecretPath  string `json:"secretPath"`
}

type GetSingleSecretByIDV3Request struct {
	ID string
}

type GetSingleSecretByNameSecretResponse struct {
	Secret EncryptedSecret `json:"secret"`
}

type GetRawSecretsV3Request struct {
	Environment            string `json:"environment"`
	WorkspaceId            string `json:"workspaceId"`
	SecretPath             string `json:"secretPath"`
	ExpandSecretReferences bool   `json:"expandSecretReferences"`
}

type RawV3Secret struct {
	ID            string `json:"id"`
	Version       int    `json:"version"`
	Workspace     string `json:"workspace"`
	Type          string `json:"type"`
	Environment   string `json:"environment"`
	SecretPath    string `json:"secretPath"`
	SecretKey     string `json:"secretKey"`
	SecretValue   string `json:"secretValue"`
	SecretComment string `json:"secretComment"`

	SecretReminderNote       string `json:"secretReminderNote"`
	SecretReminderRepeatDays int64  `json:"secretReminderRepeatDays"`
	Tags                     []struct {
		ID    string `json:"id"`
		Slug  string `json:"slug"`
		Color string `json:"color"`
		Name  string `json:"name"`
	} `json:"tags"`
}

type GetRawSecretsV3Response struct {
	Secrets []RawV3Secret `json:"secrets"`
}

type GetSingleRawSecretByNameSecretResponse struct {
	Secret RawV3Secret `json:"secret"`
}

type GetSingleSecretByIDV3Response = struct {
	Secret struct {
		ID            string `json:"id"`
		Version       int    `json:"version"`
		Workspace     string `json:"workspace"`
		Type          string `json:"type"`
		Environment   string `json:"environment"`
		SecretKey     string `json:"secretKey"`
		SecretValue   string `json:"secretValue"`
		SecretComment string `json:"secretComment"`
		SecretPath    string `json:"secretPath"`

		SecretReminderNote       string `json:"secretReminderNote"`
		SecretReminderRepeatDays int64  `json:"secretReminderRepeatDays"`
		Tags                     []struct {
			ID    string `json:"id"`
			Slug  string `json:"slug"`
			Color string `json:"color"`
			Name  string `json:"name"`
		} `json:"tags"`
	} `json:"secret"`
}

// create secrets.
type CreateRawSecretV3Request struct {
	WorkspaceID              string   `json:"workspaceId"`
	Type                     string   `json:"type"`
	Environment              string   `json:"environment"`
	SecretKey                string   `json:"secretKey"`
	SecretValue              string   `json:"secretValue"`
	SecretComment            string   `json:"secretComment"`
	SecretPath               string   `json:"secretPath"`
	SecretReminderNote       string   `json:"secretReminderNote"`
	SecretReminderRepeatDays int64    `json:"secretReminderRepeatDays"`
	TagIDs                   []string `json:"tagIds"`
}

type DeleteRawSecretV3Request struct {
	SecretName  string `json:"secretName"`
	WorkspaceId string `json:"workspaceId"`
	Environment string `json:"environment"`
	Type        string `json:"type"`
	SecretPath  string `json:"secretPath"`
}

// update secret by name api.
type UpdateRawSecretByNameV3Request struct {
	SecretName               string   `json:"secretName"`
	WorkspaceID              string   `json:"workspaceId"`
	Environment              string   `json:"environment"`
	Type                     string   `json:"type"`
	SecretPath               string   `json:"secretPath"`
	SecretReminderNote       string   `json:"secretReminderNote"`
	SecretReminderRepeatDays int64    `json:"secretReminderRepeatDays"`
	SecretValue              string   `json:"secretValue"`
	TagIDs                   []string `json:"tagIds"`
}

type CreateProjectRequest struct {
	ProjectName             string `json:"projectName"`
	ProjectDescription      string `json:"projectDescription,omitempty"`
	Slug                    string `json:"slug"`
	OrganizationSlug        string `json:"organizationSlug"`
	Template                string `json:"template,omitempty"`
	KmsSecretManagerKeyId   string `json:"kmsKeyId,omitempty"`
	ShouldCreateDefaultEnvs bool   `json:"shouldCreateDefaultEnvs"`
	HasDeleteProtection     bool   `json:"hasDeleteProtection"`
}

type DeleteProjectRequest struct {
	Slug string `json:"slug"`
}

type GetProjectRequest struct {
	Slug string `json:"slug"`
}

type GetProjectByIdRequest struct {
	ID string `json:"id"`
}

type UpdateProjectRequest struct {
	Slug                string `json:"slug"`
	ProjectName         string `json:"name"`
	ProjectDescription  string `json:"description"`
	HasDeleteProtection bool   `json:"hasDeleteProtection"`
}

type UpdateProjectAuditLogRetentionRequest struct {
	ProjectSlug string
	Days        int64 `json:"auditLogsRetentionDays"`
}

type UpdateProjectAuditLogRetentionResponse struct {
	Project Project `json:"workspace"`
}

type InviteUsersToProjectRequest struct {
	ProjectID string   `json:"projectId"`
	Usernames []string `json:"usernames"`
}

type CreateProjectUserRequest struct {
	ProjectID string   `json:"projectId"`
	Username  []string `json:"usernames"`
}

type CreateProjectUserResponse struct {
	Memberships []CreateProjectUserResponseMembers `json:"memberships"`
}

type CreateProjectUserResponseMembers struct {
	ID     string `json:"id"`
	UserId string `json:"userId"`
}

type GetProjectUserByUserNameRequest struct {
	ProjectID string `json:"projectId"`
	Username  string `json:"username"`
}

type GetProjectUserByUserNameResponse struct {
	Membership ProjectUser `json:"membership"`
}

type UpdateProjectUserRequest struct {
	ProjectID    string                          `json:"projectId"`
	MembershipID string                          `json:"membershipId"`
	Roles        []UpdateProjectUserRequestRoles `json:"roles"`
}

type UpdateProjectUserRequestRoles struct {
	Role                     string    `json:"role"`
	IsTemporary              bool      `json:"isTemporary"`
	TemporaryMode            string    `json:"temporaryMode"`
	TemporaryRange           string    `json:"temporaryRange"`
	TemporaryAccessStartTime time.Time `json:"temporaryAccessStartTime"`
}

type UpdateProjectUserResponse struct {
	Roles []struct {
		ID                       string    `json:"id"`
		Role                     string    `json:"role"`
		ProjectMembershipId      string    `json:"projectMembershipId"`
		CustomRoleId             string    `json:"customRoleId"`
		IsTemporary              bool      `json:"isTemporary"`
		TemporaryMode            string    `json:"temporaryMode"`
		TemporaryRange           string    `json:"temporaryRange"`
		TemporaryAccessStartTime time.Time `json:"temporaryAccessStartTime"`
		TemporaryAccessEndTime   time.Time `json:"temporaryAccessEndTime"`
		CreatedAt                time.Time `json:"createdAt"`
		UpdatedAt                time.Time `json:"updatedAt"`
	} `json:"roles"`
}

type DeleteProjectUserRequest struct {
	ProjectID string   `json:"projectId"`
	Username  []string `json:"usernames"`
}

type DeleteProjectUserResponse struct {
	Memberships []DeleteProjectUserResponseMembers `json:"memberships"`
}

type DeleteProjectUserResponseMembers struct {
	ID     string `json:"id"`
	UserId string `json:"userId"`
}

// identity.
type CreateProjectIdentityRequest struct {
	ProjectID  string                              `json:"projectId"`
	IdentityID string                              `json:"identityId"`
	Roles      []CreateProjectIdentityRequestRoles `json:"roles"`
}

type CreateProjectIdentityRequestRoles struct {
	Role                     string    `json:"role"`
	IsTemporary              bool      `json:"isTemporary"`
	TemporaryMode            string    `json:"temporaryMode"`
	TemporaryRange           string    `json:"temporaryRange"`
	TemporaryAccessStartTime time.Time `json:"temporaryAccessStartTime"`
}

type CreateProjectIdentityResponse struct {
	Membership CreateProjectIdentityResponseMembers `json:"identityMembership"`
}

type CreateProjectIdentityResponseMembers struct {
	ID         string `json:"id"`
	IdentityId string `json:"identityId"`
}

type GetProjectIdentityByIDRequest struct {
	ProjectID  string `json:"projectId"`
	IdentityID string `json:"identityId"`
}

type GetProjectIdentityByMembershipIDRequest struct {
	MembershipID string
}

type GetProjectIdentityByIDResponse struct {
	Membership ProjectIdentity `json:"identityMembership"`
}

type UpdateProjectIdentityRequest struct {
	ProjectID  string                              `json:"projectId"`
	IdentityID string                              `json:"identityId"`
	Roles      []UpdateProjectIdentityRequestRoles `json:"roles"`
}

type UpdateProjectIdentityRequestRoles struct {
	Role                     string    `json:"role"`
	IsTemporary              bool      `json:"isTemporary"`
	TemporaryMode            string    `json:"temporaryMode"`
	TemporaryRange           string    `json:"temporaryRange"`
	TemporaryAccessStartTime time.Time `json:"temporaryAccessStartTime"`
}

type UpdateProjectIdentityResponse struct {
	Roles []struct {
		ID                       string    `json:"id"`
		Role                     string    `json:"role"`
		ProjectMembershipId      string    `json:"projectMembershipId"`
		CustomRoleId             string    `json:"customRoleId"`
		IsTemporary              bool      `json:"isTemporary"`
		TemporaryMode            string    `json:"temporaryMode"`
		TemporaryRange           string    `json:"temporaryRange"`
		TemporaryAccessStartTime time.Time `json:"temporaryAccessStartTime"`
		TemporaryAccessEndTime   time.Time `json:"temporaryAccessEndTime"`
		CreatedAt                time.Time `json:"createdAt"`
		UpdatedAt                time.Time `json:"updatedAt"`
	} `json:"roles"`
}

type DeleteProjectIdentityRequest struct {
	ProjectID  string `json:"projectId"`
	IdentityID string `json:"identityId"`
}

type DeleteProjectIdentityResponse struct {
	Membership DeleteProjectIdentityResponseIdentities `json:"identityMembership"`
}

type DeleteProjectIdentityResponseIdentities struct {
	ID         string `json:"id"`
	IdentityID string `json:"identityId"`
}

type CreateProjectRoleRequest struct {
	ProjectSlug string                         `json:"projectSlug"`
	Slug        string                         `json:"slug"`
	Name        string                         `json:"name"`
	Description string                         `json:"description"`
	Permissions []ProjectRolePermissionRequest `json:"permissions"`
}

type CreateProjectRoleResponse struct {
	Role ProjectRole `json:"role"`
}

type CreateProjectRoleV2Request struct {
	ProjectId   string
	Slug        string                   `json:"slug"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Permissions []map[string]interface{} `json:"permissions"`
}

type CreateProjectRoleV2Response struct {
	Role ProjectRole `json:"role"`
}

type UpdateProjectRoleRequest struct {
	ProjectSlug string                         `json:"projectSlug"`
	RoleId      string                         `json:"roleId"`
	Slug        string                         `json:"slug"`
	Name        string                         `json:"name"`
	Description string                         `json:"description"`
	Permissions []ProjectRolePermissionRequest `json:"permissions"`
}

type UpdateProjectRoleResponse struct {
	Role ProjectRole `json:"role"`
}

type UpdateProjectRoleV2Request struct {
	ProjectId   string
	RoleId      string
	Slug        string                   `json:"slug"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Permissions []map[string]interface{} `json:"permissions"`
}

type UpdateProjectRoleV2Response struct {
	Role ProjectRole `json:"role"`
}

type DeleteProjectRoleRequest struct {
	ProjectSlug string `json:"projectSlug"`
	RoleId      string `json:"roleId"`
}

type DeleteProjectRoleResponse struct {
	Role ProjectRole `json:"role"`
}

type GetProjectRoleBySlugRequest struct {
	ProjectSlug string `json:"projectSlug"`
	RoleSlug    string `json:"roleSlug"`
}

type GetProjectRoleBySlugResponse struct {
	Role ProjectRole `json:"role"`
}

type GetProjectRoleBySlugV2Request struct {
	ProjectId string
	RoleSlug  string
}

type GetProjectRoleBySlugV2Response struct {
	Role ProjectRole `json:"role"`
}

type ProjectRolePermissionRequest struct {
	Action     string         `json:"action"`
	Subject    string         `json:"subject"`
	Conditions map[string]any `json:"conditions,omitempty"`
}

type ProjectSpecificPrivilegePermissionRequest struct {
	Actions    []string       `json:"actions"`
	Subject    string         `json:"subject"`
	Conditions map[string]any `json:"conditions,omitempty"`
}

type CreatePermanentProjectIdentitySpecificPrivilegeRequest struct {
	ProjectSlug string                                    `json:"projectSlug"`
	IdentityId  string                                    `json:"identityId"`
	Slug        string                                    `json:"slug,omitempty"`
	Permissions ProjectSpecificPrivilegePermissionRequest `json:"privilegePermission"`
}

type CreateTemporaryProjectIdentitySpecificPrivilegeRequest struct {
	ProjectSlug              string                                    `json:"projectSlug"`
	IdentityId               string                                    `json:"identityId"`
	Slug                     string                                    `json:"slug,omitempty"`
	Permissions              ProjectSpecificPrivilegePermissionRequest `json:"privilegePermission"`
	TemporaryMode            string                                    `json:"temporaryMode"`
	TemporaryRange           string                                    `json:"temporaryRange"`
	TemporaryAccessStartTime time.Time                                 `json:"temporaryAccessStartTime"`
}

type CreateProjectIdentitySpecificPrivilegeV2Type struct {
	IsTemporary              bool      `json:"isTemporary"`
	TemporaryMode            string    `json:"temporaryMode"`
	TemporaryRange           string    `json:"temporaryRange"`
	TemporaryAccessStartTime time.Time `json:"temporaryAccessStartTime"`
}

type CreateProjectIdentitySpecificPrivilegeV2Request struct {
	ProjectId   string                                       `json:"projectId"`
	IdentityId  string                                       `json:"identityId"`
	Slug        string                                       `json:"slug,omitempty"`
	Permissions []map[string]interface{}                     `json:"permissions"`
	Type        CreateProjectIdentitySpecificPrivilegeV2Type `json:"type"`
}

type CreateProjectIdentitySpecificPrivilegeResponse struct {
	Privilege ProjectIdentitySpecificPrivilege `json:"privilege"`
}

type CreateProjectIdentitySpecificPrivilegeV2Response struct {
	Privilege ProjectIdentitySpecificPrivilege `json:"privilege"`
}

type UpdateProjectIdentitySpecificPrivilegeRequest struct {
	ProjectSlug   string                                            `json:"projectSlug"`
	IdentityId    string                                            `json:"identityId"`
	PrivilegeSlug string                                            `json:"privilegeSlug,omitempty"`
	Details       UpdateProjectIdentitySpecificPrivilegeDataRequest `json:"privilegeDetails"`
}

type UpdateProjectIdentitySpecificPrivilegeDataRequest struct {
	Slug                     string                                    `json:"slug,omitempty"`
	Permissions              ProjectSpecificPrivilegePermissionRequest `json:"privilegePermission"`
	IsTemporary              bool                                      `json:"isTemporary"`
	TemporaryMode            string                                    `json:"temporaryMode,omitempty"`
	TemporaryRange           string                                    `json:"temporaryRange,omitempty"`
	TemporaryAccessStartTime time.Time                                 `json:"temporaryAccessStartTime,omitempty"`
}

type UpdateProjectIdentitySpecificPrivilegeV2Type struct {
	IsTemporary              bool      `json:"isTemporary"`
	TemporaryMode            string    `json:"temporaryMode,omitempty"`
	TemporaryRange           string    `json:"temporaryRange,omitempty"`
	TemporaryAccessStartTime time.Time `json:"temporaryAccessStartTime,omitempty"`
}

type UpdateProjectIdentitySpecificPrivilegeV2Request struct {
	ID          string
	Slug        string                                       `json:"slug,omitempty"`
	Permissions []map[string]interface{}                     `json:"permissions"`
	Type        UpdateProjectIdentitySpecificPrivilegeV2Type `json:"type"`
}

type UpdateProjectIdentitySpecificPrivilegeV2Response struct {
	Privilege ProjectIdentitySpecificPrivilege `json:"privilege"`
}

type UpdateProjectIdentitySpecificPrivilegeResponse struct {
	Privilege ProjectIdentitySpecificPrivilege `json:"privilege"`
}

type DeleteProjectIdentitySpecificPrivilegeRequest struct {
	ProjectSlug   string `json:"projectSlug"`
	IdentityId    string `json:"identityId"`
	PrivilegeSlug string `json:"privilegeSlug,omitempty"`
}

type DeleteProjectIdentitySpecificPrivilegeResponse struct {
	Privilege ProjectIdentitySpecificPrivilege `json:"privilege"`
}

type GetProjectIdentitySpecificPrivilegeRequest struct {
	ProjectSlug   string `json:"projectSlug"`
	IdentityID    string `json:"identityId"`
	PrivilegeSlug string `json:"privilegeSlug,omitempty"`
}

type GetProjectIdentitySpecificPrivilegeV2Request struct {
	ID string
}

type GetProjectIdentitySpecificPrivilegeResponse struct {
	Privilege ProjectIdentitySpecificPrivilege `json:"privilege"`
}

type GetProjectIdentitySpecificPrivilegeV2Response struct {
	Privilege ProjectIdentitySpecificPrivilege `json:"privilege"`
}

type ProjectGroupRole struct {
	ID                       string    `json:"id"`
	Role                     string    `json:"role"`
	CustomRoleSlug           string    `json:"customRoleSlug"`
	CustomRoleId             string    `json:"customRoleId"`
	IsTemporary              bool      `json:"isTemporary"`
	TemporaryMode            string    `json:"temporaryMode"`
	TemporaryRange           string    `json:"temporaryRange"`
	TemporaryAccessStartTime time.Time `json:"temporaryAccessStartTime"`
	TemporaryAccessEndTime   time.Time `json:"temporaryAccessEndTime"`
	CreatedAt                time.Time `json:"createdAt"`
	UpdatedAt                time.Time `json:"updatedAt"`
}

type ProjectGroup struct {
	ID      string `json:"id"`
	GroupID string `json:"groupId"`
	Roles   []ProjectGroupRole
}

type CreateProjectGroupRequestRoles struct {
	Role                     string    `json:"role"`
	IsTemporary              bool      `json:"isTemporary"`
	TemporaryMode            string    `json:"temporaryMode"`
	TemporaryRange           string    `json:"temporaryRange"`
	TemporaryAccessStartTime time.Time `json:"temporaryAccessStartTime"`
}

type CreateProjectGroupRequest struct {
	GroupIdOrName string
	ProjectId     string                           `json:"projectId"`
	Roles         []CreateProjectGroupRequestRoles `json:"roles"`
}

type CreateProjectGroupResponseMembers struct {
	ID      string `json:"id"`
	GroupID string `json:"groupId"`
}

type CreateProjectGroupResponse struct {
	Membership CreateProjectGroupResponseMembers `json:"groupMembership"`
}

type GetProjectGroupMembershipRequest struct {
	ProjectId string `json:"projectId"`
	GroupId   string `json:"groupId"`
}

type GetProjectGroupMembershipResponse struct {
	Membership ProjectGroup `json:"groupMembership"`
}

type UpdateProjectGroupRequestRoles struct {
	Role                     string    `json:"role"`
	IsTemporary              bool      `json:"isTemporary"`
	TemporaryMode            string    `json:"temporaryMode"`
	TemporaryRange           string    `json:"temporaryRange"`
	TemporaryAccessStartTime time.Time `json:"temporaryAccessStartTime"`
}

type UpdateProjectGroupRequest struct {
	ProjectId string                           `json:"projectId"`
	GroupId   string                           `json:"groupId"`
	Roles     []UpdateProjectGroupRequestRoles `json:"roles"`
}

type UpdateProjectGroupResponse struct {
	Roles []struct {
		ID                       string    `json:"id"`
		Role                     string    `json:"role"`
		CustomRoleId             string    `json:"customRoleId"`
		IsTemporary              bool      `json:"isTemporary"`
		TemporaryMode            string    `json:"temporaryMode"`
		TemporaryRange           string    `json:"temporaryRange"`
		TemporaryAccessStartTime time.Time `json:"temporaryAccessStartTime"`
		TemporaryAccessEndTime   time.Time `json:"temporaryAccessEndTime"`
		CreatedAt                time.Time `json:"createdAt"`
		UpdatedAt                time.Time `json:"updatedAt"`
	} `json:"roles"`
}

type DeleteProjectGroupRequest struct {
	ProjectId string `json:"projectId"`
	GroupId   string `json:"groupId"`
}

type DeleteProjectGroupResponseMembers struct {
	ID      string `json:"id"`
	GroupID string `json:"groupId"`
}

type DeleteProjectGroupResponse struct {
	Membership DeleteProjectGroupResponseMembers `json:"groupMembership"`
}

type GetGroupByIdRequest struct {
	ID string `json:"id"`
}

type Group struct {
	ID             string `json:"id"`
	OrgID          string `json:"orgId"`
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	Role           string `json:"role"`
	RoleId         string `json:"roleId"`
	CustomRoleSlug string `json:"customRoleSlug"`
}

type GetGroupsResponse []Group

type GetProjectTagsResponse struct {
	Tags []ProjectTag `json:"workspaceTags"`
}

type GetProjectTagsRequest struct {
	ProjectID string `json:"projectId"`
}

type CreateProjectTagRequest struct {
	Name      string `json:"name"`
	Color     string `json:"color"`
	Slug      string `json:"slug"`
	ProjectID string `json:"projectId"`
}

type CreateProjectTagResponse struct {
	Tag ProjectTag `json:"workspaceTag"`
}

type UpdateProjectTagRequest struct {
	Name      string `json:"name,omitempty"`
	Color     string `json:"color,omitempty"`
	Slug      string `json:"slug,omitempty"`
	ProjectID string `json:"projectId"`
	TagID     string `json:"tagId"`
}

type UpdateProjectTagResponse struct {
	Tag ProjectTag `json:"workspaceTag"`
}

type DeleteProjectTagRequest struct {
	ProjectID string `json:"projectId"`
	TagID     string `json:"tagId"`
}

type DeleteProjectTagResponse struct {
	Tag ProjectTag `json:"workspaceTag"`
}

type GetProjectTagByIDRequest struct {
	ProjectID string `json:"projectId"`
	TagID     string `json:"tagId"`
}

type GetProjectTagByIDResponse struct {
	Tag ProjectTag `json:"workspaceTag"`
}

type GetProjectTagBySlugRequest struct {
	ProjectID string `json:"projectId"`
	TagSlug   string `json:"tagSlug"`
}

type GetProjectTagBySlugResponse struct {
	Tag ProjectTag `json:"workspaceTag"`
}

type CreateSecretFolderRequest struct {
	Name        string `json:"name"`
	Environment string `json:"environment"`
	ProjectID   string `json:"workspaceId"`
	SecretPath  string `json:"path"`
}

type CreateSecretFolderResponse struct {
	Folder SecretFolder `json:"folder"`
}

type UpdateSecretFolderRequest struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Environment string `json:"environment"`
	ProjectID   string `json:"workspaceId"`
	SecretPath  string `json:"path"`
}

type UpdateSecretFolderResponse struct {
	Folder SecretFolder `json:"folder"`
}

type DeleteSecretFolderRequest struct {
	ID          string `json:"id"`
	Environment string `json:"environment"`
	ProjectID   string `json:"workspaceId"`
	SecretPath  string `json:"path"`
}

type DeleteSecretFolderResponse struct {
	Folder SecretFolder `json:"folder"`
}

type GetSecretFolderByIDRequest struct {
	ID string `json:"id"`
}

type GetSecretFolderByIDResponse struct {
	Folder SecretFolderByID `json:"folder"`
}

type GetSecretFolderByPathRequest struct {
	ProjectID   string
	Environment string
	SecretPath  string
}

type GetSecretFolderByPathResponse struct {
	Folder SecretFolder `json:"folder"`
}

type ListSecretFolderRequest struct {
	Environment string `json:"environment"`
	ProjectID   string `json:"workspaceId"`
	SecretPath  string `json:"path"`
}

type ListSecretFolderResponse struct {
	Folders []SecretFolder `json:"folders"`
}

type CreateProjectEnvironmentRequest struct {
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	ProjectID string `json:"workspaceId"`
	Position  int64  `json:"position,omitempty"`
}

type CreateProjectEnvironmentResponse struct {
	Environment ProjectEnvironmentWithPosition `json:"environment"`
}

type DeleteProjectEnvironmentRequest struct {
	ID        string `json:"id"`
	ProjectID string `json:"workspaceId"`
}

type DeleteProjectEnvironmentResponse struct {
	Environment ProjectEnvironmentWithPosition `json:"environment"`
}

type GetProjectEnvironmentByIDRequest struct {
	ID string
}

type GetProjectEnvironmentByIDResponse struct {
	Environment ProjectEnvironmentWithPosition `json:"environment"`
}

type UpdateProjectEnvironmentRequest struct {
	ID        string
	ProjectID string
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Position  int64  `json:"position"`
}

type UpdateProjectEnvironmentResponse struct {
	Environment ProjectEnvironmentWithPosition `json:"environment"`
}

// Different from Identity because metadata is only included on post/patch requests.
type CreateUpdateIdentity struct {
	Name                string      `json:"name"`
	HasDeleteProtection bool        `json:"hasDeleteProtection"`
	ID                  string      `json:"id"`
	AuthMethods         []string    `json:"authMethods"`
	CreatedAt           time.Time   `json:"createdAt"`
	UpdatedAt           time.Time   `json:"updatedAt"`
	Metadata            []MetaEntry `json:"metadata"`
}

type CreateMetaEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type CreateIdentityRequest struct {
	Name                string            `json:"name"`
	HasDeleteProtection bool              `json:"hasDeleteProtection"`
	OrgID               string            `json:"organizationId"`
	Role                string            `json:"role"`
	Metadata            []CreateMetaEntry `json:"metadata,omitempty"`
}

type CreateIdentityResponse struct {
	Identity CreateUpdateIdentity `json:"identity"`
}

type UpdateIdentityRequest struct {
	IdentityID          string            `json:"identityId"`
	Name                string            `json:"name,omitempty"`
	HasDeleteProtection bool              `json:"hasDeleteProtection"`
	Role                string            `json:"role,omitempty"`
	Metadata            []CreateMetaEntry `json:"metadata"`
}

type UpdateIdentityResponse struct {
	Identity CreateUpdateIdentity `json:"identity"`
}

type DeleteIdentityRequest struct {
	IdentityID string `json:"identityId"`
}

type DeleteIdentityResponse struct {
	Identity Identity `json:"identity"`
}

type GetIdentityRequest struct {
	IdentityID string `json:"identityId"`
}

type GetIdentityResponse struct {
	Identity OrgIdentity `json:"identity"`
}

type IdentityAuthTrustedIpRequest struct {
	IPAddress string `json:"ipAddress"`
}

type CreateIdentityUniversalAuthRequest struct {
	ClientSecretTrustedIPs  []IdentityAuthTrustedIpRequest `json:"clientSecretTrustedIps,omitempty"`
	AccessTokenTrustedIPs   []IdentityAuthTrustedIpRequest `json:"accessTokenTrustedIps,omitempty"`
	AccessTokenTTL          int64                          `json:"accessTokenTTL,omitempty"`
	AccessTokenMaxTTL       int64                          `json:"accessTokenMaxTTL,omitempty"`
	AccessTokenNumUsesLimit int64                          `json:"accessTokenNumUsesLimit,omitempty"`
	IdentityID              string                         `json:"identityId"`
}

type CreateIdentityUniversalAuthResponse struct {
	UniversalAuth IdentityUniversalAuth `json:"identityUniversalAuth"`
}

type UpdateIdentityUniversalAuthRequest struct {
	ClientSecretTrustedIPs  []IdentityAuthTrustedIpRequest `json:"clientSecretTrustedIps,omitempty"`
	AccessTokenTrustedIPs   []IdentityAuthTrustedIpRequest `json:"accessTokenTrustedIps,omitempty"`
	AccessTokenTTL          int64                          `json:"accessTokenTTL,omitempty"`
	AccessTokenMaxTTL       int64                          `json:"accessTokenMaxTTL,omitempty"`
	AccessTokenNumUsesLimit int64                          `json:"accessTokenNumUsesLimit,omitempty"`
	IdentityID              string                         `json:"identityId"`
}

type UpdateIdentityUniversalAuthResponse struct {
	UniversalAuth IdentityUniversalAuth `json:"identityUniversalAuth"`
}

type RevokeIdentityUniversalAuthRequest struct {
	IdentityID string `json:"identityId"`
}

type RevokeIdentityUniversalAuthResponse struct {
	UniversalAuth IdentityUniversalAuth `json:"identityUniversalAuth"`
}

type GetIdentityUniversalAuthRequest struct {
	IdentityID string `json:"identityId"`
}

type GetIdentityUniversalAuthResponse struct {
	UniversalAuth IdentityUniversalAuth `json:"identityUniversalAuth"`
}

type CreateIdentityUniversalAuthClientSecretRequest struct {
	IdentityID   string `json:"identityId"`
	Description  string `json:"description"`
	NumUsesLimit int64  `json:"numUsesLimit"`
	TTL          int64  `json:"ttl"`
}

type CreateIdentityUniversalAuthClientSecretResponse struct {
	ClientSecret     string                            `json:"clientSecret"`
	ClientSecretData IdentityUniversalAuthClientSecret `json:"clientSecretData"`
}

type GetIdentityUniversalAuthClientSecretRequest struct {
	IdentityID     string `json:"identityId"`
	ClientSecretID string `json:"clientSecretId"`
}

type GetIdentityUniversalAuthClientSecretResponse struct {
	ClientSecretData IdentityUniversalAuthClientSecret `json:"clientSecretData"`
}

type RevokeIdentityUniversalAuthClientSecretRequest struct {
	IdentityID     string `json:"identityId"`
	ClientSecretID string `json:"clientSecretId"`
}

type RevokeIdentityUniversalAuthClientSecretResponse struct {
	ClientSecretData IdentityUniversalAuthClientSecret `json:"clientSecretData"`
}

type CreateIdentityAwsAuthRequest struct {
	IdentityID              string                         `json:"identityId"`
	StsEndpoint             string                         `json:"stsEndpoint,omitempty"`
	AllowedPrincipalArns    string                         `json:"allowedPrincipalArns,omitempty"`
	AllowedAccountIDS       string                         `json:"allowedAccountIds,omitempty"`
	AccessTokenTrustedIPS   []IdentityAuthTrustedIpRequest `json:"accessTokenTrustedIps,omitempty"`
	AccessTokenTTL          int64                          `json:"accessTokenTTL"`
	AccessTokenMaxTTL       int64                          `json:"accessTokenMaxTTL"`
	AccessTokenNumUsesLimit int64                          `json:"accessTokenNumUsesLimit"`
}

type CreateIdentityAwsAuthResponse struct {
	IdentityAwsAuth IdentityAwsAuth `json:"identityAwsAuth"`
}

type UpdateIdentityAwsAuthRequest struct {
	IdentityID              string                         `json:"identityId"`
	StsEndpoint             string                         `json:"stsEndpoint,omitempty"`
	AllowedPrincipalArns    string                         `json:"allowedPrincipalArns,omitempty"`
	AllowedAccountIDS       string                         `json:"allowedAccountIds,omitempty"`
	AccessTokenTrustedIPS   []IdentityAuthTrustedIpRequest `json:"accessTokenTrustedIps,omitempty"`
	AccessTokenTTL          int64                          `json:"accessTokenTTL"`
	AccessTokenMaxTTL       int64                          `json:"accessTokenMaxTTL"`
	AccessTokenNumUsesLimit int64                          `json:"accessTokenNumUsesLimit"`
}

type UpdateIdentityAwsAuthResponse struct {
	IdentityAwsAuth IdentityAwsAuth `json:"identityAwsAuth"`
}

type GetIdentityAwsAuthRequest struct {
	IdentityID string `json:"identityId"`
}

type GetIdentityAwsAuthResponse struct {
	IdentityAwsAuth IdentityAwsAuth `json:"identityAwsAuth"`
}

type RevokeIdentityAwsAuthRequest struct {
	IdentityID string `json:"identityId"`
}

type RevokeIdentityAwsAuthResponse struct {
	IdentityAwsAuth IdentityAwsAuth `json:"identityAwsAuth"`
}

type CreateIdentityAzureAuthRequest struct {
	IdentityID                 string                         `json:"identityId"`
	TenantID                   string                         `json:"tenantId"`
	Resource                   string                         `json:"resource"`
	AllowedServicePrincipalIDS string                         `json:"allowedServicePrincipalIds,omitempty"`
	AccessTokenTrustedIPS      []IdentityAuthTrustedIpRequest `json:"accessTokenTrustedIps,omitempty"`
	AccessTokenTTL             int64                          `json:"accessTokenTTL,omitempty"`
	AccessTokenMaxTTL          int64                          `json:"accessTokenMaxTTL,omitempty"`
	AccessTokenNumUsesLimit    int64                          `json:"accessTokenNumUsesLimit,omitempty"`
}

type CreateIdentityAzureAuthResponse struct {
	IdentityAzureAuth IdentityAzureAuth `json:"identityAzureAuth"`
}

type UpdateIdentityAzureAuthRequest struct {
	IdentityID                 string                         `json:"identityId"`
	TenantID                   string                         `json:"tenantId"`
	Resource                   string                         `json:"resource,omitempty"`
	AllowedServicePrincipalIDS string                         `json:"allowedServicePrincipalIds"`
	AccessTokenTrustedIPS      []IdentityAuthTrustedIpRequest `json:"accessTokenTrustedIps,omitempty"`
	AccessTokenTTL             int64                          `json:"accessTokenTTL,omitempty"`
	AccessTokenMaxTTL          int64                          `json:"accessTokenMaxTTL,omitempty"`
	AccessTokenNumUsesLimit    int64                          `json:"accessTokenNumUsesLimit,omitempty"`
}

type UpdateIdentityAzureAuthResponse struct {
	IdentityAzureAuth IdentityAzureAuth `json:"identityAzureAuth"`
}

type GetIdentityAzureAuthRequest struct {
	IdentityID string `json:"identityId"`
}

type GetIdentityAzureAuthResponse struct {
	IdentityAzureAuth IdentityAzureAuth `json:"identityAzureAuth"`
}

type RevokeIdentityAzureAuthRequest struct {
	IdentityID string `json:"identityId"`
}

type RevokeIdentityAzureAuthResponse struct {
	IdentityAzureAuth IdentityAzureAuth `json:"identityAzureAuth"`
}

type CreateIdentityGcpAuthRequest struct {
	IdentityID              string                         `json:"identityId"`
	Type                    string                         `json:"type"`
	AllowedServiceAccounts  string                         `json:"allowedServiceAccounts,omitempty"`
	AllowedProjects         string                         `json:"allowedProjects,omitempty"`
	AllowedZones            string                         `json:"allowedZones,omitempty"`
	AccessTokenTrustedIPS   []IdentityAuthTrustedIpRequest `json:"accessTokenTrustedIps,omitempty"`
	AccessTokenTTL          int64                          `json:"accessTokenTTL,omitempty"`
	AccessTokenMaxTTL       int64                          `json:"accessTokenMaxTTL,omitempty"`
	AccessTokenNumUsesLimit int64                          `json:"accessTokenNumUsesLimit,omitempty"`
}

type CreateIdentityGcpAuthResponse struct {
	IdentityGcpAuth IdentityGcpAuth `json:"identityGcpAuth"`
}

type UpdateIdentityGcpAuthRequest struct {
	IdentityID              string                         `json:"identityId"`
	Type                    string                         `json:"type"`
	AllowedServiceAccounts  string                         `json:"allowedServiceAccounts"`
	AllowedProjects         string                         `json:"allowedProjects"`
	AllowedZones            string                         `json:"allowedZones"`
	AccessTokenTrustedIPS   []IdentityAuthTrustedIpRequest `json:"accessTokenTrustedIps,omitempty"`
	AccessTokenTTL          int64                          `json:"accessTokenTTL,omitempty"`
	AccessTokenMaxTTL       int64                          `json:"accessTokenMaxTTL,omitempty"`
	AccessTokenNumUsesLimit int64                          `json:"accessTokenNumUsesLimit,omitempty"`
}

type UpdateIdentityGcpAuthResponse struct {
	IdentityGcpAuth IdentityGcpAuth `json:"identityGcpAuth"`
}

type GetIdentityGcpAuthRequest struct {
	IdentityID string `json:"identityId"`
}

type GetIdentityGcpAuthResponse struct {
	IdentityGcpAuth IdentityGcpAuth `json:"identityGcpAuth"`
}

type RevokeIdentityGcpAuthRequest struct {
	IdentityID string `json:"identityId"`
}

type RevokeIdentityGcpAuthResponse struct {
	IdentityGcpAuth IdentityGcpAuth `json:"identityGcpAuth"`
}

type CreateIdentityKubernetesAuthRequest struct {
	IdentityID              string                         `json:"identityId"`
	KubernetesHost          string                         `json:"kubernetesHost"`
	CACERT                  string                         `json:"caCert"`
	TokenReviewerJwt        string                         `json:"tokenReviewerJwt"`
	AllowedNamespaces       string                         `json:"allowedNamespaces"`
	AllowedNames            string                         `json:"allowedNames"`
	AllowedAudience         string                         `json:"allowedAudience"`
	AccessTokenTrustedIPS   []IdentityAuthTrustedIpRequest `json:"accessTokenTrustedIps,omitempty"`
	AccessTokenTTL          int64                          `json:"accessTokenTTL,omitempty"`
	AccessTokenMaxTTL       int64                          `json:"accessTokenMaxTTL,omitempty"`
	AccessTokenNumUsesLimit int64                          `json:"accessTokenNumUsesLimit,omitempty"`
}

type CreateIdentityKubernetesAuthResponse struct {
	IdentityKubernetesAuth IdentityKubernetesAuth `json:"identityKubernetesAuth"`
}

type UpdateIdentityKubernetesAuthRequest struct {
	IdentityID              string                         `json:"identityId"`
	KubernetesHost          string                         `json:"kubernetesHost"`
	CACERT                  string                         `json:"caCert"`
	TokenReviewerJwt        string                         `json:"tokenReviewerJwt"`
	AllowedNamespaces       string                         `json:"allowedNamespaces"`
	AllowedNames            string                         `json:"allowedNames"`
	AllowedAudience         string                         `json:"allowedAudience"`
	AccessTokenTrustedIPS   []IdentityAuthTrustedIpRequest `json:"accessTokenTrustedIps,omitempty"`
	AccessTokenTTL          int64                          `json:"accessTokenTTL,omitempty"`
	AccessTokenMaxTTL       int64                          `json:"accessTokenMaxTTL,omitempty"`
	AccessTokenNumUsesLimit int64                          `json:"accessTokenNumUsesLimit,omitempty"`
}

type CreateIdentityOidcAuthResponse struct {
	IdentityOidcAuth IdentityOidcAuth `json:"identityOidcAuth"`
}

type UpdateIdentityOidcAuthResponse struct {
	IdentityOidcAuth IdentityOidcAuth `json:"identityOidcAuth"`
}

type GetIdentityOidcAuthResponse struct {
	IdentityOidcAuth IdentityOidcAuth `json:"identityOidcAuth"`
}

type GetIdentityOidcAuthRequest struct {
	IdentityID string `json:"identityId"`
}

type RevokeIdentityOidcAuthRequest struct {
	IdentityID string `json:"identityId"`
}

type RevokeIdentityOidcAuthResponse struct {
	IdentityOidcAuth IdentityOidcAuth `json:"identityOidcAuth"`
}

type CreateIdentityOidcAuthRequest struct {
	IdentityID              string                         `json:"identityId"`
	OidcDiscoveryUrl        string                         `json:"oidcDiscoveryUrl"`
	CACERT                  string                         `json:"caCert"`
	BoundIssuer             string                         `json:"boundIssuer"`
	BoundAudiences          string                         `json:"boundAudiences"`
	BoundClaims             map[string]string              `json:"boundClaims"`
	ClaimMetadataMapping    map[string]string              `json:"claimMetadataMapping"`
	BoundSubject            string                         `json:"boundSubject"`
	AccessTokenTrustedIPS   []IdentityAuthTrustedIpRequest `json:"accessTokenTrustedIps,omitempty"`
	AccessTokenTTL          int64                          `json:"accessTokenTTL,omitempty"`
	AccessTokenMaxTTL       int64                          `json:"accessTokenMaxTTL,omitempty"`
	AccessTokenNumUsesLimit int64                          `json:"accessTokenNumUsesLimit,omitempty"`
}

type UpdateIdentityOidcAuthRequest struct {
	IdentityID              string                         `json:"identityId"`
	OidcDiscoveryUrl        string                         `json:"oidcDiscoveryUrl"`
	CACERT                  string                         `json:"caCert"`
	BoundIssuer             string                         `json:"boundIssuer"`
	BoundAudiences          string                         `json:"boundAudiences"`
	BoundClaims             map[string]string              `json:"boundClaims"`
	ClaimMetadataMapping    map[string]string              `json:"claimMetadataMapping"`
	BoundSubject            string                         `json:"boundSubject"`
	AccessTokenTrustedIPS   []IdentityAuthTrustedIpRequest `json:"accessTokenTrustedIps,omitempty"`
	AccessTokenTTL          int64                          `json:"accessTokenTTL,omitempty"`
	AccessTokenMaxTTL       int64                          `json:"accessTokenMaxTTL,omitempty"`
	AccessTokenNumUsesLimit int64                          `json:"accessTokenNumUsesLimit,omitempty"`
}

type UpdateIdentityKubernetesAuthResponse struct {
	IdentityKubernetesAuth IdentityKubernetesAuth `json:"identityKubernetesAuth"`
}

type GetIdentityKubernetesAuthRequest struct {
	IdentityID string `json:"identityId"`
}

type GetIdentityKubernetesAuthResponse struct {
	IdentityKubernetesAuth IdentityKubernetesAuth `json:"identityKubernetesAuth"`
}

type RevokeIdentityKubernetesAuthRequest struct {
	IdentityID string `json:"identityId"`
}

type RevokeIdentityKubernetesAuthResponse struct {
	IdentityKubernetesAuth IdentityKubernetesAuth `json:"identityKubernetesAuth"`
}

type CreateIntegrationAuthRequest struct {
	AccessId            string              `json:"accessId,omitempty"`
	AccessToken         string              `json:"accessToken,omitempty"`
	AWSAssumeIamRoleArn string              `json:"awsAssumeIamRoleArn,omitempty"`
	RefreshToken        string              `json:"refreshToken,omitempty"`
	URL                 string              `json:"url,omitempty"`
	ProjectID           string              `json:"workspaceId"`
	Integration         IntegrationAuthType `json:"integration"`
}

type UpdateIntegrationAuthRequest struct {
	AccessId            string              `json:"accessId,omitempty"`
	AccessToken         string              `json:"accessToken,omitempty"`
	AWSAssumeIamRoleArn string              `json:"awsAssumeIamRoleArn,omitempty"`
	RefreshToken        string              `json:"refreshToken,omitempty"`
	URL                 string              `json:"url,omitempty"`
	Integration         IntegrationAuthType `json:"integration"`
	IntegrationAuthId   string              `json:"integrationAuthId"`
}

type CreateIntegrationAuthResponse struct {
	IntegrationAuth struct {
		ID string `json:"id"`
	} `json:"integrationAuth"`
}

type UpdateIntegrationAuthResponse struct {
	IntegrationAuth struct {
		ID string `json:"id"`
	} `json:"integrationAuth"`
}

type DeleteIntegrationAuthRequest struct {
	ID string `json:"id"`
}

type DeleteIntegrationAuthResponse struct {
	IntegrationAuth struct {
		ID string `json:"id"`
	} `json:"integrationAuth"`
}

type AwsTag struct {
	Key   string `tfsdk:"key" json:"key,omitempty"`
	Value string `tfsdk:"value" json:"value,omitempty"`
}

type IntegrationMetadata struct {
	InitialSyncBehavior string `json:"initialSyncBehavior,omitempty"`
	SecretPrefix        string `json:"secretPrefix"`
	SecretSuffix        string `json:"secretSuffix"`
	MappingBehavior     string `json:"mappingBehavior,omitempty"`
	ShouldAutoRedeploy  bool   `json:"shouldAutoRedeploy,omitempty"`
	MetadataSyncMode    string `json:"metadataSyncMode,omitempty"`

	SecretGCPLabel []struct {
		LabelName  string `json:"labelName,omitempty"`
		LabelValue string `json:"labelValue,omitempty"`
	} `json:"secretGCPLabel,omitempty"`
	SecretAWSTag []AwsTag `json:"secretAWSTag,omitempty"`

	GithubVisibility        string   `json:"githubVisibility,omitempty"`
	GithubVisibilityRepoIDs []string `json:"githubVisibilityRepoIds,omitempty"`
	KMSKeyID                string   `json:"kmsKeyId,omitempty"`
	ShouldDisableDelete     bool     `json:"shouldDisableDelete,omitempty"`
	ShouldEnableDelete      bool     `json:"shouldEnableDelete,omitempty"`
	ShouldMaskSecrets       bool     `json:"shouldMaskSecrets,omitempty"`
	ShouldProtectSecrets    bool     `json:"shouldProtectSecrets,omitempty"`
}
type CreateIntegrationRequest struct {
	IntegrationAuthID   string `json:"integrationAuthId"`
	App                 string `json:"app,omitempty"`
	AppID               string `json:"appId,omitempty"`
	SecretPath          string `json:"secretPath,omitempty"`
	SourceEnvironment   string `json:"sourceEnvironment,omitempty"`
	TargetEnvironment   string `json:"targetEnvironment,omitempty"`
	TargetEnvironmentID string `json:"targetEnvironmentId,omitempty"`
	TargetService       string `json:"targetService,omitempty"`
	TargetServiceID     string `json:"targetServiceId,omitempty"`
	Owner               string `json:"owner,omitempty"`
	URL                 string `json:"url,omitempty"`
	Path                string `json:"path,omitempty"`
	Region              string `json:"region,omitempty"`
	Scope               string `json:"scope,omitempty"`

	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type Integration struct {
	ID                  string              `json:"id"`
	IsActive            bool                `json:"isActive"`
	URL                 string              `json:"url"`
	App                 string              `json:"app"`
	AppID               string              `json:"appId"`
	TargetEnvironment   string              `json:"targetEnvironment"`
	TargetEnvironmentID string              `json:"targetEnvironmentId"`
	TargetService       string              `json:"targetService"`
	TargetServiceID     string              `json:"targetServiceId"`
	Owner               string              `json:"owner"`
	Path                string              `json:"path"`
	Region              string              `json:"region"`
	Scope               string              `json:"scope"`
	Integration         string              `json:"integration"`
	Metadata            IntegrationMetadata `json:"metadata,omitempty"`
	IntegrationAuthID   string              `json:"integrationAuthId"`
	EnvID               string              `json:"envId"`
	SecretPath          string              `json:"secretPath"`

	Environment struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Slug string `json:"slug"`
	} `json:"environment"`
}

type CreateIntegrationResponse struct {
	Integration Integration `json:"integration"`
}

type GetIntegrationRequest struct {
	ID string
}

type GetIntegrationResponse struct {
	Integration Integration `json:"integration"`
}

type UpdateIntegrationRequest struct {
	ID                string
	App               string                 `json:"app,omitempty"`
	AppID             string                 `json:"appId,omitempty"`
	SecretPath        string                 `json:"secretPath,omitempty"`
	TargetEnvironment string                 `json:"targetEnvironment,omitempty"`
	Owner             string                 `json:"owner,omitempty"`
	Environment       string                 `json:"environment,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	IsActive          bool                   `json:"isActive,omitempty"`
	Region            string                 `json:"region,omitempty"`
	Path              string                 `json:"path,omitempty"`
}

type UpdateIntegrationResponse struct {
	Integration Integration `json:"integration"`
}
type SecretApprovalPolicyEnvironment struct {
	Slug string `json:"slug"`
}

type SecretApprovalPolicyApprover struct {
	ID   string `json:"id"`
	Name string `json:"username"`
	Type string `json:"type"`
}

type SecretApprovalPolicy struct {
	ID                   string                            `json:"id"`
	ProjectID            string                            `json:"projectId"`
	Name                 string                            `json:"name"`
	Environment          SecretApprovalPolicyEnvironment   `json:"environment"`
	Environments         []SecretApprovalPolicyEnvironment `json:"environments"`
	SecretPath           string                            `json:"secretPath"`
	Approvers            []SecretApprovalPolicyApprover    `json:"approvers"`
	RequiredApprovals    int64                             `json:"approvals"`
	EnforcementLevel     string                            `json:"enforcementLevel"`
	AllowedSelfApprovals bool                              `json:"allowedSelfApprovals"`
}

type CreateSecretApprovalPolicyApprover struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"username"`
}

type CreateSecretApprovalPolicyRequest struct {
	ProjectID            string                               `json:"workspaceId"`
	Name                 string                               `json:"name,omitempty"`
	Environments         []string                             `json:"environments"`
	Environment          string                               `json:"environment"`
	SecretPath           string                               `json:"secretPath"`
	Approvers            []CreateSecretApprovalPolicyApprover `json:"approvers"`
	RequiredApprovals    int64                                `json:"approvals"`
	EnforcementLevel     string                               `json:"enforcementLevel"`
	AllowedSelfApprovals bool                                 `json:"allowedSelfApprovals"`
}

type CreateSecretApprovalPolicyResponse struct {
	SecretApprovalPolicy SecretApprovalPolicy `json:"approval"`
}

type GetSecretApprovalPolicyByIDRequest struct {
	ID string
}

type GetSecretApprovalPolicyByIDResponse struct {
	SecretApprovalPolicy SecretApprovalPolicy `json:"approval"`
}

type UpdateSecretApprovalPolicyApprover struct {
	Type         string   `json:"type"`
	ID           string   `json:"id"`
	Name         string   `json:"username"`
	Environments []string `json:"environments"`
}

type UpdateSecretApprovalPolicyRequest struct {
	ID                   string
	Name                 string                               `json:"name"`
	SecretPath           string                               `json:"secretPath"`
	Approvers            []UpdateSecretApprovalPolicyApprover `json:"approvers"`
	RequiredApprovals    int64                                `json:"approvals"`
	EnforcementLevel     string                               `json:"enforcementLevel"`
	AllowedSelfApprovals bool                                 `json:"allowedSelfApprovals"`
	Environments         []string                             `json:"environments"`
}

type UpdateSecretApprovalPolicyResponse struct {
	SecretApprovalPolicy SecretApprovalPolicy `json:"approval"`
}

type DeleteSecretApprovalPolicyRequest struct {
	ID string
}

type DeleteSecretApprovalPolicyResponse struct {
	SecretApprovalPolicy SecretApprovalPolicy `json:"approval"`
}

type AccessApprovalPolicyApprover struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type AccessApprovalPolicyEnvironment struct {
	Slug string `json:"slug"`
}

type AccessApprovalPolicy struct {
	ID                string                            `json:"id"`
	ProjectID         string                            `json:"projectId"`
	Name              string                            `json:"name"`
	Environments      []AccessApprovalPolicyEnvironment `json:"environments"`
	Environment       AccessApprovalPolicyEnvironment   `json:"environment"`
	SecretPath        string                            `json:"secretPath"`
	Approvers         []AccessApprovalPolicyApprover    `json:"approvers"`
	RequiredApprovals int64                             `json:"approvals"`
	EnforcementLevel  string                            `json:"enforcementLevel"`
}

type CreateAccessApprovalPolicyApprover struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"username"`
}

type CreateAccessApprovalPolicyRequest struct {
	ProjectSlug       string                               `json:"projectSlug"`
	Name              string                               `json:"name,omitempty"`
	Environments      []string                             `json:"environments"`
	Environment       string                               `json:"environment"`
	SecretPath        string                               `json:"secretPath"`
	Approvers         []CreateAccessApprovalPolicyApprover `json:"approvers"`
	RequiredApprovals int64                                `json:"approvals"`
	EnforcementLevel  string                               `json:"enforcementLevel"`
}

type CreateAccessApprovalPolicyResponse struct {
	AccessApprovalPolicy AccessApprovalPolicy `json:"approval"`
}

type GetAccessApprovalPolicyByIDRequest struct {
	ID string
}

type GetAccessApprovalPolicyByIDResponse struct {
	AccessApprovalPolicy AccessApprovalPolicy `json:"approval"`
}

type UpdateAccessApprovalPolicyApprover struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"username"`
}

type UpdateAccessApprovalPolicyRequest struct {
	ID                string
	Name              string                               `json:"name"`
	SecretPath        string                               `json:"secretPath"`
	Environments      []string                             `json:"environments"`
	Approvers         []UpdateAccessApprovalPolicyApprover `json:"approvers"`
	RequiredApprovals int64                                `json:"approvals"`
	EnforcementLevel  string                               `json:"enforcementLevel"`
}

type UpdateAccessApprovalPolicyResponse struct {
	AccessApprovalPolicy AccessApprovalPolicy `json:"approval"`
}

type DeleteAccessApprovalPolicyRequest struct {
	ID string
}

type DeleteAccessApprovalPolicyResponse struct {
	AccessApprovalPolicy AccessApprovalPolicy `json:"approval"`
}

type CreateSecretImportRequest struct {
	ProjectID     string `json:"workspaceId"`
	Environment   string `json:"environment"`
	SecretPath    string `json:"path"`
	IsReplication bool   `json:"isReplication"`
	ImportFrom    struct {
		Environment string `json:"environment"`
		SecretPath  string `json:"path"`
	} `json:"import"`
}

type CreateSecretImportResponse struct {
	SecretImport SecretImport `json:"secretImport"`
}

type UpdateSecretImportRequest struct {
	ID            string `json:"id"`
	ProjectID     string `json:"workspaceId"`
	Environment   string `json:"environment"`
	SecretPath    string `json:"path"`
	IsReplication bool   `json:"isReplication"`
	ImportFrom    struct {
		Environment string `json:"environment"`
		SecretPath  string `json:"path"`
	} `json:"import"`
}

type UpdateSecretImportResponse struct {
	SecretImport SecretImport `json:"secretImport"`
}

type DeleteSecretImportRequest struct {
	ID          string `json:"id"`
	Environment string `json:"environment"`
	ProjectID   string `json:"workspaceId"`
	SecretPath  string `json:"path"`
}

type DeleteSecretImportResponse struct {
	SecretImport SecretImport `json:"secretImport"`
}

type GetSecretImportRequest struct {
	ID          string `json:"id"`
	Environment string `json:"environment"`
	ProjectID   string `json:"workspaceId"`
	SecretPath  string `json:"path"`
}

type GetSecretImportResponse struct {
	SecretImport SecretImport `json:"secretImport"`
}

type GetSecretImportByIDRequest struct {
	ID string
}

type GetSecretImportByIDResponse struct {
	SecretImport SecretImportByID `json:"secretImport"`
}

type ListSecretImportRequest struct {
	Environment string `json:"environment"`
	ProjectID   string `json:"workspaceId"`
	SecretPath  string `json:"path"`
}

type ListSecretImportResponse struct {
	SecretImports []SecretImport `json:"secretImports"`
}

type AppConnection struct {
	Id              string `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Version         int    `json:"version"`
	OrgId           string `json:"orgId"`
	App             string `json:"app"`
	Method          string `json:"method"`
	CredentialsHash string `json:"credentialsHash"`
}

type CreateAppConnectionRequest struct {
	App         AppConnectionApp
	Description string                 `json:"description,omitempty"`
	Method      string                 `json:"method"`
	Name        string                 `json:"name"`
	Credentials map[string]interface{} `json:"credentials"`
}

type CreateAppConnectionResponse struct {
	AppConnection AppConnection `json:"appConnection"`
}

type GetAppConnectionByIdRequest struct {
	App AppConnectionApp
	ID  string
}

type GetAppConnectionByIdResponse struct {
	AppConnection AppConnection `json:"appConnection"`
}

type UpdateAppConnectionRequest struct {
	ID          string
	App         AppConnectionApp
	Description string                 `json:"description"`
	Method      string                 `json:"method"`
	Name        string                 `json:"name"`
	Credentials map[string]interface{} `json:"credentials,omitempty"`
}

type UpdateAppConnectionResponse struct {
	AppConnection AppConnection `json:"appConnection"`
}

type DeleteAppConnectionRequest struct {
	App AppConnectionApp
	ID  string
}

type DeleteAppConnectionResponse struct {
	AppConnection AppConnection `json:"appConnection"`
}

type SecretSyncConnection struct {
	ConnectionID string `json:"id"`
}

type SecretSyncEnvironment struct {
	Slug string `json:"slug"`
}

type SecretSyncFolder struct {
	Path string `json:"path"`
}

type SecretSync struct {
	ID                string                 `json:"id"`
	Name              string                 `json:"name"`
	Description       string                 `json:"description"`
	AutoSyncEnabled   bool                   `json:"isAutoSyncEnabled"`
	Version           int                    `json:"version"`
	ProjectID         string                 `json:"projectId"`
	ConnectionID      string                 `json:"connectionId"`
	Connection        SecretSyncConnection   `json:"connection"`
	Environment       SecretSyncEnvironment  `json:"environment"`
	SecretFolder      SecretSyncFolder       `json:"folder"`
	SyncOptions       map[string]interface{} `json:"syncOptions"`
	DestinationConfig map[string]interface{} `json:"destinationConfig"`
}

type CreateSecretSyncRequest struct {
	App               SecretSyncApp
	Name              string                 `json:"name"`
	ProjectID         string                 `json:"projectId"`
	ConnectionID      string                 `json:"connectionId"`
	Environment       string                 `json:"environment"`
	SecretPath        string                 `json:"secretPath"`
	AutoSyncEnabled   bool                   `json:"isAutoSyncEnabled"`
	Description       string                 `json:"description"`
	SyncOptions       map[string]interface{} `json:"syncOptions"`
	DestinationConfig map[string]interface{} `json:"destinationConfig"`
}

type CreateSecretSyncResponse struct {
	SecretSync SecretSync `json:"secretSync"`
}

type GetSecretSyncByIdRequest struct {
	App SecretSyncApp
	ID  string
}

type GetSecretSyncByIdResponse struct {
	SecretSync SecretSync `json:"secretSync"`
}

type UpdateSecretSyncRequest struct {
	App               SecretSyncApp
	ID                string
	Name              string                 `json:"name,omitempty"`
	ProjectID         string                 `json:"projectId,omitempty"`
	ConnectionID      string                 `json:"connectionId,omitempty"`
	Environment       string                 `json:"environment,omitempty"`
	SecretPath        string                 `json:"secretPath,omitempty"`
	AutoSyncEnabled   bool                   `json:"isAutoSyncEnabled,omitempty"`
	Description       string                 `json:"description"`
	SyncOptions       map[string]interface{} `json:"syncOptions,omitempty"`
	DestinationConfig map[string]interface{} `json:"destinationConfig,omitempty"`
}

type UpdateSecretSyncResponse struct {
	SecretSync SecretSync `json:"secretSync"`
}

type DeleteSecretSyncRequest struct {
	App SecretSyncApp
	ID  string
}

type DeleteSecretSyncResponse struct {
	SecretSync SecretSync `json:"secretSync"`
}

type DynamicSecret struct {
	Id               string                 `json:"id"`
	Name             string                 `json:"name"`
	Version          int                    `json:"version"`
	Type             string                 `json:"type"`
	DefaultTTL       string                 `json:"defaultTTL"`
	MaxTTL           string                 `json:"maxTTL"`
	FolderId         string                 `json:"folderId"`
	CreatedAt        string                 `json:"createdAt"`
	UpdatedAt        string                 `json:"updatedAt"`
	UsernameTemplate string                 `json:"usernameTemplate"`
	Metadata         []MetaEntry            `json:"metadata"`
	Inputs           map[string]interface{} `json:"inputs"`
}

type DynamicSecretProviderObject struct {
	Provider DynamicSecretProvider  `json:"type"`
	Inputs   map[string]interface{} `json:"inputs"`
}

type CreateDynamicSecretRequest struct {
	Provider         DynamicSecretProviderObject `json:"provider"`
	Name             string                      `json:"name"`
	ProjectSlug      string                      `json:"projectSlug"`
	EnvironmentSlug  string                      `json:"environmentSlug"`
	Path             string                      `json:"path"`
	DefaultTTL       string                      `json:"defaultTTL"`
	MaxTTL           string                      `json:"maxTTL,omitempty"`
	UsernameTemplate string                      `json:"usernameTemplate,omitempty"`
	Metadata         []MetaEntry                 `json:"metadata"`
}

type CreateDynamicSecretResponse struct {
	DynamicSecret DynamicSecret `json:"dynamicSecret"`
}

type GetDynamicSecretByNameRequest struct {
	ProjectSlug     string
	EnvironmentSlug string
	Path            string
	Name            string
}

type GetDynamicSecretByNameResponse struct {
	DynamicSecret DynamicSecret `json:"dynamicSecret"`
}

type UpdateDynamicSecretData struct {
	Inputs           map[string]interface{} `json:"inputs"`
	DefaultTTL       string                 `json:"defaultTTL"`
	MaxTTL           string                 `json:"maxTTL,omitempty"`
	NewName          string                 `json:"newName,omitempty"`
	Metadata         []MetaEntry            `json:"metadata"`
	UsernameTemplate string                 `json:"usernameTemplate,omitempty"`
}

type UpdateDynamicSecretRequest struct {
	Name            string                  `json:"name"`
	ProjectSlug     string                  `json:"projectSlug"`
	EnvironmentSlug string                  `json:"environmentSlug"`
	Path            string                  `json:"path"`
	Data            UpdateDynamicSecretData `json:"data"`
}

type UpdateDynamicSecretResponse struct {
	DynamicSecret DynamicSecret `json:"dynamicSecret"`
}

type DeleteDynamicSecretRequest struct {
	Name            string `json:"name"`
	ProjectSlug     string `json:"projectSlug"`
	EnvironmentSlug string `json:"environmentSlug"`
	Path            string `json:"path"`
}

type DeleteDynamicSecretResponse struct {
	DynamicSecret DynamicSecret `json:"dynamicSecret"`
}

type CreateGroupRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	Role string `json:"role"`
}

type UpdateGroupRequest struct {
	ID   string
	Name string `json:"name"`
	Slug string `json:"slug"`
	Role string `json:"role"`
}

type DeleteGroupRequest struct {
	ID string
}

type SecretRotationProviderObject struct {
	Provider SecretRotationProvider `json:"type"`
	Inputs   map[string]any         `json:"inputs"`
}

type SecretRotationConnection struct {
	ConnectionID string `json:"id"`
}

type SecretRotationEnvironment struct {
	Slug string `json:"slug"`
}

type SecretRotationFolder struct {
	Path string `json:"path"`
}

type SecretRotationRotateAtUtc struct {
	Hours   int64 `json:"hours"`
	Minutes int64 `json:"minutes"`
}

type SecretRotation struct {
	ID                  string                    `json:"id"`
	Name                string                    `json:"name"`
	Description         string                    `json:"description"`
	AutoRotationEnabled bool                      `json:"isAutoRotationEnabled"`
	ProjectID           string                    `json:"projectId"`
	ConnectionID        string                    `json:"connectionId"`
	Connection          SecretRotationConnection  `json:"connection"`
	Environment         SecretRotationEnvironment `json:"environment"`
	SecretFolder        SecretRotationFolder      `json:"folder"`

	RotationInterval int32                      `json:"rotationInterval"`
	RotateAtUtc      *SecretRotationRotateAtUtc `json:"rotateAtUtc,omitempty"`

	Parameters          map[string]any `json:"parameters"`
	SecretsMapping      map[string]any `json:"secretsMapping"`
	TemporaryParameters map[string]any `json:"temporaryParameters"`
}

type CreateSecretRotationRequest struct {
	Provider SecretRotationProvider

	Name                string `json:"name"`
	Description         string `json:"description"`
	AutoRotationEnabled bool   `json:"isAutoRotationEnabled"`
	ProjectID           string `json:"projectId"`
	ConnectionID        string `json:"connectionId"`
	Environment         string `json:"environment"`
	SecretPath          string `json:"secretPath"`

	RotationInterval int32                     `json:"rotationInterval"`
	RotateAtUtc      SecretRotationRotateAtUtc `json:"rotateAtUtc,omitempty"`

	Parameters          map[string]any `json:"parameters"`
	SecretsMapping      map[string]any `json:"secretsMapping"`
	TemporaryParameters map[string]any `json:"temporaryParameters,omitempty"`
}

type CreateSecretRotationResponse struct {
	SecretRotation SecretRotation `json:"secretRotation"`
}

type GetSecretRotationByIdRequest struct {
	Provider SecretRotationProvider
	ID       string
}

type GetSecretRotationByIdResponse struct {
	SecretRotation SecretRotation `json:"secretRotation"`
}

type UpdateSecretRotationRequest struct {
	Provider SecretRotationProvider
	ID       string

	Name                string `json:"name,omitempty"`
	Description         string `json:"description"`
	AutoRotationEnabled bool   `json:"isAutoRotationEnabled,omitempty"`
	ConnectionID        string `json:"connectionId,omitempty"`
	Environment         string `json:"environment,omitempty"`
	SecretPath          string `json:"secretPath,omitempty"`

	RotationInterval int32                     `json:"rotationInterval,omitempty"`
	RotateAtUtc      SecretRotationRotateAtUtc `json:"rotateAtUtc,omitempty"`

	Parameters          map[string]any `json:"parameters,omitempty"`
	SecretsMapping      map[string]any `json:"secretsMapping,omitempty"`
	TemporaryParameters map[string]any `json:"temporaryParameters,omitempty"`
}

type UpdateSecretRotationResponse struct {
	SecretRotation SecretRotation `json:"secretRotation"`
}

type DeleteSecretRotationRequest struct {
	Provider SecretRotationProvider
	ID       string
}

type DeleteSecretRotationResponse struct {
	SecretRotation SecretRotation `json:"secretRotation"`
}

type ProjectTemplate struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Roles        []Role        `json:"roles"`
	Environments []Environment `json:"environments"`
	OrgID        string        `json:"orgId"`
	CreatedAt    time.Time     `json:"createdAt"`
	UpdatedAt    time.Time     `json:"updatedAt"`
	Type         string        `json:"type"`
}

type Environment struct {
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Position int64  `json:"position"`
}

type Role struct {
	Name        string       `json:"name"`
	Slug        string       `json:"slug"`
	Permissions []Permission `json:"permissions"`
}

type Permission struct {
	Subject    string         `json:"subject"`
	Action     []string       `json:"action"`
	Conditions map[string]any `json:"conditions,omitempty"`
	Inverted   bool           `json:"inverted"`
}

type CreateProjectTemplateRequest struct {
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Type         string        `json:"type"`
	Roles        []Role        `json:"roles,omitempty"`
	Environments []Environment `json:"environments,omitempty"`
}

type CreateProjectTemplateResponse struct {
	ProjectTemplate ProjectTemplate `json:"projectTemplate"`
}

type GetProjectTemplateByIdRequest struct {
	ID string `json:"id"`
}

type GetProjectTemplateByIdResponse struct {
	ProjectTemplate ProjectTemplate `json:"projectTemplate"`
}

type UpdateProjectTemplateRequest struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Type         string        `json:"type"`
	Roles        []Role        `json:"roles,omitempty"`
	Environments []Environment `json:"environments,omitempty"`
}

type UpdateProjectTemplateResponse struct {
	ProjectTemplate ProjectTemplate `json:"projectTemplate"`
}

type DeleteProjectTemplateResponse struct {
	ProjectTemplate ProjectTemplate `json:"projectTemplate"`
}

// IdentityDetails

type IdentityOrganization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type IdentityDetails struct {
	Organization IdentityOrganization `json:"organization"`
}

type GetIdentityDetailsResponse struct {
	IdentityDetails IdentityDetails `json:"identityDetails"`
}
