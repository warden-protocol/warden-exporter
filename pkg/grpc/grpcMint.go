package grpc

import (
	"context"

	mintv1beta1 "cosmossdk.io/api/cosmos/mint/v1beta1"
)

func (c Client) Inflation(ctx context.Context) ([]byte, error) {
	client := mintv1beta1.NewQueryClient(c.conn)

	req := &mintv1beta1.QueryInflationRequest{}

	resp, err := client.Inflation(ctx, req)
	if err != nil {
		return nil, endpointError(err.Error())
	}

	return resp.Inflation, nil
}

func (c Client) AnnualProvisions(ctx context.Context) ([]byte, error) {
	client := mintv1beta1.NewQueryClient(c.conn)

	req := &mintv1beta1.QueryAnnualProvisionsRequest{}

	resp, err := client.AnnualProvisions(ctx, req)
	if err != nil {
		return nil, endpointError(err.Error())
	}

	return resp.AnnualProvisions, nil
}
