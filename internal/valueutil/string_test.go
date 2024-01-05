package valueutil

import (
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
	"testing"
)

func TestStringerOrNullAcceptsStringer(t *testing.T) {
	g := NewWithT(t)
	u := uuid.New()
	s := StringerOrNull(u)

	g.Expect(s.IsNull()).To(BeFalse())
	g.Expect(s.ValueString()).To(Equal(u.String()))
}

func TestStringerOrNullAcceptsNil(t *testing.T) {
	g := NewWithT(t)
	s := StringerOrNull(nil)

	g.Expect(s.IsNull()).To(BeTrue())
}

func TestStringerOrNullAcceptsNilInterface(t *testing.T) {
	g := NewWithT(t)
	u := (*uuid.UUID)(nil)
	s := StringerOrNull(u)

	g.Expect(s.IsNull()).To(BeTrue())
}
