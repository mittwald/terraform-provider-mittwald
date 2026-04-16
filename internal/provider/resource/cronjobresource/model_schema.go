package cronjobresource

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/common"
)

var modelDestinationCommandSchema = schema.SingleNestedAttribute{
	Optional: true,
	Attributes: map[string]schema.Attribute{
		"interpreter": schema.StringAttribute{
			MarkdownDescription: "The interpreter to use for the command. Must be a valid path to an executable within the project environment (typically, `/bin/bash` or `/usr/bin/php` should work).",
			Required:            true,
		},
		"path": schema.StringAttribute{
			MarkdownDescription: "The path to the file to run. Must be a valid path to an executable file within the project environment.",
			Required:            true,
		},
		"parameters": schema.ListAttribute{
			MarkdownDescription: "A list of parameters to pass to the command. Each parameter must be a valid string.",
			Optional:            true,
			ElementType:         types.StringType,
		},
	},
}

var modelDestinationURLSchema = schema.StringAttribute{
	Description: "The URL that should be requested by the cron job",
	Optional:    true,
}

var modelDestinationSchema = schema.SingleNestedAttribute{
	Required:            true,
	MarkdownDescription: "Models the action to be executed by the cron job. Exactly one of `url`, `command`, or `container_command` must be set.",
	Validators: []validator.Object{
		&cronjobDestinationValidator{},
	},
	Attributes: map[string]schema.Attribute{
		"url":               modelDestinationURLSchema,
		"command":           modelDestinationCommandSchema,
		"container_command": modelDestinationContainerCommandSchema,
	},
}

var modelDestinationContainerCommandSchema = schema.ListAttribute{
	MarkdownDescription: "Command and arguments to execute in a container service. This must be used together with `container`.",
	Optional:            true,
	ElementType:         types.StringType,
}

var modelContainerSchema = schema.SingleNestedAttribute{
	Optional:            true,
	MarkdownDescription: "Container target for this cronjob. This must be used together with `destination.container_command`.",
	Attributes: map[string]schema.Attribute{
		"stack_id": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "The ID of the container stack.",
			Validators: []validator.String{
				&common.UUIDValidator{},
			},
		},
		"service_id": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "The identifier of the service in the stack.",
		},
	},
	PlanModifiers: []planmodifier.Object{
		objectplanmodifier.RequiresReplace(),
	},
}
