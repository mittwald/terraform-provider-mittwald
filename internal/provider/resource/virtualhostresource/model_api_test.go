package virtualhostresource_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/ingressv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/virtualhostresource"
	. "github.com/onsi/gomega"
)

func ingressWithTLS(t *testing.T, tlsJSON string) *ingressv2.Ingress {
	t.Helper()

	var tls ingressv2.IngressTLS
	if err := json.Unmarshal([]byte(tlsJSON), &tls); err != nil {
		t.Fatalf("failed to unmarshal TLS settings: %s", err)
	}

	return &ingressv2.Ingress{
		Id:        "ingress-id",
		ProjectId: "project-id",
		Hostname:  "test.example",
		Tls:       tls,
	}
}

func TestFromAPIModelMapsACME(t *testing.T) {
	cases := []struct {
		name     string
		tlsJSON  string
		expected bool
	}{
		{name: "acme enabled", tlsJSON: `{"acme":true,"isCreated":true}`, expected: true},
		{name: "acme disabled", tlsJSON: `{"acme":false,"isCreated":false}`, expected: false},
		{name: "manual certificate", tlsJSON: `{"certificateId":"cert-id"}`, expected: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)

			model := virtualhostresource.ResourceModel{}
			diags := model.FromAPIModel(context.Background(), ingressWithTLS(t, tc.tlsJSON))

			g.Expect(diags.HasError()).To(BeFalse())
			g.Expect(model.Acme).To(Equal(types.BoolValue(tc.expected)))
		})
	}
}

func TestToUpdateTLSRequest(t *testing.T) {
	g := NewWithT(t)

	model := virtualhostresource.ResourceModel{
		ID:   types.StringValue("ingress-id"),
		Acme: types.BoolValue(false),
	}

	req := model.ToUpdateTLSRequest()

	g.Expect(req.IngressID).To(Equal("ingress-id"))
	g.Expect(req.Body.AlternativeUpdateIngressTLSRequestBodyAlternative2).To(BeNil())
	g.Expect(req.Body.AlternativeUpdateIngressTLSRequestBodyAlternative1).NotTo(BeNil())
	g.Expect(req.Body.AlternativeUpdateIngressTLSRequestBodyAlternative1.Acme).To(BeFalse())

	body, err := json.Marshal(&req.Body)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(body).To(MatchJSON(`{"acme":false}`))
}
