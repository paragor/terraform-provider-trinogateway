// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/paragor/terraform-provider-trinogateway/internal/trinogatewayclient"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &BackendResource{}
var _ resource.ResourceWithImportState = &BackendResource{}

func NewBackendResource() resource.Resource {
	return &BackendResource{}
}

// BackendResource defines the resource implementation.
type BackendResource struct {
	client trinogatewayclient.TrinoGatewayClient
}

// BackendResourceModel describes the resource data model.
type BackendResourceModel struct {
	Id           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	ProxyTo      types.String `tfsdk:"proxy_to"`
	Active       types.Bool   `tfsdk:"active"`
	RoutingGroup types.String `tfsdk:"routing_group"`
	ExternalUrl  types.String `tfsdk:"external_url"`
}

func (r *BackendResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_backend"
}

func (r *BackendResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Backend configration",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Internal id for terraform provider",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of backend",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"proxy_to": schema.StringAttribute{
				MarkdownDescription: "Backend url",
				Required:            true,
			},
			"active": schema.BoolAttribute{
				MarkdownDescription: "Backend activation",
				Required:            true,
			},
			"routing_group": schema.StringAttribute{
				MarkdownDescription: "Routing group name",
				Required:            true,
			},
			"external_url": schema.StringAttribute{
				MarkdownDescription: "If the backend URL is different from the proxyTo URL (for example if they are internal vs. external hostnames)",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *BackendResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(trinogatewayclient.TrinoGatewayClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected trinogatewayclient.TrinoGatewayClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *BackendResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BackendResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	backend := &trinogatewayclient.Backend{
		Name:         data.Name.ValueString(),
		ProxyTo:      data.ProxyTo.ValueString(),
		RoutingGroup: data.RoutingGroup.ValueString(),
		Active:       data.Active.ValueBool(),
	}
	if data.ExternalUrl.IsNull() || data.ExternalUrl.IsUnknown() {
		data.ExternalUrl = types.StringValue(data.ProxyTo.ValueString())
	}
	backend.ExternalUrl = data.ExternalUrl.ValueString()

	err := r.client.AddOrUpdateBackend(ctx, backend)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to add backend, got error: %s", err),
		)
		return
	}

	data.Id = types.StringValue(data.Name.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BackendResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BackendResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	backends, err := r.client.GetAllBackends(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list backends, got error: %s", err))
		return
	}

	var foundBackend *trinogatewayclient.Backend
	for _, backend := range backends {
		if backend.Name == data.Name.ValueString() {
			foundBackend = backend
		}
	}

	if foundBackend == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	backendDomainToTfModel(foundBackend, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BackendResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BackendResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	backend := &trinogatewayclient.Backend{
		Name:         data.Name.ValueString(),
		ProxyTo:      data.ProxyTo.ValueString(),
		RoutingGroup: data.RoutingGroup.ValueString(),
		Active:       data.Active.ValueBool(),
	}
	if data.ExternalUrl.IsNull() || data.ExternalUrl.IsUnknown() {
		data.ExternalUrl = types.StringValue(data.ProxyTo.ValueString())
	}
	backend.ExternalUrl = data.ExternalUrl.ValueString()
	err := r.client.AddOrUpdateBackend(ctx, backend)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to update backend, got error: %s", err),
		)
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BackendResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BackendResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteBackend(ctx, data.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete backend, got error: %s", err))
		return
	}
}

func (r *BackendResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	backendName := req.ID

	backends, err := r.client.GetAllBackends(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list backends, got error: %s", err))
		return
	}

	var foundBackend *trinogatewayclient.Backend
	for _, backend := range backends {
		if backend.Name == backendName {
			foundBackend = backend
		}
	}
	if foundBackend == nil {
		resp.Diagnostics.AddError("Backend not found", "Backend not found")
	}
	var data BackendResourceModel
	backendDomainToTfModel(foundBackend, &data)

	resp.State.Set(ctx, &data)
}

func backendDomainToTfModel(domainmodel *trinogatewayclient.Backend, tfmodel *BackendResourceModel) {
	tfmodel.Active = types.BoolValue(domainmodel.Active)
	tfmodel.ProxyTo = types.StringValue(domainmodel.ProxyTo)
	tfmodel.Name = types.StringValue(domainmodel.Name)
	tfmodel.RoutingGroup = types.StringValue(domainmodel.RoutingGroup)
	tfmodel.ExternalUrl = types.StringValue(domainmodel.ExternalUrl)
}
