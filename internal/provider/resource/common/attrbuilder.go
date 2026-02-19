package common

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type AttributeBuilder struct {
	resourceName string
}

func AttributeBuilderFor(name string) *AttributeBuilder {
	return &AttributeBuilder{
		resourceName: name,
	}
}

func (b *AttributeBuilder) Id() schema.Attribute {
	return schema.StringAttribute{
		Computed:            true,
		MarkdownDescription: fmt.Sprintf("The generated %s ID", b.resourceName),
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
}

func (b *AttributeBuilder) ProjectId() schema.Attribute {
	return schema.StringAttribute{
		MarkdownDescription: fmt.Sprintf("The ID of the project the %s belongs to. Must be a full UUID (not a short ID like p-XXXXXX).", b.resourceName),
		Required:            true,
		Validators: []validator.String{
			&UUIDValidator{},
		},
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
	}
}

func (b *AttributeBuilder) AppId() schema.Attribute {
	return schema.StringAttribute{
		MarkdownDescription: fmt.Sprintf("The ID of the app the %s belongs to. Must be a full UUID (not a short ID like a-XXXXXX).", b.resourceName),
		Required:            true,
		Validators: []validator.String{
			&UUIDValidator{},
		},
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
	}
}

func (b *AttributeBuilder) ServerId() schema.Attribute {
	return schema.StringAttribute{
		MarkdownDescription: fmt.Sprintf("The ID of the server the %s belongs to. Must be a full UUID (not a short ID like s-XXXXXX).", b.resourceName),
		Required:            true,
		Validators: []validator.String{
			&UUIDValidator{},
		},
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
	}
}

func (b *AttributeBuilder) ContainerId() schema.Attribute {
	return schema.StringAttribute{
		MarkdownDescription: fmt.Sprintf("The ID of the container the %s belongs to. Must be a full UUID (not a short ID like c-XXXXXX).", b.resourceName),
		Required:            true,
		Validators: []validator.String{
			&UUIDValidator{},
		},
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
	}
}

func (b *AttributeBuilder) Description() schema.Attribute {
	return schema.StringAttribute{
		Required:            true,
		MarkdownDescription: fmt.Sprintf("Description for your %s", b.resourceName),
	}
}
