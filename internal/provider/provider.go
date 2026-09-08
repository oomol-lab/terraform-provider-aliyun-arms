package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/oomol-lab/terraform-provider-aliyun-arms/internal/armsapi"
)

var _ provider.Provider = &armsProvider{}

type armsProvider struct {
	version string
}

type providerModel struct {
	Region   types.String `tfsdk:"region"`
	Profile  types.String `tfsdk:"profile"`
	Endpoint types.String `tfsdk:"endpoint"`
}

// New returns a provider factory for Terraform's protocol server.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &armsProvider{version: version}
	}
}

func (p *armsProvider) Metadata(_ context.Context, _ provider.MetadataRequest, response *provider.MetadataResponse) {
	response.TypeName = "arms"
	response.Version = p.version
}

func (p *armsProvider) Schema(_ context.Context, _ provider.SchemaRequest, response *provider.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Manages Alibaba Cloud ARMS APIs that are not yet exposed by the official Alibaba Cloud provider.",
		Attributes: map[string]schema.Attribute{
			"region": schema.StringAttribute{
				MarkdownDescription: "Alibaba Cloud region in which ARMS resources are managed.",
				Required:            true,
			},
			"profile": schema.StringAttribute{
				MarkdownDescription: "Optional named profile from the aliyun CLI configuration. When omitted, the Alibaba Cloud default credential chain is used.",
				Optional:            true,
			},
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Optional ARMS API endpoint override, primarily for testing.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
	}
}

func (p *armsProvider) Configure(ctx context.Context, request provider.ConfigureRequest, response *provider.ConfigureResponse) {
	var config providerModel
	response.Diagnostics.Append(request.Config.Get(ctx, &config)...)
	if response.Diagnostics.HasError() {
		return
	}
	if config.Region.IsUnknown() || config.Profile.IsUnknown() || config.Endpoint.IsUnknown() {
		response.Diagnostics.AddError("Unknown provider configuration", "Provider attributes must be known before ARMS resources can be configured.")
		return
	}

	client, err := armsapi.NewClient(config.Region.ValueString(), config.Profile.ValueString(), config.Endpoint.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Unable to configure ARMS client", err.Error())
		return
	}
	response.ResourceData = client
}

func (p *armsProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewPrometheusAlertRuleResource,
	}
}

func (p *armsProvider) DataSources(context.Context) []func() datasource.DataSource {
	return nil
}

func unexpectedProviderDataDiagnostic(data any) diag.Diagnostic {
	return diag.NewErrorDiagnostic(
		"Unexpected provider data",
		fmt.Sprintf("Expected an ARMS API client, got %T. This is a provider bug.", data),
	)
}
