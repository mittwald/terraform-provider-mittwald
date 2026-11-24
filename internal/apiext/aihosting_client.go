package apiext

import (
	"context"
	"errors"
	"net/http"

	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/aihostingclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/contractclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/aihostingv2"
	"github.com/mittwald/api-client-go/pkg/httperr"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiutils"
)

type AIHostingClient interface {
	aihostingclientv2.Client

	// ListAllCustomerPlans returns all AI hosting plans of a customer, across
	// all pages.
	ListAllCustomerPlans(ctx context.Context, customerID string) ([]aihostingv2.CustomerPlan, error)

	// GetPlanContractID returns the ID of the contract that belongs to the
	// given AI hosting plan.
	GetPlanContractID(ctx context.Context, customerID, planID string) (string, error)

	// FindPlanIDByContractID returns the ID of the plan (out of plans) whose
	// contract has the given contract ID, or an empty string if there is none.
	FindPlanIDByContractID(ctx context.Context, customerID, contractID string, plans []aihostingv2.CustomerPlan) (string, error)
}

type aiHostingClient struct {
	aihostingclientv2.Client
	clientSet mittwaldv2.Client
}

func NewAIHostingClient(c mittwaldv2.Client) AIHostingClient {
	return &aiHostingClient{
		Client:    c.AIHosting(),
		clientSet: c,
	}
}

func (c *aiHostingClient) ListAllCustomerPlans(ctx context.Context, customerID string) ([]aihostingv2.CustomerPlan, error) {
	return apiutils.FetchAllPages(ctx, 100, func(ctx context.Context, limit, page int64) (*[]aihostingv2.CustomerPlan, *http.Response, error) {
		plans, httpResp, err := c.CustomerGetPlans(ctx, aihostingclientv2.CustomerGetPlansRequest{
			CustomerID: customerID,
			Limit:      &limit,
			Page:       &page,
		})
		if plans == nil {
			return nil, httpResp, err
		}
		return &plans.Plans, httpResp, err
	})
}

func (c *aiHostingClient) GetPlanContractID(ctx context.Context, customerID, planID string) (string, error) {
	contract, _, err := c.clientSet.Contract().GetDetailOfContractByAIHosting(ctx, contractclientv2.GetDetailOfContractByAIHostingRequest{
		CustomerID:  customerID,
		AIHostingID: planID,
	})
	if err != nil {
		return "", err
	}

	if contract == nil {
		return "", errors.New("empty response when reading AI hosting contract")
	}

	return contract.ContractId, nil
}

func (c *aiHostingClient) FindPlanIDByContractID(ctx context.Context, customerID, contractID string, plans []aihostingv2.CustomerPlan) (string, error) {
	for _, plan := range plans {
		planContractID, err := c.GetPlanContractID(ctx, customerID, plan.PlanId)

		// A plan whose contract can't be found (e.g. because it is still
		// being provisioned) can't be the one we're looking for. The API
		// reports some missing resources as 403 instead of 404.
		notFound := new(httperr.ErrNotFound)
		permissionDenied := new(httperr.ErrPermissionDenied)
		if errors.As(err, &notFound) || errors.As(err, &permissionDenied) {
			continue
		}

		if err != nil {
			return "", err
		}

		if planContractID == contractID {
			return plan.PlanId, nil
		}
	}

	return "", nil
}
