package terraform

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func StringListToGoStringSlice(ctx context.Context, diagnostics diag.Diagnostics, stringList basetypes.ListValue) []string {
	tfStringSlice := make([]types.String, 0, len(stringList.Elements()))
	if !stringList.IsNull() && !stringList.IsUnknown() {
		diags := stringList.ElementsAs(ctx, &tfStringSlice, false)
		diagnostics.Append(diags...)
		if diagnostics.HasError() {
			return nil
		}
	}

	output := make([]string, 0, len(tfStringSlice))
	for _, el := range tfStringSlice {
		output = append(output, el.ValueString())
	}
	return output
}
