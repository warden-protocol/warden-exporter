package grpc

import (
	"errors"
	"fmt"

	"google.golang.org/grpc"

	"github.com/warden-protocol/warden-exporter/pkg/config"
)

var errEndpoint = errors.New("grpc error")

func endpointError(msg string) error {
	return fmt.Errorf("%w: %s", errEndpoint, msg)
}

type Client struct {
	cfg  config.Config
	conn *grpc.ClientConn
}

func NewClient(cfg config.Config) (Client, error) {
	client := Client{
		cfg: cfg,
	}

	conn, err := cfg.GRPCConn()
	if err != nil {
		return Client{}, endpointError(err.Error())
	}

	client.conn = conn

	return client, nil
}
