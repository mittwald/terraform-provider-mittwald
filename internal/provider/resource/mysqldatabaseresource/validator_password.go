package mysqldatabaseresource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type mysqlPasswordValidator struct {
}

func (m *mysqlPasswordValidator) Description(_ context.Context) string {
	return "Asserts that the password is not set when using password_wo and that password_wo_version is set when using password_wo."
}

func (m *mysqlPasswordValidator) MarkdownDescription(_ context.Context) string {
	return "Asserts that the password is not set when using password_wo and that password_wo_version is set when using password_wo."
}

func (m *mysqlPasswordValidator) ValidateObject(_ context.Context, request validator.ObjectRequest, response *validator.ObjectResponse) {
	password, hasPassword := request.ConfigValue.Attributes()["password"]
	writeOnlyPassword, hasWriteOnlyPassword := request.ConfigValue.Attributes()["password_wo"]
	passwordVersion, hasPasswordVersion := request.ConfigValue.Attributes()["password_wo_version"]

	if hasPassword && !password.IsNull() && hasWriteOnlyPassword && !writeOnlyPassword.IsNull() {
		response.Diagnostics.AddAttributeWarning(
			request.Path.AtName("password_wo"), "duplicate password", "password and password_wo are mutually exclusive; will prefer password_wo",
		)
	}

	if hasWriteOnlyPassword && !writeOnlyPassword.IsNull() && (!hasPasswordVersion || passwordVersion.IsNull()) {
		response.Diagnostics.AddAttributeError(
			request.Path.AtName("password_wo_version"), "missing password_wo_version", "password_wo_version is required when using password_wo",
		)
	}
}
