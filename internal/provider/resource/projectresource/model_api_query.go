package projectresource

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/articleclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/orderv2"
)

// Article attribute keys describing the resources an article provides. A
// project order takes either a named machine type or an explicit hardware
// spec, and the article says which of the two applies by way of these
// attributes.
const (
	attrMachineType  = "spec.machine_type"
	attrHardwareRAM  = "spec.hardware_spec.ram"
	attrHardwareVCPU = "spec.hardware_spec.vcpu"

	// Server articles carry their machine type in an unprefixed attribute;
	// accept it here too, so that either convention works.
	attrMachineTypeUnprefixed = "machine_type"
)

// articleSpec holds the resources an article provides, before they are mapped
// onto one of the two shapes the order API accepts.
type articleSpec struct {
	machineType *string
	ram         *float64
	vcpu        *float64
}

func (s *articleSpec) toOrderSpec() orderv2.ProjectHostingOrderSpec {
	if s.machineType != nil {
		return orderv2.ProjectHostingOrderSpec{
			AlternativeMachineTypeSpec: &orderv2.MachineTypeSpec{MachineType: s.machineType},
		}
	}

	return orderv2.ProjectHostingOrderSpec{
		AlternativeHardwareSpec: &orderv2.HardwareSpec{Ram: s.ram, Vcpu: s.vcpu},
	}
}

func (s *articleSpec) toTariffChangeSpec() orderv2.ProjectHostingTariffChangeSpec {
	if s.machineType != nil {
		return orderv2.ProjectHostingTariffChangeSpec{
			AlternativeMachineTypeSpec: &orderv2.MachineTypeSpec{MachineType: s.machineType},
		}
	}

	return orderv2.ProjectHostingTariffChangeSpec{
		AlternativeHardwareSpec: &orderv2.HardwareSpec{Ram: s.ram, Vcpu: s.vcpu},
	}
}

// QueryArticleSpec resolves the resources that the configured article provides.
func (m *ResourceModel) QueryArticleSpec(ctx context.Context, client mittwaldv2.Client) (*articleSpec, error) {
	articleID := m.ArticleID.ValueString()

	articleRequest := articleclientv2.GetArticleRequest{ArticleID: articleID}
	article, _, err := client.Article().GetArticle(ctx, articleRequest)
	if err != nil {
		return nil, fmt.Errorf("error while retrieving article %s: %w", articleID, err)
	}

	attributes := make(map[string]string, len(article.Attributes))
	for _, attr := range article.Attributes {
		if attr.Value != nil {
			attributes[attr.Key] = *attr.Value
		}
	}

	return articleSpecFromAttributes(articleID, attributes)
}

// articleSpecFromAttributes derives the resources an article provides from its
// attributes. An article either names a machine type, or describes the hardware
// it provides; the order API accepts both shapes.
func articleSpecFromAttributes(articleID string, attributes map[string]string) (*articleSpec, error) {
	for _, key := range []string{attrMachineType, attrMachineTypeUnprefixed} {
		if value, ok := attributes[key]; ok {
			return &articleSpec{machineType: &value}, nil
		}
	}

	ram, hasRAM := attributes[attrHardwareRAM]
	vcpu, hasVCPU := attributes[attrHardwareVCPU]

	if hasRAM || hasVCPU {
		spec := articleSpec{}
		var err error

		if hasRAM {
			if spec.ram, err = parseArticleQuantity(articleID, attrHardwareRAM, ram); err != nil {
				return nil, err
			}
		}

		if hasVCPU {
			if spec.vcpu, err = parseArticleQuantity(articleID, attrHardwareVCPU, vcpu); err != nil {
				return nil, err
			}
		}

		return &spec, nil
	}

	return nil, fmt.Errorf(
		"article %s does not describe the resources for a project: expected either a %q attribute, or %q and %q attributes, but the article only has these attributes: %s",
		articleID, attrMachineType, attrHardwareRAM, attrHardwareVCPU, strings.Join(sortedKeys(attributes), ", "),
	)
}

func parseArticleQuantity(articleID, key, value string) (*float64, error) {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil, fmt.Errorf("article %s has a %q attribute that is not a number: %q", articleID, key, value)
	}

	return &parsed, nil
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}

	slices.Sort(keys)

	return keys
}
