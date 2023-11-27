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
	MarkdownDescription: "Models the action to be executed by the cron job. Exactly one of `url` or `command` must be set.",
	Validators: []validator.Object{
		&cronjobDestinationValidator{},
	},
	Attributes: map[string]schema.Attribute{
		"url":     modelDestinationURLSchema,
		"command": modelDestinationCommandSchema,
	},
}
