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
		Name       string `json:"name"`
		Id         string `json:"id"`
		AuthMethod string `json:"authMethod"`
	} `json:"identity"`
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
	CustomRole *struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		Description string `json:"description"`
	} `json:"customRole,omitempty"`
}

type Identity struct {
	Name       string    `json:"name"`
	ID         string    `json:"id"`
	AuthMethod string    `json:"authMethod"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
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
	ID                 string               `json:"id"`
	Name               string               `json:"name"`
	Slug               string               `json:"slug"`
	AutoCapitalization bool                 `json:"autoCapitalization"`
	OrgID              string               `json:"orgId"`
	CreatedAt          string               `json:"createdAt"`
	UpdatedAt          string               `json:"updatedAt"`
	Version            int64                `json:"version"`
	UpgradeStatus      string               `json:"upgradeStatus"`
	Environments       []ProjectEnvironment `json:"environments"`
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

type SecretFolder struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	EnvID string `json:"envId"`
}

type SecretImport struct {
	ID         string `json:"id"`
	SecretPath string `json:"secretPath"`
	ImportPath string `json:"importPath"`
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
	Version       int    `json:"version"`
	Workspace     string `json:"workspace"`
	Type          string `json:"type"`
	Environment   string `json:"environment"`
	SecretKey     string `json:"secretKey"`
	SecretValue   string `json:"secretValue"`
	SecretComment string `json:"secretComment"`
}

type GetRawSecretsV3Response struct {
	Secrets []RawV3Secret `json:"secrets"`
}

type GetSingleRawSecretByNameSecretResponse struct {
	Secret RawV3Secret `json:"secret"`
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
	ProjectName      string `json:"projectName"`
	Slug             string `json:"slug"`
	OrganizationSlug string `json:"organizationSlug"`
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
	Slug        string `json:"slug"`
	ProjectName string `json:"name"`
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

type CreateProjectIdentitySpecificPrivilegeResponse struct {
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

type GetProjectIdentitySpecificPrivilegeResponse struct {
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
	ProjectId string                           `json:"projectId"`
	GroupId   string                           `json:"groupId"`
	Roles     []CreateProjectGroupRequestRoles `json:"roles"`
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
	ID     string `json:"id"`
	OrgID  string `json:"orgId"`
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Role   string `json:"role"`
	RoleId string `json:"roleId"`
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
}

type CreateProjectEnvironmentResponse struct {
	Environment ProjectEnvironment `json:"environment"`
}

type DeleteProjectEnvironmentRequest struct {
	ID        string `json:"id"`
	ProjectID string `json:"workspaceId"`
}

type DeleteProjectEnvironmentResponse struct {
	Environment ProjectEnvironment `json:"environment"`
}

type GetProjectEnvironmentByIDRequest struct {
	ID        string `json:"id"`
	ProjectID string `json:"workspaceId"`
}

type GetProjectEnvironmentByIDResponse struct {
	Environment ProjectEnvironment `json:"environment"`
}

type UpdateProjectEnvironmentRequest struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	ProjectID string `json:"workspaceId"`
}

type UpdateProjectEnvironmentResponse struct {
	Environment ProjectEnvironment `json:"environment"`
}

type CreateIdentityRequest struct {
	Name  string `json:"name"`
	OrgID string `json:"organizationId"`
	Role  string `json:"role"`
}

type CreateIdentityResponse struct {
	Identity Identity `json:"identity"`
}

type UpdateIdentityRequest struct {
	IdentityID string `json:"identityId"`
	Name       string `json:"name,omitempty"`
	Role       string `json:"role,omitempty"`
}

type UpdateIdentityResponse struct {
	Identity Identity `json:"identity"`
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

type SecretApprovalPolicyEnvironment struct {
	Slug string `json:"slug"`
}

type SecretApprovalPolicyApprover struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type SecretApprovalPolicy struct {
	ID                string                          `json:"id"`
	ProjectID         string                          `json:"projectId"`
	Name              string                          `json:"name"`
	Environment       SecretApprovalPolicyEnvironment `json:"environment"`
	SecretPath        string                          `json:"secretPath"`
	Approvers         []SecretApprovalPolicyApprover  `json:"approvers"`
	RequiredApprovals int64                           `json:"approvals"`
	EnforcementLevel  string                          `json:"enforcementLevel"`
}

type CreateSecretApprovalPolicyApprover struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

type CreateSecretApprovalPolicyRequest struct {
	ProjectID         string                               `json:"workspaceId"`
	Name              string                               `json:"name,omitempty"`
	Environment       string                               `json:"environment"`
	SecretPath        string                               `json:"secretPath"`
	Approvers         []CreateSecretApprovalPolicyApprover `json:"approvers"`
	RequiredApprovals int64                                `json:"approvals"`
	EnforcementLevel  string                               `json:"enforcementLevel"`
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
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

type UpdateSecretApprovalPolicyRequest struct {
	ID                string
	Name              string                               `json:"name"`
	SecretPath        string                               `json:"secretPath"`
	Approvers         []UpdateSecretApprovalPolicyApprover `json:"approvers"`
	RequiredApprovals int64                                `json:"approvals"`
	EnforcementLevel  string                               `json:"enforcementLevel"`
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
	ID                string                          `json:"id"`
	ProjectID         string                          `json:"projectId"`
	Name              string                          `json:"name"`
	Environment       AccessApprovalPolicyEnvironment `json:"environment"`
	SecretPath        string                          `json:"secretPath"`
	Approvers         []AccessApprovalPolicyApprover  `json:"approvers"`
	RequiredApprovals int64                           `json:"approvals"`
	EnforcementLevel  string                          `json:"enforcementLevel"`
}

type CreateAccessApprovalPolicyApprover struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

type CreateAccessApprovalPolicyRequest struct {
	ProjectSlug       string                               `json:"projectSlug"`
	Name              string                               `json:"name,omitempty"`
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
	Name string `json:"name"`
}

type UpdateAccessApprovalPolicyRequest struct {
	ID                string
	Name              string                               `json:"name"`
	SecretPath        string                               `json:"secretPath"`
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

type GetSecretImportByIDRequest struct {
	ID          string `json:"id"`
	Environment string `json:"environment"`
	ProjectID   string `json:"workspaceId"`
	SecretPath  string `json:"path"`
}

type GetSecretImportByIDResponse struct {
	SecretImport SecretImport `json:"secretImport"`
}

type ListSecretImportRequest struct {
	Environment string `json:"environment"`
	ProjectID   string `json:"workspaceId"`
	SecretPath  string `json:"path"`
}

type ListSecretImportResponse struct {
	SecretImports []SecretImport `json:"secretImports"`
}
