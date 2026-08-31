package containerstackresource

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	. "github.com/onsi/gomega"
)

func TestWithExplicitNullFieldOnEmptyBody(t *testing.T) {
	g := NewWithT(t)

	req, err := http.NewRequest(http.MethodPatch, "https://example.invalid/", io.NopCloser(bytes.NewReader([]byte(`{}`))))
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(withExplicitNullField("updateSchedule")(req)).To(Succeed())

	body, err := io.ReadAll(req.Body)
	g.Expect(err).NotTo(HaveOccurred())

	var decoded map[string]json.RawMessage
	g.Expect(json.Unmarshal(body, &decoded)).To(Succeed())
	g.Expect(decoded).To(HaveKey("updateSchedule"))
	g.Expect(string(decoded["updateSchedule"])).To(Equal("null"))
	g.Expect(req.ContentLength).To(Equal(int64(len(body))))
}

func TestWithExplicitNullFieldPreservesOtherFields(t *testing.T) {
	g := NewWithT(t)

	original := `{"description":"my stack","services":{"web":{}}}`
	req, err := http.NewRequest(http.MethodPatch, "https://example.invalid/", io.NopCloser(bytes.NewReader([]byte(original))))
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(withExplicitNullField("updateSchedule")(req)).To(Succeed())

	body, err := io.ReadAll(req.Body)
	g.Expect(err).NotTo(HaveOccurred())

	var decoded map[string]json.RawMessage
	g.Expect(json.Unmarshal(body, &decoded)).To(Succeed())
	g.Expect(string(decoded["updateSchedule"])).To(Equal("null"))
	g.Expect(string(decoded["description"])).To(Equal(`"my stack"`))
	g.Expect(decoded).To(HaveKey("services"))
}
