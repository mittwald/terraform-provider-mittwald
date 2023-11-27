package cronjobresource

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var modelDestinationCommandSchema = schema.SingleNestedAttribute{
	Optional: true,
	Attributes: map[string]schema.Attribute{
		"interpreter": schema.StringAttribute{
			Required: true,
		},
		"path": schema.StringAttribute{
			Required: true,
		},
		"parameters": schema.ListAttribute{
			Optional:    true,
			ElementType: types.StringType,
		},
	},
}

var modelDestinationURLSchema = schema.StringAttribute{
	Optional: true,
}

var modelDestinationSchema = schema.SingleNestedAttribute{
	Required: true,
	Validators: []validator.Object{
		&cronjobDestinationValidator{},
	},
	Attributes: map[string]schema.Attribute{
		"url":     modelDestinationURLSchema,
		"command": modelDestinationCommandSchema,
	},
}
