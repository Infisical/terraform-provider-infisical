package resource

import (
	"testing"
	"time"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func strPtr(s string) *string { return &s }

func TestBuildGroupAssignmentRoles(t *testing.T) {
	roles, diags := buildGroupAssignmentRoles([]groupAssignmentRoleModel{
		{RoleSlug: types.StringValue("member"), IsTemporary: types.BoolValue(false)},
		{RoleSlug: types.StringValue("admin"), IsTemporary: types.BoolValue(true), TemporaryAccessStartTime: types.StringValue("2026-01-02T03:04:05Z")},
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if roles[0].TemporaryMode != "" || roles[0].TemporaryRange != "" || roles[0].TemporaryAccessStartTime != nil {
		t.Errorf("a permanent role must carry no temporary fields, got %+v", roles[0])
	}

	temporary := roles[1]
	if temporary.TemporaryMode != TEMPORARY_MODE_RELATIVE || temporary.TemporaryRange != TEMPORARY_RANGE_DEFAULT {
		t.Errorf("a temporary role defaults to relative mode and a %s range, got %+v", TEMPORARY_RANGE_DEFAULT, temporary)
	}
	if temporary.TemporaryAccessStartTime == nil || !temporary.TemporaryAccessStartTime.Equal(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)) {
		t.Errorf("unexpected start time: %v", temporary.TemporaryAccessStartTime)
	}
}

func TestBuildGroupAssignmentRolesRejectsBadTemporaryRoles(t *testing.T) {
	for name, role := range map[string]groupAssignmentRoleModel{
		"missing start time": {RoleSlug: types.StringValue("admin"), IsTemporary: types.BoolValue(true), TemporaryAccessStartTime: types.StringNull()},
		"invalid start time": {RoleSlug: types.StringValue("admin"), IsTemporary: types.BoolValue(true), TemporaryAccessStartTime: types.StringValue("tomorrow")},
	} {
		t.Run(name, func(t *testing.T) {
			if _, diags := buildGroupAssignmentRoles([]groupAssignmentRoleModel{role}); !diags.HasError() {
				t.Error("expected an error diagnostic")
			}
		})
	}
}

func TestGroupAssignmentRolesFromAPI(t *testing.T) {
	start := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	apiRoles := []infisical.OrgGroupMembershipRole{
		{Role: "custom", CustomRoleSlug: strPtr("deployer")},
		{Role: "admin", IsTemporary: true, TemporaryRange: strPtr(TEMPORARY_RANGE_DEFAULT), TemporaryAccessStartTime: &start},
	}

	t.Run("prior state left the range unset", func(t *testing.T) {
		roles := groupAssignmentRolesFromAPI(apiRoles, []groupAssignmentRoleModel{
			{RoleSlug: types.StringValue("admin"), IsTemporary: types.BoolValue(true), TemporaryRange: types.StringNull()},
		})

		if roles[0].RoleSlug.ValueString() != "deployer" {
			t.Errorf("a custom role maps to its custom slug, got %s", roles[0].RoleSlug)
		}
		if !roles[0].TemporaryRange.IsNull() || !roles[0].TemporaryAccessStartTime.IsNull() || roles[0].IsTemporary.ValueBool() {
			t.Errorf("a permanent role has no temporary values, got %+v", roles[0])
		}
		if !roles[1].TemporaryRange.IsNull() {
			t.Errorf("the API's default range must stay null when the config left it unset, got %s", roles[1].TemporaryRange)
		}
		if roles[1].TemporaryAccessStartTime.ValueString() != "2026-01-02T03:04:05Z" {
			t.Errorf("unexpected start time: %s", roles[1].TemporaryAccessStartTime)
		}
	})

	t.Run("no prior state, as on import", func(t *testing.T) {
		roles := groupAssignmentRolesFromAPI(apiRoles, nil)
		if roles[1].TemporaryRange.ValueString() != TEMPORARY_RANGE_DEFAULT {
			t.Errorf("without prior state the range is reported as-is, got %s", roles[1].TemporaryRange)
		}
	})
}
