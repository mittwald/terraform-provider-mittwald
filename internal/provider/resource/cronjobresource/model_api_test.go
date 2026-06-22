package cronjobresource_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/cronjobv2"
	cronjobresource "github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/cronjobresource"
	. "github.com/onsi/gomega"
)

func paramList(values ...string) types.List {
	elems := make([]attr.Value, 0, len(values))
	for _, v := range values {
		elems = append(elems, types.StringValue(v))
	}
	return types.ListValueMust(types.StringType, elems)
}

// TestParametersAsStrDoesNotAddQuotes is the regression test for
// https://github.com/mittwald/terraform-provider-mittwald/issues/354:
// a plain parameter such as "scheduler:run" must be serialized verbatim and
// must NOT be wrapped in extra quotes like '"scheduler:run"'.
func TestParametersAsStrDoesNotAddQuotes(t *testing.T) {
	g := NewWithT(t)

	cmd := cronjobresource.ResourceDestinationCommandModel{
		Interpreter: types.StringValue("/usr/bin/php"),
		Path:        types.StringValue("/html/console"),
		Parameters:  paramList("scheduler:run"),
	}

	apiModel := cmd.AsAPIModel()

	g.Expect(apiModel.Parameters).ToNot(BeNil())
	g.Expect(*apiModel.Parameters).To(Equal("scheduler:run"))
}

func TestParametersRoundTrip(t *testing.T) {
	ctx := context.Background()

	testCases := map[string][]string{
		"simple":             {"messenger:consume", "async", "-vvv"},
		"with colon":         {"scheduler:run"},
		"with leading dash":  {"--verbose", "-n"},
		"value with space":   {"--message", "hello world"},
		"value with quote":   {"it's", "fine"},
		"empty list":         {},
		"path-like argument": {"/var/www/html/bin/console", "cache:clear"},
	}

	for name, params := range testCases {
		t.Run(name, func(t *testing.T) {
			g := NewWithT(t)

			cmd := cronjobresource.ResourceDestinationCommandModel{
				Interpreter: types.StringValue("/usr/bin/php"),
				Path:        types.StringValue("/html/console"),
				Parameters:  paramList(params...),
			}

			// Serialize to the API representation (single string)...
			apiModel := cmd.AsAPIModel()

			// ...and parse it back into a fresh model.
			parsed := cronjobresource.ResourceDestinationCommandModel{}
			diags := parsed.FromAPIModel(ctx, &cronjobv2.CronjobCommand{
				Interpreter: "/usr/bin/php",
				Path:        "/html/console",
				Parameters:  apiModel.Parameters,
			})

			g.Expect(diags.HasError()).To(BeFalse(), "unexpected diagnostics: %v", diags)
			g.Expect(parsed.Parameters.Equal(paramList(params...))).To(
				BeTrue(),
				"round-trip mismatch: got %s, want %s", parsed.Parameters, paramList(params...),
			)
		})
	}
}

// TestParametersFromAPIModelNil ensures a missing parameters value maps to a
// null list rather than an empty one.
func TestParametersFromAPIModelNil(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	parsed := cronjobresource.ResourceDestinationCommandModel{}
	diags := parsed.FromAPIModel(ctx, &cronjobv2.CronjobCommand{
		Interpreter: "/usr/bin/php",
		Path:        "/html/console",
		Parameters:  nil,
	})

	g.Expect(diags.HasError()).To(BeFalse())
	g.Expect(parsed.Parameters.IsNull()).To(BeTrue())
}
