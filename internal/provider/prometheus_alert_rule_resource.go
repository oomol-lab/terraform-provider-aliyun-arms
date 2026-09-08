package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/oomol-lab/terraform-provider-aliyun-arms/internal/armsapi"
)

const createdByTag = "CreatedBy"

var (
	_ resource.Resource                = &prometheusAlertRuleResource{}
	_ resource.ResourceWithConfigure   = &prometheusAlertRuleResource{}
	_ resource.ResourceWithImportState = &prometheusAlertRuleResource{}
)

type prometheusAlertRuleResource struct {
	client armsapi.AlertRuleService
}

type prometheusAlertRuleModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	ClusterID       types.String `tfsdk:"cluster_id"`
	PromQL          types.String `tfsdk:"promql"`
	DurationMinutes types.Int64  `tfsdk:"duration_minutes"`
	Level           types.String `tfsdk:"level"`
	Message         types.String `tfsdk:"message"`
	Status          types.String `tfsdk:"status"`
	Labels          types.Map    `tfsdk:"labels"`
	Annotations     types.Map    `tfsdk:"annotations"`
	Tags            types.Map    `tfsdk:"tags"`
	NoDataRevision  types.Int64  `tfsdk:"no_data_revision"`
}

func NewPrometheusAlertRuleResource() resource.Resource {
	return &prometheusAlertRuleResource{}
}

func (r *prometheusAlertRuleResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_prometheus_alert_rule"
}

func (r *prometheusAlertRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "A custom-PromQL alert rule managed through the ARMS 2019-08-08 CreateOrUpdateAlertRule API.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ARMS alert rule ID.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Unique alert rule name.",
				Required:            true,
			},
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "Managed Service for Prometheus cluster ID.",
				Required:            true,
			},
			"promql": schema.StringAttribute{
				MarkdownDescription: "PromQL expression evaluated by the alert rule.",
				Required:            true,
			},
			"duration_minutes": schema.Int64Attribute{
				MarkdownDescription: "Number of minutes the expression must remain true before the alert fires.",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.Between(0, 1440),
				},
			},
			"level": schema.StringAttribute{
				MarkdownDescription: "ARMS severity level.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("P1", "P2", "P3", "P4", "Default"),
				},
			},
			"message": schema.StringAttribute{
				MarkdownDescription: "Alert message. Go template label references are supported by ARMS.",
				Required:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Desired alert rule status.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("RUNNING", "STOPPED"),
				},
			},
			"labels": schema.MapAttribute{
				MarkdownDescription: "Prometheus labels attached to alerts.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"annotations": schema.MapAttribute{
				MarkdownDescription: "Prometheus annotations attached to alerts.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"tags": schema.MapAttribute{
				MarkdownDescription: "Alibaba Cloud resource tags. Defaults to CreatedBy=Terraform.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				Default: mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{
					createdByTag: types.StringValue("Terraform"),
				})),
			},
			"no_data_revision": schema.Int64Attribute{
				MarkdownDescription: "Value used when metric data is missing: 0 fills zero, 1 fills one, and 2 fills null without firing.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(2),
				Validators: []validator.Int64{
					int64validator.OneOf(0, 1, 2),
				},
			},
		},
	}
}

func (r *prometheusAlertRuleResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}
	client, ok := request.ProviderData.(armsapi.AlertRuleService)
	if !ok {
		response.Diagnostics.Append(unexpectedProviderDataDiagnostic(request.ProviderData))
		return
	}
	r.client = client
}

func (r *prometheusAlertRuleResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var plan prometheusAlertRuleModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	rule := plan.toAPI(ctx, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}
	id, err := r.client.CreateOrUpdate(rule)
	if err != nil {
		response.Diagnostics.AddError("Unable to create Prometheus alert rule", err.Error())
		return
	}
	plan.ID = types.StringValue(strconv.FormatInt(id, 10))
	response.Diagnostics.Append(response.State.Set(ctx, &plan)...)
}

func (r *prometheusAlertRuleResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state prometheusAlertRuleModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		response.Diagnostics.AddError("Invalid Prometheus alert rule ID", err.Error())
		return
	}
	rule, err := r.client.Get(id)
	if err != nil {
		response.Diagnostics.AddError("Unable to read Prometheus alert rule", err.Error())
		return
	}
	if rule == nil {
		response.State.RemoveResource(ctx)
		return
	}

	state.fromAPI(ctx, rule, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}
	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func (r *prometheusAlertRuleResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan prometheusAlertRuleModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}
	id, err := strconv.ParseInt(plan.ID.ValueString(), 10, 64)
	if err != nil {
		response.Diagnostics.AddError("Invalid Prometheus alert rule ID", err.Error())
		return
	}

	rule := plan.toAPI(ctx, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}
	rule.ID = id
	if _, err := r.client.CreateOrUpdate(rule); err != nil {
		response.Diagnostics.AddError("Unable to update Prometheus alert rule", err.Error())
		return
	}
	response.Diagnostics.Append(response.State.Set(ctx, &plan)...)
}

func (r *prometheusAlertRuleResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state prometheusAlertRuleModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}
	id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		response.Diagnostics.AddError("Invalid Prometheus alert rule ID", err.Error())
		return
	}
	existing, err := r.client.Get(id)
	if err != nil {
		response.Diagnostics.AddError("Unable to check Prometheus alert rule before deletion", err.Error())
		return
	}
	if existing == nil {
		return
	}
	if err := r.client.Delete(id); err != nil {
		response.Diagnostics.AddError("Unable to delete Prometheus alert rule", err.Error())
	}
}

func (r *prometheusAlertRuleResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	if _, err := strconv.ParseInt(request.ID, 10, 64); err != nil {
		response.Diagnostics.AddError("Invalid import ID", "Expected a numeric ARMS alert rule ID.")
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), request, response)
}

func (model prometheusAlertRuleModel) toAPI(ctx context.Context, diagnostics *diag.Diagnostics) armsapi.AlertRule {
	labels := mapValue(ctx, model.Labels, diagnostics)
	annotations := mapValue(ctx, model.Annotations, diagnostics)
	tags := mapValue(ctx, model.Tags, diagnostics)
	return armsapi.AlertRule{
		Name:            model.Name.ValueString(),
		ClusterID:       model.ClusterID.ValueString(),
		PromQL:          model.PromQL.ValueString(),
		DurationMinutes: model.DurationMinutes.ValueInt64(),
		Level:           model.Level.ValueString(),
		Message:         model.Message.ValueString(),
		Status:          model.Status.ValueString(),
		Labels:          labels,
		Annotations:     annotations,
		Tags:            tags,
		NoDataRevision:  model.NoDataRevision.ValueInt64(),
	}
}

func (model *prometheusAlertRuleModel) fromAPI(ctx context.Context, rule *armsapi.AlertRule, diagnostics *diag.Diagnostics) {
	model.ID = types.StringValue(strconv.FormatInt(rule.ID, 10))
	model.Name = types.StringValue(rule.Name)
	model.ClusterID = types.StringValue(rule.ClusterID)
	model.PromQL = types.StringValue(rule.PromQL)
	model.DurationMinutes = types.Int64Value(rule.DurationMinutes)
	model.Level = types.StringValue(rule.Level)
	model.Message = types.StringValue(rule.Message)
	model.Status = types.StringValue(rule.Status)
	model.Labels = mapFromAPI(ctx, rule.Labels, diagnostics)
	model.Annotations = mapFromAPI(ctx, rule.Annotations, diagnostics)
	model.Tags = mapFromAPI(ctx, rule.Tags, diagnostics)
	// GetAlertRules does not expose DataConfig. Preserve the configured value
	// in state; imported resources receive the documented default on next plan.
}

func mapValue(ctx context.Context, value types.Map, diagnostics *diag.Diagnostics) map[string]string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	result := make(map[string]string, len(value.Elements()))
	diagnostics.Append(value.ElementsAs(ctx, &result, false)...)
	return result
}

func mapFromAPI(ctx context.Context, values map[string]string, diagnostics *diag.Diagnostics) types.Map {
	if len(values) == 0 {
		return types.MapNull(types.StringType)
	}
	elements := make(map[string]attr.Value, len(values))
	for key, value := range values {
		elements[key] = types.StringValue(value)
	}
	result, diags := types.MapValue(types.StringType, elements)
	diagnostics.Append(diags...)
	_ = ctx
	return result
}
