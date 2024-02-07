package mittwaldv2

import "encoding/json"

// These types were apparently not generated by oapi-codegen, so we need to
// define them ourselves.

type DeMittwaldV1SignupRemovingLastOwnerNotAllowedErrorName string
type DeMittwaldV1SignupSecondFactorRequiredErrorName string

type DeMittwaldV1ConversationShareableAggregateReference0Aggregate string
type DeMittwaldV1ConversationShareableAggregateReference0Domain string
type DeMittwaldV1ConversationShareableAggregateReference1Aggregate string
type DeMittwaldV1ConversationShareableAggregateReference1Domain string
type DeMittwaldV1ConversationShareableAggregateReference2Aggregate string
type DeMittwaldV1ConversationShareableAggregateReference2Domain string
type DeMittwaldV1ConversationShareableAggregateReference3Aggregate string
type DeMittwaldV1ConversationShareableAggregateReference3Domain string

type AppPatchInstallationSystemSoftwareItem = struct {
	SystemSoftwareVersion *string                                    `json:"systemSoftwareVersion,omitempty"`
	UpdatePolicy          *DeMittwaldV1AppSystemSoftwareUpdatePolicy `json:"updatePolicy,omitempty"`
}
type AppPatchInstallationSystemSoftware = map[string]AppPatchInstallationSystemSoftwareItem

func (t *CronjobUpdateCronjobJSONBody_Destination) FromDeMittwaldV1CronjobCronjobUrl(v DeMittwaldV1CronjobCronjobUrl) error {
	b, err := json.Marshal(v)
	t.union = b
	return err
}

func (t *CronjobUpdateCronjobJSONBody_Destination) FromDeMittwaldV1CronjobCronjobCommand(v DeMittwaldV1CronjobCronjobCommand) error {
	b, err := json.Marshal(v)
	t.union = b
	return err
}
