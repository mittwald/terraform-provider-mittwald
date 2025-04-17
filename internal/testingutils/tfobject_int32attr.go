package testingutils

import (
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/onsi/gomega"
)

var _ gomega.OmegaMatcher = &HaveInt32AttrMatcher{}

type HaveInt32AttrMatcher struct {
	key   string
	value types.Int32
}

func HaveInt32Attr(key string, value any) *HaveInt32AttrMatcher {
	switch v := value.(type) {
	case int32:
		return &HaveInt32AttrMatcher{
			key:   key,
			value: types.Int32Value(v),
		}
	case types.Int32:
		return &HaveInt32AttrMatcher{
			key:   key,
			value: v,
		}
	default:
		panic(fmt.Sprintf("HaveStringAttr matcher expects a string or types.String, got %T", value))
	}
}

func (h HaveInt32AttrMatcher) Match(actual interface{}) (success bool, err error) {
	if msg := h.FailureMessage(actual); msg != "" {
		return false, errors.New(msg)
	}

	return true, nil
}

func (h HaveInt32AttrMatcher) FailureMessage(actual interface{}) (message string) {
	obj, ok := actual.(types.Object)
	if !ok {
		return fmt.Sprintf("HaveStringAttr matcher expects an Object, %T given", actual)
	}

	attr, ok := obj.Attributes()[h.key]
	if !ok {
		return fmt.Sprintf("attribute %s not found", h.key)
	}

	attrStr, ok := attr.(types.Int32)
	if !ok {
		return fmt.Sprintf("attribute %s is not an int32", h.key)
	}

	if attrStr != h.value {
		return fmt.Sprintf("attribute %s has value %s, expected %s", h.key, attrStr, h.value)
	}

	return ""
}

func (h HaveInt32AttrMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return "negated " + h.FailureMessage(actual)
}
