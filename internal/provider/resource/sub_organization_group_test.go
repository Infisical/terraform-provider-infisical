package resource

import (
	"testing"
	"time"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func strPtr(s string) *string { return &s }

// Values the API fills in or reformats keep the user's form, otherwise every plan shows drift.
func TestSubOrganizationGroupRolesFromAPIKeepsPriorForm(t *testing.T) {
	start := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	apiRoles := []infisical.OrgGroupMembershipRole{
		{Role: "member"},
		{Role: "custom", CustomRoleSlug: strPtr("auditor")},
		{Role: "admin", IsTemporary: true, TemporaryRange: strPtr(TEMPORARY_RANGE_DEFAULT), TemporaryAccessStartTime: &start},
	}
	prior := []subOrganizationGroupRole{
		{
			RoleSlug:                 types.StringValue("admin"),
			IsTemporary:              types.BoolValue(true),
			TemporaryRange:           types.StringNull(),
			TemporaryAccessStartTime: types.StringValue("2026-10-01T11:00:00+02:00"),
		},
	}

	roles := subOrganizationGroupRolesFromAPI(apiRoles, prior)
	if len(roles) != 3 {
		t.Fatalf("expected 3 roles, got %d", len(roles))
	}

	if roles[0].RoleSlug.ValueString() != "member" || !roles[0].TemporaryRange.IsNull() {
		t.Errorf("unexpected permanent role %+v", roles[0])
	}

	if roles[1].RoleSlug.ValueString() != "auditor" {
		t.Errorf("expected custom role to map to its slug, got %s", roles[1].RoleSlug.ValueString())
	}

	if !roles[2].TemporaryRange.IsNull() {
		t.Errorf("expected defaulted temporary_range to stay null, got %s", roles[2].TemporaryRange.ValueString())
	}
	if roles[2].TemporaryAccessStartTime.ValueString() != "2026-10-01T11:00:00+02:00" {
		t.Errorf("expected the same instant to keep the configured offset, got %s", roles[2].TemporaryAccessStartTime.ValueString())
	}
}

func TestSubOrganizationGroupRolesFromAPIReportsDrift(t *testing.T) {
	start := time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC)
	apiRoles := []infisical.OrgGroupMembershipRole{
		{Role: "admin", IsTemporary: true, TemporaryRange: strPtr("2h"), TemporaryAccessStartTime: &start},
	}
	prior := []subOrganizationGroupRole{
		{
			RoleSlug:                 types.StringValue("admin"),
			IsTemporary:              types.BoolValue(true),
			TemporaryRange:           types.StringNull(),
			TemporaryAccessStartTime: types.StringValue("2026-10-01T09:00:00Z"),
		},
	}

	roles := subOrganizationGroupRolesFromAPI(apiRoles, prior)
	if roles[0].TemporaryRange.ValueString() != "2h" {
		t.Errorf("expected a changed range to surface, got %s", roles[0].TemporaryRange.ValueString())
	}
	if roles[0].TemporaryAccessStartTime.ValueString() != "2026-10-02T09:00:00Z" {
		t.Errorf("expected a changed start time to surface, got %s", roles[0].TemporaryAccessStartTime.ValueString())
	}
}

// The API only takes UTC, so offsets get converted before sending.
func TestBuildSubOrganizationGroupRolesSendsUTC(t *testing.T) {
	roles, diags := buildSubOrganizationGroupRoles([]subOrganizationGroupRole{
		{
			RoleSlug:                 types.StringValue("admin"),
			IsTemporary:              types.BoolValue(true),
			TemporaryRange:           types.StringNull(),
			TemporaryAccessStartTime: types.StringValue("2026-10-01T11:00:00+02:00"),
		},
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	start := roles[0].TemporaryAccessStartTime
	if start.Location() != time.UTC || !start.Equal(time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)) {
		t.Errorf("expected 2026-10-01T09:00:00Z, got %s", start.Format(time.RFC3339))
	}
	if roles[0].TemporaryRange != TEMPORARY_RANGE_DEFAULT {
		t.Errorf("expected the default range, got %s", roles[0].TemporaryRange)
	}
}
