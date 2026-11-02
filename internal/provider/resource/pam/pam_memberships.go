package resource

import (
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func NewPamUserResource() resource.Resource {
	return &PamMembershipBaseResource{Scope: pamMembershipScopeProduct, Kind: infisical.PamMemberKindUser, ResourceTypeName: "_pam_user"}
}

func NewPamGroupResource() resource.Resource {
	return &PamMembershipBaseResource{Scope: pamMembershipScopeProduct, Kind: infisical.PamMemberKindGroup, ResourceTypeName: "_pam_group"}
}

func NewPamIdentityResource() resource.Resource {
	return &PamMembershipBaseResource{Scope: pamMembershipScopeProduct, Kind: infisical.PamMemberKindIdentity, ResourceTypeName: "_pam_identity"}
}

func NewPamFolderUserResource() resource.Resource {
	return &PamMembershipBaseResource{Scope: pamMembershipScopeFolder, Kind: infisical.PamMemberKindUser, ResourceTypeName: "_pam_folder_user"}
}

func NewPamFolderGroupResource() resource.Resource {
	return &PamMembershipBaseResource{Scope: pamMembershipScopeFolder, Kind: infisical.PamMemberKindGroup, ResourceTypeName: "_pam_folder_group"}
}

func NewPamFolderIdentityResource() resource.Resource {
	return &PamMembershipBaseResource{Scope: pamMembershipScopeFolder, Kind: infisical.PamMemberKindIdentity, ResourceTypeName: "_pam_folder_identity"}
}

func NewPamAccountUserResource() resource.Resource {
	return &PamMembershipBaseResource{Scope: pamMembershipScopeAccount, Kind: infisical.PamMemberKindUser, ResourceTypeName: "_pam_account_user"}
}

func NewPamAccountGroupResource() resource.Resource {
	return &PamMembershipBaseResource{Scope: pamMembershipScopeAccount, Kind: infisical.PamMemberKindGroup, ResourceTypeName: "_pam_account_group"}
}

func NewPamAccountIdentityResource() resource.Resource {
	return &PamMembershipBaseResource{Scope: pamMembershipScopeAccount, Kind: infisical.PamMemberKindIdentity, ResourceTypeName: "_pam_account_identity"}
}
