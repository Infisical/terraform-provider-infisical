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

type UniversalMachineIdentityAuthResponse struct {
	AccessToken       string `json:"accessToken"`
	ExpiresIn         int    `json:"expiresIn"`
	AccessTokenMaxTTL int    `json:"accessTokenMaxTTL"`
	TokenType         string `json:"tokenType"`
}

type OidcMachineIdentityAuthResponse struct {
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
	WorkspaceID   string   `json:"workspaceId"`
	Type          string   `json:"type"`
	Environment   string   `json:"environment"`
	SecretKey     string   `json:"secretKey"`
	SecretValue   string   `json:"secretValue"`
	SecretComment string   `json:"secretComment"`
	SecretPath    string   `json:"secretPath"`
	TagIDs        []string `json:"tagIds"`
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
	SecretName  string   `json:"secretName"`
	WorkspaceID string   `json:"workspaceId"`
	Environment string   `json:"environment"`
	Type        string   `json:"type"`
	SecretPath  string   `json:"secretPath"`
	SecretValue string   `json:"secretValue"`
	TagIDs      []string `json:"tagIds"`
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
	IdentityId    string `json:"identityId"`
	PrivilegeSlug string `json:"privilegeSlug,omitempty"`
}

type GetProjectIdentitySpecificPrivilegeResponse struct {
	Privilege ProjectIdentitySpecificPrivilege `json:"privilege"`
}

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
