package testingutils

import (
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/onsi/gomega"
)

var _ gomega.OmegaMatcher = &HaveStringAttrMatcher{}

type HaveStringAttrMatcher struct {
	key   string
	value types.String
}

func HaveStringAttr(key string, value any) *HaveStringAttrMatcher {
	switch v := value.(type) {
	case string:
		return &HaveStringAttrMatcher{
			key:   key,
			value: types.StringValue(v),
		}
	case types.String:
		return &HaveStringAttrMatcher{
			key:   key,
			value: v,
		}
	default:
		panic(fmt.Sprintf("HaveStringAttr matcher expects a string or types.String, got %T", value))
	}
}

func (h HaveStringAttrMatcher) Match(actual interface{}) (success bool, err error) {
	if msg := h.FailureMessage(actual); msg != "" {
		return false, errors.New(msg)
	}

	return true, nil
}

func (h HaveStringAttrMatcher) FailureMessage(actual interface{}) (message string) {
	obj, ok := actual.(types.Object)
	if !ok {
		return fmt.Sprintf("HaveStringAttr matcher expects an Object, %T given", actual)
	}

	attr, ok := obj.Attributes()[h.key]
	if !ok {
		return fmt.Sprintf("attribute %s not found", h.key)
	}

	attrStr, ok := attr.(types.String)
	if !ok {
		return fmt.Sprintf("attribute %s is not a string", h.key)
	}

	if attrStr != h.value {
		return fmt.Sprintf("attribute %s has value %s, expected %s", h.key, attrStr, h.value)
	}

	return ""
}

func (h HaveStringAttrMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return "negated " + h.FailureMessage(actual)
}
