package config

import (
	"crypto/tls"
	"errors"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var errConfig = errors.New("config error")

func configError(msg string) error {
	return fmt.Errorf("%w: %s", errConfig, msg)
}

type Config struct {
	Addr             string `env:"GRPC_ADDR" envDefault:"grpc.buenavista.wardenprotocol.org:443"`
	TLS              bool   `env:"GRPC_TLS_ENABLED" envDefault:"true"`
	Timeout          int    `env:"GRPC_TIMEOUT_SECONDS" envDefault:"5"`
	ChainID          string `env:"CHAIN_ID" envDefault:"buenavista-1"`
	WardenMetrics    bool   `env:"WARDEN_METRICS" envDefault:"true"`
	ValidatorMetrics bool   `env:"VALIDATOR_METRICS" envDefault:"true"`
}

func (c Config) GRPCConn() (*grpc.ClientConn, error) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	transportCreds := grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))

	if !c.TLS {
		transportCreds = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	conn, err := grpc.Dial(
		c.Addr,
		transportCreds,
		grpc.WithDefaultCallOptions(grpc.ForceCodec(codec.NewProtoCodec(nil).GRPCCodec())),
	)
	if err != nil {
		return nil, configError(err.Error())
	}

	return conn, nil
}
