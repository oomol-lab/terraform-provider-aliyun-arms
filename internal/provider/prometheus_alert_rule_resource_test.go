package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestResourceRequestAddsTerraformTag(t *testing.T) {
	t.Parallel()

	diagnostics := diag.Diagnostics{}
	rule := (prometheusAlertRuleModel{
		Name:            types.StringValue("rule"),
		ClusterID:       types.StringValue("cluster"),
		PromQL:          types.StringValue("up == 0"),
		DurationMinutes: types.Int64Value(5),
		Level:           types.StringValue("P2"),
		Message:         types.StringValue("down"),
		Status:          types.StringValue("RUNNING"),
		Tags: types.MapValueMust(types.StringType, map[string]attr.Value{
			createdByTag: types.StringValue("Terraform"),
		}),
		NoDataRevision: types.Int64Value(2),
	}).toAPI(context.Background(), &diagnostics)
	if diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
	if rule.Tags[createdByTag] != "Terraform" {
		t.Fatalf("tags = %#v", rule.Tags)
	}
}
